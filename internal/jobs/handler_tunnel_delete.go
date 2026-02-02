package jobs

import (
	"context"
	"log/slog"

	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
)

// TunnelDeleteHandler handles tunnel_delete jobs
type TunnelDeleteHandler struct {
	db            *db.DB
	dockerManager *docker.Manager
	tunnelService domain.TunnelService
	logger        *slog.Logger
}

// NewTunnelDeleteHandler creates a new tunnel delete handler
func NewTunnelDeleteHandler(
	database *db.DB,
	dockerMgr *docker.Manager,
	tunnelSvc domain.TunnelService,
	logger *slog.Logger,
) *TunnelDeleteHandler {
	return &TunnelDeleteHandler{
		db:            database,
		dockerManager: dockerMgr,
		tunnelService: tunnelSvc,
		logger:        logger,
	}
}

// Handle processes a tunnel_delete job
func (h *TunnelDeleteHandler) Handle(ctx context.Context, job *db.Job, progress *ProgressTracker) error {
	progress.Update(5, "Getting app details...")

	// Get app
	app, err := h.db.GetApp(job.AppID)
	if err != nil {
		return err
	}

	// Check if app has a tunnel to delete
	if app.TunnelID == "" && app.TunnelMode == "" {
		progress.Update(100, "No tunnel to delete")
		h.logger.Info("app has no tunnel to delete", "app_id", app.ID)
		return nil
	}

	progress.Update(10, "Stopping tunnel container...")
	progress.Update(30, "Deleting tunnel from Cloudflare...")

	// Delete tunnel via tunnel service (handles container removal, provider API deletion, and cleanup)
	// This is the main deletion step - it handles:
	// 1. Stopping and removing tunnel container
	// 2. Waiting for Cloudflare to detect disconnection
	// 3. Provider API deletion with retries
	// 4. DNS record cleanup
	// 5. Database cleanup
	// 6. Compose file updates
	if err := h.tunnelService.DeleteTunnel(ctx, app.ID, app.NodeID); err != nil {
		return err
	}

	progress.Update(90, "Cleaning up configuration...")
	progress.Update(95, "Tunnel deleted successfully")

	// Log success
	h.logger.Info("tunnel deleted via background job", "app_id", app.ID)

	progress.Update(100, "Complete")
	return nil
}
