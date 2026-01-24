package system

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/selfhostly/internal/docker"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemStats represents comprehensive system and container statistics
type SystemStats struct {
	NodeID     string          `json:"node_id"`
	NodeName   string          `json:"node_name"`
	CPU        CPUStats        `json:"cpu"`
	Memory     MemoryStats     `json:"memory"`
	Disk       DiskStats       `json:"disk"`
	Docker     DockerStats     `json:"docker"`
	Containers []ContainerInfo `json:"containers"`
	Timestamp  time.Time       `json:"timestamp"`
}

// CPUStats represents CPU usage statistics
type CPUStats struct {
	UsagePercent float64 `json:"usage_percent"`
	Cores        int     `json:"cores"`
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	Total        uint64  `json:"total_bytes"`
	Used         uint64  `json:"used_bytes"`
	Free         uint64  `json:"free_bytes"`
	Available    uint64  `json:"available_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

// DiskStats represents disk usage statistics
type DiskStats struct {
	Total        uint64  `json:"total_bytes"`
	Used         uint64  `json:"used_bytes"`
	Free         uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
	Path         string  `json:"path"`
}

// DockerStats represents Docker daemon statistics
type DockerStats struct {
	TotalContainers int    `json:"total_containers"`
	Running         int    `json:"running"`
	Stopped         int    `json:"stopped"`
	Paused          int    `json:"paused"`
	Images          int    `json:"images"`
	Version         string `json:"version"`
}

// ContainerInfo represents detailed container information with resource stats
type ContainerInfo struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	AppName      string  `json:"app_name"`
	Status       string  `json:"status"`
	State        string  `json:"state"`
	CPUPercent   float64 `json:"cpu_percent"`
	MemoryUsage  uint64  `json:"memory_usage_bytes"`
	MemoryLimit  uint64  `json:"memory_limit_bytes"`
	NetworkRx    uint64  `json:"network_rx_bytes"`
	NetworkTx    uint64  `json:"network_tx_bytes"`
	BlockRead    uint64  `json:"block_read_bytes"`
	BlockWrite   uint64  `json:"block_write_bytes"`
	CreatedAt    string  `json:"created_at"`
	RestartCount int     `json:"restart_count"`
}

// Collector collects system and container statistics
type Collector struct {
	appsDir         string
	dockerManager   *docker.Manager
	commandExecutor docker.CommandExecutor
}

// NewCollector creates a new system stats collector
func NewCollector(appsDir string, dockerManager *docker.Manager) *Collector {
	return &Collector{
		appsDir:         appsDir,
		dockerManager:   dockerManager,
		commandExecutor: docker.NewRealCommandExecutor(),
	}
}

// GetSystemStats retrieves comprehensive system statistics
func (c *Collector) GetSystemStats() (*SystemStats, error) {
	slog.Debug("collecting system statistics")

	// Get node information
	nodeID, nodeName := c.getNodeInfo()

	// Collect all stats in parallel for performance
	cpuStats := c.getCPUStats()
	memStats := c.getMemoryStats()
	diskStats := c.getDiskStats("/")
	dockerStats := c.getDockerDaemonStats()
	containers := c.getAllContainerStats()

	stats := &SystemStats{
		NodeID:     nodeID,
		NodeName:   nodeName,
		CPU:        cpuStats,
		Memory:     memStats,
		Disk:       diskStats,
		Docker:     dockerStats,
		Containers: containers,
		Timestamp:  time.Now(),
	}

	slog.Debug("system statistics collected successfully",
		"cpu_usage", cpuStats.UsagePercent,
		"memory_usage", memStats.UsagePercent,
		"disk_usage", diskStats.UsagePercent,
		"total_containers", len(containers))

	return stats, nil
}

// getNodeInfo retrieves node identification information
func (c *Collector) getNodeInfo() (string, string) {
	hostname, err := os.Hostname()
	if err != nil {
		slog.Warn("failed to get hostname", "error", err)
		hostname = "unknown"
	}

	// For future multi-node support, this could come from config
	nodeID := hostname
	nodeName := hostname

	return nodeID, nodeName
}

// getCPUStats retrieves CPU usage statistics
func (c *Collector) getCPUStats() CPUStats {
	// Get CPU count
	cores, err := cpu.Counts(true)
	if err != nil {
		slog.Warn("failed to get CPU count", "error", err)
		cores = 1
	}

	// Get CPU usage percentage (averaged over 1 second)
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		slog.Warn("failed to get CPU usage", "error", err)
		return CPUStats{UsagePercent: 0, Cores: cores}
	}

	usagePercent := 0.0
	if len(percentages) > 0 {
		usagePercent = percentages[0]
	}

	return CPUStats{
		UsagePercent: usagePercent,
		Cores:        cores,
	}
}

// getMemoryStats retrieves memory usage statistics
func (c *Collector) getMemoryStats() MemoryStats {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		slog.Warn("failed to get memory stats", "error", err)
		return MemoryStats{}
	}

	return MemoryStats{
		Total:        vmStat.Total,
		Used:         vmStat.Used,
		Free:         vmStat.Free,
		Available:    vmStat.Available,
		UsagePercent: vmStat.UsedPercent,
	}
}

// getDiskStats retrieves disk usage statistics for a given path
func (c *Collector) getDiskStats(path string) DiskStats {
	usage, err := disk.Usage(path)
	if err != nil {
		slog.Warn("failed to get disk stats", "path", path, "error", err)
		return DiskStats{Path: path}
	}

	return DiskStats{
		Total:        usage.Total,
		Used:         usage.Used,
		Free:         usage.Free,
		UsagePercent: usage.UsedPercent,
		Path:         path,
	}
}

// getDockerDaemonStats retrieves Docker daemon statistics
func (c *Collector) getDockerDaemonStats() DockerStats {
	stats := DockerStats{}

	// Get Docker version
	versionOutput, err := c.commandExecutor.ExecuteCommand("docker", "version", "--format", "{{.Server.Version}}")
	if err != nil {
		slog.Warn("failed to get docker version", "error", err)
		stats.Version = "unknown"
	} else {
		stats.Version = strings.TrimSpace(string(versionOutput))
	}

	// Get container counts by state
	psOutput, err := c.commandExecutor.ExecuteCommand("docker", "ps", "-a", "--format", "{{.State}}")
	if err != nil {
		slog.Warn("failed to get docker container states", "error", err)
		return stats
	}

	states := strings.Split(strings.TrimSpace(string(psOutput)), "\n")
	for _, state := range states {
		state = strings.TrimSpace(strings.ToLower(state))
		if state == "" {
			continue
		}
		stats.TotalContainers++
		switch state {
		case "running":
			stats.Running++
		case "paused":
			stats.Paused++
		default:
			stats.Stopped++
		}
	}

	// Get image count
	imagesOutput, err := c.commandExecutor.ExecuteCommand("docker", "images", "-q")
	if err != nil {
		slog.Warn("failed to get docker images", "error", err)
	} else {
		imageLines := strings.Split(strings.TrimSpace(string(imagesOutput)), "\n")
		if len(imageLines) > 0 && imageLines[0] != "" {
			stats.Images = len(imageLines)
		}
	}

	return stats
}

// getAllContainerStats retrieves statistics for all containers system-wide
func (c *Collector) getAllContainerStats() []ContainerInfo {
	var allContainers []ContainerInfo

	// Get all container IDs on the system
	output, err := c.commandExecutor.ExecuteCommand("docker", "ps", "-a", "-q")
	if err != nil {
		slog.Warn("failed to get all container IDs", "error", err)
		return allContainers
	}

	containerIDs := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Build a map of Selfhostly-managed apps
	managedApps := c.getManagedAppsMap()

	for _, containerID := range containerIDs {
		containerID = strings.TrimSpace(containerID)
		if containerID == "" {
			continue
		}

		// Get container project/app name from labels
		appName := c.getContainerAppName(containerID, managedApps)

		containerInfo := c.getContainerInfo(containerID, appName)
		if containerInfo != nil {
			allContainers = append(allContainers, *containerInfo)
		}
	}

	slog.Debug("collected container stats", "count", len(allContainers))
	return allContainers
}

// getManagedAppsMap returns a set of Selfhostly-managed app names
func (c *Collector) getManagedAppsMap() map[string]bool {
	managedApps := make(map[string]bool)

	entries, err := os.ReadDir(c.appsDir)
	if err != nil {
		return managedApps
	}

	for _, entry := range entries {
		if entry.IsDir() {
			composePath := filepath.Join(c.appsDir, entry.Name(), "docker-compose.yml")
			if _, err := os.Stat(composePath); err == nil {
				managedApps[entry.Name()] = true
			}
		}
	}

	return managedApps
}

// getContainerAppName determines the app/project name for a container
func (c *Collector) getContainerAppName(containerID string, managedApps map[string]bool) string {
	// Try to get the compose project name from labels
	output, err := c.commandExecutor.ExecuteCommand(
		"docker", "inspect", "--format", "{{index .Config.Labels \"com.docker.compose.project\"}}", containerID)
	if err == nil {
		projectName := strings.TrimSpace(string(output))
		if projectName != "" && projectName != "<no value>" {
			// Check if this is a managed app
			if managedApps[projectName] {
				return projectName
			}
			// Return project name with indicator it's external
			return projectName
		}
	}

	// If no project label, mark as unmanaged
	return "unmanaged"
}

// getContainerInfo retrieves detailed information for a specific container
func (c *Collector) getContainerInfo(containerID, appName string) *ContainerInfo {
	// Get container inspect data
	inspectOutput, err := c.commandExecutor.ExecuteCommand(
		"docker", "inspect", "--format",
		"{{.Name}}|{{.State.Status}}|{{.State.Running}}|{{.Created}}|{{.RestartCount}}",
		containerID)
	if err != nil {
		slog.Debug("failed to inspect container", "containerID", containerID, "error", err)
		return nil
	}

	parts := strings.Split(strings.TrimSpace(string(inspectOutput)), "|")
	if len(parts) < 5 {
		return nil
	}

	containerName := strings.TrimPrefix(parts[0], "/")
	status := parts[1]
	isRunning := parts[2] == "true"
	createdAt := parts[3]
	restartCount := 0
	fmt.Sscanf(parts[4], "%d", &restartCount)

	state := "stopped"
	if isRunning {
		state = "running"
	} else if status == "paused" {
		state = "paused"
	}

	containerInfo := &ContainerInfo{
		ID:           containerID,
		Name:         containerName,
		AppName:      appName,
		Status:       status,
		State:        state,
		CreatedAt:    createdAt,
		RestartCount: restartCount,
	}

	// Only get resource stats for running containers
	if isRunning {
		c.populateContainerStats(containerInfo, containerID)
	}

	return containerInfo
}

// populateContainerStats adds resource usage statistics to container info
func (c *Collector) populateContainerStats(containerInfo *ContainerInfo, containerID string) {
	// Get stats (no-stream for single snapshot)
	statsOutput, err := c.commandExecutor.ExecuteCommand(
		"docker", "stats", containerID, "--no-stream", "--no-trunc", "--format",
		"{{.CPUPerc}}|{{.MemUsage}}|{{.NetIO}}|{{.BlockIO}}")
	if err != nil {
		slog.Debug("failed to get container stats", "containerID", containerID, "error", err)
		return
	}

	rawStats := strings.TrimSpace(string(statsOutput))
	parts := strings.Split(rawStats, "|")
	
	if len(parts) < 4 {
		slog.Warn("unexpected stats format", "containerID", containerID, "parts", len(parts), "output", rawStats)
		return
	}

	// Parse CPU percentage
	containerInfo.CPUPercent = parsePercentage(parts[0])

	// Parse memory usage (e.g., "100MiB / 2GiB")
	memParts := strings.Split(parts[1], " / ")
	if len(memParts) == 2 {
		containerInfo.MemoryUsage = parseBytes(strings.TrimSpace(memParts[0]))
		containerInfo.MemoryLimit = parseBytes(strings.TrimSpace(memParts[1]))
	}

	// Parse network I/O (e.g., "1.2MB / 3.4MB")
	netParts := strings.Split(parts[2], " / ")
	if len(netParts) == 2 {
		containerInfo.NetworkRx = parseBytes(strings.TrimSpace(netParts[0]))
		containerInfo.NetworkTx = parseBytes(strings.TrimSpace(netParts[1]))
	}

	// Parse block I/O (e.g., "5.6MB / 7.8MB")
	blockParts := strings.Split(parts[3], " / ")
	if len(blockParts) == 2 {
		containerInfo.BlockRead = parseBytes(strings.TrimSpace(blockParts[0]))
		containerInfo.BlockWrite = parseBytes(strings.TrimSpace(blockParts[1]))
	}
}

// Helper functions from docker/stats.go
func parsePercentage(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	var val float64
	fmt.Sscanf(s, "%f", &val)
	return val
}

func parseBytes(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "0B" || s == "0" {
		return 0
	}

	// Handle formats like "0B", "123.4MiB", "1.2GB", etc.
	var numStr string
	var unit string
	
	// Extract numeric part (including decimal point)
	for i, ch := range s {
		if (ch >= '0' && ch <= '9') || ch == '.' {
			numStr += string(ch)
		} else {
			unit = s[i:]
			break
		}
	}

	if numStr == "" {
		return 0
	}

	var num float64
	n, err := fmt.Sscanf(numStr, "%f", &num)
	if err != nil || n != 1 {
		return 0
	}

	// Normalize unit (remove spaces and convert to uppercase)
	unit = strings.ToUpper(strings.TrimSpace(unit))
	
	// Handle both SI (MB, GB) and binary (MiB, GiB) units
	var multiplier float64
	switch unit {
	case "B":
		multiplier = 1
	case "K", "KB":
		multiplier = 1000
	case "KIB":
		multiplier = 1024
	case "M", "MB":
		multiplier = 1000 * 1000
	case "MIB":
		multiplier = 1024 * 1024
	case "G", "GB":
		multiplier = 1000 * 1000 * 1000
	case "GIB":
		multiplier = 1024 * 1024 * 1024
	case "T", "TB":
		multiplier = 1000 * 1000 * 1000 * 1000
	case "TIB":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		multiplier = 1
	}

	return uint64(num * multiplier)
}
