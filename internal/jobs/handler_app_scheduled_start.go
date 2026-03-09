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

// AppScheduledStartHandler handles scheduled app start jobs
type AppScheduledStartHandler struct {
	database      *db.DB
	dockerManager *docker.Manager
	logger        *slog.Logger
}

// NewAppScheduledStartHandler creates a new AppScheduledStartHandler
func NewAppScheduledStartHandler(database *db.DB, dockerManager *docker.Manager, logger *slog.Logger) JobHandler {
	return &AppScheduledStartHandler{
		database:      database,
		dockerManager: dockerManager,
		logger:        logger,
	}
}

// Handle implements the JobHandler interface for scheduled app start
func (h *AppScheduledStartHandler) Handle(ctx context.Context, job *db.Job, progress *ProgressTracker) error {
	// Parse payload
	var payload struct {
		Name string `json:"name"`
	}
	if job.Payload != nil {
		if err := json.Unmarshal([]byte(*job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to parse app_scheduled_start payload: %w", err)
		}
	}

	// Get app from database
	app, err := h.database.GetApp(job.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	progress.Update(10, "Starting application...")

	// Check if app is already running
	if app.Status == constants.AppStatusRunning {
		progress.Update(100, "Application already running")
		h.logger.Info("App already running, skipping start", "app_id", job.AppID, "app_name", app.Name)
		return nil
	}

	// Start the app
	if err := h.dockerManager.StartApp(app.Name); err != nil {
		// Update app to error state
		app.Status = constants.AppStatusError
		errorMsg := err.Error()
		app.ErrorMessage = &errorMsg
		
		if updateErr := h.database.UpdateApp(app); updateErr != nil {
			h.logger.Warn("Failed to update app to error state", "app_id", app.ID, "error", updateErr)
		}
		
		return fmt.Errorf("failed to start app: %w", err)
	}

	progress.Update(60, "Application started")

	// Update app status
	app.Status = constants.AppStatusRunning
	app.ErrorMessage = nil

	if err := h.database.UpdateApp(app); err != nil {
		h.logger.Warn("Failed to update app status", "app_id", app.ID, "error", err)
		return fmt.Errorf("failed to update app status: %w", err)
	}

	progress.Update(100, "Application started successfully")

	h.logger.Info("Scheduled start completed successfully", 
		"app_id", app.ID, 
		"app_name", app.Name,
		"job_id", job.ID)

	return nil
}
