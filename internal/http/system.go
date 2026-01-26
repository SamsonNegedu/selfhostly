package http

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/httputil"
	"github.com/selfhostly/internal/validation"
)

// getSystemStats returns comprehensive system and container statistics from specified nodes
func (s *Server) getSystemStats(c *gin.Context) {
	// Extract node_ids from query parameter
	nodeIDs := httputil.ParseNodeIDs(c)

	stats, err := s.systemService.GetSystemStats(c.Request.Context(), nodeIDs)
	if err != nil {
		s.handleServiceError(c, "get system stats", err)
		return
	}

	// Return array of stats (one per node)
	c.JSON(http.StatusOK, stats)
}

// getLocalSystemStats returns system stats from the local node only (for inter-node calls)
func (s *Server) getLocalSystemStats(c *gin.Context) {
	// Log internal endpoint calls for debugging
	slog.DebugContext(c.Request.Context(), "getLocalSystemStats internal request",
		"local_node_id", s.config.Node.ID,
		"remote_addr", c.RemoteIP(),
		"user_agent", c.GetHeader("User-Agent"))

	// Validate that the configured node ID exists in the database
	_, err := s.database.GetNode(s.config.Node.ID)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "configured node ID not found in database",
			"config_node_id", s.config.Node.ID,
			"error", err)
		// Continue anyway - GetSystemStats will handle the error
	}

	// Get stats for local node only (empty nodeIDs means all nodes, but we want local only)
	stats, err := s.systemService.GetSystemStats(c.Request.Context(), []string{s.config.Node.ID})
	if err != nil {
		s.handleServiceError(c, "get local system stats", err)
		return
	}

	// Return single stats object (should only be one since we requested local node only)
	if len(stats) > 0 {
		c.JSON(http.StatusOK, stats[0])
	} else {
		c.JSON(http.StatusOK, gin.H{"error": "No stats available"})
	}
}

// restartContainer restarts a specific container by ID
func (s *Server) restartContainer(c *gin.Context) {
	containerID, err := httputil.ValidateAndGetContainerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Get node_id from query parameter (default to local node if not specified)
	nodeID := httputil.GetNodeIDOrDefault(c, s.config.Node.ID)

	if err := s.systemService.RestartContainer(c.Request.Context(), containerID, nodeID); err != nil {
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
	containerID, err := httputil.ValidateAndGetContainerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Get node_id from query parameter (default to local node if not specified)
	nodeID := httputil.GetNodeIDOrDefault(c, s.config.Node.ID)

	if err := s.systemService.StopContainer(c.Request.Context(), containerID, nodeID); err != nil {
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
	containerID, err := httputil.ValidateAndGetContainerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Get node_id from query parameter (default to local node if not specified)
	nodeID := httputil.GetNodeIDOrDefault(c, s.config.Node.ID)

	if err := s.systemService.DeleteContainer(c.Request.Context(), containerID, nodeID); err != nil {
		s.handleServiceError(c, "delete container", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Container deleted successfully",
		"container_id": containerID,
	})
}
