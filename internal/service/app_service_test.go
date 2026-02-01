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

// setupTestAppService creates a test app service with in-memory database
func setupTestAppService(t *testing.T) (domain.AppService, *db.DB, func()) {
	return setupTestAppServiceWithMocks(t, nil)
}

// setupTestAppServiceWithMocks creates a test app service with mocked dependencies
func setupTestAppServiceWithMocks(t *testing.T, mockExecutor docker.CommandExecutor) (domain.AppService, *db.DB, func()) {
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
		AppsDir:    tmpAppsDir,
		AutoStart:  false,
		Cloudflare: config.CloudflareConfig{},
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
	tunnelService := NewTunnelService(database, dockerManager, cfg, logger)
	service := NewAppService(database, dockerManager, cfg, logger, tunnelService)

	cleanup := func() {
		database.Close()
		os.Remove(tmpDB.Name())
		os.RemoveAll(tmpAppsDir)
	}

	return service, database, cleanup
}

func TestAppService_CreateApp(t *testing.T) {
	service, _, cleanup := setupTestAppService(t)
	defer cleanup()

	ctx := context.Background()
	req := domain.CreateAppRequest{
		Name:           "test-app",
		Description:    "Test application",
		ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
		IngressRules:   []db.IngressRule{},
	}

	// Create app
	app, err := service.CreateApp(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify app was created with correct values
	if app.Name != req.Name {
		t.Errorf("Expected app name '%s', got '%s'", req.Name, app.Name)
	}

	if app.Description != req.Description {
		t.Errorf("Expected description '%s', got '%s'", req.Description, app.Description)
	}

	if app.ComposeContent != req.ComposeContent {
		t.Errorf("Expected compose content to match")
	}

	if app.Status != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", app.Status)
	}

	if app.ID == "" {
		t.Error("Expected app ID to be generated")
	}
}

func TestAppService_GetApp(t *testing.T) {
	service, _, cleanup := setupTestAppService(t)
	defer cleanup()

	ctx := context.Background()

	// Create an app first
	req := domain.CreateAppRequest{
		Name:           "test-app",
		Description:    "Test application",
		ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
	}

	createdApp, err := service.CreateApp(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Retrieve the app
	retrievedApp, err := service.GetApp(ctx, createdApp.ID, createdApp.NodeID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify retrieved app matches created app
	if retrievedApp.ID != createdApp.ID {
		t.Errorf("Expected ID '%s', got '%s'", createdApp.ID, retrievedApp.ID)
	}

	if retrievedApp.Name != createdApp.Name {
		t.Errorf("Expected name '%s', got '%s'", createdApp.Name, retrievedApp.Name)
	}
}

func TestAppService_GetApp_NotFound(t *testing.T) {
	service, _, cleanup := setupTestAppService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to get non-existent app
	_, err := service.GetApp(ctx, "non-existent-id", "test-node-id")
	if err == nil {
		t.Error("Expected error for non-existent app, got nil")
	}

	// Verify it's a not found error
	if !domain.IsNotFoundError(err) {
		t.Errorf("Expected not found error, got %v", err)
	}
}

func TestAppService_ListApps(t *testing.T) {
	service, _, cleanup := setupTestAppService(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple apps
	apps := []domain.CreateAppRequest{
		{
			Name:           "app1",
			Description:    "First app",
			ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
		},
		{
			Name:           "app2",
			Description:    "Second app",
			ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
		},
	}

	for _, req := range apps {
		_, err := service.CreateApp(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create app '%s': %v", req.Name, err)
		}
	}

	// List all apps
	appsList, err := service.ListApps(ctx, []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify we have both apps
	if len(appsList) != 2 {
		t.Errorf("Expected 2 apps, got %d", len(appsList))
	}
}

func TestAppService_UpdateApp(t *testing.T) {
	service, _, cleanup := setupTestAppService(t)
	defer cleanup()

	ctx := context.Background()

	// Create app first
	createReq := domain.CreateAppRequest{
		Name:           "test-app",
		Description:    "Original description",
		ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
	}

	createdApp, err := service.CreateApp(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Update app (don't rename, just update description and compose)
	updateReq := domain.UpdateAppRequest{
		Description:    "Updated description",
		ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:alpine",
	}

	updatedApp, err := service.UpdateApp(ctx, createdApp.ID, createdApp.NodeID, updateReq)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify updates were applied
	if updatedApp.Description != updateReq.Description {
		t.Errorf("Expected description '%s', got '%s'", updateReq.Description, updatedApp.Description)
	}

	if updatedApp.ComposeContent != updateReq.ComposeContent {
		t.Errorf("Expected compose content to be updated")
	}
}

func TestAppService_DeleteApp(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, _, cleanup := setupTestAppServiceWithMocks(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	// Create app first
	req := domain.CreateAppRequest{
		Name:           "test-app",
		Description:    "Test application",
		ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
	}

	createdApp, err := service.CreateApp(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Mock Docker stop command for cleanup
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "down"}, []byte("success"))

	// Delete the app
	err = service.DeleteApp(ctx, createdApp.ID, createdApp.NodeID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify app no longer exists
	_, err = service.GetApp(ctx, createdApp.ID, createdApp.NodeID)
	if err == nil {
		t.Error("Expected error when getting deleted app, got nil")
	}

	if !domain.IsNotFoundError(err) {
		t.Errorf("Expected not found error, got %v", err)
	}
}

// TestAppService_StartApp tests starting an app with mocked Docker commands
func TestAppService_StartApp(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, _, cleanup := setupTestAppServiceWithMocks(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	// Create app first
	req := domain.CreateAppRequest{
		Name:           "test-app",
		Description:    "Test application",
		ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
	}

	createdApp, err := service.CreateApp(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Mock successful docker compose up command
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d"}, []byte("success"))

	// Start the app
	updatedApp, err := service.StartApp(ctx, createdApp.ID, createdApp.NodeID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify status was updated
	if updatedApp.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", updatedApp.Status)
	}

	// Verify docker command was executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d"}) {
		t.Error("Expected docker compose up command to be executed")
	}
}

// TestAppService_StartApp_NotFound tests starting a non-existent app
func TestAppService_StartApp_NotFound(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, _, cleanup := setupTestAppServiceWithMocks(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	_, err := service.StartApp(ctx, "non-existent-id", "test-node-id")
	if err == nil {
		t.Error("Expected error for non-existent app, got nil")
	}

	if !domain.IsNotFoundError(err) {
		t.Errorf("Expected not found error, got %v", err)
	}
}

// TestAppService_StartApp_DockerError tests error handling when Docker command fails
func TestAppService_StartApp_DockerError(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestAppServiceWithMocks(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	// Create app first
	req := domain.CreateAppRequest{
		Name:           "test-app",
		Description:    "Test application",
		ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
	}

	createdApp, err := service.CreateApp(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Mock docker command failure
	dockerError := errors.New("docker compose failed")
	mockExecutor.SetMockError("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d"}, dockerError)

	// Try to start the app
	_, err = service.StartApp(ctx, createdApp.ID, createdApp.NodeID)
	if err == nil {
		t.Error("Expected error when Docker command fails, got nil")
	}

	// Verify app status was updated to error
	app, err := database.GetApp(createdApp.ID)
	if err != nil {
		t.Fatalf("Failed to get app: %v", err)
	}

	if app.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", app.Status)
	}

	if app.ErrorMessage == nil {
		t.Error("Expected error message to be set")
	}
}

// TestAppService_StopApp tests stopping an app with mocked Docker commands
func TestAppService_StopApp(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, db, cleanup := setupTestAppServiceWithMocks(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	// Create app first
	req := domain.CreateAppRequest{
		Name:           "test-app",
		Description:    "Test application",
		ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
	}

	createdApp, err := service.CreateApp(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Set app status to running
	createdApp.Status = "running"
	if err := db.UpdateApp(createdApp); err != nil {
		t.Fatalf("Failed to update app status: %v", err)
	}

	// Mock successful docker compose down command
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "down"}, []byte("success"))

	// Stop the app
	updatedApp, err := service.StopApp(ctx, createdApp.ID, createdApp.NodeID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify status was updated
	if updatedApp.Status != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", updatedApp.Status)
	}

	// Verify docker command was executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "down"}) {
		t.Error("Expected docker compose down command to be executed")
	}
}

// TestAppService_StopApp_DockerError tests error handling when Docker stop command fails
func TestAppService_StopApp_DockerError(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestAppServiceWithMocks(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	// Create app first
	req := domain.CreateAppRequest{
		Name:           "test-app",
		Description:    "Test application",
		ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
	}

	createdApp, err := service.CreateApp(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Set app status to running
	createdApp.Status = "running"
	if err := database.UpdateApp(createdApp); err != nil {
		t.Fatalf("Failed to update app status: %v", err)
	}

	// Mock docker command failure
	dockerError := errors.New("docker compose down failed")
	mockExecutor.SetMockError("docker", []string{"compose", "-f", "docker-compose.yml", "down"}, dockerError)

	// Try to stop the app
	_, err = service.StopApp(ctx, createdApp.ID, createdApp.NodeID)
	if err == nil {
		t.Error("Expected error when Docker command fails, got nil")
	}

	// Verify app status was updated to error
	app, err := database.GetApp(createdApp.ID)
	if err != nil {
		t.Fatalf("Failed to get app: %v", err)
	}

	if app.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", app.Status)
	}
}

// TestAppService_UpdateAppContainers tests updating app containers with mocked Docker commands
func TestAppService_UpdateAppContainers(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, _, cleanup := setupTestAppServiceWithMocks(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	// Create app first
	req := domain.CreateAppRequest{
		Name:           "test-app",
		Description:    "Test application",
		ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
	}

	createdApp, err := service.CreateApp(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Mock successful docker compose commands
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "pull", "--ignore-buildable"}, []byte("success"))
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d", "--build"}, []byte("success"))

	// Update containers
	updatedApp, err := service.UpdateAppContainers(ctx, createdApp.ID, createdApp.NodeID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify status was updated to running
	if updatedApp.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", updatedApp.Status)
	}

	// Verify docker commands were executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d", "--build"}) {
		t.Error("Expected docker compose up command to be executed")
	}
}

// TestAppService_UpdateAppContainers_DockerError tests error handling when Docker update fails
func TestAppService_UpdateAppContainers_DockerError(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, db, cleanup := setupTestAppServiceWithMocks(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	// Create app first
	req := domain.CreateAppRequest{
		Name:           "test-app",
		Description:    "Test application",
		ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
	}

	createdApp, err := service.CreateApp(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Mock docker command failure
	dockerError := errors.New("docker compose up failed")
	mockExecutor.SetMockError("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d", "--build"}, dockerError)

	// Try to update containers
	_, err = service.UpdateAppContainers(ctx, createdApp.ID, createdApp.NodeID)
	if err == nil {
		t.Error("Expected error when Docker command fails, got nil")
	}

	// Verify app status was updated to error
	app, err := db.GetApp(createdApp.ID)
	if err != nil {
		t.Fatalf("Failed to get app: %v", err)
	}

	if app.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", app.Status)
	}
}

// TestAppService_RestartCloudflared tests restarting cloudflared with mocked Docker commands
func TestAppService_RestartCloudflared(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, _, cleanup := setupTestAppServiceWithMocks(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	// Create app first
	req := domain.CreateAppRequest{
		Name:           "test-app",
		Description:    "Test application",
		ComposeContent: "version: '3'\nservices:\n  web:\n    image: nginx:latest",
	}

	createdApp, err := service.CreateApp(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Mock successful docker compose restart command
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "restart", "cloudflared"}, []byte("success"))

	// Restart cloudflared (using app's node_id)
	err = service.RestartCloudflared(ctx, createdApp.ID, createdApp.NodeID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify docker command was executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "restart", "cloudflared"}) {
		t.Error("Expected docker compose restart command to be executed")
	}
}

// Note: Cloudflare API tests are in tunnel_service_test.go since app_service
// creates TunnelManager internally and doesn't support dependency injection.
// Docker command tests are fully covered above with mocked CommandExecutor.
