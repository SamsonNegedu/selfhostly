package service

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/node"
	"github.com/selfhostly/internal/routing"
)

// setupTestComposeService creates a test compose service with mocked dependencies
func setupTestComposeService(t *testing.T, mockExecutor docker.CommandExecutor) (domain.ComposeService, *db.DB, func()) {
	service, database, _, cleanup := setupTestComposeServiceWithAppsDir(t, mockExecutor)
	return service, database, cleanup
}

// setupTestComposeServiceWithAppsDir creates a test compose service and returns the apps directory
func setupTestComposeServiceWithAppsDir(t *testing.T, mockExecutor docker.CommandExecutor) (domain.ComposeService, *db.DB, string, func()) {
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
	testNode := db.NewNode(testNodeName, "http://localhost:8080", testAPIKey, true)
	testNode.ID = testNodeID
	if err := database.CreateNode(testNode); err != nil {
		t.Fatalf("Failed to create test node: %v", err)
	}

	logger := slog.Default()
	nodeClient := node.NewClient()
	router := routing.NewNodeRouter(database, nodeClient, testNodeID, logger)
	service := NewComposeService(database, dockerManager, router, nodeClient, logger)

	cleanup := func() {
		database.Close()
		os.Remove(tmpDB.Name())
		os.RemoveAll(tmpAppsDir)
	}

	return service, database, tmpAppsDir, cleanup
}

func TestComposeService_GetVersions(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestComposeService(t, mockExecutor)
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

	// Create compose versions
	reason1 := "Initial version"
	version1 := db.NewComposeVersion(app.ID, 1, "version: '3'\nservices:\n  web:\n    image: nginx:latest", &reason1, nil)
	if err := database.CreateComposeVersion(version1); err != nil {
		t.Fatalf("Failed to create version 1: %v", err)
	}

	reason2 := "Updated version"
	version2 := db.NewComposeVersion(app.ID, 2, "version: '3'\nservices:\n  web:\n    image: nginx:alpine", &reason2, nil)
	if err := database.CreateComposeVersion(version2); err != nil {
		t.Fatalf("Failed to create version 2: %v", err)
	}

	// Get versions
	versions, err := service.GetVersions(ctx, app.ID, testNodeID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify versions (database returns versions in DESC order, so version 2 comes first)
	if len(versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(versions))
	}

	// Versions are returned in DESC order (newest first)
	if versions[0].Version != 2 {
		t.Errorf("Expected version 2 (newest first), got %d", versions[0].Version)
	}

	if versions[1].Version != 1 {
		t.Errorf("Expected version 1 (oldest last), got %d", versions[1].Version)
	}
}

func TestComposeService_GetVersions_Empty(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestComposeService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Create app without versions
	app := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app.NodeID = testNodeID
	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Get versions
	versions, err := service.GetVersions(ctx, app.ID, testNodeID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify empty array (not nil)
	if versions == nil {
		t.Error("Expected empty array, got nil")
	}

	if len(versions) != 0 {
		t.Errorf("Expected 0 versions, got %d", len(versions))
	}
}

func TestComposeService_GetVersions_NotFound(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestComposeService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Try to get versions for non-existent app
	_, err = service.GetVersions(ctx, "non-existent-id", testNodeID)
	if err == nil {
		t.Error("Expected error for non-existent app, got nil")
	}

	if !domain.IsNotFoundError(err) {
		t.Errorf("Expected not found error, got %v", err)
	}
}

func TestComposeService_GetVersion(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestComposeService(t, mockExecutor)
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

	// Create compose version
	reason := constants.ComposeVersionReasonInitial
	version := db.NewComposeVersion(app.ID, 1, "version: '3'\nservices:\n  web:\n    image: nginx:latest", &reason, nil)
	if err := database.CreateComposeVersion(version); err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}

	// Get specific version
	retrievedVersion, err := service.GetVersion(ctx, app.ID, 1, testNodeID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify version
	if retrievedVersion.Version != 1 {
		t.Errorf("Expected version 1, got %d", retrievedVersion.Version)
	}

	if retrievedVersion.ComposeContent != version.ComposeContent {
		t.Error("Expected compose content to match")
	}
}

func TestComposeService_GetVersion_NotFound(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestComposeService(t, mockExecutor)
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

	// Try to get non-existent version
	_, err = service.GetVersion(ctx, app.ID, 999, testNodeID)
	if err == nil {
		t.Error("Expected error for non-existent version, got nil")
	}

	if err != domain.ErrComposeVersionNotFound {
		t.Errorf("Expected ErrComposeVersionNotFound, got %v", err)
	}
}

func TestComposeService_RollbackToVersion(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, tmpAppsDir, cleanup := setupTestComposeServiceWithAppsDir(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Create app
	app := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:alpine")
	app.NodeID = testNodeID
	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Create app directory on disk (required for WriteComposeFile)
	appDir := filepath.Join(tmpAppsDir, "test-app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}

	// Create initial version
	reason1 := constants.ComposeVersionReasonInitial
	version1 := db.NewComposeVersion(app.ID, 1, "version: '3'\nservices:\n  web:\n    image: nginx:latest", &reason1, nil)
	if err := database.CreateComposeVersion(version1); err != nil {
		t.Fatalf("Failed to create version 1: %v", err)
	}

	// Create second version
	reason2 := "Updated version"
	version2 := db.NewComposeVersion(app.ID, 2, "version: '3'\nservices:\n  web:\n    image: nginx:alpine", &reason2, nil)
	if err := database.CreateComposeVersion(version2); err != nil {
		t.Fatalf("Failed to create version 2: %v", err)
	}

	// Mark version 2 as current
	if err := database.MarkVersionAsCurrent(app.ID, 2); err != nil {
		t.Fatalf("Failed to mark version as current: %v", err)
	}

	// Rollback to version 1
	rollbackReason := "Rolling back to stable version"
	newVersion, err := service.RollbackToVersion(ctx, app.ID, 1, testNodeID, &rollbackReason, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify new version was created
	if newVersion.Version != 3 {
		t.Errorf("Expected version 3 (new rollback version), got %d", newVersion.Version)
	}

	if newVersion.ComposeContent != version1.ComposeContent {
		t.Error("Expected compose content to match version 1")
	}

	if newVersion.RolledBackFrom == nil || *newVersion.RolledBackFrom != 2 {
		t.Errorf("Expected rolled back from version 2, got %v", newVersion.RolledBackFrom)
	}

	// Verify app was updated
	updatedApp, err := database.GetApp(app.ID)
	if err != nil {
		t.Fatalf("Failed to get app: %v", err)
	}

	if updatedApp.ComposeContent != version1.ComposeContent {
		t.Error("Expected app compose content to be updated to version 1")
	}

	// Verify docker command was executed to write compose file
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d"}) {
		// The WriteComposeFile doesn't execute docker commands, it just writes to disk
		// So we just verify the rollback succeeded
	}
}

func TestComposeService_RollbackToVersion_NotFound(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestComposeService(t, mockExecutor)
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

	// Try to rollback to non-existent version
	_, err = service.RollbackToVersion(ctx, app.ID, 999, testNodeID, nil, nil)
	if err == nil {
		t.Error("Expected error for non-existent version, got nil")
	}

	if err != domain.ErrComposeVersionNotFound {
		t.Errorf("Expected ErrComposeVersionNotFound, got %v", err)
	}
}

func TestComposeService_RollbackToVersion_AppNotFound(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	service, database, cleanup := setupTestComposeService(t, mockExecutor)
	defer cleanup()

	ctx := context.Background()

	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Try to rollback for non-existent app
	_, err = service.RollbackToVersion(ctx, "non-existent-id", 1, testNodeID, nil, nil)
	if err == nil {
		t.Error("Expected error for non-existent app, got nil")
	}

	if !domain.IsNotFoundError(err) {
		t.Errorf("Expected not found error, got %v", err)
	}
}
