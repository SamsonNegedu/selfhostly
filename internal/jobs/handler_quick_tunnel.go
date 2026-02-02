package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
)

// QuickTunnelHandler handles quick_tunnel jobs
type QuickTunnelHandler struct {
	db            *db.DB
	dockerManager *docker.Manager
	tunnelService domain.TunnelService
	logger        *slog.Logger
}

// NewQuickTunnelHandler creates a new quick tunnel handler
func NewQuickTunnelHandler(
	database *db.DB,
	dockerMgr *docker.Manager,
	tunnelSvc domain.TunnelService,
	logger *slog.Logger,
) *QuickTunnelHandler {
	return &QuickTunnelHandler{
		db:            database,
		dockerManager: dockerMgr,
		tunnelService: tunnelSvc,
		logger:        logger,
	}
}

// Handle processes a quick_tunnel job
func (h *QuickTunnelHandler) Handle(ctx context.Context, job *db.Job, progress *ProgressTracker) error {
	// Parse payload
	var payload QuickTunnelPayload
	if job.Payload != nil {
		if err := json.Unmarshal([]byte(*job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to parse quick_tunnel payload: %w", err)
		}
	}

	if payload.Service == "" || payload.Port == 0 {
		return fmt.Errorf("invalid payload: service and port are required")
	}

	progress.Update(10, "Retrieving app details...")

	// Get the app from database
	app, err := h.db.GetApp(job.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	// Check if we're recreating an existing quick tunnel
	isRecreating := app.TunnelMode == constants.TunnelModeQuick

	progress.Update(20, "Parsing compose configuration...")

	compose, err := docker.ParseCompose([]byte(app.ComposeContent))
	if err != nil {
		return fmt.Errorf("invalid compose file: %w", err)
	}

	progress.Update(30, "Allocating metrics port...")

	// If recreating, try to reuse the existing metrics port
	var metricsPort int
	if isRecreating {
		if existingPort, ok := docker.ExtractQuickTunnelMetricsHostPort(app.ComposeContent); ok {
			metricsPort = existingPort
		}
	}
	// If not recreating or failed to extract port, allocate a new one
	if metricsPort == 0 {
		metricsPort, err = h.tunnelService.NextFreeQuickTunnelMetricsPort()
		if err != nil {
			// Use fallback port if allocation fails
			metricsPort = constants.QuickTunnelMetricsPort
		}
	}

	progress.Update(40, "Configuring Quick Tunnel container...")

	// Remove existing tunnel service if recreating
	if isRecreating {
		docker.RemoveTunnelService(compose)
	}

	// Create tunnel container config
	containerConfig, err := h.tunnelService.CreateQuickTunnelConfig(payload.Service, payload.Port, metricsPort)
	if err != nil {
		return fmt.Errorf("failed to create Quick Tunnel config: %w", err)
	}

	// Inject tunnel container
	networks := docker.ExtractNetworks(compose)
	network := ""
	if len(networks) > 0 {
		network = networks[0]
	}
	injected, err := docker.InjectTunnelContainer(compose, app.Name, containerConfig, network)
	if err != nil {
		return fmt.Errorf("failed to inject Quick Tunnel container: %w", err)
	}
	if !injected {
		return fmt.Errorf("Quick Tunnel container was not injected")
	}

	progress.Update(50, "Updating compose file...")

	// Marshal and save compose content
	composeBytes, err := docker.MarshalComposeFile(compose)
	if err != nil {
		return fmt.Errorf("failed to marshal compose file: %w", err)
	}
	app.ComposeContent = string(composeBytes)
	app.TunnelMode = constants.TunnelModeQuick
	app.TunnelID = ""
	app.TunnelToken = ""
	app.TunnelDomain = ""
	app.PublicURL = ""
	app.UpdatedAt = time.Now()
	if err := h.db.UpdateApp(app); err != nil {
		return fmt.Errorf("failed to update app in database: %w", err)
	}

	progress.Update(60, "Saving compose version...")

	// Create compose version
	latestVersion, _ := h.db.GetLatestVersionNumber(job.AppID)
	_ = h.db.MarkAllVersionsAsNotCurrent(job.AppID)
	updateReason := constants.ComposeVersionReasonQuickTunnel
	newVersion := db.NewComposeVersion(job.AppID, latestVersion+1, app.ComposeContent, &updateReason, nil)
	_ = h.db.CreateComposeVersion(newVersion)

	progress.Update(70, "Writing compose file to disk...")

	// Write compose file
	if err := h.dockerManager.WriteComposeFile(app.Name, app.ComposeContent); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	progress.Update(80, "Restarting containers...")

	// Update or start app containers
	if isRecreating {
		if err := h.dockerManager.UpdateApp(app.Name); err != nil {
			return fmt.Errorf("failed to update app containers: %w", err)
		}
		// Force recreate tunnel container
		if err := h.dockerManager.ForceRecreateTunnel(app.Name); err != nil {
			return fmt.Errorf("failed to force-recreate tunnel: %w", err)
		}
	} else {
		if err := h.dockerManager.StartApp(app.Name); err != nil {
			return fmt.Errorf("failed to start app: %w", err)
		}
	}

	// Update app status
	app.Status = constants.AppStatusRunning
	app.ErrorMessage = nil
	app.UpdatedAt = time.Now()
	_ = h.db.UpdateApp(app)

	progress.Update(90, "Extracting Quick Tunnel URL...")

	// Extract Quick Tunnel URL
	quickURL, err := h.tunnelService.ExtractQuickTunnelURL(ctx, job.AppID, app.NodeID)
	if err == nil && quickURL != "" {
		app.PublicURL = quickURL
		app.TunnelDomain = strings.TrimPrefix(quickURL, "https://")
		app.UpdatedAt = time.Now()
		_ = h.db.UpdateApp(app)
	}

	progress.Update(100, "Quick Tunnel created successfully")
	return nil
}
