package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
)

// AppUpdateHandler handles app_update jobs
type AppUpdateHandler struct {
	db            *db.DB
	dockerManager *docker.Manager
	logger        *slog.Logger
}

// NewAppUpdateHandler creates a new app update handler
func NewAppUpdateHandler(
	database *db.DB,
	dockerMgr *docker.Manager,
	logger *slog.Logger,
) *AppUpdateHandler {
	return &AppUpdateHandler{
		db:            database,
		dockerManager: dockerMgr,
		logger:        logger,
	}
}

// Handle processes an app_update job
func (h *AppUpdateHandler) Handle(ctx context.Context, job *db.Job, progress *ProgressTracker) error {
	// Get app details
	app, err := h.db.GetApp(job.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	progress.Update(5, "Preparing to update...")

	// Create progress callback that forwards to our tracker
	progressCallback := func(pct int, msg string) {
		// Docker progress is 0-100, map it to our overall progress (5-95)
		overallProgress := 5 + (pct * 90 / 100)
		progress.Update(overallProgress, msg)
	}

	// Pull latest images and rebuild (this is the slow operation)
	if err := h.dockerManager.UpdateAppWithProgress(ctx, app.Name, progressCallback); err != nil {
		return fmt.Errorf("failed to update app: %w", err)
	}

	progress.Update(97, "Updating app status...")

	// Update app status in database
	app.Status = constants.AppStatusRunning
	if err := h.db.UpdateApp(app); err != nil {
		h.logger.Warn("failed to update app status", "app_id", app.ID, "error", err)
	}

	progress.Update(100, "App updated successfully")
	return nil
}
