package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/validation"
)

// getSystemStats returns comprehensive system and container statistics
func (s *Server) getSystemStats(c *gin.Context) {
	stats, err := s.systemService.GetSystemStats(c.Request.Context())
	if err != nil {
		s.handleServiceError(c, "get system stats", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// restartContainer restarts a specific container by ID
func (s *Server) restartContainer(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid container ID"})
		return
	}

	if err := s.systemService.RestartContainer(c.Request.Context(), containerID); err != nil {
		s.handleServiceError(c, "restart container", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Container restarted successfully",
		"container_id": containerID,
	})
}

// stopContainer stops a specific container by ID
func (s *Server) stopContainer(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid container ID"})
		return
	}

	if err := s.systemService.StopContainer(c.Request.Context(), containerID); err != nil {
		s.handleServiceError(c, "stop container", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Container stopped successfully",
		"container_id": containerID,
	})
}

// getDebugDockerStats returns raw docker stats output for debugging
func (s *Server) getDebugDockerStats(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Container ID is required"})
		return
	}

	// Validate container ID format
	if err := validation.ValidateContainerID(containerID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid container ID format",
			Details: err.Error(),
		})
		return
	}

	executor := s.dockerManager.GetCommandExecutor()
	
	// Get raw docker stats output
	statsOutput, err := executor.ExecuteCommand(
		"docker", "stats", containerID, "--no-stream", "--no-trunc", "--format",
		"{{.CPUPerc}}|{{.MemUsage}}|{{.NetIO}}|{{.BlockIO}}")
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get docker stats",
			Details: err.Error(),
		})
		return
	}

	// Also get container name for context
	inspectOutput, _ := executor.ExecuteCommand(
		"docker", "inspect", "--format", "{{.Name}}", containerID)

	c.JSON(http.StatusOK, gin.H{
		"container_id":   containerID,
		"container_name": string(inspectOutput),
		"raw_output":     string(statsOutput),
		"note":           "This shows the raw docker stats output to help debug parsing issues",
	})
}

// deleteContainer removes a container by ID
func (s *Server) deleteContainer(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid container ID"})
		return
	}

	if err := s.systemService.DeleteContainer(c.Request.Context(), containerID); err != nil {
		s.handleServiceError(c, "delete container", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Container deleted successfully",
		"container_id": containerID,
	})
}
