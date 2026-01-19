package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
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

// CreateDNSRecordRequest represents a DNS record creation request
type CreateDNSRecordRequest struct {
	Type    string `json:"type"`
	Proxied bool   `json:"proxied"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

// CreateDNSRecordResponse represents a DNS record creation response
type CreateDNSRecordResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Messages []string `json:"messages"`
	Result   struct {
		ID      string `json:"id"`
		ZoneID  string `json:"zone_id"`
		Type    string `json:"type"`
		Name    string `json:"name"`
		Content string `json:"content"`
		Proxied bool   `json:"proxied"`
	} `json:"result"`
}

// IngressRule represents an ingress rule
type IngressRule struct {
	Service       string                 `json:"service"`
	Hostname      string                 `json:"hostname,omitempty"`
	Path          string                 `json:"path,omitempty"`
	OriginRequest map[string]interface{} `json:"originRequest,omitempty"`
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

// DeleteDNSRecordsForTunnel deletes all DNS records associated with a tunnel
func (m *Manager) DeleteDNSRecordsForTunnel(tunnelID string) error {
	// Get all zones
	zonesURL := fmt.Sprintf("%s/zones", apiBaseURL)

	req, err := http.NewRequest("GET", zonesURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create zones request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.config.APIToken)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get zones: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read zones response: %w", err)
	}

	var zonesData struct {
		Success bool `json:"success"`
		Result  []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
		Errors []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &zonesData); err != nil {
		return fmt.Errorf("failed to unmarshal zones response: %w", err)
	}

	if !zonesData.Success {
		return fmt.Errorf("failed to get zones: %v", zonesData.Errors)
	}

	// Search for DNS records related to this tunnel across all zones
	var errors []error
	for _, zone := range zonesData.Result {
		recordsURL := fmt.Sprintf("%s/zones/%s/dns_records?type=CNAME", apiBaseURL, zone.ID)

		req, err := http.NewRequest("GET", recordsURL, nil)
		if err != nil {
			slog.Warn("Failed to create DNS records request", "zone", zone.Name, "error", err)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+m.config.APIToken)

		resp, err := m.client.Do(req)
		if err != nil {
			slog.Warn("Failed to get DNS records", "zone", zone.Name, "error", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			slog.Warn("Failed to read DNS records response", "zone", zone.Name, "error", err)
			continue
		}

		var recordsData ListDNSRecordsResponse
		if err := json.Unmarshal(body, &recordsData); err != nil {
			slog.Warn("Failed to unmarshal DNS records response", "zone", zone.Name, "error", err)
			continue
		}

		if !recordsData.Success {
			slog.Warn("Failed to get DNS records", "zone", zone.Name, "errors", recordsData.Errors)
			continue
		}

		// Delete records that reference this tunnel
		for _, record := range recordsData.Result {
			if strings.Contains(record.Content, fmt.Sprintf("%s.cfargotunnel.com", tunnelID)) {
				deleteURL := fmt.Sprintf("%s/zones/%s/dns_records/%s", apiBaseURL, zone.ID, record.ID)

				req, err := http.NewRequest("DELETE", deleteURL, nil)
				if err != nil {
					slog.Warn("Failed to create DNS record delete request", "zone", zone.Name, "record", record.Name, "error", err)
					errors = append(errors, err)
					continue
				}

				req.Header.Set("Authorization", "Bearer "+m.config.APIToken)

				resp, err := m.client.Do(req)
				if err != nil {
					slog.Warn("Failed to delete DNS record", "zone", zone.Name, "record", record.Name, "error", err)
					errors = append(errors, err)
					continue
				}
				resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					slog.Warn("Failed to delete DNS record, non-200 status", "zone", zone.Name, "record", record.Name, "status", resp.StatusCode)
					errors = append(errors, fmt.Errorf("failed to delete DNS record %s, status: %d", record.Name, resp.StatusCode))
					continue
				}

				slog.Info("DNS record deleted successfully", "zone", zone.Name, "record", record.Name, "recordID", record.ID)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors while deleting DNS records, but deletion continued", len(errors))
	}

	return nil
}

// DeleteTunnel deletes a Cloudflare tunnel
func (m *Manager) DeleteTunnel(tunnelID string) error {
	// First, clean up associated DNS records
	if err := m.DeleteDNSRecordsForTunnel(tunnelID); err != nil {
		slog.Warn("Failed to clean up DNS records for tunnel, continuing with tunnel deletion", "tunnelID", tunnelID, "error", err)
	}

	// Then delete the tunnel itself
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

	slog.Info("Tunnel deleted successfully", "tunnelID", tunnelID)
	return nil
}

// CreateIngressConfiguration creates or updates tunnel ingress configuration
func (m *Manager) CreateIngressConfiguration(tunnelID string, ingressRules []IngressRule) error {
	url := fmt.Sprintf("%s/accounts/%s/cfd_tunnel/%s/configurations", apiBaseURL, m.config.AccountID, tunnelID)

	reqBody := CreateIngressRequest{
		Config: TunnelConfig{
			Ingress: ingressRules,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create ingress configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create ingress configuration, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetZoneID retrieves the zone ID for a given domain
func (m *Manager) GetZoneID(domain string) (string, error) {
	url := fmt.Sprintf("%s/zones?name=%s", apiBaseURL, domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.config.APIToken)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get zone ID: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var respData struct {
		Success bool `json:"success"`
		Result  []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !respData.Success || len(respData.Result) == 0 {
		return "", fmt.Errorf("no zone found for domain: %s", domain)
	}

	return respData.Result[0].ID, nil
}

// ListDNSRecordsResponse represents a list of DNS records response
type ListDNSRecordsResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Messages []string `json:"messages"`
	Result   []struct {
		ID      string `json:"id"`
		ZoneID  string `json:"zone_id"`
		Type    string `json:"type"`
		Name    string `json:"name"`
		Content string `json:"content"`
		Proxied bool   `json:"proxied"`
	} `json:"result"`
}

// GetDNSRecord retrieves a DNS record by name and type
func (m *Manager) GetDNSRecord(zoneID, hostname, recordType string) (*ListDNSRecordsResponse, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records?type=%s&name=%s", apiBaseURL, zoneID, recordType, hostname)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.config.APIToken)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS record: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var respData ListDNSRecordsResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !respData.Success {
		return nil, fmt.Errorf("failed to get DNS record: %v", respData.Errors)
	}

	return &respData, nil
}

// UpdateDNSRecord updates an existing DNS record
func (m *Manager) UpdateDNSRecord(zoneID, recordID, hostname, tunnelID string) error {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", apiBaseURL, zoneID, recordID)

	tunnelDomain := fmt.Sprintf("%s.cfargotunnel.com", tunnelID)
	reqBody := CreateDNSRecordRequest{
		Type:    "CNAME",
		Proxied: true,
		Name:    hostname,
		Content: tunnelDomain,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update DNS record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update DNS record, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateDNSRecord creates a DNS record for a tunnel (idempotent)
func (m *Manager) CreateDNSRecord(zoneID, hostname, tunnelID string) (string, error) {
	tunnelDomain := fmt.Sprintf("%s.cfargotunnel.com", tunnelID)

	// First check if a DNS record with this name already exists
	existingRecords, err := m.GetDNSRecord(zoneID, hostname, "CNAME")
	if err == nil && len(existingRecords.Result) > 0 {
		// Record already exists, update it instead of creating a new one
		recordID := existingRecords.Result[0].ID
		err = m.UpdateDNSRecord(zoneID, recordID, hostname, tunnelID)
		if err != nil {
			return "", fmt.Errorf("failed to update existing DNS record: %w", err)
		}
		return recordID, nil
	}

	// No existing record found, create a new one
	url := fmt.Sprintf("%s/zones/%s/dns_records", apiBaseURL, zoneID)

	reqBody := CreateDNSRecordRequest{
		Type:    "CNAME",
		Proxied: true,
		Name:    hostname,
		Content: tunnelDomain,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create DNS record: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var respData CreateDNSRecordResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !respData.Success {
		return "", fmt.Errorf("failed to create DNS record: %v", respData.Errors)
	}

	return respData.Result.ID, nil
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
