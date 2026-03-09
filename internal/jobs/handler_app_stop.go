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
		app.Status = constants.AppStatusError
		errorMsg := err.Error()
		app.ErrorMessage = &errorMsg

		if updateErr := h.database.UpdateApp(app); updateErr != nil {
			h.logger.Warn("Failed to update app to error state", "app_id", app.ID, "error", updateErr)
		}

		return fmt.Errorf("failed to stop app: %w", err)
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
