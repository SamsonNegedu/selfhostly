package http

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/system"
)

// getSystemStats returns comprehensive system and container statistics
func (s *Server) getSystemStats(c *gin.Context) {
	slog.DebugContext(c.Request.Context(), "fetching system statistics")

	// Create system stats collector
	collector := system.NewCollector(s.config.AppsDir, s.dockerManager)

	// Get system stats
	stats, err := collector.GetSystemStats()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get system stats", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve system statistics",
			Details: err.Error(),
		})
		return
	}

	slog.DebugContext(c.Request.Context(), "system statistics retrieved successfully",
		"cpu", stats.CPU.UsagePercent,
		"memory", stats.Memory.UsagePercent,
		"containers", len(stats.Containers))

	c.JSON(http.StatusOK, stats)
}

// restartContainer restarts a specific container by ID
func (s *Server) restartContainer(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid container ID"})
		return
	}

	slog.InfoContext(c.Request.Context(), "restarting container", "containerID", containerID)

	dockerManager := docker.NewManager(s.config.AppsDir)
	if err := dockerManager.RestartContainer(containerID); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to restart container",
			"containerID", containerID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to restart container",
			Details: err.Error(),
		})
		return
	}

	slog.InfoContext(c.Request.Context(), "container restarted successfully", "containerID", containerID)
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

	slog.InfoContext(c.Request.Context(), "stopping container", "containerID", containerID)

	dockerManager := docker.NewManager(s.config.AppsDir)
	if err := dockerManager.StopContainer(containerID); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to stop container",
			"containerID", containerID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to stop container",
			Details: err.Error(),
		})
		return
	}

	slog.InfoContext(c.Request.Context(), "container stopped successfully", "containerID", containerID)
	c.JSON(http.StatusOK, gin.H{
		"message":      "Container stopped successfully",
		"container_id": containerID,
	})
}

// getDebugDockerStats returns raw docker stats output for debugging
func (s *Server) getDebugDockerStats(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid container ID"})
		return
	}

	slog.InfoContext(c.Request.Context(), "fetching debug docker stats", "containerID", containerID)

	dockerManager := docker.NewManager(s.config.AppsDir)
	executor := dockerManager.GetCommandExecutor()
	
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
		"note":          "This shows the raw docker stats output to help debug parsing issues",
	})
}
