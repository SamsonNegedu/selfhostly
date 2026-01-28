package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/http"
	"github.com/selfhostly/internal/logger"
)

func main() {
	// Show current working directory for debugging
	cwd, _ := os.Getwd()
	
	// Load .env file - can be overridden with ENV_FILE environment variable
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	
	if err := godotenv.Load(envFile); err != nil {
		// Use default logger temporarily before config is loaded
		slog.Warn("No .env file found", "file", envFile, "cwd", cwd, "error", err)
	} else {
		slog.Info("Loaded .env file successfully", "file", envFile)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize structured logger based on environment
	// This sets slog as the default logger, so we can use slog directly throughout
	logger.InitLogger(cfg.Environment)
	
	slog.Info("Application starting", "cwd", cwd, "environment", cfg.Environment)

	// Debug: show auth configuration
	slog.Info("Auth configuration", "enabled", cfg.Auth.Enabled)
	if cfg.Auth.Enabled {
		clientID := cfg.Auth.GitHub.ClientID
		if len(clientID) > 8 {
			clientID = clientID[:8]
		}
		slog.Info("GitHub OAuth configured", "client_id_prefix", clientID+"...")
		
		// Show allowed users count (but not the actual usernames for security)
		if len(cfg.Auth.GitHub.AllowedUsers) > 0 {
			slog.Info("GitHub whitelist configured", "allowed_users_count", len(cfg.Auth.GitHub.AllowedUsers))
		} else {
			slog.Warn("GitHub auth enabled but no allowed users configured - all access will be denied")
		}
	}

	// Initialize database
	database, err := db.Init(cfg.DatabasePath)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Initialize node (bootstrap for primary nodes)
	if err := database.InitNode(cfg); err != nil {
		slog.Error("Failed to initialize node", "error", err)
		os.Exit(1)
	}

	// Verify node setup
	nodes, err := database.GetAllNodes()
	if err != nil {
		slog.Error("Failed to query nodes", "error", err)
		os.Exit(1)
	}

	// For PRIMARY nodes: should have at least one node (itself)
	// For SECONDARY nodes: may have zero nodes (until registered)
	if len(nodes) == 0 && cfg.Node.IsPrimary {
		// This should never happen for primary nodes
		slog.Error("No nodes found on primary node - this is a critical error")
		os.Exit(1)
	} else if len(nodes) == 0 {
		// Secondary node not yet registered - show registration information
		slog.Warn("SECONDARY NODE - NOT YET REGISTERED")
		slog.Info("Use these details to register this node on the primary",
			"node_id", cfg.Node.ID,
			"node_name", cfg.Node.Name,
			"api_endpoint", cfg.Node.APIEndpoint,
			"api_key", cfg.Node.APIKey,
			"register_url", cfg.Node.PrimaryNodeURL+"/nodes")
		
		if cfg.Node.APIEndpoint == "" {
			slog.Warn("NODE_API_ENDPOINT not set - using placeholder",
				"placeholder", "http://<this-server-ip>"+cfg.ServerAddress)
		}
		
		slog.Info("CRITICAL: Copy the Node ID above - required for heartbeat authentication!")
		slog.Info("IMPORTANT: Save NODE_ID and NODE_API_KEY to .env to keep them consistent")
		slog.Info("INFO: Server will start but cannot manage apps until registered")
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
			slog.Info("Running as primary node", "node_name", primaryNode.Name)
		} else {
			slog.Info("Running as secondary node", "node_name", cfg.Node.Name)
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
				slog.Warn("Apps without node assignment", "count", unassignedCount)
			} else if len(apps) > 0 {
				slog.Info("All apps have valid node assignments", "total_apps", len(apps))
			}
		}
	}

	// Create HTTP server
	server := http.NewServer(cfg, database)

	// Start server
	slog.Info("Starting server", "address", cfg.ServerAddress)
	if err := server.Run(); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
