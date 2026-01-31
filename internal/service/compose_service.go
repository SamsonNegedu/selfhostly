package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/node"
	"github.com/selfhostly/internal/routing"
)

// composeService implements the ComposeService interface
type composeService struct {
	database      *db.DB
	dockerManager *docker.Manager
	router        *routing.NodeRouter
	nodeClient    *node.Client
	logger        *slog.Logger
}

// NewComposeService creates a new compose service
func NewComposeService(
	database *db.DB,
	dockerManager *docker.Manager,
	router *routing.NodeRouter,
	nodeClient *node.Client,
	logger *slog.Logger,
) domain.ComposeService {
	return &composeService{
		database:      database,
		dockerManager: dockerManager,
		router:        router,
		nodeClient:    nodeClient,
		logger:        logger,
	}
}

// GetVersions retrieves all compose versions for an app (local only)
func (s *composeService) GetVersions(ctx context.Context, appID string, nodeID string) ([]*db.ComposeVersion, error) {
	s.logger.DebugContext(ctx, "getting compose versions", "appID", appID, "nodeID", nodeID)
	if _, err := s.database.GetApp(appID); err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}
	versions, err := s.database.GetComposeVersionsByAppID(appID)
	if err != nil {
		return nil, domain.WrapDatabaseOperation("get compose versions", err)
	}
	if versions == nil {
		versions = []*db.ComposeVersion{}
	}
	return versions, nil
}

// GetVersion retrieves a specific compose version (local only)
func (s *composeService) GetVersion(ctx context.Context, appID string, version int, nodeID string) (*db.ComposeVersion, error) {
	s.logger.DebugContext(ctx, "getting compose version", "appID", appID, "version", version, "nodeID", nodeID)
	if _, err := s.database.GetApp(appID); err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}
	composeVersion, err := s.database.GetComposeVersion(appID, version)
	if err != nil {
		return nil, domain.ErrComposeVersionNotFound
	}
	return composeVersion, nil
}

// RollbackToVersion rolls back to a specific compose version (local only)
func (s *composeService) RollbackToVersion(ctx context.Context, appID string, version int, nodeID string, reason *string, changedBy *string) (*db.ComposeVersion, error) {
	s.logger.InfoContext(ctx, "rolling back to version", "appID", appID, "version", version, "nodeID", nodeID)
	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}
	targetComposeVersion, err := s.database.GetComposeVersion(appID, version)
	if err != nil {
		return nil, domain.ErrComposeVersionNotFound
	}
	currentVersionNumber, err := s.database.GetLatestVersionNumber(appID)
	if err != nil {
		return nil, domain.WrapDatabaseOperation("get latest version", err)
	}
	newVersionNumber := currentVersionNumber + 1
	rolledBackFrom := currentVersionNumber
	changeReason := reason
	if changeReason == nil {
		r := fmt.Sprintf("Rolled back to version %d", version)
		changeReason = &r
	}
	newVersion := db.NewComposeVersion(appID, newVersionNumber, targetComposeVersion.ComposeContent, changeReason, changedBy)
	newVersion.RolledBackFrom = &rolledBackFrom
	if err := s.database.MarkAllVersionsAsNotCurrent(appID); err != nil {
		return nil, domain.WrapDatabaseOperation("mark versions as not current", err)
	}
	if err := s.database.CreateComposeVersion(newVersion); err != nil {
		return nil, domain.WrapDatabaseOperation("create compose version", err)
	}
	app.ComposeContent = targetComposeVersion.ComposeContent
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		return nil, domain.WrapDatabaseOperation("update app", err)
	}
	if err := s.dockerManager.WriteComposeFile(app.Name, app.ComposeContent); err != nil {
		return nil, domain.WrapContainerOperationFailed("write compose file", err)
	}
	s.logger.InfoContext(ctx, "rolled back compose version", "app", app.Name, "appID", appID, "fromVersion", version, "toVersion", newVersionNumber)
	return newVersion, nil
}
