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

// AppScheduledStopHandler handles scheduled app stop jobs
type AppScheduledStopHandler struct {
	database      *db.DB
	dockerManager *docker.Manager
	logger        *slog.Logger
}

// NewAppScheduledStopHandler creates a new AppScheduledStopHandler
func NewAppScheduledStopHandler(database *db.DB, dockerManager *docker.Manager, logger *slog.Logger) JobHandler {
	return &AppScheduledStopHandler{
		database:      database,
		dockerManager: dockerManager,
		logger:        logger,
	}
}

// Handle implements the JobHandler interface for scheduled app stop
func (h *AppScheduledStopHandler) Handle(ctx context.Context, job *db.Job, progress *ProgressTracker) error {
	// Parse payload
	var payload struct {
		Name string `json:"name"`
	}
	if job.Payload != nil {
		if err := json.Unmarshal([]byte(*job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to parse app_scheduled_stop payload: %w", err)
		}
	}

	// Get app from database
	app, err := h.database.GetApp(job.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	progress.Update(10, "Stopping application...")

	// Check if app is already stopped
	if app.Status == constants.AppStatusStopped {
		progress.Update(100, "Application already stopped")
		h.logger.Info("App already stopped, skipping stop", "app_id", job.AppID, "app_name", app.Name)
		return nil
	}

	// Stop the app
	if err := h.dockerManager.StopApp(app.Name); err != nil {
		// Update app to error state
		app.Status = constants.AppStatusError
		errorMsg := err.Error()
		app.ErrorMessage = &errorMsg
		
		if updateErr := h.database.UpdateApp(app); updateErr != nil {
			h.logger.Warn("Failed to update app to error state", "app_id", app.ID, "error", updateErr)
		}
		
		return fmt.Errorf("failed to stop app: %w", err)
	}

	progress.Update(60, "Application stopped")

	// Update app status
	app.Status = constants.AppStatusStopped
	app.ErrorMessage = nil

	if err := h.database.UpdateApp(app); err != nil {
		h.logger.Warn("Failed to update app status", "app_id", app.ID, "error", err)
		return fmt.Errorf("failed to update app status: %w", err)
	}

	progress.Update(100, "Application stopped successfully")

	h.logger.Info("Scheduled stop completed successfully", 
		"app_id", app.ID, 
		"app_name", app.Name,
		"job_id", job.ID)

	return nil
}
