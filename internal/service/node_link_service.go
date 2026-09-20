package service

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"strings"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

type nodeLinkService struct {
	database *db.DB
	config   *config.Config
	logger   *slog.Logger
}

// NewNodeLinkService creates the service for nodes that dial out to this primary
func NewNodeLinkService(database *db.DB, cfg *config.Config, logger *slog.Logger) domain.NodeLinkService {
	return &nodeLinkService{database: database, config: cfg, logger: logger}
}

func linkEndpoint(nodeID string) string {
	return constants.LinkEndpointScheme + "://" + nodeID
}

func isLinkNode(n *db.Node) bool {
	return n != nil && strings.HasPrefix(n.APIEndpoint, constants.LinkEndpointScheme+"://")
}

func constantTimeEqual(a, b string) bool {
	return a != "" && b != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// Admit authenticates a node that dialled out and records it. A known node proves itself with its key
// (constant time); a new node needs a single-use join token or the shared registration token. A node
// that used to be reached directly and presents its key here switches to the link, keeping its identity.
func (s *nodeLinkService) Admit(ctx context.Context, req domain.LinkRequest) (*domain.LinkAdmission, error) {
	existing, err := s.database.GetNode(req.ID)
	if err == nil && existing != nil {
		if !constantTimeEqual(existing.APIKey, req.Key) {
			s.logger.WarnContext(ctx, "node link refused: key does not match the one on record", "node_id", req.ID)
			return nil, domain.ErrLinkUnauthorized
		}
		admission := &domain.LinkAdmission{Node: existing}
		if endpoint := linkEndpoint(req.ID); existing.APIEndpoint != endpoint {
			existing.APIEndpoint = endpoint
			if err := s.database.UpdateNode(existing); err != nil {
				return nil, domain.WrapDatabaseOperation("update node", err)
			}
			admission.Switched = true
			s.logger.InfoContext(ctx, "node switched to the outbound link", "node_id", req.ID)
		}
		return admission, nil
	}

	authenticated := req.Token != "" && constantTimeEqual(req.Token, s.config.Node.RegistrationToken)
	if !authenticated && req.Token != "" {
		ok, err := s.database.ConsumeJoinToken(req.Token, req.ID)
		if err != nil {
			return nil, domain.WrapDatabaseOperation("check join token", err)
		}
		authenticated = ok
	}
	if !authenticated {
		s.logger.WarnContext(ctx, "node link refused: unknown node without a valid token", "node_id", req.ID)
		return nil, domain.ErrLinkUnauthorized
	}
	if other, err := s.database.GetNodeByName(req.Name); err == nil && other != nil {
		return nil, domain.ErrLinkNameTaken
	}

	n := db.NewNodeWithID(req.ID, req.Name, linkEndpoint(req.ID), req.Key, false)
	n.Status = constants.NodeStatusOffline // becomes online when the link comes up and answers
	if err := s.database.CreateNode(n); err != nil {
		return nil, domain.WrapDatabaseOperation("register node", err)
	}
	s.logger.InfoContext(ctx, "node registered over the outbound link", "node_id", req.ID, "name", req.Name)
	return &domain.LinkAdmission{Node: n, Created: true}, nil
}

func (s *nodeLinkService) LinkedNode(_ context.Context, nodeID string) (*db.Node, error) {
	n, err := s.database.GetNode(nodeID)
	if err != nil || !isLinkNode(n) {
		return nil, nil
	}
	return n, nil
}

func (s *nodeLinkService) MarkOffline(ctx context.Context, nodeID string) error {
	n, err := s.LinkedNode(ctx, nodeID)
	if err != nil || n == nil {
		return err
	}
	n.Status = constants.NodeStatusOffline
	if err := s.database.UpdateNode(n); err != nil {
		return fmt.Errorf("mark node offline: %w", err)
	}
	return nil
}
