package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
)

// AppCreateHandler handles app_create jobs
type AppCreateHandler struct {
	db            *db.DB
	dockerManager *docker.Manager
	appService    domain.AppService
	tunnelService domain.TunnelService
	logger        *slog.Logger
}

// NewAppCreateHandler creates a new app create handler
func NewAppCreateHandler(
	database *db.DB,
	dockerMgr *docker.Manager,
	appSvc domain.AppService,
	tunnelSvc domain.TunnelService,
	logger *slog.Logger,
) *AppCreateHandler {
	return &AppCreateHandler{
		db:            database,
		dockerManager: dockerMgr,
		appService:    appSvc,
		tunnelService: tunnelSvc,
		logger:        logger,
	}
}

// Handle processes an app_create job
func (h *AppCreateHandler) Handle(ctx context.Context, job *db.Job, progress *ProgressTracker) error {
	// Parse payload
	var payload struct {
		Name               string              `json:"name"`
		TunnelMode         string              `json:"tunnel_mode,omitempty"`
		IngressRules       []map[string]string `json:"ingress_rules,omitempty"`
		CreatedTunnelAppID string              `json:"created_tunnel_app_id,omitempty"`
	}
	if job.Payload != nil {
		if err := json.Unmarshal([]byte(*job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to parse app_create payload: %w", err)
		}
	}

	// Get app from database
	app, err := h.db.GetApp(job.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	progress.Update(10, "Starting containers...")

	// Start app (SLOW: docker pull/build/up)
	if err := h.dockerManager.StartApp(app.Name); err != nil {
		// Update app to error state
		app.Status = constants.AppStatusError
		errorMsg := err.Error()
		app.ErrorMessage = &errorMsg
		if updateErr := h.db.UpdateApp(app); updateErr != nil {
			h.logger.Warn("failed to update app to error state", "app_id", app.ID, "error", updateErr)
		}
		return fmt.Errorf("failed to start app: %w", err)
	}

	progress.Update(60, "Containers started")

	// Extract Quick Tunnel URL if applicable (SLOW: 15s wait)
	if payload.TunnelMode == constants.TunnelModeQuick {
		progress.Update(70, "Extracting Quick Tunnel URL...")

		quickURL, err := h.tunnelService.ExtractQuickTunnelURL(ctx, app.ID, app.NodeID)
		if err != nil {
			h.logger.Warn("failed to extract Quick Tunnel URL", "app", app.Name, "error", err)
		} else if quickURL != "" {
			app.PublicURL = quickURL
			app.TunnelDomain = strings.TrimPrefix(quickURL, "https://")
			if err := h.db.UpdateApp(app); err != nil {
				h.logger.Warn("failed to save Quick Tunnel URL", "app", app.Name, "error", err)
			} else {
				h.logger.Info("Quick Tunnel URL captured", "app", app.Name, "public_url", quickURL)
			}
		}
	}

	// Restart tunnel container if needed (custom tunnel with ingress rules)
	if len(payload.IngressRules) > 0 && payload.CreatedTunnelAppID != "" && payload.TunnelMode == constants.TunnelModeCustom {
		progress.Update(85, "Restarting tunnel container...")
		if err := h.dockerManager.RestartTunnelService(app.Name); err != nil {
			h.logger.Warn("failed to restart tunnel container", "app", app.Name, "error", err)
		} else {
			h.logger.Info("tunnel container restarted", "app", app.Name)
		}
	}

	progress.Update(95, "Updating app status...")

	// Update app status to running
	app.Status = constants.AppStatusRunning
	app.ErrorMessage = nil
	if err := h.db.UpdateApp(app); err != nil {
		h.logger.Warn("failed to update app status to running", "app_id", app.ID, "error", err)
	}

	progress.Update(100, "App started successfully")
	return nil
}
