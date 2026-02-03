package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/httputil"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// detailForError returns a short, user-facing detail string. Uses only domain message (never Cause) to avoid leaking DB/driver internals.
func detailForError(err error) string {
	return domain.PublicMessage(err)
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
	// Validate Quick Tunnel params when tunnel_mode is "quick"
	if req.TunnelMode == constants.TunnelModeQuick {
		if strings.TrimSpace(req.QuickTunnelService) == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "quick_tunnel_service is required for Quick Tunnel mode"})
			return
		}
		if req.QuickTunnelPort < constants.MinPort || req.QuickTunnelPort > constants.MaxPort {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("quick_tunnel_port must be between %d and %d", constants.MinPort, constants.MaxPort)})
			return
		}
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID", Details: domain.PublicMessage(err)})
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID", Details: domain.PublicMessage(err)})
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID", Details: domain.PublicMessage(err)})
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

	// Create background job for app update (async operation)
	job, err := s.appService.UpdateAppContainersAsync(c.Request.Context(), id)
	if err != nil {
		s.handleServiceError(c, "create update job", err)
		return
	}

	// Return 202 Accepted with job info
	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  job.ID,
		"status":  job.Status,
		"message": "App update started in background",
	})
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

	// Get optional service filter from query parameter
	service := c.Query("service")

	logs, err := s.systemService.GetAppLogs(c.Request.Context(), id, nodeID, service)
	if err != nil {
		s.handleServiceError(c, "get app logs", err)
		return
	}

	c.Header("Content-Type", "text/plain")
	c.Data(http.StatusOK, "text/plain", logs)
}

// getAppServices returns the list of service names for an app
func (s *Server) getAppServices(c *gin.Context) {
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

	services, err := s.systemService.GetAppServices(c.Request.Context(), id, nodeID)
	if err != nil {
		s.handleServiceError(c, "get app services", err)
		return
	}

	c.JSON(http.StatusOK, services)
}

// restartAppService restarts a specific service within an app
func (s *Server) restartAppService(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	serviceName := c.Param("service")
	if serviceName == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Service name is required"})
		return
	}

	// Get node_id from middleware (already validated)
	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	if err := s.appService.RestartAppService(c.Request.Context(), id, nodeID, serviceName); err != nil {
		s.handleServiceError(c, "restart app service", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Service %s restarted successfully", serviceName),
		"service": serviceName,
	})
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

// getQuickTunnelURL runs Quick Tunnel URL extraction on the node that hosts the app and returns the URL.
func (s *Server) getQuickTunnelURL(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	url, err := s.appService.GetQuickTunnelURL(c.Request.Context(), id, nodeID)
	if err != nil {
		s.handleServiceError(c, "get quick tunnel URL", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// createQuickTunnelRequest is the body for POST /api/apps/:id/quick-tunnel
type createQuickTunnelRequest struct {
	Service string `json:"service" binding:"required"`
	Port    int    `json:"port" binding:"required,min=1,max=65535"`
}

// createQuickTunnelForApp adds a Quick Tunnel to an app that has no tunnel.
func (s *Server) createQuickTunnelForApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	nodeID := getNodeIDFromContext(c)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "node_id is required"})
		return
	}

	var req createQuickTunnelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.WarnContext(c.Request.Context(), "invalid create quick tunnel request", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Details: fmt.Sprintf("service (required) and port (%d-%d) are required", constants.MinPort, constants.MaxPort)})
		return
	}
	if strings.TrimSpace(req.Service) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "service is required"})
		return
	}

	// Create background job for Quick Tunnel creation (async operation)
	job, err := s.appService.CreateQuickTunnelForAppAsync(c.Request.Context(), id, strings.TrimSpace(req.Service), req.Port)
	if err != nil {
		s.handleServiceError(c, "create quick tunnel job", err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  job.ID,
		"app_id":  job.AppID,
		"status":  job.Status,
		"message": "Quick Tunnel creation started in background",
	})
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID", Details: domain.PublicMessage(err)})
		return
	}

	version, err := httputil.ValidateAndGetVersion(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid version", Details: domain.PublicMessage(err)})
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID", Details: domain.PublicMessage(err)})
		return
	}

	targetVersion, err := httputil.ValidateAndGetVersion(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid version", Details: domain.PublicMessage(err)})
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
