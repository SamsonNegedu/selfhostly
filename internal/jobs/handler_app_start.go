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

	var dockerLog func(string)
	if jl := progress.JobLog(); jl != nil {
		dockerLog = jl.WriteLine
	}

	if err := h.dockerManager.StartAppWithLogs(ctx, app.Name, dockerLog); err != nil {
		// Check actual container status to determine if start partially succeeded
		status, statusErr := h.dockerManager.GetAppStatus(app.Name)
		if statusErr != nil {
			h.logger.Warn("failed to check app status after start failure", "app", app.Name, "error", statusErr)
			// Can't determine status, mark as error
			app.Status = constants.AppStatusError
		} else if status == constants.AppStatusRunning {
			// Containers are running despite error, mark as running
			h.logger.Info("start reported error but containers are running", "app", app.Name)
			app.Status = constants.AppStatusRunning
		} else {
			// Containers not running, mark as error
			app.Status = constants.AppStatusError
		}

		errorMsg := fmt.Sprintf("Start failed: %s", err.Error())
		app.ErrorMessage = &errorMsg

		if updateErr := h.database.UpdateApp(app); updateErr != nil {
			h.logger.Warn("Failed to update app state after start failure", "app_id", app.ID, "error", updateErr)
		}

		// Log full error for debugging
		h.logger.ErrorContext(ctx, "docker start failed", "app", app.Name, "error", err, "final_status", app.Status)
		return fmt.Errorf("Docker start failed: %w", err)
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
