package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/db"
)

// getJob retrieves a job by ID
func (s *Server) getJob(c *gin.Context) {
	jobID := c.Param("id")

	job, err := s.database.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Job not found",
			Details: "Could not find job with the specified ID",
		})
		return
	}

	c.JSON(http.StatusOK, job)
}

// getAppJobs retrieves recent jobs for an app
func (s *Server) getAppJobs(c *gin.Context) {
	appID := c.Param("id")

	// Get last 10 jobs for this app
	jobs, err := s.database.GetJobsByAppID(appID, 10)
	if err != nil {
		s.handleServiceError(c, "get app jobs", err)
		return
	}

	// Return empty array if no jobs found (not an error)
	if jobs == nil {
		jobs = []*db.Job{}
	}

	c.JSON(http.StatusOK, jobs)
}
