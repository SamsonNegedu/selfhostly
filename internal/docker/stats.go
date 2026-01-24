package docker

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// AppStats represents aggregated resource statistics for an app
type AppStats struct {
	AppName     string          `json:"app_name"`
	TotalCPU    float64         `json:"total_cpu_percent"`
	TotalMemory uint64          `json:"total_memory_bytes"`
	MemoryLimit uint64          `json:"memory_limit_bytes"`
	Containers  []ContainerStat `json:"containers"`
	Timestamp   time.Time       `json:"timestamp"`
}

// ContainerStat represents resource statistics for a single container
type ContainerStat struct {
	ContainerID   string  `json:"container_id"`
	ContainerName string  `json:"container_name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   uint64  `json:"memory_usage_bytes"`
	MemoryLimit   uint64  `json:"memory_limit_bytes"`
	NetworkRx     uint64  `json:"network_rx_bytes"`
	NetworkTx     uint64  `json:"network_tx_bytes"`
	BlockRead     uint64  `json:"block_read_bytes"`
	BlockWrite    uint64  `json:"block_write_bytes"`
}

// GetAppStats retrieves real-time resource statistics for all containers in an app
func (m *Manager) GetAppStats(name string) (*AppStats, error) {
	appPath := filepath.Join(m.appsDir, name)

	// Get list of container IDs for this app
	output, err := m.commandExecutor.ExecuteCommandInDir(appPath,
		"docker", "compose", "-f", "docker-compose.yml", "ps", "-q")
	if err != nil {
		// If docker compose ps fails (app stopped, no compose file, etc.), return empty stats
		// This is not an error condition - just means no containers are running
		return &AppStats{
			AppName:    name,
			Containers: []ContainerStat{},
			Timestamp:  time.Now(),
		}, nil
	}

	containerIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(containerIDs) == 0 || (len(containerIDs) == 1 && containerIDs[0] == "") {
		// No running containers
		return &AppStats{
			AppName:    name,
			Containers: []ContainerStat{},
			Timestamp:  time.Now(),
		}, nil
	}

	var totalCPU float64
	var totalMemory, totalMemoryLimit uint64
	var containerStats []ContainerStat

	for _, containerID := range containerIDs {
		containerID = strings.TrimSpace(containerID)
		if containerID == "" {
			continue
		}

		// Get container name
		nameOutput, err := m.commandExecutor.ExecuteCommand(
			"docker", "inspect", "--format", "{{.Name}}", containerID)
		if err != nil {
			continue
		}
		containerName := strings.TrimPrefix(strings.TrimSpace(string(nameOutput)), "/")

		// Get stats for each container (--no-stream for single snapshot, minimal overhead)
		statsOutput, err := m.commandExecutor.ExecuteCommand(
			"docker", "stats", containerID, "--no-stream", "--no-trunc", "--format",
			"{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}|{{.NetIO}}|{{.BlockIO}}")
		if err != nil {
			// Container might have stopped, skip it
			continue
		}

		stat := parseContainerStats(containerID, containerName, string(statsOutput))
		containerStats = append(containerStats, stat)
		totalCPU += stat.CPUPercent
		totalMemory += stat.MemoryUsage
		totalMemoryLimit += stat.MemoryLimit
	}

	return &AppStats{
		AppName:     name,
		TotalCPU:    totalCPU,
		TotalMemory: totalMemory,
		MemoryLimit: totalMemoryLimit,
		Containers:  containerStats,
		Timestamp:   time.Now(),
	}, nil
}

// parseContainerStats parses Docker stats output
// Format: "CPUPerc|MemUsage|MemPerc|NetIO|BlockIO"
// Example: "0.50%|100MiB / 2GiB|4.88%|1.2MB / 3.4MB|5.6MB / 7.8MB"
func parseContainerStats(containerID, containerName, output string) ContainerStat {
	stat := ContainerStat{
		ContainerID:   containerID,
		ContainerName: containerName,
	}

	parts := strings.Split(strings.TrimSpace(output), "|")
	if len(parts) < 5 {
		return stat
	}

	// Parse CPU percentage
	stat.CPUPercent = parsePercentage(parts[0])

	// Parse memory usage (e.g., "100MiB / 2GiB")
	memParts := strings.Split(parts[1], " / ")
	if len(memParts) == 2 {
		stat.MemoryUsage = parseBytes(strings.TrimSpace(memParts[0]))
		stat.MemoryLimit = parseBytes(strings.TrimSpace(memParts[1]))
	}

	// Parse network I/O (e.g., "1.2MB / 3.4MB")
	netParts := strings.Split(parts[3], " / ")
	if len(netParts) == 2 {
		stat.NetworkRx = parseBytes(strings.TrimSpace(netParts[0]))
		stat.NetworkTx = parseBytes(strings.TrimSpace(netParts[1]))
	}

	// Parse block I/O (e.g., "5.6MB / 7.8MB")
	blockParts := strings.Split(parts[4], " / ")
	if len(blockParts) == 2 {
		stat.BlockRead = parseBytes(strings.TrimSpace(blockParts[0]))
		stat.BlockWrite = parseBytes(strings.TrimSpace(blockParts[1]))
	}

	return stat
}

// parsePercentage parses percentage string (e.g., "0.50%" -> 0.50)
func parsePercentage(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

// parseBytes parses byte strings with units (e.g., "100MiB", "2GiB", "1.2MB")
func parseBytes(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "0B" {
		return 0
	}

	// Extract number and unit
	var numStr string
	var unit string
	for i, ch := range s {
		if (ch >= '0' && ch <= '9') || ch == '.' {
			numStr += string(ch)
		} else {
			unit = s[i:]
			break
		}
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	// Convert to bytes based on unit
	unit = strings.ToUpper(strings.TrimSpace(unit))
	switch unit {
	case "B":
		return uint64(num)
	case "KB", "KIB":
		return uint64(num * 1024)
	case "MB", "MIB":
		return uint64(num * 1024 * 1024)
	case "GB", "GIB":
		return uint64(num * 1024 * 1024 * 1024)
	case "TB", "TIB":
		return uint64(num * 1024 * 1024 * 1024 * 1024)
	default:
		return uint64(num)
	}
}
