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

	// Create HTTP server
	server := http.NewServer(cfg, database)

	// Start server
	log.Printf("Starting server on %s", cfg.ServerAddress)
	if err := server.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
