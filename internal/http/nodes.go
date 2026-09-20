package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

// NodeResponse represents a node without sensitive information (API key excluded)
type NodeResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	APIEndpoint string     `json:"api_endpoint"`
	IsPrimary   bool       `json:"is_primary"`
	Status      string     `json:"status"`
	LastSeen    *time.Time `json:"last_seen"`
	// Health of the last checks. LastLatencyMs is 0 until a check has succeeded.
	LastHealthCheck     *time.Time `json:"last_health_check"`
	LastLatencyMs       int        `json:"last_latency_ms"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// toNodeResponse converts a db.Node to NodeResponse (excluding API key)
func toNodeResponse(node *db.Node) *NodeResponse {
	return &NodeResponse{
		ID:                  node.ID,
		Name:                node.Name,
		APIEndpoint:         node.APIEndpoint,
		IsPrimary:           node.IsPrimary,
		Status:              node.Status,
		LastSeen:            node.LastSeen,
		LastHealthCheck:     node.LastHealthCheck,
		LastLatencyMs:       node.LastLatencyMs,
		ConsecutiveFailures: node.ConsecutiveFailures,
		CreatedAt:           node.CreatedAt,
		UpdatedAt:           node.UpdatedAt,
	}
}

// toNodeResponseList converts a slice of db.Node to NodeResponse
func toNodeResponseList(nodes []*db.Node) []*NodeResponse {
	result := make([]*NodeResponse, len(nodes))
	for i, node := range nodes {
		result[i] = toNodeResponse(node)
	}
	return result
}

// listNodes returns all nodes in the cluster (API keys excluded for security)
func (s *Server) listNodes(c *gin.Context) {
	nodes, err := s.nodeService.ListNodes(c.Request.Context())
	if err != nil {
		s.handleServiceError(c, "list nodes", err)
		return
	}

	// Convert to response format without API keys
	c.JSON(http.StatusOK, toNodeResponseList(nodes))
}

// registerNode registers a new node in the cluster (API key excluded from response for security)
func (s *Server) registerNode(c *gin.Context) {
	var req domain.RegisterNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Details: domain.PublicMessage(err),
		})
		return
	}

	node, err := s.nodeService.RegisterNode(c.Request.Context(), req)
	if err != nil {
		s.handleServiceError(c, "register node", err)
		return
	}

	// Return response without API key
	setAuditTarget(c, targetNode, node.ID, node.Name)
	c.JSON(http.StatusCreated, toNodeResponse(node))
}

// getNode returns details of a specific node (API key excluded for security)
func (s *Server) getNode(c *gin.Context) {
	nodeID := c.Param("id")

	node, err := s.nodeService.GetNode(c.Request.Context(), nodeID)
	if err != nil {
		s.handleServiceError(c, "get node", err)
		return
	}

	// Return response without API key
	c.JSON(http.StatusOK, toNodeResponse(node))
}

// updateNode updates a node's information (API key excluded from response for security)
func (s *Server) updateNode(c *gin.Context) {
	nodeID := c.Param("id")

	var req domain.UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Details: domain.PublicMessage(err),
		})
		return
	}

	node, err := s.nodeService.UpdateNode(c.Request.Context(), nodeID, req)
	if err != nil {
		s.handleServiceError(c, "update node", err)
		return
	}

	// Return response without API key
	c.JSON(http.StatusOK, toNodeResponse(node))
}

// deleteNode removes a node from the cluster
func (s *Server) deleteNode(c *gin.Context) {
	nodeID := c.Param("id")

	err := s.nodeService.DeleteNode(c.Request.Context(), nodeID, c.Query("force") == "true")
	if err != nil {
		s.handleServiceError(c, "delete node", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Node deleted successfully",
		"nodeID":  nodeID,
	})
}

// checkNodeHealth performs a health check on a specific node
func (s *Server) checkNodeHealth(c *gin.Context) {
	nodeID := c.Param("id")

	err := s.nodeService.HealthCheckNode(c.Request.Context(), nodeID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Node health check failed",
			Details: domain.PublicMessage(err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Node is healthy",
		"nodeID":  nodeID,
	})
}

// getCurrentNodeInfo returns information about the current node (API key excluded for security)
func (s *Server) getCurrentNodeInfo(c *gin.Context) {
	node, err := s.nodeService.GetCurrentNodeInfo(c.Request.Context())
	if err != nil {
		s.handleServiceError(c, "get current node info", err)
		return
	}

	// Return response without API key
	c.JSON(http.StatusOK, toNodeResponse(node))
}

// getSettingsForNode returns settings (for inter-node sync)
// This endpoint is protected by node authentication middleware
func (s *Server) getSettingsForNode(c *gin.Context) {
	// This is an internal endpoint for secondary nodes to fetch settings
	// Only the primary node should respond to this

	settings, err := s.nodeService.GetSettings(c.Request.Context())
	if err != nil {
		s.handleServiceError(c, "get settings", err)
		return
	}

	c.JSON(http.StatusOK, settings)
}

// createJoinToken issues a single-use token a new secondary node can present instead of the
// shared registration token. It is shown once and only its hash is kept.
func (s *Server) createJoinToken(c *gin.Context) {
	if !s.config.Node.IsPrimary {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Join tokens are issued by the primary node"})
		return
	}
	tok, expires, err := s.securityService.CreateJoinToken(c.Request.Context())
	if err != nil {
		s.handleServiceError(c, "create join token", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"token":      tok,
		"expires_at": expires,
		"usage":      "On the secondary set REGISTRATION_TOKEN to this value and PRIMARY_NODE_URL to the primary's URL. The token works once.",
	})
}

// revokeSessions invalidates every browser session issued so far; everyone must log in again.
func (s *Server) revokeSessions(c *gin.Context) {
	at, err := s.securityService.RevokeSessions(c.Request.Context())
	if err != nil {
		s.handleServiceError(c, "revoke sessions", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "All sessions issued before now are revoked", "revoked_before": at})
}

// listAudit returns the most recent state-changing requests
func (s *Server) listAudit(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	entries, err := s.securityService.ListAudit(c.Request.Context(), limit)
	if err != nil {
		s.handleServiceError(c, "read audit log", err)
		return
	}
	c.JSON(http.StatusOK, entries)
}
