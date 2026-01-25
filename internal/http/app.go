package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/domain"
)

const (
	statusStopped  = "stopped"
	statusRunning  = "running"
	statusUpdating = "updating"
	statusError    = "error"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// handleServiceError handles errors from service layer
func (s *Server) handleServiceError(c *gin.Context, operation string, err error) {
	if domain.IsNotFoundError(err) {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Resource not found", Details: err.Error()})
		return
	}

	if domain.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Validation error", Details: err.Error()})
		return
	}

	slog.ErrorContext(c.Request.Context(), "service error", "operation", operation, "error", err)
	c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("Failed to %s", operation), Details: err.Error()})
}

// createApp creates a new app
func (s *Server) createApp(c *gin.Context) {
	var req domain.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.WarnContext(c.Request.Context(), "invalid create app request", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request format"})
		return
	}

	app, err := s.appService.CreateApp(c.Request.Context(), req)
	if err != nil {
		s.handleServiceError(c, "create app", err)
		return
	}

	c.JSON(http.StatusCreated, app)
}

// getApp returns a single app
func (s *Server) getApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.appService.GetApp(c.Request.Context(), id)
	if err != nil {
		s.handleServiceError(c, "get app", err)
		return
	}

	c.JSON(http.StatusOK, app)
}

// updateApp updates an app
func (s *Server) updateApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	var req domain.UpdateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.WarnContext(c.Request.Context(), "invalid update app request", "appID", id, "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request format"})
		return
	}

	app, err := s.appService.UpdateApp(c.Request.Context(), id, req)
	if err != nil {
		s.handleServiceError(c, "update app", err)
		return
	}

	c.JSON(http.StatusOK, app)
}

// deleteApp deletes an app using the comprehensive cleanup system
func (s *Server) deleteApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	if err := s.appService.DeleteApp(c.Request.Context(), id); err != nil {
		s.handleServiceError(c, "delete app", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "App deleted successfully",
		"appID":   id,
	})
}

// startApp starts an app
func (s *Server) startApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.appService.StartApp(c.Request.Context(), id)
	if err != nil {
		s.handleServiceError(c, "start app", err)
		return
	}

	c.JSON(http.StatusOK, app)
}

// stopApp stops an app
func (s *Server) stopApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.appService.StopApp(c.Request.Context(), id)
	if err != nil {
		s.handleServiceError(c, "stop app", err)
		return
	}

	c.JSON(http.StatusOK, app)
}

// updateAppContainers updates app containers with zero downtime
func (s *Server) updateAppContainers(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.appService.UpdateAppContainers(c.Request.Context(), id)
	if err != nil {
		s.handleServiceError(c, "update app containers", err)
		return
	}

	c.JSON(http.StatusOK, app)
}

// getAppLogs returns app logs
func (s *Server) getAppLogs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	logs, err := s.systemService.GetAppLogs(c.Request.Context(), id)
	if err != nil {
		s.handleServiceError(c, "get app logs", err)
		return
	}

	c.Header("Content-Type", "text/plain")
	c.Data(http.StatusOK, "text/plain", logs)
}

// getAppStats returns real-time resource statistics for an app
func (s *Server) getAppStats(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	stats, err := s.systemService.GetAppStats(c.Request.Context(), id)
	if err != nil {
		s.handleServiceError(c, "get app stats", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// repairApp repairs an app's compose file if needed (e.g., adds missing cloudflared token)
func (s *Server) repairApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.appService.RepairApp(c.Request.Context(), id)
	if err != nil {
		s.handleServiceError(c, "repair app", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "App repaired successfully",
		"app":     app,
	})
}

// listApps returns all apps
func (s *Server) listApps(c *gin.Context) {
	apps, err := s.appService.ListApps(c.Request.Context())
	if err != nil {
		s.handleServiceError(c, "list apps", err)
		return
	}

	c.JSON(http.StatusOK, apps)
}

// RollbackRequest represents a rollback request with optional metadata
type RollbackRequest struct {
	ChangeReason *string `json:"change_reason"`
}

// getComposeVersions returns all compose versions for an app
func (s *Server) getComposeVersions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	versions, err := s.composeService.GetVersions(c.Request.Context(), id)
	if err != nil {
		s.handleServiceError(c, "get compose versions", err)
		return
	}

	c.JSON(http.StatusOK, versions)
}

// getComposeVersion returns a specific compose version
func (s *Server) getComposeVersion(c *gin.Context) {
	id := c.Param("id")
	versionParam := c.Param("version")
	if id == "" || versionParam == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID or version"})
		return
	}

	// Parse version number
	var version int
	if _, err := fmt.Sscanf(versionParam, "%d", &version); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid version number"})
		return
	}

	composeVersion, err := s.composeService.GetVersion(c.Request.Context(), id, version)
	if err != nil {
		s.handleServiceError(c, "get compose version", err)
		return
	}

	c.JSON(http.StatusOK, composeVersion)
}

// rollbackToVersion rolls back to a specific compose version
func (s *Server) rollbackToVersion(c *gin.Context) {
	id := c.Param("id")
	versionParam := c.Param("version")
	if id == "" || versionParam == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID or version"})
		return
	}

	// Parse version number
	var targetVersion int
	if _, err := fmt.Sscanf(versionParam, "%d", &targetVersion); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid version number"})
		return
	}

	// Get optional rollback request body
	var req RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Body is optional, so just use empty request if binding fails
		req = RollbackRequest{}
	}

	// Get authenticated user info
	user, userExists := getUserFromContext(c)
	var changedBy *string
	if userExists && user.Name != "" {
		changedBy = &user.Name
	}

	newVersion, err := s.composeService.RollbackToVersion(c.Request.Context(), id, targetVersion, req.ChangeReason, changedBy)
	if err != nil {
		s.handleServiceError(c, "rollback compose version", err)
		return
	}

	// Get updated app
	app, _ := s.appService.GetApp(c.Request.Context(), id)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Rolled back successfully",
		"app":          app,
		"new_version":  newVersion,
		"from_version": targetVersion,
	})
}
