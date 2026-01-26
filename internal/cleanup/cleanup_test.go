package cleanup

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/selfhostly/internal/cloudflare"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
)

// setupTestCleanupManager creates a test cleanup manager with mocked dependencies
func setupTestCleanupManager(t *testing.T, mockExecutor docker.CommandExecutor, mockHTTPClient cloudflare.HTTPClient) (*CleanupManager, *db.DB, func()) {
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

	// Set up Cloudflare settings
	apiToken := "test-api-token"
	accountID := "test-account-id"
	settings := &db.Settings{
		CloudflareAPIToken: &apiToken,
		CloudflareAccountID: &accountID,
	}

	// Create mocked Cloudflare Manager with mock HTTP client
	var tunnelManager *cloudflare.TunnelManager
	if mockHTTPClient != nil {
		mockManager := cloudflare.NewManagerWithClient(apiToken, accountID, mockHTTPClient)
		tunnelManager = cloudflare.NewTunnelManagerWithManager(mockManager, database)
	} else {
		tunnelManager = cloudflare.NewTunnelManager(apiToken, accountID, database)
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

	manager := NewCleanupManager(dockerManager, database, settings, tunnelManager)

	cleanup := func() {
		database.Close()
		os.Remove(tmpDB.Name())
		os.RemoveAll(tmpAppsDir)
	}

	return manager, database, cleanup
}

func TestCleanupManager_CleanupApp(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	mockHTTPClient := cloudflare.NewMockHTTPClient()
	manager, database, cleanup := setupTestCleanupManager(t, mockExecutor, mockHTTPClient)
	defer cleanup()

	// Get the test node ID
	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Create app with tunnel
	app := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app.TunnelID = "tunnel-123"
	app.TunnelToken = "tunnel-token-456"
	app.NodeID = testNodeID // Assign to test node
	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Create tunnel record
	tunnel := db.NewCloudflareTunnel(app.ID, app.TunnelID, app.Name, app.TunnelToken, "test-account-id")
	tunnel.IsActive = true
	tunnel.Status = "active"
	if err := database.CreateCloudflareTunnel(tunnel); err != nil {
		t.Fatalf("Failed to create tunnel: %v", err)
	}

	// Mock Docker stop command
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "down"}, []byte("success"))

	// Mock Cloudflare API responses for DNS cleanup
	zonesURL := "https://api.cloudflare.com/client/v4/zones"
	zonesResponse := map[string]interface{}{
		"success": true,
		"result": []map[string]interface{}{
			{
				"id":   "zone-123",
				"name": "example.com",
			},
		},
	}
	if err := mockHTTPClient.SetJSONMockResponse(zonesURL, http.StatusOK, zonesResponse); err != nil {
		t.Fatalf("Failed to set mock response: %v", err)
	}

	// Mock DNS records list (empty)
	dnsRecordsURL := "https://api.cloudflare.com/client/v4/zones/zone-123/dns_records?type=CNAME"
	dnsRecordsResponse := map[string]interface{}{
		"success": true,
		"result":  []interface{}{},
	}
	if err := mockHTTPClient.SetJSONMockResponse(dnsRecordsURL, http.StatusOK, dnsRecordsResponse); err != nil {
		t.Fatalf("Failed to set mock response: %v", err)
	}

	// Mock tunnel deletion
	deleteTunnelURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/test-account-id/cfd_tunnel/%s", app.TunnelID)
	mockHTTPClient.SetMockResponse(deleteTunnelURL, cloudflare.MockResponse{
		StatusCode: http.StatusOK,
		Body:       `{"success": true}`,
	})

	// Perform cleanup
	results, err := manager.CleanupApp(app)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify cleanup results
	if len(results) == 0 {
		t.Error("Expected cleanup results, got empty")
	}

	// Verify Docker command was executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "down"}) {
		t.Error("Expected docker compose down command to be executed")
	}

	// Verify Cloudflare API calls were made
	if !mockHTTPClient.AssertRequestMade("GET", zonesURL) {
		t.Error("Expected GET request to get zones")
	}

	if !mockHTTPClient.AssertRequestMade("DELETE", deleteTunnelURL) {
		t.Error("Expected DELETE request to delete tunnel")
	}

	// Verify app was deleted from database
	_, err = database.GetApp(app.ID)
	if err == nil {
		t.Error("Expected app to be deleted from database")
	}
}

func TestCleanupManager_CleanupApp_NoTunnel(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	manager, database, cleanup := setupTestCleanupManager(t, mockExecutor, nil)
	defer cleanup()

	// Get the test node ID
	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Create app without tunnel
	app := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app.NodeID = testNodeID // Assign to test node
	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Mock Docker stop command
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "down"}, []byte("success"))

	// Perform cleanup
	results, err := manager.CleanupApp(app)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify cleanup results
	if len(results) == 0 {
		t.Error("Expected cleanup results, got empty")
	}

	// Verify Docker command was executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "down"}) {
		t.Error("Expected docker compose down command to be executed")
	}
}

func TestCleanupManager_CleanupApp_PartialFailure(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	mockHTTPClient := cloudflare.NewMockHTTPClient()
	manager, database, cleanup := setupTestCleanupManager(t, mockExecutor, mockHTTPClient)
	defer cleanup()

	// Get the test node ID
	nodes, err := database.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to get test node: %v", err)
	}
	testNodeID := nodes[0].ID

	// Create app with tunnel
	app := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app.TunnelID = "tunnel-123"
	app.NodeID = testNodeID // Assign to test node
	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Mock Docker stop command failure
	dockerError := fmt.Errorf("docker compose down failed")
	mockExecutor.SetMockError("docker", []string{"compose", "-f", "docker-compose.yml", "down"}, dockerError)

	// Mock Cloudflare API responses (successful)
	zonesURL := "https://api.cloudflare.com/client/v4/zones"
	zonesResponse := map[string]interface{}{
		"success": true,
		"result":  []interface{}{},
	}
	if err := mockHTTPClient.SetJSONMockResponse(zonesURL, http.StatusOK, zonesResponse); err != nil {
		t.Fatalf("Failed to set mock response: %v", err)
	}

	deleteTunnelURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/test-account-id/cfd_tunnel/%s", app.TunnelID)
	mockHTTPClient.SetMockResponse(deleteTunnelURL, cloudflare.MockResponse{
		StatusCode: http.StatusOK,
		Body:       `{"success": true}`,
	})

	// Perform cleanup - should continue despite Docker failure
	results, _ := manager.CleanupApp(app)
	// Cleanup should continue even with errors
	if results == nil {
		t.Error("Expected cleanup results even with errors")
	}

	// Verify that some steps succeeded despite Docker failure
	successCount, failedCount, _ := manager.GetSummary()
	if successCount == 0 {
		t.Error("Expected at least some cleanup steps to succeed")
	}

	if failedCount == 0 {
		t.Error("Expected at least one cleanup step to fail")
	}
}

func TestCleanupManager_GetSummary(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	manager, database, cleanup := setupTestCleanupManager(t, mockExecutor, nil)
	defer cleanup()

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

	// Mock Docker stop command
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "down"}, []byte("success"))

	// Perform cleanup
	_, err = manager.CleanupApp(app)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Get summary
	successCount, failedCount, duration := manager.GetSummary()

	if successCount == 0 {
		t.Error("Expected at least one successful cleanup step")
	}

	if duration <= 0 {
		t.Error("Expected duration to be greater than 0")
	}

	// Verify summary matches results
	results := manager.GetResults()
	if len(results) != successCount+failedCount {
		t.Errorf("Expected total steps to match summary: got %d results, summary says %d+%d", len(results), successCount, failedCount)
	}
}

func TestCleanupManager_GetResults(t *testing.T) {
	mockExecutor := docker.NewMockCommandExecutor()
	manager, database, cleanup := setupTestCleanupManager(t, mockExecutor, nil)
	defer cleanup()

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

	// Mock Docker stop command
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "down"}, []byte("success"))

	// Perform cleanup
	_, err = manager.CleanupApp(app)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Get results
	results := manager.GetResults()

	if len(results) == 0 {
		t.Error("Expected cleanup results, got empty")
	}

	// Verify result structure
	for _, result := range results {
		if result.Step == "" {
			t.Error("Expected result step to be set")
		}

		if result.Duration <= 0 {
			t.Error("Expected result duration to be greater than 0")
		}
	}
}
