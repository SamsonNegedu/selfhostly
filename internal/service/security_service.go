package service

import (
	"context"
	"time"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

type securityService struct {
	database *db.DB
}

// NewSecurityService creates the service for join tokens, session revocation and the audit log
func NewSecurityService(database *db.DB) domain.SecurityService {
	return &securityService{database: database}
}

func (s *securityService) CreateJoinToken(_ context.Context) (string, time.Time, error) {
	tok, expires, err := s.database.CreateJoinToken(constants.JoinTokenTTL)
	if err != nil {
		return "", time.Time{}, domain.WrapDatabaseOperation("create join token", err)
	}
	return tok, expires, nil
}

func (s *securityService) RevokeSessions(_ context.Context) (time.Time, error) {
	at, err := s.database.RevokeSessions()
	if err != nil {
		return time.Time{}, domain.WrapDatabaseOperation("revoke sessions", err)
	}
	return at, nil
}

func (s *securityService) ListAudit(_ context.Context, limit int) ([]db.AuditEntry, error) {
	entries, err := s.database.ListAudit(limit)
	if err != nil {
		return nil, domain.WrapDatabaseOperation("read audit log", err)
	}
	return entries, nil
}

func (s *securityService) RecordAudit(_ context.Context, entry db.AuditEntry) error {
	return s.database.InsertAudit(entry)
}

func (s *securityService) AuditTargetName(_ context.Context, targetType, id string) string {
	switch targetType {
	case domain.AuditTargetApp:
		if app, err := s.database.GetApp(id); err == nil && app != nil {
			return app.Name
		}
	case domain.AuditTargetNode:
		if node, err := s.database.GetNode(id); err == nil && node != nil {
			return node.Name
		}
	}
	return ""
}
