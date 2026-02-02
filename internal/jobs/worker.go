package jobs

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
)

// Worker polls for pending jobs and processes them
type Worker struct {
	processor    *Processor
	db           *db.DB
	pollInterval time.Duration
	logger       *slog.Logger
	workerID     string // Unique ID for this worker instance

	// State management for graceful shutdown
	currentJobID string
	mu           sync.RWMutex
}

// NewWorker creates a new job worker
func NewWorker(processor *Processor, database *db.DB, pollInterval time.Duration, logger *slog.Logger) *Worker {
	return &Worker{
		processor:    processor,
		db:           database,
		pollInterval: pollInterval,
		logger:       logger,
		workerID:     uuid.New().String(), // Generate unique worker ID
	}
}

// Start begins the worker's main loop
func (w *Worker) Start(ctx context.Context) error {
	w.logger.Info("job worker starting", "poll_interval", w.pollInterval)

	// On startup, recover from stale jobs (from previous crashes)
	if err := w.recoverStaleJobs(); err != nil {
		w.logger.Error("failed to recover stale jobs", "error", err)
		// Don't fail startup, just log the error
	}

	// Start cleanup routine in background
	go w.cleanupLoop(ctx)

	// Main processing loop
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("job worker shutting down gracefully")
			return w.gracefulShutdown()
		case <-ticker.C:
			w.processPendingJobs(ctx)
		}
	}
}

// recoverStaleJobs marks stale "running" jobs as failed on startup
func (w *Worker) recoverStaleJobs() error {
	w.logger.Info("checking for stale jobs", "threshold", constants.JobStaleThreshold)

	if err := w.db.MarkStaleJobsAsFailed(constants.JobStaleThreshold); err != nil {
		return err
	}

	return nil
}

// gracefulShutdown waits for the current job to finish or times out
func (w *Worker) gracefulShutdown() error {
	w.mu.RLock()
	currentJobID := w.currentJobID
	w.mu.RUnlock()

	if currentJobID == "" {
		w.logger.Info("no job running, shutdown complete")
		return nil
	}

	w.logger.Info("waiting for current job to complete", "job_id", currentJobID, "timeout", constants.JobGracefulShutdownTimeout)

	// Wait for current job with timeout
	deadline := time.Now().Add(constants.JobGracefulShutdownTimeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		w.mu.RLock()
		stillRunning := w.currentJobID == currentJobID
		w.mu.RUnlock()

		if !stillRunning {
			w.logger.Info("current job completed before shutdown")
			return nil
		}

		<-ticker.C
	}

	// Timeout reached, mark job as failed
	w.logger.Warn("shutdown timeout reached, marking current job as failed", "job_id", currentJobID)
	errorMsg := "Worker shutdown before job completion"
	return w.db.UpdateJobCompleted(currentJobID, constants.JobStatusFailed, nil, &errorMsg)
}

// processPendingJobs processes a single pending job if worker is idle
// Uses atomic claiming to prevent race conditions
func (w *Worker) processPendingJobs(ctx context.Context) {
	// Check if already processing a job
	w.mu.RLock()
	busy := w.currentJobID != ""
	w.mu.RUnlock()

	if busy {
		return // Already processing a job
	}

	// Atomically claim a pending job
	job, err := w.db.ClaimPendingJob(w.workerID)
	if err != nil {
		w.logger.Error("failed to claim pending job", "error", err)
		return
	}

	if job == nil {
		return // No job available
	}

	// Mark as current job
	w.mu.Lock()
	w.currentJobID = job.ID
	w.mu.Unlock()

	// Ensure we clear current job and release claim when done
	defer func() {
		w.mu.Lock()
		w.currentJobID = ""
		w.mu.Unlock()
		
		// Release claim if job failed (in case of crash)
		// If job completed successfully, claim is already cleared
	}()

	// Process the job
	w.logger.Info("starting job processing", "job_id", job.ID, "type", job.Type, "app_id", job.AppID, "worker_id", w.workerID)
	startTime := time.Now()

	if err := w.processor.ProcessJob(ctx, job); err != nil {
		w.logger.Error("job processing failed", "job_id", job.ID, "error", err, "duration", time.Since(startTime))
		// Release claim on failure
		if releaseErr := w.db.ReleaseJobClaim(job.ID); releaseErr != nil {
			w.logger.Warn("failed to release job claim", "job_id", job.ID, "error", releaseErr)
		}
	} else {
		w.logger.Info("job processing completed", "job_id", job.ID, "duration", time.Since(startTime))
	}
}

// cleanupLoop periodically cleans up old completed/failed jobs
func (w *Worker) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(constants.JobHistoryCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Debug("job cleanup routine stopped")
			return
		case <-ticker.C:
			w.performCleanup()
		}
	}
}

// performCleanup cleans up old job records using optimized single-query approach
func (w *Worker) performCleanup() {
	w.logger.Debug("cleaning up old job records")

	if err := w.db.CleanupAllOldCompletedJobs(constants.JobHistoryKeepCount); err != nil {
		w.logger.Error("failed to cleanup old jobs", "error", err)
		return
	}

	w.logger.Debug("job cleanup completed")
}
