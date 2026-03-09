package http

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/db"
)

// getAppSchedule returns the schedule for an app
func (s *Server) getAppSchedule(c *gin.Context) {
	appID := c.Param("id")
	nodeID := c.Query("node_id")

	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	schedule, err := s.scheduleService.GetSchedule(c.Request.Context(), appID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get schedule", "app_id", appID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get schedule"})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// createOrUpdateAppSchedule creates or updates a schedule for an app
func (s *Server) createOrUpdateAppSchedule(c *gin.Context) {
	appID := c.Param("id")
	nodeID := c.Query("node_id")

	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	var req struct {
		StartCron string `json:"start_cron"`
		StopCron  string `json:"stop_cron"`
		Timezone  string `json:"timezone"`
		Enabled   bool   `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		slog.WarnContext(c.Request.Context(), "invalid schedule request", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	if req.Timezone == "" {
		req.Timezone = "UTC"
	}

	existingSchedule, err := s.scheduleService.GetSchedule(c.Request.Context(), appID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to check for existing schedule", "app_id", appID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to check for existing schedule"})
		return
	}

	var schedule *db.AppSchedule
	if existingSchedule != nil {
		schedule, err = s.scheduleService.UpdateSchedule(
			c.Request.Context(),
			appID,
			req.StartCron,
			req.StopCron,
			req.Timezone,
			req.Enabled,
		)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to update schedule", "app_id", appID, "error", err)
			s.handleServiceError(c, "update schedule", err)
			return
		}
	} else {
		schedule, err = s.scheduleService.CreateSchedule(
			c.Request.Context(),
			appID,
			req.StartCron,
			req.StopCron,
			req.Timezone,
			req.Enabled,
		)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to create schedule", "app_id", appID, "error", err)
			s.handleServiceError(c, "create schedule", err)
			return
		}
	}

	if err := s.scheduler.UpdateSchedule(schedule); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update scheduler", "app_id", appID, "error", err)
	}

	c.JSON(http.StatusOK, schedule)
}

// deleteAppSchedule deletes a schedule for an app
func (s *Server) deleteAppSchedule(c *gin.Context) {
	appID := c.Param("id")
	nodeID := c.Query("node_id")

	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	if err := s.scheduleService.DeleteSchedule(c.Request.Context(), appID); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to delete schedule", "app_id", appID, "error", err)
		s.handleServiceError(c, "delete schedule", err)
		return
	}

	s.scheduler.RemoveSchedule(appID)

	c.Status(http.StatusNoContent)
}

// testAppSchedule validates a cron expression and returns next run times
func (s *Server) testAppSchedule(c *gin.Context) {
	appID := c.Param("id")
	nodeID := c.Query("node_id")

	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	var req struct {
		StartCron string `json:"start_cron"`
		StopCron  string `json:"stop_cron"`
		Timezone  string `json:"timezone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		slog.WarnContext(c.Request.Context(), "invalid test schedule request", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	if req.Timezone == "" {
		req.Timezone = "UTC"
	}

	if req.StartCron != "" {
		if err := s.scheduleService.ValidateCronExpression(req.StartCron); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid start cron expression", Details: err.Error()})
			return
		}
	}

	if req.StopCron != "" {
		if err := s.scheduleService.ValidateCronExpression(req.StopCron); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid stop cron expression", Details: err.Error()})
			return
		}
	}

	// Calculate next run times based on the provided cron expressions (not from database)
	nextRuns, err := s.scheduleService.CalculateNextRunTimes(c.Request.Context(), appID, req.StartCron, req.StopCron, req.Timezone)
	if err != nil {
		// Check if it's a validation error (client error) or server error
		errMsg := err.Error()
		if strings.Contains(errMsg, "cannot be the same") || 
		   strings.Contains(errMsg, "must occur after") ||
		   strings.Contains(errMsg, "invalid") || 
		   strings.Contains(errMsg, "failed to parse") ||
		   strings.Contains(errMsg, "failed to load timezone") {
			slog.WarnContext(c.Request.Context(), "invalid schedule configuration", "app_id", appID, "error", err)
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}
		
		slog.ErrorContext(c.Request.Context(), "failed to calculate next run times", "app_id", appID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to calculate next run times"})
		return
	}

	c.JSON(http.StatusOK, nextRuns)
}

// getAppScheduleNextRuns returns the next run times for an existing schedule
func (s *Server) getAppScheduleNextRuns(c *gin.Context) {
	appID := c.Param("id")
	nodeID := c.Query("node_id")

	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	nextRuns, err := s.scheduleService.GetNextRunTimes(c.Request.Context(), appID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get next run times", "app_id", appID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get next run times"})
		return
	}

	c.JSON(http.StatusOK, nextRuns)
}
