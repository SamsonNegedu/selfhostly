package http

import (
	"net/http"
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
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// toNodeResponse converts a db.Node to NodeResponse (excluding API key)
func toNodeResponse(node *db.Node) *NodeResponse {
	return &NodeResponse{
		ID:          node.ID,
		Name:        node.Name,
		APIEndpoint: node.APIEndpoint,
		IsPrimary:   node.IsPrimary,
		Status:      node.Status,
		LastSeen:    node.LastSeen,
		CreatedAt:   node.CreatedAt,
		UpdatedAt:   node.UpdatedAt,
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list nodes",
			Details: err.Error(),
		})
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
			Details: err.Error(),
		})
		return
	}

	node, err := s.nodeService.RegisterNode(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to register node",
			Details: err.Error(),
		})
		return
	}

	// Return response without API key
	c.JSON(http.StatusCreated, toNodeResponse(node))
}

// getNode returns details of a specific node (API key excluded for security)
func (s *Server) getNode(c *gin.Context) {
	nodeID := c.Param("id")

	node, err := s.nodeService.GetNode(c.Request.Context(), nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Node not found",
			Details: err.Error(),
		})
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
			Details: err.Error(),
		})
		return
	}

	node, err := s.nodeService.UpdateNode(c.Request.Context(), nodeID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update node",
			Details: err.Error(),
		})
		return
	}

	// Return response without API key
	c.JSON(http.StatusOK, toNodeResponse(node))
}

// deleteNode removes a node from the cluster
func (s *Server) deleteNode(c *gin.Context) {
	nodeID := c.Param("id")

	err := s.nodeService.DeleteNode(c.Request.Context(), nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete node",
			Details: err.Error(),
		})
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
			Details: err.Error(),
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get current node info",
			Details: err.Error(),
		})
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

	settings, err := s.database.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get settings",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, settings)
}
