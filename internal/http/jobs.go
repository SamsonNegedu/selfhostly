package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/domain"
)

// handleJobError answers a missing job with the message the UI expects and leaves anything else to handleServiceError
func (s *Server) handleJobError(c *gin.Context, operation string, err error) {
	if domain.IsNotFoundError(err) {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Job not found",
			Details: "Could not find job with the specified ID",
		})
		return
	}
	s.handleServiceError(c, operation, err)
}

// getJob retrieves a job by ID
func (s *Server) getJob(c *gin.Context) {
	jobID := c.Param("id")

	job, err := s.jobService.GetJob(c.Request.Context(), jobID)
	if err != nil {
		s.handleJobError(c, "get job", err)
		return
	}

	c.JSON(http.StatusOK, job)
}

// getAppJobs retrieves recent jobs for an app
func (s *Server) getAppJobs(c *gin.Context) {
	appID := c.Param("id")

	// Get last 10 jobs for this app
	jobs, err := s.jobService.ListAppJobs(c.Request.Context(), appID, 10)
	if err != nil {
		s.handleServiceError(c, "get app jobs", err)
		return
	}

	c.JSON(http.StatusOK, jobs)
}
