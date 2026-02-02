package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
)

// bootstrapSingleNode handles the initial setup for primary node creation
// and migration of existing apps to multi-node architecture
func bootstrapSingleNode(db *sql.DB, cfg *config.Config) error {
	slog.Info("Bootstrapping node...")

	// Check if this node already has a record
	var existingNodeCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM nodes WHERE id = ?", cfg.Node.ID).Scan(&existingNodeCount); err != nil {
		return fmt.Errorf("failed to check for existing node: %w", err)
	}

	// If this node already exists in the database, skip bootstrap
	if existingNodeCount > 0 {
		slog.Info("Node already exists in database - skipping bootstrap", "node_name", cfg.Node.Name)
		return nil
	}

	// Handle SECONDARY nodes: Create their own local node record
	if !cfg.Node.IsPrimary {
		slog.Info("Skipping bootstrap - not configured as primary node")
		slog.Info("To register this as a secondary node, use the /nodes API on the primary")

		// Determine API endpoint - prefer NODE_API_ENDPOINT for multi-node setups
		apiEndpoint := cfg.Node.APIEndpoint
		if apiEndpoint == "" {
			// Fallback: use localhost (only for single-machine testing)
			apiEndpoint = "http://localhost" + cfg.ServerAddress
			slog.Warn("NODE_API_ENDPOINT not set - using fallback", "endpoint", apiEndpoint)
			slog.Info("For multi-node setups, set NODE_API_ENDPOINT to this node's reachable URL")
		}

		// Create a local node record for this secondary node
		secondaryNode := NewNode(cfg.Node.Name, apiEndpoint, cfg.Node.APIKey, false)
		// CRITICAL: Use the node ID from config, not the auto-generated one
		secondaryNode.ID = cfg.Node.ID

		_, err := db.Exec(
			`INSERT INTO nodes (id, name, api_endpoint, api_key, is_primary, status, created_at, updated_at, last_seen)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			secondaryNode.ID, secondaryNode.Name, secondaryNode.APIEndpoint,
			secondaryNode.APIKey, 0, secondaryNode.Status,
			secondaryNode.CreatedAt, secondaryNode.UpdatedAt, secondaryNode.LastSeen,
		)
		if err != nil {
			slog.Warn("Failed to create local node record", "error", err)
		} else {
			slog.Info("Created local node record", "node_name", secondaryNode.Name, "endpoint", apiEndpoint)
		}

		return nil
	}

	// Check if nodes table already has entries
	var nodeCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&nodeCount); err != nil {
		return err
	}

	if nodeCount > 0 {
		slog.Info("Found existing nodes - skipping bootstrap", "node_count", nodeCount)

		// Even though we're not bootstrapping, check if any apps need node assignment
		var primaryNodeID string
		if err := db.QueryRow("SELECT id FROM nodes WHERE is_primary = 1 LIMIT 1").Scan(&primaryNodeID); err == nil {
			// Assign any unassigned apps to primary node
			if err := assignUnassignedAppsToNode(db, primaryNodeID); err != nil {
				slog.Warn("Failed to assign apps to node", "error", err)
			}
		} else {
			slog.Warn("Could not find primary node to assign apps", "error", err)
		}

		return nil
	}

	// Check for split-brain: another primary node exists
	if cfg.Node.PrimaryNodeURL != "" {
		// If PRIMARY_NODE_URL is set, check if that primary exists
		err := checkPrimaryNodeExists(cfg.Node.PrimaryNodeURL, cfg.Node.APIKey)
		if err == nil {
			// Another primary exists and is reachable
			slog.Warn("Another primary node already exists!", "primary_url", cfg.Node.PrimaryNodeURL)
			slog.Error("Cannot create second primary node - this would cause split-brain")
			slog.Info("Set NODE_IS_PRIMARY=false to run as secondary node")
			return fmt.Errorf("cluster already has a primary node at %s", cfg.Node.PrimaryNodeURL)
		}
		// Primary node is unreachable - log warning but continue
		slog.Warn("PRIMARY_NODE_URL is set but unreachable")
		slog.Warn("Proceeding with primary bootstrap - ensure this is intentional")
	}

	// Check if there are any apps in the system
	var appCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM apps").Scan(&appCount); err != nil {
		return err
	}

	// Determine if this is a migration (existing apps) or new primary node
	if appCount > 0 {
		slog.Info("Migrating to multi-node architecture...", "app_count", appCount)
		slog.Info("Found existing apps - creating primary node entry")
	} else {
		slog.Info("Initializing primary node for new installation...")
	}

	// Determine API endpoint - prefer NODE_API_ENDPOINT for multi-node setups
	apiEndpoint := cfg.Node.APIEndpoint
	if apiEndpoint == "" {
		// Fallback: use localhost (only for single-machine testing)
		apiEndpoint = "http://localhost" + cfg.ServerAddress
		slog.Warn("NODE_API_ENDPOINT not set - using fallback", "endpoint", apiEndpoint)
		slog.Info("For multi-node setups, set NODE_API_ENDPOINT to this node's reachable URL")
	}

	// Create the primary node for this installation using the config's node ID
	primaryNode := NewNode(cfg.Node.Name, apiEndpoint, cfg.Node.APIKey, true)
	// CRITICAL: Use the node ID from config, not the auto-generated one
	primaryNode.ID = cfg.Node.ID

	// Insert the primary node
	_, err := db.Exec(
		`INSERT INTO nodes (id, name, api_endpoint, api_key, is_primary, status, created_at, updated_at, last_seen)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		primaryNode.ID, primaryNode.Name, primaryNode.APIEndpoint,
		primaryNode.APIKey, 1, primaryNode.Status,
		primaryNode.CreatedAt, primaryNode.UpdatedAt, primaryNode.LastSeen,
	)
	if err != nil {
		return fmt.Errorf("failed to create primary node: %w", err)
	}

	slog.Info("Created primary node", "node_name", primaryNode.Name, "node_id", primaryNode.ID)

	// Migrate existing apps to the newly created primary node
	if appCount > 0 {
		if err := assignUnassignedAppsToNode(db, primaryNode.ID); err != nil {
			return err
		}
	} else {
		slog.Info("Primary node initialized - ready for apps")
	}

	return nil
}

// assignUnassignedAppsToNode assigns apps without node_id to the specified node
func assignUnassignedAppsToNode(db *sql.DB, nodeID string) error {
	// Check if there are any unassigned apps
	var unassignedCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM apps WHERE node_id IS NULL OR node_id = ''").Scan(&unassignedCount); err != nil {
		return fmt.Errorf("failed to count unassigned apps: %w", err)
	}

	// Nothing to do if all apps are assigned
	if unassignedCount == 0 {
		return nil
	}

	// Get node name for logging
	var nodeName string
	if err := db.QueryRow("SELECT name FROM nodes WHERE id = ?", nodeID).Scan(&nodeName); err != nil {
		return fmt.Errorf("failed to get node name: %w", err)
	}

	slog.Info("Found apps without node assignment - assigning to node", "unassigned_count", unassignedCount, "node_name", nodeName)

	// Update only unassigned apps
	result, err := db.Exec(
		`UPDATE apps SET node_id = ? WHERE node_id IS NULL OR node_id = ''`,
		nodeID,
	)
	if err != nil {
		return fmt.Errorf("failed to assign apps to node: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	slog.Info("Assigned apps to node", "rows_affected", rowsAffected, "node_name", nodeName)

	return nil
}

// checkPrimaryNodeExists checks if a primary node is reachable at the given URL
func checkPrimaryNodeExists(primaryURL, apiKey string) error {
	client := &http.Client{Timeout: constants.HTTPClientTimeout}
	req, err := http.NewRequest("GET", primaryURL+"/api/health", nil)
	if err != nil {
		return err
	}

	// Add node auth header if we have an API key
	if apiKey != "" {
		req.Header.Set("X-Node-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil // Primary is reachable
	}

	return fmt.Errorf("primary returned status %d", resp.StatusCode)
}
