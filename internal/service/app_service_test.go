package service

import (
	"context"
	"io/ioutil"
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
	// Create temp database
	tmpDB, err := ioutil.TempFile("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp database: %v", err)
	}
	tmpDB.Close()

	database, err := db.Init(tmpDB.Name())
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Create temp apps directory
	tmpAppsDir, err := ioutil.TempDir("", "test-apps-")
	if err != nil {
		t.Fatalf("Failed to create temp apps directory: %v", err)
	}

	// Create real docker manager (file operations only, no actual docker commands)
	dockerManager := docker.NewManager(tmpAppsDir)

	cfg := &config.Config{
		AppsDir:    tmpAppsDir,
		AutoStart:  false,
		Cloudflare: config.CloudflareConfig{},
	}

	logger := slog.Default()
	service := NewAppService(database, dockerManager, cfg, logger)

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
	retrievedApp, err := service.GetApp(ctx, createdApp.ID)
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
	_, err := service.GetApp(ctx, "non-existent-id")
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
	appsList, err := service.ListApps(ctx)
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

	updatedApp, err := service.UpdateApp(ctx, createdApp.ID, updateReq)
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
	service, _, cleanup := setupTestAppService(t)
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

	// Delete the app
	err = service.DeleteApp(ctx, createdApp.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify app no longer exists
	_, err = service.GetApp(ctx, createdApp.ID)
	if err == nil {
		t.Error("Expected error when getting deleted app, got nil")
	}

	if !domain.IsNotFoundError(err) {
		t.Errorf("Expected not found error, got %v", err)
	}
}

// Note: StartApp and StopApp tests require actual Docker, so they're integration tests
// For proper unit testing, we would need to inject mocked ContainerOrchestrator interface
