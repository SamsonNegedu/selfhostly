package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/http"
)

func main() {
	// Show current working directory for debugging
	cwd, _ := os.Getwd()
	log.Printf("Current working directory: %s", cwd)

	// Load .env file if it exists (optional, won't error if missing)
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found in %s: %v", cwd, err)
	} else {
		log.Println("Loaded .env file successfully")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Debug: show auth configuration
	log.Printf("Auth enabled: %v", cfg.Auth.Enabled)
	if cfg.Auth.Enabled {
		clientID := cfg.Auth.GitHub.ClientID
		if len(clientID) > 8 {
			clientID = clientID[:8]
		}
		log.Printf("GitHub OAuth configured: ClientID=%s...", clientID)
		
		// Show allowed users count (but not the actual usernames for security)
		if len(cfg.Auth.GitHub.AllowedUsers) > 0 {
			log.Printf("GitHub whitelist configured: %d user(s) allowed", len(cfg.Auth.GitHub.AllowedUsers))
		} else {
			log.Printf("WARNING: GitHub auth enabled but no allowed users configured - all access will be denied")
		}
	}

	// Initialize database
	database, err := db.Init(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize node (bootstrap for primary nodes)
	if err := database.InitNode(cfg); err != nil {
		log.Fatalf("Failed to initialize node: %v", err)
	}

	// Verify node setup
	nodes, err := database.GetAllNodes()
	if err != nil {
		log.Fatalf("Failed to query nodes: %v", err)
	}

	// For PRIMARY nodes: should have at least one node (itself)
	// For SECONDARY nodes: may have zero nodes (until registered)
	if len(nodes) == 0 && cfg.Node.IsPrimary {
		// This should never happen for primary nodes
		log.Fatal("❌ No nodes found on primary node - this is a critical error")
	} else if len(nodes) == 0 {
		// Secondary node not yet registered - show registration information
		log.Println("")
		log.Println("═══════════════════════════════════════════════════════════")
		log.Println("⚠️  SECONDARY NODE - NOT YET REGISTERED")
		log.Println("═══════════════════════════════════════════════════════════")
		log.Println("")
		log.Println("📝 Use these details to register this node on the primary:")
		log.Println("")
		log.Printf("   Node Name:     %s", cfg.Node.Name)
		log.Printf("   API Endpoint:  http://<this-server-ip>%s", cfg.ServerAddress)
		log.Printf("   API Key:       %s", cfg.Node.APIKey)
		log.Println("")
		log.Printf("💡 Register at:   %s/nodes", cfg.Node.PrimaryNodeURL)
		log.Println("")
		log.Println("ℹ️  Server will start but cannot manage apps until registered")
		log.Println("═══════════════════════════════════════════════════════════")
		log.Println("")
	} else {
		// Find primary or current node
		var primaryNode *db.Node
		for _, node := range nodes {
			if node.IsPrimary {
				primaryNode = node
				break
			}
		}

		if primaryNode != nil {
			log.Printf("✓ Running as: %s (Primary)", primaryNode.Name)
		} else {
			log.Printf("✓ Running as: %s (Secondary)", cfg.Node.Name)
		}

		// Verify all apps have node assignments (for migration verification)
		apps, err := database.GetAllApps()
		if err == nil {
			unassignedCount := 0
			for _, app := range apps {
				if app.NodeID == "" {
					unassignedCount++
				}
			}

			if unassignedCount > 0 {
				log.Printf("⚠️  Warning: %d apps without node assignment", unassignedCount)
			} else if len(apps) > 0 {
				log.Printf("✓ All %d apps have valid node assignments", len(apps))
			}
		}
	}

	// Create HTTP server
	server := http.NewServer(cfg, database)

	// Start server
	log.Printf("Starting server on %s", cfg.ServerAddress)
	if err := server.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
