package http

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/httputil"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// detailForError returns a short, user-facing detail string (avoids redundant chained messages like "CODE: message: cause").
func detailForError(err error) string {
	var de *domain.DomainError
	if errors.As(err, &de) && de.Message != "" {
		return de.Message
	}
	return err.Error()
}

// handleServiceError handles errors from service layer
func (s *Server) handleServiceError(c *gin.Context, operation string, err error) {
	if domain.IsNotFoundError(err) {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Resource not found", Details: detailForError(err)})
		return
	}

	if domain.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Validation error", Details: detailForError(err)})
		return
	}

	slog.ErrorContext(c.Request.Context(), "service error", "operation", operation, "error", err)
	c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("Failed to %s", operation), Details: detailForError(err)})
}

// getNodeIDFromContext extracts node_id from context, checking both possible keys
// Returns empty string if not found (callers should validate)
func getNodeIDFromContext(c *gin.Context) string {
	// Check for node_id_param (from requireNodeIDMiddleware)
	if nodeID, exists := c.Get("node_id_param"); exists {
		if id, ok := nodeID.(string); ok && id != "" {
			return id
		}
	}
	
	// Check for node_id (from nodeAuthMiddleware)
	if nodeID, exists := c.Get("node_id"); exists {
		if id, ok := nodeID.(string); ok && id != "" {
			return id
		}
	}
	
	return ""
}

// createApp creates a new app
func (s *Server) createApp(c *gin.Context) {
	var req domain.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.WarnContext(c.Request.Context(), "invalid create app request", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request format"})
		return
	}
	// When node auth was used, composite auth set node_id_param; override req.NodeID so target is local
	if nodeID := getNodeIDFromContext(c); nodeID != "" {
		req.NodeID = nodeID
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
	id, err := httputil.ValidateAndGetAppID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}
	
	app, err := s.appService.GetApp(c.Request.Context(), id, nodeID)
	if err != nil {
		s.handleServiceError(c, "get app", err)
		return
	}

	c.JSON(http.StatusOK, app)
}

// updateApp updates an app
func (s *Server) updateApp(c *gin.Context) {
	id, err := httputil.ValidateAndGetAppID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	var req domain.UpdateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.WarnContext(c.Request.Context(), "invalid update app request", "appID", id, "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request format"})
		return
	}

	app, err := s.appService.UpdateApp(c.Request.Context(), id, nodeID, req)
	if err != nil {
		s.handleServiceError(c, "update app", err)
		return
	}

	c.JSON(http.StatusOK, app)
}

// deleteApp deletes an app using the comprehensive cleanup system
func (s *Server) deleteApp(c *gin.Context) {
	id, err := httputil.ValidateAndGetAppID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	if err := s.appService.DeleteApp(c.Request.Context(), id, nodeID); err != nil {
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

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	app, err := s.appService.StartApp(c.Request.Context(), id, nodeID)
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

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	app, err := s.appService.StopApp(c.Request.Context(), id, nodeID)
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

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	app, err := s.appService.UpdateAppContainers(c.Request.Context(), id, nodeID)
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

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	logs, err := s.systemService.GetAppLogs(c.Request.Context(), id, nodeID)
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

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	stats, err := s.systemService.GetAppStats(c.Request.Context(), id, nodeID)
	if err != nil {
		s.handleServiceError(c, "get app stats", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// listApps returns all apps
func (s *Server) listApps(c *gin.Context) {
	var nodeIDs []string
	if scope, ok := c.Get("request_scope"); ok && scope == "local" {
		nodeIDs = []string{s.config.Node.ID}
	} else {
		nodeIDs = httputil.ParseNodeIDs(c)
	}

	apps, err := s.appService.ListApps(c.Request.Context(), nodeIDs)
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

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	versions, err := s.composeService.GetVersions(c.Request.Context(), id, nodeID)
	if err != nil {
		s.handleServiceError(c, "get compose versions", err)
		return
	}

	c.JSON(http.StatusOK, versions)
}

// getComposeVersion returns a specific compose version
func (s *Server) getComposeVersion(c *gin.Context) {
	id, err := httputil.ValidateAndGetAppID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	version, err := httputil.ValidateAndGetVersion(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	composeVersion, err := s.composeService.GetVersion(c.Request.Context(), id, version, nodeID)
	if err != nil {
		s.handleServiceError(c, "get compose version", err)
		return
	}

	c.JSON(http.StatusOK, composeVersion)
}

// rollbackToVersion rolls back to a specific compose version
func (s *Server) rollbackToVersion(c *gin.Context) {
	id, err := httputil.ValidateAndGetAppID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	targetVersion, err := httputil.ValidateAndGetVersion(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
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

	newVersion, err := s.composeService.RollbackToVersion(c.Request.Context(), id, targetVersion, nodeID, req.ChangeReason, changedBy)
	if err != nil {
		s.handleServiceError(c, "rollback compose version", err)
		return
	}

	// Get updated app
	var app *db.App
	if nodeID != "" {
		app, _ = s.appService.GetApp(c.Request.Context(), id, nodeID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Rolled back successfully",
		"app":          app,
		"new_version":  newVersion,
		"from_version": targetVersion,
	})
}

