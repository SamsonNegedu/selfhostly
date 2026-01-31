package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
)

// setupTestSystemService creates a test system service with mocked dependencies
func setupTestSystemService(t *testing.T, mockExecutor docker.CommandExecutor) (domain.SystemService, *db.DB, func()) {
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

	// Use mocked executor if provided, otherwise create real one
	var dockerManager *docker.Manager
	if mockExecutor != nil {
		dockerManager = docker.NewManagerWithExecutor(tmpAppsDir, mockExecutor)
	} else {
		dockerManager = docker.NewManager(tmpAppsDir)
	}

	// Set up node configuration for multi-node support
	testNodeID := "test-node-id"
	testNodeName := "test-node"
	testAPIKey := "test-api-key"
	cfg := &config.Config{
		AppsDir: tmpAppsDir,
		Node: config.NodeConfig{
			ID:        testNodeID,
			Name:      testNodeName,
			IsPrimary: true,
			APIKey:    testAPIKey,
		},
	}

	// Create a test node in the database
	testNode := db.NewNode(testNodeName, "http://localhost:8080", testAPIKey, true)
	testNode.ID = testNodeID
	if err := database.CreateNode(testNode); err != nil {
		t.Fatalf("Failed to create test node: %v", err)
	}

	logger := slog.Default()
	service := NewSystemService(database, dockerManager, cfg, logger)

	cleanup := func() {
		database.Close()
		os.Remove(tmpDB.Name())
		os.RemoveAll(tmpAppsDir)
	}

	return service, database, cleanup
}

func TestSystemService_GetAppStats(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestSystemService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	// Get the test node ID
	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Create app
	app := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app.Status = "running"
	app.NodeID = testNodeID // Assign to test node
	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Mock docker compose ps command (returns container IDs)
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "ps", "-q"}, []byte("container-123\n"))

	// Mock docker inspect command (returns container name)
	mockExecutor.SetMockOutput("docker", []string{"inspect", "--format", "{{.Name}}", "container-123"}, []byte("/test-app-web-1\n"))

	// Mock docker stats command (returns stats)
	statsOutput := "1.5%|100MiB / 2GiB|4.88%|1.2MB / 3.4MB|5.6MB / 7.8MB"
	mockExecutor.SetMockOutput("docker", []string{"stats", "container-123", "--no-stream", "--no-trunc", "--format", "{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}|{{.NetIO}}|{{.BlockIO}}"}, []byte(statsOutput))

	// Get app stats
	stats, err := service.GetAppStats(ctx, app.ID, testNodeID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify stats
	if stats.AppName != app.Name {
		t.Errorf("Expected app name '%s', got '%s'", app.Name, stats.AppName)
	}

	if len(stats.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(stats.Containers))
	}

	if stats.Containers[0].ID != "container-123" {
		t.Errorf("Expected container ID 'container-123', got '%s'", stats.Containers[0].ID)
	}
}

func TestSystemService_GetAppStats_StoppedApp(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestSystemService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	// Get the test node ID
	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Create stopped app
	app := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app.Status = "stopped"
	app.NodeID = testNodeID // Assign to test node
	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Get app stats
	stats, err := service.GetAppStats(ctx, app.ID, testNodeID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify empty stats for stopped app
	if stats.Status != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", stats.Status)
	}

	if len(stats.Containers) != 0 {
		t.Errorf("Expected 0 containers for stopped app, got %d", len(stats.Containers))
	}

	if stats.TotalCPUPercent != 0 {
		t.Errorf("Expected 0 CPU for stopped app, got %f", stats.TotalCPUPercent)
	}
}

func TestSystemService_GetAppStats_NotFound(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestSystemService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Try to get stats for non-existent app
	_, err = service.GetAppStats(ctx, "non-existent-id", testNodeID)
	if err == nil {
		t.Error("Expected error for non-existent app, got nil")
	}

	if !domain.IsNotFoundError(err) {
		t.Errorf("Expected not found error, got %v", err)
	}
}

func TestSystemService_GetAppLogs(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestSystemService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	// Get the test node ID
	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Create app
	app := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app.NodeID = testNodeID // Assign to test node
	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Mock docker compose logs command
	expectedLogs := "log line 1\nlog line 2\nlog line 3"
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "logs", "--tail=100"}, []byte(expectedLogs))

	// Get app logs
	logs, err := service.GetAppLogs(ctx, app.ID, testNodeID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify logs were retrieved
	if len(logs) == 0 {
		t.Error("Expected logs to be returned")
	}

	// Verify docker command was executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "logs", "--tail=100"}) {
		t.Error("Expected docker compose logs command to be executed")
	}
}

func TestSystemService_GetAppLogs_NotFound(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestSystemService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Try to get logs for non-existent app
	_, err = service.GetAppLogs(ctx, "non-existent-id", testNodeID)
	if err == nil {
		t.Error("Expected error for non-existent app, got nil")
	}

	if !domain.IsNotFoundError(err) {
		t.Errorf("Expected not found error, got %v", err)
	}
}

func TestSystemService_RestartContainer(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, _, cleanup := setupTestSystemService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	containerID := "abc123def456"

	// Mock successful docker restart command
	mockExecutor.SetMockOutput("docker", []string{"restart", containerID}, []byte("success"))

	// Restart container
	err := service.RestartContainer(ctx, containerID, "test-node-id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify docker command was executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"restart", containerID}) {
		t.Error("Expected docker restart command to be executed")
	}
}

func TestSystemService_RestartContainer_InvalidID(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, _, cleanup := setupTestSystemService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	// Try to restart with invalid container ID
	err := service.RestartContainer(ctx, "", "test-node-id")
	if err == nil {
		t.Error("Expected error for invalid container ID, got nil")
	}
}

func TestSystemService_RestartContainer_DockerError(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, _, cleanup := setupTestSystemService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	containerID := "abc123def456"

	// Mock docker command failure
	dockerError := errors.New("container not found")
	mockExecutor.SetMockError("docker", []string{"restart", containerID}, dockerError)

	// Try to restart container
	err := service.RestartContainer(ctx, containerID, "test-node-id")
	if err == nil {
		t.Error("Expected error when Docker command fails, got nil")
	}
}

func TestSystemService_StopContainer(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, _, cleanup := setupTestSystemService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	containerID := "abc123def456"

	// Mock successful docker stop command
	mockExecutor.SetMockOutput("docker", []string{"stop", containerID}, []byte("success"))

	// Stop container
	err := service.StopContainer(ctx, containerID, "test-node-id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify docker command was executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"stop", containerID}) {
		t.Error("Expected docker stop command to be executed")
	}
}

func TestSystemService_StopContainer_DockerError(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, _, cleanup := setupTestSystemService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	containerID := "abc123def456"

	// Mock docker command failure
	dockerError := errors.New("container not found")
	mockExecutor.SetMockError("docker", []string{"stop", containerID}, dockerError)

	// Try to stop container
	err := service.StopContainer(ctx, containerID, "test-node-id")
	if err == nil {
		t.Error("Expected error when Docker command fails, got nil")
	}
}

func TestSystemService_DeleteContainer(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, _, cleanup := setupTestSystemService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	containerID := "abc123def456"

	// Mock successful docker rm command
	mockExecutor.SetMockOutput("docker", []string{"rm", "-f", containerID}, []byte("success"))

	// Delete container
	err := service.DeleteContainer(ctx, containerID, "test-node-id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify docker command was executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"rm", "-f", containerID}) {
		t.Error("Expected docker rm command to be executed")
	}
}

func TestSystemService_DeleteContainer_DockerError(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, _, cleanup := setupTestSystemService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	containerID := "abc123def456"

	// Mock docker command failure
	dockerError := errors.New("container not found")
	mockExecutor.SetMockError("docker", []string{"rm", "-f", containerID}, dockerError)

	// Try to delete container
	err := service.DeleteContainer(ctx, containerID, "test-node-id")
	if err == nil {
		t.Error("Expected error when Docker command fails, got nil")
	}
}

// Note: GetSystemStats tests would require mocking the system.Collector which is more complex.
// For now, we focus on testing the service methods that use Docker commands directly.
