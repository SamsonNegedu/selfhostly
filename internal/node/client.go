package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/selfhostly/internal/apipaths"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

// Client handles communication with other nodes
type Client struct {
	httpClient     *http.Client
	circuitBreaker *CircuitBreaker
}

// NewClient creates a new inter-node API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
		circuitBreaker: NewCircuitBreaker(),
	}
}

// setNodeAuthHeaders sets the required authentication headers for inter-node requests
func (c *Client) setNodeAuthHeaders(req *http.Request, node *db.Node) {
	req.Header.Set("X-Node-ID", node.ID)
	req.Header.Set("X-Node-API-Key", node.APIKey)
}

// GetApps fetches all apps from a remote node
func (c *Client) GetApps(node *db.Node) ([]*db.App, error) {
	// Check circuit breaker
	if c.circuitBreaker.IsOpen(node.ID) {
		stats := c.circuitBreaker.GetStats(node.ID)
		return nil, &CircuitOpenError{NodeID: node.ID, Stats: stats}
	}

	req, err := http.NewRequest("GET", node.APIEndpoint+apipaths.Apps, nil)
	if err != nil {
		c.circuitBreaker.RecordFailure(node.ID)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add node authentication
	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.circuitBreaker.RecordFailure(node.ID)
		return nil, fmt.Errorf("failed to fetch apps from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.circuitBreaker.RecordFailure(node.ID)
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var apps []*db.App
	if err := json.NewDecoder(resp.Body).Decode(&apps); err != nil {
		c.circuitBreaker.RecordFailure(node.ID)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Add node information to each app for display
	for _, app := range apps {
		app.NodeID = node.ID
	}

	// Record success
	c.circuitBreaker.RecordSuccess(node.ID)
	return apps, nil
}

// GetApp fetches a specific app from a remote node
func (c *Client) GetApp(node *db.Node, appID string) (*db.App, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+apipaths.AppByID(appID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch app from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var app db.App
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &app, nil
}

// CreateApp creates an app on a remote node
func (c *Client) CreateApp(node *db.Node, reqData interface{}) (*db.App, error) {
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", node.APIEndpoint+apipaths.Apps, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create app on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var app db.App
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &app, nil
}

// StartApp starts an app on a remote node
func (c *Client) StartApp(node *db.Node, appID string) error {
	return c.appAction(node, appID, "start")
}

// StopApp stops an app on a remote node
func (c *Client) StopApp(node *db.Node, appID string) error {
	return c.appAction(node, appID, "stop")
}

// UpdateApp updates an app on a remote node
func (c *Client) UpdateApp(node *db.Node, appID string, reqData interface{}) (*db.App, error) {
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", node.APIEndpoint+apipaths.AppByID(appID), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update app on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var app db.App
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &app, nil
}

// DeleteApp deletes an app from a remote node
func (c *Client) DeleteApp(node *db.Node, appID string) error {
	req, err := http.NewRequest("DELETE", node.APIEndpoint+apipaths.AppByID(appID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete app on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// appAction performs a start/stop action on an app
func (c *Client) appAction(node *db.Node, appID, action string) error {
	var path string
	switch action {
	case "start":
		path = node.APIEndpoint + apipaths.AppStart(appID)
	case "stop":
		path = node.APIEndpoint + apipaths.AppStop(appID)
	default:
		path = fmt.Sprintf("%s/api/apps/%s/%s", node.APIEndpoint, appID, action)
	}
	req, err := http.NewRequest("POST", path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to %s app on node %s: %w", action, node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateAppContainers triggers a container update on a remote node
func (c *Client) UpdateAppContainers(node *db.Node, appID string) (*db.App, error) {
	req, err := http.NewRequest("POST", node.APIEndpoint+apipaths.AppUpdateContainers(appID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update app containers on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var app db.App
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &app, nil
}

// GetQuickTunnelURL fetches the Quick Tunnel URL for an app from a remote node.
// The node runs extraction locally and returns the trycloudflare.com URL.
func (c *Client) GetQuickTunnelURL(node *db.Node, appID string) (string, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+apipaths.AppQuickTunnelURL(appID), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get quick tunnel URL from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var out struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	return out.URL, nil
}

// CreateQuickTunnelForApp adds a Quick Tunnel to an app that has no tunnel on a remote node.
func (c *Client) CreateQuickTunnelForApp(node *db.Node, appID string, service string, port int) (*db.App, error) {
	body := map[string]interface{}{"service": service, "port": port}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", node.APIEndpoint+apipaths.AppQuickTunnel(appID), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create quick tunnel on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var app db.App
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	app.NodeID = node.ID
	return &app, nil
}

// SwitchAppToCustomTunnel forwards switch-to-custom to a remote node so that node processes the request (its DB, its Cloudflare config).
func (c *Client) SwitchAppToCustomTunnel(node *db.Node, appID string, body interface{}) (*db.App, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequest("POST", node.APIEndpoint+apipaths.TunnelSwitchToCustom(appID), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setNodeAuthHeaders(req, node)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to switch to custom tunnel on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var app db.App
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	app.NodeID = node.ID
	return &app, nil
}

// CreateTunnelForApp forwards create-tunnel (custom domain) to a remote node so that node processes the request (its DB, its Cloudflare config).
func (c *Client) CreateTunnelForApp(node *db.Node, appID string, body interface{}) (*db.App, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	url := node.APIEndpoint + apipaths.TunnelByApp(appID) + "?node_id=" + node.ID
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setNodeAuthHeaders(req, node)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create tunnel for app on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var app db.App
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	app.NodeID = node.ID
	return &app, nil
}

// GetSystemStats fetches system statistics from a remote node
func (c *Client) GetSystemStats(node *db.Node) (map[string]interface{}, error) {
	// Check circuit breaker
	if c.circuitBreaker.IsOpen(node.ID) {
		stats := c.circuitBreaker.GetStats(node.ID)
		return nil, &CircuitOpenError{NodeID: node.ID, Stats: stats}
	}

	req, err := http.NewRequest("GET", node.APIEndpoint+apipaths.SystemStats, nil)
	if err != nil {
		c.circuitBreaker.RecordFailure(node.ID)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.circuitBreaker.RecordFailure(node.ID)
		return nil, fmt.Errorf("failed to fetch stats from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.circuitBreaker.RecordFailure(node.ID)
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		c.circuitBreaker.RecordFailure(node.ID)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Record success
	c.circuitBreaker.RecordSuccess(node.ID)
	return stats, nil
}

// HealthCheck performs a health check on a remote node
func (c *Client) HealthCheck(node *db.Node) error {
	// Check circuit breaker
	if c.circuitBreaker.IsOpen(node.ID) {
		stats := c.circuitBreaker.GetStats(node.ID)
		return &CircuitOpenError{NodeID: node.ID, Stats: stats}
	}

	req, err := http.NewRequest("GET", node.APIEndpoint+apipaths.Health, nil)
	if err != nil {
		c.circuitBreaker.RecordFailure(node.ID)
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.circuitBreaker.RecordFailure(node.ID)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.circuitBreaker.RecordFailure(node.ID)
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	// Record success
	c.circuitBreaker.RecordSuccess(node.ID)
	return nil
}

// GetSettings fetches settings from the primary node (for secondary nodes)
func (c *Client) GetSettings(node *db.Node) (*db.Settings, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+apipaths.Settings, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch settings from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var settings db.Settings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &settings, nil
}

// GetTunnels fetches all tunnels from a remote node
func (c *Client) GetTunnels(node *db.Node) ([]*db.CloudflareTunnel, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+apipaths.TunnelsList, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tunnels from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var tunnels []*db.CloudflareTunnel
	if err := json.NewDecoder(resp.Body).Decode(&tunnels); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tunnels, nil
}

// RestartContainer restarts a container on a remote node
func (c *Client) RestartContainer(node *db.Node, containerID string) error {
	req, err := http.NewRequest("POST", node.APIEndpoint+apipaths.ContainerRestart(containerID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to restart container on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// StopContainer stops a container on a remote node
func (c *Client) StopContainer(node *db.Node, containerID string) error {
	req, err := http.NewRequest("POST", node.APIEndpoint+"/api/system/containers/"+containerID+"/stop", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop container on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteContainer deletes a container on a remote node
func (c *Client) DeleteContainer(node *db.Node, containerID string) error {
	req, err := http.NewRequest("DELETE", node.APIEndpoint+apipaths.Container(containerID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete container on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetComposeVersions fetches all compose versions for an app from a remote node
func (c *Client) GetComposeVersions(node *db.Node, appID string) ([]*db.ComposeVersion, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+apipaths.AppComposeVersions(appID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch compose versions from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var versions []*db.ComposeVersion
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return versions, nil
}

// GetComposeVersion fetches a specific compose version for an app from a remote node
func (c *Client) GetComposeVersion(node *db.Node, appID string, version int) (*db.ComposeVersion, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+apipaths.AppComposeVersion(appID, version), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch compose version from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var composeVersion *db.ComposeVersion
	if err := json.NewDecoder(resp.Body).Decode(&composeVersion); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return composeVersion, nil
}

// RollbackComposeVersion rolls back an app to a specific compose version on a remote node
func (c *Client) RollbackComposeVersion(node *db.Node, appID string, version int, reason *string, changedBy *string) (*db.ComposeVersion, error) {
	// Prepare request body
	body := make(map[string]interface{})
	if reason != nil {
		body["change_reason"] = *reason
	}
	if changedBy != nil {
		body["changed_by"] = *changedBy
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", node.APIEndpoint+apipaths.AppComposeRollback(appID, version), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to rollback compose version on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		NewVersion *db.ComposeVersion `json:"new_version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.NewVersion, nil
}

// GetAppLogs fetches logs for an app from a remote node
func (c *Client) GetAppLogs(node *db.Node, appID string) ([]byte, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+apipaths.AppLogs(appID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch app logs from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	logs, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return logs, nil
}

// GetAppStats fetches stats for an app from a remote node
func (c *Client) GetAppStats(node *db.Node, appID string) (*domain.AppStats, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+apipaths.AppStats(appID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch app stats from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var stats *domain.AppStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return stats, nil
}

// GetTunnelByAppID fetches tunnel for an app from a remote node
func (c *Client) GetTunnelByAppID(node *db.Node, appID string) (*db.CloudflareTunnel, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+apipaths.TunnelByApp(appID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tunnel from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Remote returns the same envelope as primary: { tunnel, app_id, tunnel_mode, node_id, public_url }.
	var envelope struct {
		Tunnel *db.CloudflareTunnel `json:"tunnel"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if envelope.Tunnel == nil {
		return nil, domain.ErrTunnelNotFound
	}
	return envelope.Tunnel, nil
}

// SyncTunnelStatus syncs tunnel status on a remote node
func (c *Client) SyncTunnelStatus(node *db.Node, appID string) error {
	req, err := http.NewRequest("POST", node.APIEndpoint+apipaths.TunnelSync(appID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to sync tunnel on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateTunnelIngress updates tunnel ingress rules on a remote node
func (c *Client) UpdateTunnelIngress(node *db.Node, appID string, req domain.UpdateIngressRequest) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("PUT", node.APIEndpoint+apipaths.TunnelIngress(appID), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	c.setNodeAuthHeaders(httpReq, node)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to update tunnel ingress on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateTunnelDNSRecord creates a DNS record for a tunnel on a remote node
func (c *Client) CreateTunnelDNSRecord(node *db.Node, appID string, req domain.CreateDNSRequest) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", node.APIEndpoint+apipaths.TunnelDNS(appID), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	c.setNodeAuthHeaders(httpReq, node)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to create DNS record on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteTunnel deletes a tunnel on a remote node
func (c *Client) DeleteTunnel(node *db.Node, appID string) error {
	req, err := http.NewRequest("DELETE", node.APIEndpoint+apipaths.TunnelByApp(appID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete tunnel on node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
