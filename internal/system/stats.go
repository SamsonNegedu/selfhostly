package system

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/selfhostly/internal/db"
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
	IsManaged    bool    `json:"is_managed"`     // Whether container belongs to an app managed by our system
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
	database        *db.DB
}

// NewCollector creates a new system stats collector
func NewCollector(appsDir string, dockerManager *docker.Manager, database *db.DB) *Collector {
	return &Collector{
		appsDir:         appsDir,
		dockerManager:   dockerManager,
		commandExecutor: docker.NewRealCommandExecutor(),
		database:        database,
	}
}

// GetSystemStats retrieves comprehensive system statistics
func (c *Collector) GetSystemStats() (*SystemStats, error) {
	slog.Debug("collecting system statistics")

	// Get node information
	nodeID, nodeName := c.getNodeInfo()

	// Collect all stats in parallel using goroutines for better performance
	var cpuStats CPUStats
	var memStats MemoryStats
	var diskStats DiskStats
	var dockerStats DockerStats
	var containers []ContainerInfo

	// Use sync.WaitGroup for proper synchronization
	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		cpuStats = c.getCPUStats()
	}()

	go func() {
		defer wg.Done()
		memStats = c.getMemoryStats()
	}()

	go func() {
		defer wg.Done()
		diskStats = c.getDiskStats("/")
	}()

	go func() {
		defer wg.Done()
		dockerStats = c.getDockerDaemonStats()
	}()

	go func() {
		defer wg.Done()
		containers = c.getAllContainerStats()
	}()

	// Wait for all goroutines to complete
	wg.Wait()

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

	// Get CPU usage percentage (instant measurement, no blocking)
	// Using 0 duration returns the percentage calculated since the last call
	// This is much faster than blocking for 1 second
	percentages, err := cpu.Percent(0, false)
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
	// Initialize as empty slice (not nil) so it serializes as [] instead of null in JSON
	allContainers := []ContainerInfo{}

	// Get all container IDs on the system
	output, err := c.commandExecutor.ExecuteCommand("docker", "ps", "-a", "-q")
	if err != nil {
		slog.Warn("failed to get all container IDs", "error", err)
		return allContainers
	}

	containerIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(containerIDs) == 0 || (len(containerIDs) == 1 && strings.TrimSpace(containerIDs[0]) == "") {
		return allContainers
	}

	// Clean up container IDs
	var validContainerIDs []string
	for _, id := range containerIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			validContainerIDs = append(validContainerIDs, id)
		}
	}

	if len(validContainerIDs) == 0 {
		return allContainers
	}

	// Build a map of Selfhostly-managed apps
	managedApps := c.getManagedAppsMap()

	// OPTIMIZATION: Batch all docker inspect calls into a single command
	// This is MUCH faster than calling docker inspect for each container individually
	inspectData := c.batchGetContainerInspectData(validContainerIDs)
	slog.Debug("batch inspect completed", "requested", len(validContainerIDs), "received", len(inspectData))

	// Get stats for ALL running containers in ONE command (much faster)
	statsMap := c.getAllRunningContainerStats()

	for _, containerID := range validContainerIDs {
		inspect, hasInspect := inspectData[containerID]
		if !hasInspect {
			slog.Debug("container not found in inspect data", "containerID", containerID)
			continue
		}

		// Determine app name from inspect data
		appName := c.determineAppNameFromInspect(inspect, managedApps)

		containerInfo := c.buildContainerInfo(containerID, appName, inspect, managedApps)
		if containerInfo != nil {
			// Apply pre-fetched stats if container is running
			if stats, found := statsMap[containerID]; found {
				containerInfo.CPUPercent = stats.CPUPercent
				containerInfo.MemoryUsage = stats.MemoryUsage
				containerInfo.MemoryLimit = stats.MemoryLimit
				containerInfo.NetworkRx = stats.NetworkRx
				containerInfo.NetworkTx = stats.NetworkTx
				containerInfo.BlockRead = stats.BlockRead
				containerInfo.BlockWrite = stats.BlockWrite
			}
			allContainers = append(allContainers, *containerInfo)
		}
	}

	slog.Debug("collected container stats", "count", len(allContainers), "inspected", len(inspectData))
	return allContainers
}

// ContainerStats holds resource usage statistics
type ContainerStats struct {
	CPUPercent  float64
	MemoryUsage uint64
	MemoryLimit uint64
	NetworkRx   uint64
	NetworkTx   uint64
	BlockRead   uint64
	BlockWrite  uint64
}

// ContainerInspectData holds data from docker inspect
type ContainerInspectData struct {
	Name         string
	ProjectLabel string
	ServiceLabel string
	Status       string
	IsRunning    bool
	CreatedAt    string
	RestartCount int
}

// batchGetContainerInspectData gets inspect data for all containers in a single command
func (c *Collector) batchGetContainerInspectData(containerIDs []string) map[string]ContainerInspectData {
	inspectMap := make(map[string]ContainerInspectData)

	if len(containerIDs) == 0 {
		return inspectMap
	}

	// Build docker inspect command for all containers at once
	// Note: Using slice of .Id to get short ID (first 12 chars) to match with docker ps -q output
	args := []string{"inspect", "--format", 
		"{{slice .Id 0 12}}|{{.Name}}|{{index .Config.Labels \"com.docker.compose.project\"}}|{{index .Config.Labels \"com.docker.compose.service\"}}|{{.State.Status}}|{{.State.Running}}|{{.Created}}|{{.RestartCount}}"}
	args = append(args, containerIDs...)

	output, err := c.commandExecutor.ExecuteCommand("docker", args...)
	if err != nil {
		slog.Warn("failed to batch inspect containers", "error", err, "containerCount", len(containerIDs))
		return inspectMap
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	slog.Debug("parsed batch inspect output", "lineCount", len(lines), "containerCount", len(containerIDs))
	
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) < 8 {
			slog.Debug("skipping invalid inspect line", "line", line, "partCount", len(parts))
			continue
		}

		containerID := strings.TrimSpace(parts[0])
		name := strings.TrimPrefix(strings.TrimSpace(parts[1]), "/")
		projectLabel := strings.TrimSpace(parts[2])
		serviceLabel := strings.TrimSpace(parts[3])
		status := strings.TrimSpace(parts[4])
		isRunning := strings.TrimSpace(parts[5]) == "true"
		createdAt := strings.TrimSpace(parts[6])
		restartCount := 0
		fmt.Sscanf(parts[7], "%d", &restartCount)

		inspectMap[containerID] = ContainerInspectData{
			Name:         name,
			ProjectLabel: projectLabel,
			ServiceLabel: serviceLabel,
			Status:       status,
			IsRunning:    isRunning,
			CreatedAt:    createdAt,
			RestartCount: restartCount,
		}
	}

	return inspectMap
}

// determineAppNameFromInspect determines the app/project name from inspect data
func (c *Collector) determineAppNameFromInspect(inspect ContainerInspectData, managedApps map[string]bool) string {
	// Check if container has Docker Compose labels
	hasComposeLabels := inspect.ServiceLabel != "" && inspect.ServiceLabel != "<no value>"

	// Strategy 1: Check Docker Compose project label (most reliable)
	if inspect.ProjectLabel != "" && inspect.ProjectLabel != "<no value>" {
		// Check if this is a managed app
		if managedApps[inspect.ProjectLabel] {
			return inspect.ProjectLabel
		}
		// Return project name even if not managed (external compose project)
		return inspect.ProjectLabel
	}

	// Strategy 2: Match container name pattern, but ONLY if container has Compose labels
	if hasComposeLabels {
		for appName := range managedApps {
			// Check if container name starts with app name followed by a hyphen
			if strings.HasPrefix(inspect.Name, appName+"-") {
				return appName
			}
			// Also check exact match (for containers with explicit container_name)
			if inspect.Name == appName {
				return appName
			}
		}
	}

	// No match found - container is unmanaged
	return "unmanaged"
}

// buildContainerInfo builds a ContainerInfo from inspect data
func (c *Collector) buildContainerInfo(containerID, appName string, inspect ContainerInspectData, managedApps map[string]bool) *ContainerInfo {
	state := "stopped"
	if inspect.IsRunning {
		state = "running"
	} else if inspect.Status == "paused" {
		state = "paused"
	}

	return &ContainerInfo{
		ID:           containerID,
		Name:         inspect.Name,
		AppName:      appName,
		IsManaged:    managedApps[appName],
		Status:       inspect.Status,
		State:        state,
		CreatedAt:    inspect.CreatedAt,
		RestartCount: inspect.RestartCount,
	}
}

// getAllRunningContainerStats gets stats for ALL running containers in one command
func (c *Collector) getAllRunningContainerStats() map[string]ContainerStats {
	statsMap := make(map[string]ContainerStats)

	// Get stats for ALL containers at once (much faster than per-container)
	// Note: NOT using --no-trunc so we get short IDs (12 chars) to match with docker ps -q
	statsOutput, err := c.commandExecutor.ExecuteCommand(
		"docker", "stats", "--no-stream", "--format",
		"{{.ID}}|{{.CPUPerc}}|{{.MemUsage}}|{{.NetIO}}|{{.BlockIO}}")
	if err != nil {
		slog.Warn("failed to get bulk container stats", "error", err)
		return statsMap
	}

	lines := strings.Split(strings.TrimSpace(string(statsOutput)), "\n")
	slog.Debug("parsed bulk stats output", "lineCount", len(lines))
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}

		containerID := parts[0]
		stats := ContainerStats{
			CPUPercent: parsePercentage(parts[1]),
		}

		// Parse memory usage (e.g., "100MiB / 2GiB")
		memParts := strings.Split(parts[2], " / ")
		if len(memParts) == 2 {
			stats.MemoryUsage = parseBytes(strings.TrimSpace(memParts[0]))
			stats.MemoryLimit = parseBytes(strings.TrimSpace(memParts[1]))
		}

		// Parse network I/O (e.g., "1.2MB / 3.4MB")
		netParts := strings.Split(parts[3], " / ")
		if len(netParts) == 2 {
			stats.NetworkRx = parseBytes(strings.TrimSpace(netParts[0]))
			stats.NetworkTx = parseBytes(strings.TrimSpace(netParts[1]))
		}

		// Parse block I/O (e.g., "5.6MB / 7.8MB")
		blockParts := strings.Split(parts[4], " / ")
		if len(blockParts) == 2 {
			stats.BlockRead = parseBytes(strings.TrimSpace(blockParts[0]))
			stats.BlockWrite = parseBytes(strings.TrimSpace(blockParts[1]))
		}

		statsMap[containerID] = stats
	}

	return statsMap
}

// getManagedAppsMap returns a set of Selfhostly-managed app names from the database
func (c *Collector) getManagedAppsMap() map[string]bool {
	managedApps := make(map[string]bool)

	// Use database as source of truth for managed apps
	if c.database != nil {
		apps, err := c.database.GetAllApps()
		if err == nil {
			for _, app := range apps {
				managedApps[app.Name] = true
			}
			slog.Debug("loaded managed apps from database", "count", len(managedApps))
			return managedApps
		}
		slog.Warn("failed to get apps from database, falling back to directory scan", "error", err)
	}

	// Fallback to directory structure if database is unavailable
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
