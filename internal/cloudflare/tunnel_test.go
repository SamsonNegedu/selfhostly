package cloudflare

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateTunnel(t *testing.T) {
	mockClient := NewMockHTTPClient()
	manager := NewManagerWithClient("test-token", "test-account", mockClient)
	
	appName := "test-app"
	
	// Set up mock response
	response := CreateTunnelResponse{
		Success: true,
		Result: struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Token  string `json:"token,omitempty"`
			Status string `json:"status"`
		}{
			ID:     "tunnel-123",
			Name:   appName,
			Token:  "token-456",
			Status: "active",
		},
	}
	
	mockClient.SetJSONMockResponse(
		"https://api.cloudflare.com/client/v4/accounts/test-account/cfd_tunnel",
		http.StatusOK,
		response,
	)
	
	tunnelID, token, err := manager.CreateTunnel(appName)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if tunnelID != "tunnel-123" {
		t.Errorf("Expected tunnel ID 'tunnel-123', got %s", tunnelID)
	}
	
	if token != "token-456" {
		t.Errorf("Expected token 'token-456', got %s", token)
	}
	
	// Verify the request was made correctly
	if !mockClient.AssertRequestMade("POST", "https://api.cloudflare.com/client/v4/accounts/test-account/cfd_tunnel") {
		t.Error("Expected POST request to create tunnel")
	}
	
	// Verify request body
	body := mockClient.GetRequestBody("POST", "https://api.cloudflare.com/client/v4/accounts/test-account/cfd_tunnel")
	var reqBody CreateTunnelRequest
	if err := json.Unmarshal([]byte(body), &reqBody); err != nil {
		t.Fatalf("Failed to unmarshal request body: %v", err)
	}
	
	if reqBody.Name != appName {
		t.Errorf("Expected name %s, got %s", appName, reqBody.Name)
	}
}

func TestCreateTunnelError(t *testing.T) {
	mockClient := NewMockHTTPClient()
	manager := NewManagerWithClient("test-token", "test-account", mockClient)
	
	appName := "test-app"
	
	// Set up mock error response
	response := CreateTunnelResponse{
		Success: false,
		Errors: []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{
			{
				Code:    1001,
				Message: "Invalid request",
			},
		},
	}
	
	mockClient.SetJSONMockResponse(
		"https://api.cloudflare.com/client/v4/accounts/test-account/cfd_tunnel",
		http.StatusBadRequest,
		response,
	)
	
	tunnelID, token, err := manager.CreateTunnel(appName)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	if tunnelID != "" {
		t.Errorf("Expected empty tunnel ID on error, got %s", tunnelID)
	}
	
	if token != "" {
		t.Errorf("Expected empty token on error, got %s", token)
	}
}

func TestCreateIngressConfiguration(t *testing.T) {
	mockClient := NewMockHTTPClient()
	manager := NewManagerWithClient("test-token", "test-account", mockClient)
	
	tunnelID := "tunnel-123"
	ingressRules := []IngressRule{
		{
			Service: "http://localhost:8080",
			Hostname: "example.com",
		},
	}
	
	// Set up mock response
	response := struct {
		Success bool `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}{
		Success: true,
	}
	
	mockClient.SetJSONMockResponse(
		"https://api.cloudflare.com/client/v4/accounts/test-account/cfd_tunnel/tunnel-123/configurations",
		http.StatusOK,
		response,
	)
	
	err := manager.CreateIngressConfiguration(tunnelID, ingressRules)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Verify the request was made correctly
	if !mockClient.AssertRequestMade("PUT", "https://api.cloudflare.com/client/v4/accounts/test-account/cfd_tunnel/tunnel-123/configurations") {
		t.Error("Expected PUT request to create ingress configuration")
	}
}

func TestCreateIngressConfigurationError(t *testing.T) {
	mockClient := NewMockHTTPClient()
	manager := NewManagerWithClient("test-token", "test-account", mockClient)
	
	tunnelID := "tunnel-123"
	ingressRules := []IngressRule{
		{
			Service: "http://localhost:8080",
			Hostname: "example.com",
		},
	}
	
	// Set up mock error response
	response := struct {
		Success bool `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}{
		Success: false,
		Errors: []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{
			{
				Code:    1002,
				Message: "Invalid tunnel",
			},
		},
	}
	
	mockClient.SetJSONMockResponse(
		"https://api.cloudflare.com/client/v4/accounts/test-account/cfd_tunnel/tunnel-123/configurations",
		http.StatusNotFound,
		response,
	)
	
	err := manager.CreateIngressConfiguration(tunnelID, ingressRules)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestDeleteTunnel(t *testing.T) {
	mockClient := NewMockHTTPClient()
	manager := NewManagerWithClient("test-token", "test-account", mockClient)
	
	tunnelID := "tunnel-123"
	
	// Set up mock response
	response := struct {
		Success bool `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}{
		Success: true,
	}
	
	mockClient.SetJSONMockResponse(
		"https://api.cloudflare.com/client/v4/accounts/test-account/cfd_tunnel/tunnel-123",
		http.StatusOK,
		response,
	)
	
	err := manager.DeleteTunnel(tunnelID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Verify the request was made correctly
	if !mockClient.AssertRequestMade("DELETE", "https://api.cloudflare.com/client/v4/accounts/test-account/cfd_tunnel/tunnel-123") {
		t.Error("Expected DELETE request to delete tunnel")
	}
}
