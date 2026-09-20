package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
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

	reg, err := s.nodeService.AutoRegisterNode(c.Request.Context(), domain.AutoRegisterRequest(req))
	if errors.Is(err, domain.ErrRegistrationUnauthorized) {
		slog.Warn("node registration rejected: invalid token", "node_name", req.Name, "remote_addr", c.ClientIP())
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Invalid registration token",
			Details: "The token is wrong, expired, or was already used",
		})
		return
	}
	if err != nil {
		s.handleServiceError(c, "register node", err)
		return
	}

	if !reg.Created {
		c.JSON(http.StatusOK, gin.H{
			"message": "Node already registered - updated successfully",
			"node_id": req.ID,
			"status":  "updated",
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "Node registered successfully",
		"node_id": req.ID,
		"status":  reg.Node.Status,
	})
}
