package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

const (
	apiBaseURL = "https://api.cloudflare.com/client/v4"
)

// APICredentials holds Cloudflare API credentials
type APICredentials struct {
	APIToken  string
	AccountID string
}

// CreateTunnelRequest represents a create tunnel request
type CreateTunnelRequest struct {
	Name         string `json:"name"`
	TunnelSecret string `json:"tunnelSecret,omitempty"`
}

// CreateTunnelResponse represents a create tunnel response
type CreateTunnelResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Messages []string `json:"messages"`
	Result   struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Token  string `json:"token,omitempty"`
		Status string `json:"status"`
	} `json:"result"`
}

// CreateIngressRequest represents an ingress configuration request
type CreateIngressRequest struct {
	Config TunnelConfig `json:"config"`
}

// TunnelConfig represents tunnel configuration
type TunnelConfig struct {
	Ingress []IngressRule `json:"ingress"`
}

// IngressRule represents an ingress rule
type IngressRule struct {
	Service  string `json:"service"`
	Hostname string `json:"hostname,omitempty"`
	Path     string `json:"path,omitempty"`
}

// Manager handles Cloudflare tunnel operations
type Manager struct {
	config *APICredentials
	client *http.Client
}

// NewManager creates a new Cloudflare tunnel manager
func NewManager(apiToken, accountID string) *Manager {
	return &Manager{
		config: &APICredentials{
			APIToken:  apiToken,
			AccountID: accountID,
		},
		client: &http.Client{},
	}
}

// CreateTunnel creates a new Cloudflare tunnel
func (m *Manager) CreateTunnel(appName string) (tunnelID, token string, err error) {
	url := fmt.Sprintf("%s/accounts/%s/cfd_tunnel", apiBaseURL, m.config.AccountID)

	reqBody := CreateTunnelRequest{
		Name: appName,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to create tunnel: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	var respData CreateTunnelResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !respData.Success {
		return "", "", fmt.Errorf("cloudflare API error: %v", respData.Errors)
	}

	tunnelID = respData.Result.ID

	// Extract token from the response
	if respData.Result.Token == "" {
		return tunnelID, "", fmt.Errorf("tunnel token is empty in create response")
	}
	token = respData.Result.Token

	slog.Info("Tunnel creation successful", "tunnelID", tunnelID, "tokenLength", len(token))
	return tunnelID, token, nil
}

// DeleteTunnel deletes a Cloudflare tunnel
func (m *Manager) DeleteTunnel(tunnelID string) error {
	url := fmt.Sprintf("%s/accounts/%s/cfd_tunnel/%s", apiBaseURL, m.config.AccountID, tunnelID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.config.APIToken)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete tunnel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete tunnel, status: %d", resp.StatusCode)
	}

	return nil
}

// CreatePublicRoute creates a public route for the tunnel
func (m *Manager) CreatePublicRoute(tunnelID, service string) (publicURL string, err error) {
	// In a real implementation, this would configure the tunnel's ingress rules
	// For now, we return a placeholder URL
	return fmt.Sprintf("https://%s.trycloudflare.com", tunnelID), nil
}

// GetTunnelToken gets only the tunnel token (for cases where we need to retrieve it later)
func (m *Manager) GetTunnelToken(tunnelID string) (string, error) {
	url := fmt.Sprintf("%s/accounts/%s/cfd_tunnel/%s", apiBaseURL, m.config.AccountID, tunnelID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.config.APIToken)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get tunnel: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var respData CreateTunnelResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !respData.Success {
		return "", fmt.Errorf("cloudflare API error: %v", respData.Errors)
	}

	if respData.Result.Token == "" {
		return "", fmt.Errorf("tunnel token is empty in response")
	}

	return respData.Result.Token, nil
}
