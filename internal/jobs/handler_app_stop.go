package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
)

// AppStopHandler handles app stop jobs
type AppStopHandler struct {
	database      *db.DB
	dockerManager *docker.Manager
	logger        *slog.Logger
}

// NewAppStopHandler creates a new AppStopHandler
func NewAppStopHandler(database *db.DB, dockerManager *docker.Manager, logger *slog.Logger) JobHandler {
	return &AppStopHandler{
		database:      database,
		dockerManager: dockerManager,
		logger:        logger,
	}
}

// Handle implements the JobHandler interface for app stop
func (h *AppStopHandler) Handle(ctx context.Context, job *db.Job, progress *ProgressTracker) error {
	var payload struct {
		Name string `json:"name"`
	}
	if job.Payload != nil {
		if err := json.Unmarshal([]byte(*job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to parse app_stop payload: %w", err)
		}
	}

	app, err := h.database.GetApp(job.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	progress.Update(10, "Stopping application...")

	if app.Status == constants.AppStatusStopped {
		progress.Update(100, "Application already stopped")
		h.logger.Info("App already stopped, skipping stop", "app_id", job.AppID, "app_name", app.Name)
		return nil
	}

	if err := h.dockerManager.StopApp(app.Name); err != nil {
		// Check actual container status to determine if stop partially succeeded
		status, statusErr := h.dockerManager.GetAppStatus(app.Name)
		if statusErr != nil {
			h.logger.Warn("failed to check app status after stop failure", "app", app.Name, "error", statusErr)
			// Can't determine status, mark as error
			app.Status = constants.AppStatusError
		} else if status == constants.AppStatusStopped {
			// Containers are stopped despite error, mark as stopped
			h.logger.Info("stop reported error but containers are stopped", "app", app.Name)
			app.Status = constants.AppStatusStopped
		} else if status == constants.AppStatusRunning {
			// Containers still running, keep running status
			h.logger.Info("stop failed and containers still running", "app", app.Name)
			app.Status = constants.AppStatusRunning
		} else {
			// Unknown state, mark as error
			app.Status = constants.AppStatusError
		}

		errorMsg := fmt.Sprintf("Stop failed: %s", err.Error())
		app.ErrorMessage = &errorMsg

		if updateErr := h.database.UpdateApp(app); updateErr != nil {
			h.logger.Warn("Failed to update app state after stop failure", "app_id", app.ID, "error", updateErr)
		}

		// Log full error for debugging
		h.logger.ErrorContext(ctx, "docker stop failed", "app", app.Name, "error", err, "final_status", app.Status)
		return fmt.Errorf("Docker stop failed: %w", err)
	}

	progress.Update(60, "Application stopped")

	app.Status = constants.AppStatusStopped
	app.ErrorMessage = nil

	if err := h.database.UpdateApp(app); err != nil {
		h.logger.Warn("Failed to update app status", "app_id", app.ID, "error", err)
		return fmt.Errorf("failed to update app status: %w", err)
	}

	progress.Update(100, "Application stopped successfully")

	h.logger.Info("Stop completed successfully",
		"app_id", app.ID,
		"app_name", app.Name,
		"job_id", job.ID)

	return nil
}
