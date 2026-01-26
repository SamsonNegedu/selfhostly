package system

import (
	"os"
	"testing"

	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
)

// setupTestCollector creates a test collector with mocked dependencies
func setupTestCollector(t *testing.T, mockExecutor docker.CommandExecutor) (*Collector, *db.DB, func()) {
	// Create temp database
	tmpDB, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp database: %v", err)
	}
	tmpDB.Close()

	database, err := db.Init(tmpDB.Name())
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Create temp apps directory
	tmpAppsDir, err := os.MkdirTemp("", "test-apps-")
	if err != nil {
		t.Fatalf("Failed to create temp apps directory: %v", err)
	}

	// Create docker manager with mocked executor
	dockerManager := docker.NewManagerWithExecutor(tmpAppsDir, mockExecutor)

	// Create a test node in the database
	testNodeID := "test-node-id"
	testNodeName := "test-node"
	testAPIKey := "test-api-key"
	testNode := db.NewNode(testNodeName, "http://localhost:8080", testAPIKey, true)
	testNode.ID = testNodeID
	if err := database.CreateNode(testNode); err != nil {
		t.Fatalf("Failed to create test node: %v", err)
	}

	// Create collector with mocked executor
	collector := NewCollectorWithExecutor(tmpAppsDir, dockerManager, database, mockExecutor, testNodeID, testNodeName)

	cleanup := func() {
		database.Close()
		os.Remove(tmpDB.Name())
		os.RemoveAll(tmpAppsDir)
	}

	return collector, database, cleanup
}

func TestCollector_GetDockerDaemonStats(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	collector, _, cleanup := setupTestCollector(t, mockExecutor)
	defer cleanup()

	// Mock Docker version command
	mockExecutor.SetMockOutput("docker", []string{"version", "--format", "{{.Server.Version}}"}, []byte("24.0.0\n"))

	// Mock docker ps -a command (container states)
	mockExecutor.SetMockOutput("docker", []string{"ps", "-a", "--format", "{{.State}}"}, []byte("running\nstopped\nrunning\npaused\n"))

	// Mock docker images -q command
	mockExecutor.SetMockOutput("docker", []string{"images", "-q"}, []byte("image1\nimage2\nimage3\n"))

	stats := collector.getDockerDaemonStats()

	// Verify Docker version
	if stats.Version != "24.0.0" {
		t.Errorf("Expected version '24.0.0', got '%s'", stats.Version)
	}

	// Verify container counts
	if stats.TotalContainers != 4 {
		t.Errorf("Expected 4 total containers, got %d", stats.TotalContainers)
	}

	if stats.Running != 2 {
		t.Errorf("Expected 2 running containers, got %d", stats.Running)
	}

	if stats.Stopped != 1 {
		t.Errorf("Expected 1 stopped container, got %d", stats.Stopped)
	}

	if stats.Paused != 1 {
		t.Errorf("Expected 1 paused container, got %d", stats.Paused)
	}

	// Verify image count
	if stats.Images != 3 {
		t.Errorf("Expected 3 images, got %d", stats.Images)
	}

	// Verify Docker commands were executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"version", "--format", "{{.Server.Version}}"}) {
		t.Error("Expected docker version command to be executed")
	}

	if !mockExecutor.AssertCommandExecuted("docker", []string{"ps", "-a", "--format", "{{.State}}"}) {
		t.Error("Expected docker ps command to be executed")
	}
}

func TestCollector_GetDockerDaemonStats_EmptyContainers(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	collector, _, cleanup := setupTestCollector(t, mockExecutor)
	defer cleanup()

	// Mock Docker version command
	mockExecutor.SetMockOutput("docker", []string{"version", "--format", "{{.Server.Version}}"}, []byte("24.0.0\n"))

	// Mock empty container list
	mockExecutor.SetMockOutput("docker", []string{"ps", "-a", "--format", "{{.State}}"}, []byte(""))

	// Mock empty images list
	mockExecutor.SetMockOutput("docker", []string{"images", "-q"}, []byte(""))

	stats := collector.getDockerDaemonStats()

	if stats.TotalContainers != 0 {
		t.Errorf("Expected 0 containers, got %d", stats.TotalContainers)
	}

	if stats.Images != 0 {
		t.Errorf("Expected 0 images, got %d", stats.Images)
	}
}

func TestCollector_BatchGetContainerInspectData(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	collector, _, cleanup := setupTestCollector(t, mockExecutor)
	defer cleanup()

	containerIDs := []string{"abc123def456", "def456ghi789"}

	// Mock docker inspect output
	inspectOutput := "abc123def456|/test-app-web-1|test-app|web|running|true|2024-01-01T00:00:00Z|0\n" +
		"def456ghi789|/test-app-db-1|test-app|db|stopped|false|2024-01-01T00:00:00Z|2\n"

	mockExecutor.SetMockOutput("docker", []string{"inspect", "--format", "{{slice .Id 0 12}}|{{.Name}}|{{index .Config.Labels \"com.docker.compose.project\"}}|{{index .Config.Labels \"com.docker.compose.service\"}}|{{.State.Status}}|{{.State.Running}}|{{.Created}}|{{.RestartCount}}", "abc123def456", "def456ghi789"}, []byte(inspectOutput))

	inspectData := collector.batchGetContainerInspectData(containerIDs)

	if len(inspectData) != 2 {
		t.Errorf("Expected 2 containers in inspect data, got %d", len(inspectData))
	}

	// Verify first container
	container1, exists := inspectData["abc123def456"]
	if !exists {
		t.Error("Expected container abc123def456 in inspect data")
	} else {
		if container1.Name != "test-app-web-1" {
			t.Errorf("Expected name 'test-app-web-1', got '%s'", container1.Name)
		}
		if container1.ProjectLabel != "test-app" {
			t.Errorf("Expected project 'test-app', got '%s'", container1.ProjectLabel)
		}
		if container1.ServiceLabel != "web" {
			t.Errorf("Expected service 'web', got '%s'", container1.ServiceLabel)
		}
		if !container1.IsRunning {
			t.Error("Expected container to be running")
		}
		if container1.RestartCount != 0 {
			t.Errorf("Expected restart count 0, got %d", container1.RestartCount)
		}
	}

	// Verify second container
	container2, exists := inspectData["def456ghi789"]
	if !exists {
		t.Error("Expected container def456ghi789 in inspect data")
	} else {
		if container2.Status != "stopped" {
			t.Errorf("Expected status 'stopped', got '%s'", container2.Status)
		}
		if container2.IsRunning {
			t.Error("Expected container to be stopped")
		}
		if container2.RestartCount != 2 {
			t.Errorf("Expected restart count 2, got %d", container2.RestartCount)
		}
	}
}

func TestCollector_BatchGetContainerInspectData_Empty(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	collector, _, cleanup := setupTestCollector(t, mockExecutor)
	defer cleanup()

	inspectData := collector.batchGetContainerInspectData([]string{})

	if len(inspectData) != 0 {
		t.Errorf("Expected empty inspect data, got %d entries", len(inspectData))
	}
}

func TestCollector_GetAllRunningContainerStats(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	collector, _, cleanup := setupTestCollector(t, mockExecutor)
	defer cleanup()

	// Mock docker stats output
	statsOutput := "abc123def456|1.5%|100MiB / 2GiB|1.2MB / 3.4MB|5.6MB / 7.8MB\n" +
		"def456ghi789|2.3%|200MiB / 4GiB|2.1MB / 4.5MB|6.7MB / 8.9MB\n"

	mockExecutor.SetMockOutput("docker", []string{"stats", "--no-stream", "--format", "{{.ID}}|{{.CPUPerc}}|{{.MemUsage}}|{{.NetIO}}|{{.BlockIO}}"}, []byte(statsOutput))

	statsMap := collector.getAllRunningContainerStats()

	if len(statsMap) != 2 {
		t.Errorf("Expected 2 containers in stats map, got %d", len(statsMap))
	}

	// Verify first container stats
	stats1, exists := statsMap["abc123def456"]
	if !exists {
		t.Error("Expected container abc123def456 in stats map")
	} else {
		if stats1.CPUPercent != 1.5 {
			t.Errorf("Expected CPU 1.5%%, got %f", stats1.CPUPercent)
		}
		// Memory: 100MiB = 100 * 1024 * 1024 = 104857600 bytes
		if stats1.MemoryUsage != 104857600 {
			t.Errorf("Expected memory usage 104857600 bytes, got %d", stats1.MemoryUsage)
		}
		// Memory limit: 2GiB = 2 * 1024 * 1024 * 1024 = 2147483648 bytes
		if stats1.MemoryLimit != 2147483648 {
			t.Errorf("Expected memory limit 2147483648 bytes, got %d", stats1.MemoryLimit)
		}
	}

	// Verify docker command was executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"stats", "--no-stream", "--format", "{{.ID}}|{{.CPUPerc}}|{{.MemUsage}}|{{.NetIO}}|{{.BlockIO}}"}) {
		t.Error("Expected docker stats command to be executed")
	}
}

func TestCollector_GetAllContainerStats(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	collector, database, cleanup := setupTestCollector(t, mockExecutor)
	defer cleanup()

	// Get the test node ID
	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Create a managed app in database
	app := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app.NodeID = testNodeID // Assign to test node
	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Mock docker ps -a -q command
	mockExecutor.SetMockOutput("docker", []string{"ps", "-a", "-q"}, []byte("abc123def456\ndef456ghi789\n"))

	// Mock docker inspect output
	inspectOutput := "abc123def456|/test-app-web-1|test-app|web|running|true|2024-01-01T00:00:00Z|0\n" +
		"def456ghi789|/unmanaged-container|unmanaged||stopped|false|2024-01-01T00:00:00Z|0\n"

	mockExecutor.SetMockOutput("docker", []string{"inspect", "--format", "{{slice .Id 0 12}}|{{.Name}}|{{index .Config.Labels \"com.docker.compose.project\"}}|{{index .Config.Labels \"com.docker.compose.service\"}}|{{.State.Status}}|{{.State.Running}}|{{.Created}}|{{.RestartCount}}", "abc123def456", "def456ghi789"}, []byte(inspectOutput))

	// Mock docker stats output (only running containers)
	statsOutput := "abc123def456|1.5%|100MiB / 2GiB|1.2MB / 3.4MB|5.6MB / 7.8MB\n"

	mockExecutor.SetMockOutput("docker", []string{"stats", "--no-stream", "--format", "{{.ID}}|{{.CPUPerc}}|{{.MemUsage}}|{{.NetIO}}|{{.BlockIO}}"}, []byte(statsOutput))

	containers := collector.getAllContainerStats("test-node-id")

	if len(containers) != 2 {
		t.Errorf("Expected 2 containers, got %d", len(containers))
	}

	// Verify managed container
	managedFound := false
	for _, container := range containers {
		if container.ID == "abc123def456" {
			managedFound = true
			if container.AppName != "test-app" {
				t.Errorf("Expected app name 'test-app', got '%s'", container.AppName)
			}
			if !container.IsManaged {
				t.Error("Expected container to be marked as managed")
			}
			if container.State != "running" {
				t.Errorf("Expected state 'running', got '%s'", container.State)
			}
			if container.CPUPercent != 1.5 {
				t.Errorf("Expected CPU 1.5%%, got %f", container.CPUPercent)
			}
		}
	}

	if !managedFound {
		t.Error("Expected to find managed container")
	}

	// Verify unmanaged container
	unmanagedFound := false
	for _, container := range containers {
		if container.ID == "def456ghi789" {
			unmanagedFound = true
			if container.AppName != "unmanaged" {
				t.Errorf("Expected app name 'unmanaged', got '%s'", container.AppName)
			}
			if container.IsManaged {
				t.Error("Expected container to be marked as unmanaged")
			}
		}
	}

	if !unmanagedFound {
		t.Error("Expected to find unmanaged container")
	}
}

func TestCollector_GetAllContainerStats_Empty(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	collector, _, cleanup := setupTestCollector(t, mockExecutor)
	defer cleanup()

	// Mock empty container list
	mockExecutor.SetMockOutput("docker", []string{"ps", "-a", "-q"}, []byte(""))

	containers := collector.getAllContainerStats("test-node-id")

	if len(containers) != 0 {
		t.Errorf("Expected 0 containers, got %d", len(containers))
	}

	// Should return empty slice, not nil
	if containers == nil {
		t.Error("Expected empty slice, got nil")
	}
}

func TestCollector_DetermineAppNameFromInspect(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	collector, database, cleanup := setupTestCollector(t, mockExecutor)
	defer cleanup()

	// Get the test node ID
	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Create managed apps
	app1 := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app1.NodeID = testNodeID // Assign to test node
	app2 := db.NewApp("another-app", "Another app", "version: '3'\nservices:\n  api:\n    image: node:latest")
	app2.NodeID = testNodeID // Assign to test node
	database.CreateApp(app1)
	database.CreateApp(app2)

	managedApps := collector.getManagedAppsMap()

	// Test with project label (most reliable)
	inspect := ContainerInspectData{
		ProjectLabel: "test-app",
		ServiceLabel:  "web",
	}
	appName := collector.determineAppNameFromInspect(inspect, managedApps)
	if appName != "test-app" {
		t.Errorf("Expected app name 'test-app', got '%s'", appName)
	}

	// Test with container name pattern (has compose labels)
	inspect2 := ContainerInspectData{
		Name:         "test-app-web-1",
		ServiceLabel: "web",
	}
	appName2 := collector.determineAppNameFromInspect(inspect2, managedApps)
	if appName2 != "test-app" {
		t.Errorf("Expected app name 'test-app', got '%s'", appName2)
	}

	// Test unmanaged container (no compose labels)
	inspect3 := ContainerInspectData{
		Name:         "some-random-container",
		ServiceLabel: "",
	}
	appName3 := collector.determineAppNameFromInspect(inspect3, managedApps)
	if appName3 != "unmanaged" {
		t.Errorf("Expected app name 'unmanaged', got '%s'", appName3)
	}
}

func TestCollector_BuildContainerInfo(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	collector, database, cleanup := setupTestCollector(t, mockExecutor)
	defer cleanup()

	// Get the test node ID
	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Create managed app
	app := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app.NodeID = testNodeID // Assign to test node
	database.CreateApp(app)

	managedApps := collector.getManagedAppsMap()

	// Test running container
	inspect := ContainerInspectData{
		Name:         "test-app-web-1",
		Status:       "running",
		IsRunning:    true,
		CreatedAt:    "2024-01-01T00:00:00Z",
		RestartCount: 5,
	}

	containerInfo := collector.buildContainerInfo("abc123def456", "test-app", "test-node-id", inspect, managedApps)

	if containerInfo == nil {
		t.Fatal("Expected container info, got nil")
	}

	if containerInfo.ID != "abc123def456" {
		t.Errorf("Expected ID 'abc123def456', got '%s'", containerInfo.ID)
	}

	if containerInfo.Name != "test-app-web-1" {
		t.Errorf("Expected name 'test-app-web-1', got '%s'", containerInfo.Name)
	}

	if containerInfo.AppName != "test-app" {
		t.Errorf("Expected app name 'test-app', got '%s'", containerInfo.AppName)
	}

	if !containerInfo.IsManaged {
		t.Error("Expected container to be marked as managed")
	}

	if containerInfo.State != "running" {
		t.Errorf("Expected state 'running', got '%s'", containerInfo.State)
	}

	if containerInfo.RestartCount != 5 {
		t.Errorf("Expected restart count 5, got %d", containerInfo.RestartCount)
	}

	// Test paused container
	inspect2 := ContainerInspectData{
		Name:         "test-app-web-1",
		Status:       "paused",
		IsRunning:    false,
		CreatedAt:    "2024-01-01T00:00:00Z",
		RestartCount: 0,
	}

	containerInfo2 := collector.buildContainerInfo("def456ghi789", "test-app", "test-node-id", inspect2, managedApps)

	if containerInfo2.State != "paused" {
		t.Errorf("Expected state 'paused', got '%s'", containerInfo2.State)
	}

	// Test stopped container
	inspect3 := ContainerInspectData{
		Name:         "test-app-web-1",
		Status:       "stopped",
		IsRunning:    false,
		CreatedAt:    "2024-01-01T00:00:00Z",
		RestartCount: 0,
	}

	containerInfo3 := collector.buildContainerInfo("ghi789jkl012", "test-app", "test-node-id", inspect3, managedApps)

	if containerInfo3.State != "stopped" {
		t.Errorf("Expected state 'stopped', got '%s'", containerInfo3.State)
	}
}

func TestCollector_GetManagedAppsMap(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	collector, database, cleanup := setupTestCollector(t, mockExecutor)
	defer cleanup()

	// Create apps in database
	app1 := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app2 := db.NewApp("another-app", "Another app", "version: '3'\nservices:\n  api:\n    image: node:latest")
	database.CreateApp(app1)
	database.CreateApp(app2)

	managedApps := collector.getManagedAppsMap()

	if len(managedApps) != 2 {
		t.Errorf("Expected 2 managed apps, got %d", len(managedApps))
	}

	if !managedApps["test-app"] {
		t.Error("Expected 'test-app' to be in managed apps")
	}

	if !managedApps["another-app"] {
		t.Error("Expected 'another-app' to be in managed apps")
	}
}

func TestParsePercentage(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"1.5%", 1.5},
		{"0.50%", 0.50},
		{"100%", 100.0},
		{"0%", 0.0},
		{"  2.3%  ", 2.3},
		{"invalid", 0.0},
		{"", 0.0},
	}

	for _, tt := range tests {
		result := parsePercentage(tt.input)
		if result != tt.expected {
			t.Errorf("parsePercentage(%q) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestParseBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"100B", 100},
		{"1KB", 1000},
		{"1KiB", 1024},
		{"1MB", 1000 * 1000},
		{"1MiB", 1024 * 1024},
		{"1GB", 1000 * 1000 * 1000},
		{"1GiB", 1024 * 1024 * 1024},
		{"2GiB", 2 * 1024 * 1024 * 1024},
		{"100MiB", 100 * 1024 * 1024},
		{"1.5MB", uint64(1.5 * 1000 * 1000)},
		{"0B", 0},
		{"0", 0},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		result := parseBytes(tt.input)
		if result != tt.expected {
			t.Errorf("parseBytes(%q) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestCollector_GetNodeInfo(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	collector, _, cleanup := setupTestCollector(t, mockExecutor)
	defer cleanup()

	nodeID, nodeName := collector.getNodeInfo()

	// Should return the values passed to the constructor
	if nodeID != "test-node-id" {
		t.Errorf("Expected node ID 'test-node-id', got '%s'", nodeID)
	}

	if nodeName != "test-node" {
		t.Errorf("Expected node name 'test-node', got '%s'", nodeName)
	}
}
