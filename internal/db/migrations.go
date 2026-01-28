package db

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/selfhostly/internal/config"
)

// bootstrapSingleNode handles the initial setup for primary node creation
// and migration of existing apps to multi-node architecture
func bootstrapSingleNode(db *sql.DB, cfg *config.Config) error {
	log.Println("Bootstrapping node...")

	// Check if this node already has a record
	var existingNodeCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM nodes WHERE id = ?", cfg.Node.ID).Scan(&existingNodeCount); err != nil {
		return fmt.Errorf("failed to check for existing node: %w", err)
	}

	// If this node already exists in the database, skip bootstrap
	if existingNodeCount > 0 {
		log.Printf("INFO: Node %s already exists in database - skipping bootstrap", cfg.Node.Name)
		return nil
	}

	// Handle SECONDARY nodes: Create their own local node record
	if !cfg.Node.IsPrimary {
		log.Println("INFO: Skipping bootstrap - not configured as primary node")
		log.Println("INFO: To register this as a secondary node, use the /nodes API on the primary")

		// Determine API endpoint - prefer NODE_API_ENDPOINT for multi-node setups
		apiEndpoint := cfg.Node.APIEndpoint
		if apiEndpoint == "" {
			// Fallback: use localhost (only for single-machine testing)
			apiEndpoint = "http://localhost" + cfg.ServerAddress
			log.Printf("WARNING: NODE_API_ENDPOINT not set - using %s", apiEndpoint)
			log.Println("INFO: For multi-node setups, set NODE_API_ENDPOINT to this node's reachable URL")
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
			log.Printf("WARNING: Failed to create local node record: %v", err)
		} else {
			log.Printf("Created local node record for: %s (endpoint: %s)", secondaryNode.Name, apiEndpoint)
		}

		return nil
	}

	// Check if nodes table already has entries
	var nodeCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&nodeCount); err != nil {
		return err
	}

	if nodeCount > 0 {
		log.Printf("INFO: Found %d existing node(s) - skipping bootstrap", nodeCount)

		// Even though we're not bootstrapping, check if any apps need node assignment
		var primaryNodeID string
		if err := db.QueryRow("SELECT id FROM nodes WHERE is_primary = 1 LIMIT 1").Scan(&primaryNodeID); err == nil {
			// Assign any unassigned apps to primary node
			if err := assignUnassignedAppsToNode(db, primaryNodeID); err != nil {
				log.Printf("WARNING: Failed to assign apps to node: %v", err)
			}
		} else {
			log.Printf("WARNING: Could not find primary node to assign apps: %v", err)
		}

		return nil
	}

	// Check for split-brain: another primary node exists
	if cfg.Node.PrimaryNodeURL != "" {
		// If PRIMARY_NODE_URL is set, check if that primary exists
		err := checkPrimaryNodeExists(cfg.Node.PrimaryNodeURL, cfg.Node.APIKey)
		if err == nil {
			// Another primary exists and is reachable
			log.Println("WARNING: Another primary node already exists!")
			log.Printf("WARNING: Primary found at: %s", cfg.Node.PrimaryNodeURL)
			log.Println("ERROR: Cannot create second primary node - this would cause split-brain")
			log.Println("INFO: Set NODE_IS_PRIMARY=false to run as secondary node")
			return fmt.Errorf("cluster already has a primary node at %s", cfg.Node.PrimaryNodeURL)
		}
		// Primary node is unreachable - log warning but continue
		log.Println("WARNING: PRIMARY_NODE_URL is set but unreachable")
		log.Println("WARNING: Proceeding with primary bootstrap - ensure this is intentional")
	}

	// Check if there are any apps in the system
	var appCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM apps").Scan(&appCount); err != nil {
		return err
	}

	// Determine if this is a migration (existing apps) or new primary node
	if appCount > 0 {
		log.Println("Migrating to multi-node architecture...")
		log.Printf("Found %d existing apps - creating primary node entry", appCount)
	} else {
		log.Println("Initializing primary node for new installation...")
	}

	// Determine API endpoint - prefer NODE_API_ENDPOINT for multi-node setups
	apiEndpoint := cfg.Node.APIEndpoint
	if apiEndpoint == "" {
		// Fallback: use localhost (only for single-machine testing)
		apiEndpoint = "http://localhost" + cfg.ServerAddress
		log.Printf("WARNING: NODE_API_ENDPOINT not set - using %s", apiEndpoint)
		log.Println("INFO: For multi-node setups, set NODE_API_ENDPOINT to this node's reachable URL")
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

	log.Printf("Created primary node: %s (ID: %s)", primaryNode.Name, primaryNode.ID)

	// Migrate existing apps to the newly created primary node
	if appCount > 0 {
		if err := assignUnassignedAppsToNode(db, primaryNode.ID); err != nil {
			return err
		}
	} else {
		log.Println("Primary node initialized - ready for apps")
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

	log.Printf("Found %d apps without node assignment - assigning to %s...", unassignedCount, nodeName)

	// Update only unassigned apps
	result, err := db.Exec(
		`UPDATE apps SET node_id = ? WHERE node_id IS NULL OR node_id = ''`,
		nodeID,
	)
	if err != nil {
		return fmt.Errorf("failed to assign apps to node: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Assigned %d apps to %s", rowsAffected, nodeName)

	return nil
}

// checkPrimaryNodeExists checks if a primary node is reachable at the given URL
func checkPrimaryNodeExists(primaryURL, apiKey string) error {
	client := &http.Client{Timeout: 5 * time.Second}
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
