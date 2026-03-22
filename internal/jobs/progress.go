package jobs

import (
	"log/slog"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
)

// ProgressTracker provides a simple interface for updating job progress
type ProgressTracker struct {
	jobID  string
	db     *db.DB
	logger *slog.Logger
	jobLog *JobLogWriter // optional: merged into deployment log stream as [step] lines
}

// NewProgressTracker creates a new progress tracker for a job.
// jobLog may be nil (tests); when set, Update/UpdateMessage also append "[step] …" lines for the deployment log UI.
func NewProgressTracker(jobID string, database *db.DB, logger *slog.Logger, jobLog *JobLogWriter) *ProgressTracker {
	return &ProgressTracker{
		jobID:  jobID,
		db:     database,
		logger: logger,
		jobLog: jobLog,
	}
}

// JobLog returns the shared job log writer, or nil.
func (pt *ProgressTracker) JobLog() *JobLogWriter {
	return pt.jobLog
}

// Update updates the job's progress and message
func (pt *ProgressTracker) Update(progress int, message string) {
	// Clamp progress to 0-100
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	if err := pt.db.UpdateJobStatus(pt.jobID, constants.JobStatusRunning, progress, &message); err != nil {
		pt.logger.Error("failed to update job progress", "job_id", pt.jobID, "error", err)
	} else {
		pt.logger.Debug("job progress updated", "job_id", pt.jobID, "progress", progress, "message", message)
		if pt.jobLog != nil && message != "" {
			pt.jobLog.WriteLine("[step] " + message)
		}
	}
}

// UpdateMessage updates just the message without changing progress
func (pt *ProgressTracker) UpdateMessage(message string) {
	// Get current progress first
	job, err := pt.db.GetJob(pt.jobID)
	if err != nil {
		pt.logger.Error("failed to get job for message update", "job_id", pt.jobID, "error", err)
		return
	}

	if err := pt.db.UpdateJobStatus(pt.jobID, constants.JobStatusRunning, job.Progress, &message); err != nil {
		pt.logger.Error("failed to update job message", "job_id", pt.jobID, "error", err)
	} else if pt.jobLog != nil && message != "" {
		pt.jobLog.WriteLine("[step] " + message)
	}
}
