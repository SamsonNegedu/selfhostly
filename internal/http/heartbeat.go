package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/apipaths"
)

// sendNodeHeartbeat allows a node to announce it's online to the primary
// This resets health check failures and marks the node as online
// Protected by node authentication middleware (X-Node-ID, X-Node-API-Key)
func (s *Server) sendNodeHeartbeat(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Node ID is required"})
		return
	}

	if err := s.nodeService.NodeHeartbeat(c.Request.Context(), nodeID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to process heartbeat",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Heartbeat received",
		"nodeID":  nodeID,
	})
}

// manualCheckNode triggers a manual health check on a specific node
// Useful for immediately checking a node that may have come back online
// Protected by user authentication (GitHub OAuth)
func (s *Server) manualCheckNode(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Node ID is required"})
		return
	}

	// Trigger health check
	err := s.nodeService.HealthCheckNode(c.Request.Context(), nodeID)
	
	// Fetch updated node status
	node, getErr := s.nodeService.GetNode(c.Request.Context(), nodeID)
	if getErr != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get node status after health check",
			Details: getErr.Error(),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Health check completed with errors",
			"node":    toNodeResponse(node),
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Health check completed successfully",
		"node":    toNodeResponse(node),
	})
}

// sendStartupHeartbeat sends a heartbeat to the primary node when this secondary node starts
// This is called as a goroutine from startBackgroundTasks()
func (s *Server) sendStartupHeartbeat() {
	// Wait a bit for server to fully initialize
	time.Sleep(2 * time.Second)
	
	primaryURL := s.config.Node.PrimaryNodeURL
	// Use internal API endpoint which requires node authentication
	heartbeatURL := primaryURL + apipaths.NodeHeartbeat(s.config.Node.ID)
	
	slog.Info("sending startup heartbeat to primary", "url", heartbeatURL)
	
	// Send heartbeat with node authentication headers
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", heartbeatURL, nil)
	if err != nil {
		slog.Warn("failed to create heartbeat request", "error", err)
		return
	}
	
	// Add node authentication headers
	req.Header.Set("X-Node-ID", s.config.Node.ID)
	req.Header.Set("X-Node-API-Key", s.config.Node.APIKey)
	
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("failed to send heartbeat to primary", "error", err, "url", heartbeatURL)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		slog.Info("startup heartbeat sent successfully to primary")
	} else {
		body := make([]byte, 512)
		n, _ := resp.Body.Read(body)
		slog.Warn("heartbeat failed", "status", resp.StatusCode, "response", string(body[:n]))
	}
}
