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

	var dockerLog func(string)
	if jl := progress.JobLog(); jl != nil {
		dockerLog = jl.WriteLine
	}

	// Create progress callback that forwards to our tracker
	progressCallback := func(pct int, msg string) {
		// Docker progress is 0-100, map it to our overall progress (5-95)
		overallProgress := 5 + (pct * 90 / 100)
		progress.Update(overallProgress, msg)
	}

	// Pull latest images and rebuild (this is the slow operation)
	if err := h.dockerManager.UpdateAppWithProgress(ctx, app.Name, progressCallback, dockerLog); err != nil {
		// Check if containers are still running before marking as error
		// If update fails but old containers are still up, keep the running status
		status, statusErr := h.dockerManager.GetAppStatus(app.Name)
		if statusErr != nil {
			h.logger.Warn("failed to check app status after update failure", "app", app.Name, "error", statusErr)
			// Can't determine status, mark as error
			app.Status = constants.AppStatusError
		} else if status == constants.AppStatusRunning {
			// Old containers are still running, keep running status but set error message
			h.logger.Info("update failed but containers still running", "app", app.Name, "status", status)
			app.Status = constants.AppStatusRunning
		} else {
			// No running containers, mark as error
			app.Status = constants.AppStatusError
		}
		
		errorMsg := fmt.Sprintf("Update failed: %s. Previous version may still be running.", err.Error())
		app.ErrorMessage = &errorMsg
		if updateErr := h.db.UpdateApp(app); updateErr != nil {
			h.logger.Warn("failed to update app state after update failure", "app_id", app.ID, "error", updateErr)
		}
		// Log full error for debugging
		h.logger.ErrorContext(ctx, "docker update failed", "app", app.Name, "error", err, "final_status", app.Status)
		return fmt.Errorf("Docker update failed: %w", err)
	}

	progress.Update(97, "Updating app status...")

	// Update app status in database
	app.Status = constants.AppStatusRunning
	app.ErrorMessage = nil // Clear any previous error
	if err := h.db.UpdateApp(app); err != nil {
		h.logger.Warn("failed to update app status", "app_id", app.ID, "error", err)
	}

	progress.Update(100, "App updated successfully")
	return nil
}
