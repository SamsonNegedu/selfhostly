package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

type jobService struct {
	database *db.DB
}

// NewJobService creates the service that reads deployment jobs and their logs
func NewJobService(database *db.DB) domain.JobService {
	return &jobService{database: database}
}

func (s *jobService) GetJob(_ context.Context, jobID string) (*db.Job, error) {
	job, err := s.database.GetJob(jobID)
	if err != nil {
		return nil, wrapJobError("get job", jobID, err)
	}
	return job, nil
}

func (s *jobService) ListAppJobs(_ context.Context, appID string, limit int) ([]*db.Job, error) {
	jobs, err := s.database.GetJobsByAppID(appID, limit)
	if err != nil {
		return nil, domain.WrapDatabaseOperation("list app jobs", err)
	}
	if jobs == nil {
		jobs = []*db.Job{}
	}
	return jobs, nil
}

func (s *jobService) GetJobLogsAfter(_ context.Context, jobID string, after int64, limit int) ([]db.JobLogLine, error) {
	lines, err := s.database.GetJobLogsAfter(jobID, after, limit)
	if err != nil {
		return nil, wrapJobError("get job logs", jobID, err)
	}
	return lines, nil
}

func wrapJobError(operation, jobID string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.WrapJobNotFound(jobID, err)
	}
	return domain.WrapDatabaseOperation(operation, err)
}
