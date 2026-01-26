package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/selfhostly/internal/cloudflare"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

// setupTestTunnelService creates a test tunnel service with mocked Cloudflare API
func setupTestTunnelService(t *testing.T) (domain.TunnelService, *db.DB, *cloudflare.MockHTTPClient, func()) {
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

	// Create mock HTTP client
	mockHTTPClient := cloudflare.NewMockHTTPClient()

	// Set up Cloudflare settings
	apiToken := "test-api-token"
	accountID := "test-account-id"
	if err := database.UpdateSettings(&db.Settings{
		CloudflareAPIToken: &apiToken,
		CloudflareAccountID: &accountID,
	}); err != nil {
		t.Fatalf("Failed to update settings: %v", err)
	}

	logger := slog.Default()

	// Create mocked Cloudflare Manager with mock HTTP client
	mockManager := cloudflare.NewManagerWithClient(apiToken, accountID, mockHTTPClient)
	mockTunnelManager := cloudflare.NewTunnelManagerWithManager(mockManager, database)

	// Create service with mocked tunnel manager
	service := NewTunnelServiceWithManager(database, logger, mockTunnelManager)

	cleanup := func() {
		database.Close()
		os.Remove(tmpDB.Name())
	}

	return service, database, mockHTTPClient, cleanup
}

// createTestAppWithTunnel creates an app and tunnel for testing
func createTestAppWithTunnel(t *testing.T, database *db.DB) (*db.App, *db.CloudflareTunnel) {
	// Create app
	app := db.NewApp("test-app", "Test application", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app.TunnelID = "tunnel-123"
	app.TunnelToken = "tunnel-token-456"
	app.Status = "stopped"

	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Create tunnel record
	tunnel := db.NewCloudflareTunnel(app.ID, app.TunnelID, app.Name, app.TunnelToken, "test-account-id")
	tunnel.IsActive = true
	tunnel.Status = "active"
	tunnel.CreatedAt = time.Now()
	tunnel.UpdatedAt = time.Now()

	if err := database.CreateCloudflareTunnel(tunnel); err != nil {
		t.Fatalf("Failed to create tunnel: %v", err)
	}

	return app, tunnel
}

func TestTunnelService_GetTunnelByAppID(t *testing.T) {
	service, database, _, cleanup := setupTestTunnelService(t)
	defer cleanup()

	ctx := context.Background()

	// Create test app and tunnel
	app, tunnel := createTestAppWithTunnel(t, database)

	// Get tunnel by app ID
	retrievedTunnel, err := service.GetTunnelByAppID(ctx, app.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify tunnel matches
	if retrievedTunnel.TunnelID != tunnel.TunnelID {
		t.Errorf("Expected tunnel ID '%s', got '%s'", tunnel.TunnelID, retrievedTunnel.TunnelID)
	}

	if retrievedTunnel.AppID != app.ID {
		t.Errorf("Expected app ID '%s', got '%s'", app.ID, retrievedTunnel.AppID)
	}
}

func TestTunnelService_GetTunnelByAppID_NotFound(t *testing.T) {
	service, _, _, cleanup := setupTestTunnelService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to get tunnel for non-existent app
	_, err := service.GetTunnelByAppID(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent tunnel, got nil")
	}
}

func TestTunnelService_ListActiveTunnels(t *testing.T) {
	service, database, _, cleanup := setupTestTunnelService(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple apps and tunnels
	_, tunnel1 := createTestAppWithTunnel(t, database)

	app2 := db.NewApp("test-app-2", "Second app", "version: '3'\nservices:\n  web:\n    image: nginx:latest")
	app2.TunnelID = "tunnel-456"
	app2.TunnelToken = "tunnel-token-789"
	if err := database.CreateApp(app2); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	tunnel2 := db.NewCloudflareTunnel(app2.ID, app2.TunnelID, app2.Name, app2.TunnelToken, "test-account-id")
	tunnel2.IsActive = true
	tunnel2.Status = "active"
	if err := database.CreateCloudflareTunnel(tunnel2); err != nil {
		t.Fatalf("Failed to create tunnel: %v", err)
	}

	// List active tunnels
	tunnels, err := service.ListActiveTunnels(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify we have both tunnels
	if len(tunnels) != 2 {
		t.Errorf("Expected 2 tunnels, got %d", len(tunnels))
	}

	// Verify tunnel IDs
	tunnelIDs := make(map[string]bool)
	for _, tunnel := range tunnels {
		tunnelIDs[tunnel.TunnelID] = true
	}

	if !tunnelIDs[tunnel1.TunnelID] {
		t.Errorf("Expected tunnel %s to be in list", tunnel1.TunnelID)
	}

	if !tunnelIDs[tunnel2.TunnelID] {
		t.Errorf("Expected tunnel %s to be in list", tunnel2.TunnelID)
	}
}

func TestTunnelService_UpdateTunnelIngress(t *testing.T) {
	service, database, mockHTTPClient, cleanup := setupTestTunnelService(t)
	defer cleanup()

	ctx := context.Background()

	// Create test app and tunnel
	app, tunnel := createTestAppWithTunnel(t, database)

	// Set up mock Cloudflare API response for zone ID lookup (required for DNS record creation)
	zoneURL := "https://api.cloudflare.com/client/v4/zones?name=example.com"
	mockHTTPClient.SetMockResponse(zoneURL, cloudflare.MockResponse{
		StatusCode: http.StatusOK,
		Body:       `{"success": true, "result": [{"id": "zone-123", "name": "example.com"}]}`,
	})

	// Set up mock Cloudflare API response for DNS record creation
	dnsURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/zone-123/dns_records")
	mockHTTPClient.SetMockResponse(dnsURL, cloudflare.MockResponse{
		StatusCode: http.StatusOK,
		Body:       `{"success": true, "result": {"id": "dns-record-123"}}`,
	})

	// Set up mock Cloudflare API response for ingress configuration
	ingressURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/test-account-id/cfd_tunnel/%s/configurations", tunnel.TunnelID)
	mockHTTPClient.SetMockResponse(ingressURL, cloudflare.MockResponse{
		StatusCode: http.StatusOK,
		Body:       `{"success": true}`,
	})

	// Update ingress
	req := domain.UpdateIngressRequest{
		IngressRules: []db.IngressRule{
			{
				Service: "http://localhost:8080",
				Hostname: stringPtr("example.com"),
			},
			{
				Service: "http_status:404",
			},
		},
		Hostname: "example.com",
	}

	err := service.UpdateTunnelIngress(ctx, app.ID, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify Cloudflare API was called
	if !mockHTTPClient.AssertRequestMade("PUT", ingressURL) {
		t.Error("Expected PUT request to update ingress configuration")
	}
}

func TestTunnelService_UpdateTunnelIngress_NotFound(t *testing.T) {
	service, _, _, cleanup := setupTestTunnelService(t)
	defer cleanup()

	ctx := context.Background()

	req := domain.UpdateIngressRequest{
		IngressRules: []db.IngressRule{
			{
				Service: "http://localhost:8080",
			},
		},
	}

	// Try to update ingress for non-existent app
	err := service.UpdateTunnelIngress(ctx, "non-existent-id", req)
	if err == nil {
		t.Error("Expected error for non-existent tunnel, got nil")
	}
}

func TestTunnelService_CreateDNSRecord(t *testing.T) {
	service, database, mockHTTPClient, cleanup := setupTestTunnelService(t)
	defer cleanup()

	ctx := context.Background()

	// Create test app and tunnel
	app, tunnel := createTestAppWithTunnel(t, database)

	// Set up mock Cloudflare API responses
	zoneID := "zone-123"
	getZoneURL := "https://api.cloudflare.com/client/v4/zones?name=example.com"
	zoneResponse := map[string]interface{}{
		"success": true,
		"result": []map[string]interface{}{
			{
				"id":   zoneID,
				"name": "example.com",
			},
		},
	}
	if err := mockHTTPClient.SetJSONMockResponse(getZoneURL, http.StatusOK, zoneResponse); err != nil {
		t.Fatalf("Failed to set mock response: %v", err)
	}

	// Mock DNS record check (no existing record)
	getDNSURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=CNAME&name=app.example.com", zoneID)
	dnsCheckResponse := map[string]interface{}{
		"success": true,
		"result":  []interface{}{},
	}
	if err := mockHTTPClient.SetJSONMockResponse(getDNSURL, http.StatusOK, dnsCheckResponse); err != nil {
		t.Fatalf("Failed to set mock response: %v", err)
	}

	// Mock DNS record creation
	createDNSURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)
	createDNSResponse := map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"id":      "dns-record-123",
			"zone_id": zoneID,
			"type":    "CNAME",
			"name":    "app.example.com",
			"content": fmt.Sprintf("%s.cfargotunnel.com", tunnel.TunnelID),
			"proxied": true,
		},
	}
	if err := mockHTTPClient.SetJSONMockResponse(createDNSURL, http.StatusOK, createDNSResponse); err != nil {
		t.Fatalf("Failed to set mock response: %v", err)
	}

	// Create DNS record
	req := domain.CreateDNSRequest{
		Hostname: "app.example.com",
		Domain:   "example.com",
	}

	err := service.CreateDNSRecord(ctx, app.ID, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify Cloudflare API calls were made
	if !mockHTTPClient.AssertRequestMade("GET", getZoneURL) {
		t.Error("Expected GET request to get zone ID")
	}

	if !mockHTTPClient.AssertRequestMade("POST", createDNSURL) {
		t.Error("Expected POST request to create DNS record")
	}
}

func TestTunnelService_CreateDNSRecord_ZoneNotFound(t *testing.T) {
	service, database, mockHTTPClient, cleanup := setupTestTunnelService(t)
	defer cleanup()

	ctx := context.Background()

	// Create test app and tunnel
	app, _ := createTestAppWithTunnel(t, database)

	// Set up mock Cloudflare API response for zone (not found)
	getZoneURL := "https://api.cloudflare.com/client/v4/zones?name=example.com"
	zoneResponse := map[string]interface{}{
		"success": true,
		"result":  []interface{}{},
	}
	if err := mockHTTPClient.SetJSONMockResponse(getZoneURL, http.StatusOK, zoneResponse); err != nil {
		t.Fatalf("Failed to set mock response: %v", err)
	}

	// Try to create DNS record
	req := domain.CreateDNSRequest{
		Hostname: "app.example.com",
		Domain:   "example.com",
	}

	err := service.CreateDNSRecord(ctx, app.ID, req)
	if err == nil {
		t.Error("Expected error when zone not found, got nil")
	}
}

func TestTunnelService_DeleteTunnel(t *testing.T) {
	service, database, mockHTTPClient, cleanup := setupTestTunnelService(t)
	defer cleanup()

	ctx := context.Background()

	// Create test app and tunnel
	app, tunnel := createTestAppWithTunnel(t, database)

	// Set up mock Cloudflare API responses for DNS cleanup
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
	deleteTunnelURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/test-account-id/cfd_tunnel/%s", tunnel.TunnelID)
	mockHTTPClient.SetMockResponse(deleteTunnelURL, cloudflare.MockResponse{
		StatusCode: http.StatusOK,
		Body:       `{"success": true}`,
	})

	// Delete tunnel
	err := service.DeleteTunnel(ctx, app.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify tunnel was marked as inactive
	tunnelRecord, err := database.GetCloudflareTunnelByAppID(app.ID)
	if err != nil {
		t.Fatalf("Failed to get tunnel: %v", err)
	}

	if tunnelRecord.IsActive {
		t.Error("Expected tunnel to be marked as inactive")
	}

	if tunnelRecord.Status != "deleted" {
		t.Errorf("Expected status 'deleted', got '%s'", tunnelRecord.Status)
	}
}

func TestTunnelService_SyncTunnelStatus(t *testing.T) {
	service, database, _, cleanup := setupTestTunnelService(t)
	defer cleanup()

	ctx := context.Background()

	// Create test app and tunnel
	app, _ := createTestAppWithTunnel(t, database)

	// Sync tunnel status
	err := service.SyncTunnelStatus(ctx, app.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestTunnelService_SyncTunnelStatus_NotFound(t *testing.T) {
	service, _, _, cleanup := setupTestTunnelService(t)
	defer cleanup()

	ctx := context.Background()

	// Try to sync non-existent tunnel
	err := service.SyncTunnelStatus(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent tunnel, got nil")
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
