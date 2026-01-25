package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
)

// composeService implements the ComposeService interface
type composeService struct {
	database      *db.DB
	dockerManager *docker.Manager
	logger        *slog.Logger
}

// NewComposeService creates a new compose service
func NewComposeService(
	database *db.DB,
	dockerManager *docker.Manager,
	logger *slog.Logger,
) domain.ComposeService {
	return &composeService{
		database:      database,
		dockerManager: dockerManager,
		logger:        logger,
	}
}

// GetVersions retrieves all compose versions for an app
func (s *composeService) GetVersions(ctx context.Context, appID string) ([]*db.ComposeVersion, error) {
	s.logger.DebugContext(ctx, "getting compose versions", "appID", appID)

	// Verify app exists
	_, err := s.database.GetApp(appID)
	if err != nil {
		s.logger.DebugContext(ctx, "app not found", "appID", appID)
		return nil, domain.WrapAppNotFound(appID, err)
	}

	versions, err := s.database.GetComposeVersionsByAppID(appID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to retrieve compose versions", "appID", appID, "error", err)
		return nil, domain.WrapDatabaseOperation("get compose versions", err)
	}

	// Return empty array instead of null if no versions
	if versions == nil {
		versions = []*db.ComposeVersion{}
	}

	return versions, nil
}

// GetVersion retrieves a specific compose version
func (s *composeService) GetVersion(ctx context.Context, appID string, version int) (*db.ComposeVersion, error) {
	s.logger.DebugContext(ctx, "getting compose version", "appID", appID, "version", version)

	// Verify app exists
	_, err := s.database.GetApp(appID)
	if err != nil {
		s.logger.DebugContext(ctx, "app not found", "appID", appID)
		return nil, domain.WrapAppNotFound(appID, err)
	}

	composeVersion, err := s.database.GetComposeVersion(appID, version)
	if err != nil {
		s.logger.DebugContext(ctx, "compose version not found", "appID", appID, "version", version)
		return nil, domain.ErrComposeVersionNotFound
	}

	return composeVersion, nil
}

// RollbackToVersion rolls back to a specific compose version
func (s *composeService) RollbackToVersion(ctx context.Context, appID string, version int, reason *string, changedBy *string) (*db.ComposeVersion, error) {
	s.logger.InfoContext(ctx, "rolling back to version", "appID", appID, "version", version)

	// Get the app
	app, err := s.database.GetApp(appID)
	if err != nil {
		s.logger.DebugContext(ctx, "app not found for rollback", "appID", appID)
		return nil, domain.WrapAppNotFound(appID, err)
	}

	// Get the target version
	targetComposeVersion, err := s.database.GetComposeVersion(appID, version)
	if err != nil {
		s.logger.DebugContext(ctx, "target compose version not found", "appID", appID, "version", version)
		return nil, domain.ErrComposeVersionNotFound
	}

	// Get current version number
	currentVersionNumber, err := s.database.GetLatestVersionNumber(appID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get current version number", "appID", appID, "error", err)
		return nil, domain.WrapDatabaseOperation("get latest version", err)
	}

	// Create a new version with the rolled-back content
	newVersionNumber := currentVersionNumber + 1
	rolledBackFrom := currentVersionNumber
	changeReason := reason
	if changeReason == nil {
		r := fmt.Sprintf("Rolled back to version %d", version)
		changeReason = &r
	}

	newVersion := db.NewComposeVersion(appID, newVersionNumber, targetComposeVersion.ComposeContent, changeReason, changedBy)
	newVersion.RolledBackFrom = &rolledBackFrom

	// Mark all versions as not current
	if err := s.database.MarkAllVersionsAsNotCurrent(appID); err != nil {
		s.logger.ErrorContext(ctx, "failed to mark versions as not current", "appID", appID, "error", err)
		return nil, domain.WrapDatabaseOperation("mark versions as not current", err)
	}

	// Create the new version
	if err := s.database.CreateComposeVersion(newVersion); err != nil {
		s.logger.ErrorContext(ctx, "failed to create rolled-back version", "appID", appID, "error", err)
		return nil, domain.WrapDatabaseOperation("create compose version", err)
	}

	// Update the app with the rolled-back content
	app.ComposeContent = targetComposeVersion.ComposeContent
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		s.logger.ErrorContext(ctx, "failed to update app with rolled-back content", "appID", appID, "error", err)
		return nil, domain.WrapDatabaseOperation("update app", err)
	}

	// Update compose file on disk
	if err := s.dockerManager.WriteComposeFile(app.Name, app.ComposeContent); err != nil {
		s.logger.ErrorContext(ctx, "failed to update compose file on disk", "app", app.Name, "error", err)
		return nil, domain.WrapContainerOperationFailed("write compose file", err)
	}

	s.logger.InfoContext(ctx, "rolled back compose version", "app", app.Name, "appID", appID, "fromVersion", version, "toVersion", newVersionNumber)
	return newVersion, nil
}
