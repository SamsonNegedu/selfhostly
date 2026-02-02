package jobs

import (
	"context"
	"fmt"

	"github.com/selfhostly/internal/db"
)

// JobHandler defines the interface for job handlers
type JobHandler interface {
	Handle(ctx context.Context, job *db.Job, progress *ProgressTracker) error
}

// HandlerRegistry maps job types to their handlers
type HandlerRegistry struct {
	handlers map[string]JobHandler
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]JobHandler),
	}
}

// Register registers a handler for a specific job type
func (r *HandlerRegistry) Register(jobType string, handler JobHandler) {
	r.handlers[jobType] = handler
}

// GetHandler retrieves a handler for a job type
func (r *HandlerRegistry) GetHandler(jobType string) (JobHandler, error) {
	handler, ok := r.handlers[jobType]
	if !ok {
		return nil, fmt.Errorf("unknown job type: %s", jobType)
	}
	return handler, nil
}

// HasHandler checks if a handler exists for a job type
func (r *HandlerRegistry) HasHandler(jobType string) bool {
	_, ok := r.handlers[jobType]
	return ok
}
