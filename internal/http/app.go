package http

import (
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

	app, err := s.appService.CreateApp(c.Request.Context(), req)
	if err != nil {
		s.handleServiceError(c, "create app", err)
		return
	}

	c.JSON(http.StatusCreated, app)
}

// createLocalApp creates a new app on the local node (for inter-node calls)
func (s *Server) createLocalApp(c *gin.Context) {
	var req domain.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.WarnContext(c.Request.Context(), "invalid create app request", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request format"})
		return
	}

	// Force NodeID to be the local node for internal endpoints
	req.NodeID = s.config.Node.ID

	app, err := s.appService.CreateApp(c.Request.Context(), req)
	if err != nil {
		s.handleServiceError(c, "create local app", err)
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
	// Extract node_ids from query parameter
	nodeIDs := httputil.ParseNodeIDs(c)

	apps, err := s.appService.ListApps(c.Request.Context(), nodeIDs)
	if err != nil {
		s.handleServiceError(c, "list apps", err)
		return
	}

	c.JSON(http.StatusOK, apps)
}

// listLocalApps returns only apps from the local node (for inter-node calls)
func (s *Server) listLocalApps(c *gin.Context) {
	apps, err := s.database.GetAllApps()
	if err != nil {
		s.handleServiceError(c, "list local apps", err)
		return
	}

	c.JSON(http.StatusOK, apps)
}

// getLocalApp returns a single app from the local node (for inter-node calls)
func (s *Server) getLocalApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	// Fetch directly from local database (internal endpoints are local-only)
	app, err := s.database.GetApp(id)
	if err != nil {
		s.handleServiceError(c, "get local app", err)
		return
	}

	c.JSON(http.StatusOK, app)
}

// updateLocalApp updates an app on the local node (for inter-node calls)
func (s *Server) updateLocalApp(c *gin.Context) {
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

	// Use local node ID for internal endpoints
	app, err := s.appService.UpdateApp(c.Request.Context(), id, s.config.Node.ID, req)
	if err != nil {
		s.handleServiceError(c, "update local app", err)
		return
	}

	c.JSON(http.StatusOK, app)
}

// deleteLocalApp deletes an app on the local node (for inter-node calls)
func (s *Server) deleteLocalApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	// Use local node ID for internal endpoints
	if err := s.appService.DeleteApp(c.Request.Context(), id, s.config.Node.ID); err != nil {
		s.handleServiceError(c, "delete local app", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "App deleted successfully",
		"appID":   id,
	})
}

// startLocalApp starts an app on the local node (for inter-node calls)
func (s *Server) startLocalApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	// Use local node ID for internal endpoints
	app, err := s.appService.StartApp(c.Request.Context(), id, s.config.Node.ID)
	if err != nil {
		s.handleServiceError(c, "start local app", err)
		return
	}

	c.JSON(http.StatusOK, app)
}

// stopLocalApp stops an app on the local node (for inter-node calls)
func (s *Server) stopLocalApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	// Use local node ID for internal endpoints
	app, err := s.appService.StopApp(c.Request.Context(), id, s.config.Node.ID)
	if err != nil {
		s.handleServiceError(c, "stop local app", err)
		return
	}

	c.JSON(http.StatusOK, app)
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

	composeVersion, err := s.composeService.GetVersion(c.Request.Context(), id, version)
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
	nodeID := getNodeIDFromContext(c)
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
