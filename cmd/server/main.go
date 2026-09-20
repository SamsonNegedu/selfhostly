package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/http"
	"github.com/selfhostly/internal/logger"
	"github.com/selfhostly/internal/secrets"
)

// auditRetention is how long audit records are kept
const auditRetention = 90 * 24 * time.Hour

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "doctor":
			os.Exit(runDoctor(os.Args[2:]))
		case "backup":
			os.Exit(runBackup(os.Args[2:]))
		case "join-token":
			os.Exit(runJoinToken(os.Args[2:]))
		}
	}

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
	logger.InitLogger(cfg.Environment, cfg.LogJSON)

	slog.Info("Application starting", "cwd", cwd, "environment", cfg.Environment, "security_mode", cfg.Security.Mode)

	// Refuse to start insecurely when enforcing; otherwise report the same problems as warnings
	fatal := false
	for _, issue := range cfg.ValidateStartup() {
		if issue.Fatal {
			slog.Error("configuration problem", "issue", issue.Message)
			fatal = true
		} else {
			slog.Warn("configuration warning", "issue", issue.Message)
		}
	}
	if fatal {
		logEffectiveConfig(cfg)
		slog.Error("refusing to start: fix the problems above, or set SECURITY_MODE=warn to start anyway")
		os.Exit(1)
	}
	if !cfg.Enforcing() {
		slog.Warn("SECURITY_MODE=warn: policy violations are logged, not blocked. Run `selfhostly doctor --audit-apps`, then set SECURITY_MODE=enforce")
	}

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

	// Secrets at rest: reading accepts plaintext and ciphertext; writes are encrypted when enabled
	dataDir := filepath.Dir(cfg.DatabasePath)
	secretKey, found := secrets.FindKey(cfg.Security.SettingsEncryptionKey, dataDir)
	if !found {
		// Never invent a new key next to data that was encrypted with a different one
		if database.HasSealedSecrets() {
			slog.Error("stored secrets are encrypted but no key is available: restore secrets.key into " + dataDir +
				" (or set SETTINGS_ENCRYPTION_KEY) from your backup. Nothing was changed")
			os.Exit(1)
		}
		secretKey, err = secrets.CreateKey(dataDir)
		if err != nil {
			slog.Error("Failed to create secrets key", "error", err)
			os.Exit(1)
		}
	}
	box, err := secrets.NewBox(secretKey, cfg.Security.EncryptSecretsAtRest)
	if err != nil {
		slog.Error("Failed to initialise secrets", "error", err)
		os.Exit(1)
	}
	database.SetSecretBox(box)

	// The database knows who this node is: use its ID unless one was set explicitly
	if id, err := database.PrimaryNodeID(); err == nil && cfg.AdoptPrimaryIdentity(id, true) {
		slog.Info("node id adopted from the database", "node_id", id)
	}

	// Logged here, after the database has been consulted, so it shows what this start really uses
	logEffectiveConfig(cfg)

	// Initialize node (bootstrap for primary nodes)
	if err := database.InitNode(cfg); err != nil {
		slog.Error("Failed to initialize node", "error", err)
		os.Exit(1)
	}

	if err := database.MigrateSecrets(); err != nil {
		slog.Error("Failed to encrypt secrets at rest", "error", err)
		os.Exit(1)
	}
	if err := database.PruneAudit(auditRetention); err != nil {
		slog.Warn("Failed to prune audit log", "error", err)
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
			"api_key_source", "NODE_API_KEY, or the node-api-key file next to the database",
			"register_url", cfg.Node.PrimaryNodeURL+"/nodes")

		if cfg.Node.APIEndpoint == "" {
			slog.Warn("NODE_API_ENDPOINT not set - using placeholder",
				"placeholder", "http://<this-server-ip>"+cfg.ServerAddress)
		}

		slog.Info("CRITICAL: Copy the Node ID above - required for heartbeat authentication!")
		slog.Info("NODE_ID and NODE_API_KEY are persisted next to the database, so they survive restarts")
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

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("Starting server", "address", cfg.ServerAddress)
		if err := server.Run(); err != nil {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal or server error
	select {
	case <-ctx.Done():
		slog.Info("Received shutdown signal, gracefully shutting down...")
		stop() // Stop signal notifications
	case err := <-serverErr:
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}

	// Graceful shutdown with 30 second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown with error", "error", err)
		os.Exit(1)
	}

	slog.Info("Server shutdown complete")
}
