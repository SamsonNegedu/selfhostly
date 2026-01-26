package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/selfhostly/internal/db"
)

// Client handles communication with other nodes
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new inter-node API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// setNodeAuthHeaders sets the required authentication headers for inter-node requests
func (c *Client) setNodeAuthHeaders(req *http.Request, node *db.Node) {
	req.Header.Set("X-Node-ID", node.ID)
	req.Header.Set("X-Node-API-Key", node.APIKey)
}

// GetApps fetches all apps from a remote node
func (c *Client) GetApps(node *db.Node) ([]*db.App, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+"/api/internal/apps", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add node authentication
	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch apps from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var apps []*db.App
	if err := json.NewDecoder(resp.Body).Decode(&apps); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Add node information to each app for display
	for _, app := range apps {
		app.NodeID = node.ID
	}

	return apps, nil
}

// GetApp fetches a specific app from a remote node
func (c *Client) GetApp(node *db.Node, appID string) (*db.App, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+"/api/internal/apps/"+appID, nil)
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

	req, err := http.NewRequest("POST", node.APIEndpoint+"/api/internal/apps", bytes.NewBuffer(jsonData))
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

	req, err := http.NewRequest("PUT", node.APIEndpoint+"/api/internal/apps/"+appID, bytes.NewBuffer(jsonData))
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
	req, err := http.NewRequest("DELETE", node.APIEndpoint+"/api/internal/apps/"+appID, nil)
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
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/internal/apps/%s/%s", node.APIEndpoint, appID, action), nil)
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
	req, err := http.NewRequest("POST", node.APIEndpoint+"/api/internal/apps/"+appID+"/update", nil)
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

// GetSystemStats fetches system statistics from a remote node
func (c *Client) GetSystemStats(node *db.Node) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+"/api/internal/system/stats", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats from node %s: %w", node.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return stats, nil
}

// HealthCheck performs a health check on a remote node
func (c *Client) HealthCheck(node *db.Node) error {
	req, err := http.NewRequest("GET", node.APIEndpoint+"/api/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setNodeAuthHeaders(req, node)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetSettings fetches settings from the primary node (for secondary nodes)
func (c *Client) GetSettings(node *db.Node) (*db.Settings, error) {
	req, err := http.NewRequest("GET", node.APIEndpoint+"/api/internal/settings", nil)
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
	req, err := http.NewRequest("GET", node.APIEndpoint+"/api/internal/cloudflare/tunnels", nil)
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
	req, err := http.NewRequest("POST", node.APIEndpoint+"/api/system/containers/"+containerID+"/restart", nil)
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
	req, err := http.NewRequest("DELETE", node.APIEndpoint+"/api/system/containers/"+containerID, nil)
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
