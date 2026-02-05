package jobs

import (
	"context"
	"log/slog"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
)

// Processor handles the execution of background jobs
type Processor struct {
	registry *HandlerRegistry
	db       *db.DB
	logger   *slog.Logger
}

// NewProcessor creates a new job processor with registered handlers
func NewProcessor(
	database *db.DB,
	dockerMgr *docker.Manager,
	appSvc domain.AppService,
	tunnelSvc domain.TunnelService,
	logger *slog.Logger,
) *Processor {
	registry := NewHandlerRegistry()
	
	// Register all handlers
	registry.Register(constants.JobTypeAppCreate, NewAppCreateHandler(database, dockerMgr, appSvc, tunnelSvc, logger))
	registry.Register(constants.JobTypeAppUpdate, NewAppUpdateHandler(database, dockerMgr, logger))
	registry.Register(constants.JobTypeTunnelCreate, NewTunnelCreateHandler(database, dockerMgr, appSvc, tunnelSvc, logger))
	registry.Register(constants.JobTypeTunnelDelete, NewTunnelDeleteHandler(database, dockerMgr, tunnelSvc, logger))
	registry.Register(constants.JobTypeQuickTunnel, NewQuickTunnelHandler(database, dockerMgr, tunnelSvc, logger))
	
	return &Processor{
		registry: registry,
		db:       database,
		logger:   logger,
	}
}

// ProcessJob processes a single job based on its type using the handler registry
// Note: Job should already be marked as "running" by ClaimPendingJob
func (p *Processor) ProcessJob(ctx context.Context, job *db.Job) error {
	p.logger.InfoContext(ctx, "processing job", "job_id", job.ID, "type", job.Type, "app_id", job.AppID)

	// Job is already marked as running by ClaimPendingJob, so we can proceed directly

	// Create progress tracker
	progress := NewProgressTracker(job.ID, p.db, p.logger)

	// Get handler from registry
	handler, err := p.registry.GetHandler(job.Type)
	if err != nil {
		p.logger.ErrorContext(ctx, "unknown job type", "job_id", job.ID, "type", job.Type, "error", err)
		errorMsg := err.Error()
		return p.db.UpdateJobCompleted(job.ID, constants.JobStatusFailed, nil, &errorMsg)
	}

	// Process the job
	err = handler.Handle(ctx, job, progress)

	// Update job status based on result
	if err != nil {
		p.logger.ErrorContext(ctx, "job failed", "job_id", job.ID, "type", job.Type, "error", err)
		errorMsg := err.Error()
		return p.db.UpdateJobCompleted(job.ID, constants.JobStatusFailed, nil, &errorMsg)
	}

	p.logger.InfoContext(ctx, "job completed successfully", "job_id", job.ID, "type", job.Type)
	return p.db.UpdateJobCompleted(job.ID, constants.JobStatusCompleted, nil, nil)
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}
