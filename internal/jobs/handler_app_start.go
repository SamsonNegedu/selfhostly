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

// AppStartHandler handles app start jobs
type AppStartHandler struct {
	database      *db.DB
	dockerManager *docker.Manager
	logger        *slog.Logger
}

// NewAppStartHandler creates a new AppStartHandler
func NewAppStartHandler(database *db.DB, dockerManager *docker.Manager, logger *slog.Logger) JobHandler {
	return &AppStartHandler{
		database:      database,
		dockerManager: dockerManager,
		logger:        logger,
	}
}

// Handle implements the JobHandler interface for app start
func (h *AppStartHandler) Handle(ctx context.Context, job *db.Job, progress *ProgressTracker) error {
	var payload struct {
		Name string `json:"name"`
	}
	if job.Payload != nil {
		if err := json.Unmarshal([]byte(*job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to parse app_start payload: %w", err)
		}
	}

	app, err := h.database.GetApp(job.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	progress.Update(10, "Starting application...")

	if app.Status == constants.AppStatusRunning {
		progress.Update(100, "Application already running")
		h.logger.Info("App already running, skipping start", "app_id", job.AppID, "app_name", app.Name)
		return nil
	}

	if err := h.dockerManager.StartApp(app.Name); err != nil {
		app.Status = constants.AppStatusError
		errorMsg := err.Error()
		app.ErrorMessage = &errorMsg

		if updateErr := h.database.UpdateApp(app); updateErr != nil {
			h.logger.Warn("Failed to update app to error state", "app_id", app.ID, "error", updateErr)
		}

		return fmt.Errorf("failed to start app: %w", err)
	}

	progress.Update(60, "Application started")

	app.Status = constants.AppStatusRunning
	app.ErrorMessage = nil

	if err := h.database.UpdateApp(app); err != nil {
		h.logger.Warn("Failed to update app status", "app_id", app.ID, "error", err)
		return fmt.Errorf("failed to update app status: %w", err)
	}

	progress.Update(100, "Application started successfully")

	h.logger.Info("Start completed successfully",
		"app_id", app.ID,
		"app_name", app.Name,
		"job_id", job.ID)

	return nil
}
