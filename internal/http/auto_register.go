package http

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

// AutoRegisterRequest represents a node auto-registration request
type AutoRegisterRequest struct {
	ID          string `json:"id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	APIEndpoint string `json:"api_endpoint" binding:"required"`
	APIKey      string `json:"api_key" binding:"required"`
	Token       string `json:"token" binding:"required"` // Registration token for authentication
}

// autoRegisterNode handles auto-registration of secondary nodes
// Protected by node authentication middleware - uses registration token
func (s *Server) autoRegisterNode(c *gin.Context) {
	var req AutoRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Details: domain.PublicMessage(err),
		})
		return
	}

	// Validate registration token
	if s.config.Node.RegistrationToken == "" {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Auto-registration not configured",
			Details: "REGISTRATION_TOKEN not set on primary node",
		})
		return
	}

	if req.Token != s.config.Node.RegistrationToken {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Invalid registration token",
			Details: "The provided registration token does not match the primary node's token",
		})
		return
	}

	// Check if node with this ID already exists
	existingNodeByID, err := s.database.GetNode(req.ID)
	if err == nil && existingNodeByID != nil {
		// Node already registered - update it instead
		existingNodeByID.Name = req.Name
		existingNodeByID.APIEndpoint = req.APIEndpoint
		existingNodeByID.APIKey = req.APIKey
		existingNodeByID.Status = "online"

		if err := s.database.UpdateNode(existingNodeByID); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to update existing node",
				Details: domain.PublicMessage(err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Node already registered - updated successfully",
			"node_id": req.ID,
			"status":  "updated",
		})
		return
	}

	// Check if node with this name already exists
	existingNode, err := s.database.GetNodeByName(req.Name)
	if err == nil && existingNode != nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "Node name already exists",
			Details: "A node with this name is already registered with a different ID",
		})
		return
	}

	// Create new node with the provided details
	newNode := db.NewNodeWithID(req.ID, req.Name, req.APIEndpoint, req.APIKey, false)

	// Perform initial health check using nodeService
	if err := s.nodeService.HealthCheckNode(c.Request.Context(), newNode.ID); err != nil {
		slog.Warn("health check failed for auto-registered node", "name", req.Name, "error", err)
		newNode.Status = "unreachable"
	} else {
		newNode.Status = "online"
	}

	// Save to database
	if err := s.database.CreateNode(newNode); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to register node",
			Details: domain.PublicMessage(err),
		})
		return
	}

	slog.Info("node auto-registered successfully", "id", req.ID, "name", req.Name, "status", newNode.Status)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Node registered successfully",
		"node_id": req.ID,
		"status":  newNode.Status,
	})
}
