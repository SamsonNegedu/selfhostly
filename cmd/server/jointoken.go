package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
)

// runJoinToken prints a single-use token a new secondary node presents to join this cluster. It is
// meant to be run on the primary (`docker exec selfhostly-primary ./selfhostly join-token`).
//
// While the server is running it asks the server itself, over its local port, using the gateway key
// already in the container's environment: the server is the only process that writes the database,
// so this never opens a second connection to it (concurrent processes on one SQLite file can fail on
// some bind mounts). Only when the server is not reachable does it open the database directly, which
// is then safe. The token goes to stdout on its own line; everything else goes to stderr.
func runJoinToken(args []string) int {
	fs := flag.NewFlagSet("join-token", flag.ExitOnError)
	ttl := fs.Duration("ttl", constants.JoinTokenTTL, "how long the token stays valid (direct database mode only)")
	_ = fs.Parse(args)

	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	_ = godotenv.Load(envFile)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration: %v\n", err)
		return 1
	}
	if !cfg.Node.IsPrimary {
		fmt.Fprintln(os.Stderr, "join tokens are issued by the primary node, and this node is a secondary")
		return 1
	}

	token, expires, viaServer, err := tokenFromRunningServer(cfg)
	switch {
	case err == nil:
	case errors.Is(err, errServerNotRunning):
		token, expires, err = tokenFromDatabase(cfg, *ttl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(os.Stderr, "could not create a token: %v\n", err)
		return 1
	}
	_ = viaServer
	fmt.Fprintf(os.Stderr, "single-use join token, valid until %s\n", expires.Local().Format(time.RFC1123))
	fmt.Println(token)
	return 0
}

var errServerNotRunning = errors.New("the server is not running")

// tokenFromRunningServer asks the local server for a token. It returns errServerNotRunning only when
// nothing is listening; any other failure (bad key, server error) is reported as is, never papered
// over by touching the database behind the server's back.
func tokenFromRunningServer(cfg *config.Config) (string, time.Time, bool, error) {
	_, port, err := net.SplitHostPort(cfg.ServerAddress)
	if err != nil || port == "" {
		port = "8080"
	}
	url := "http://127.0.0.1:" + port + "/api/nodes/join-tokens"
	if cfg.Node.GatewayAPIKey == "" {
		// Without the key the server would demand a user login: fall back to the database
		if !portOpen(port) {
			return "", time.Time{}, false, errServerNotRunning
		}
		return "", time.Time{}, false, errors.New("GATEWAY_API_KEY is not set, so the running server cannot be asked; " +
			"stop the primary and run this again, or use the API with a logged-in session")
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(nil))
	if err != nil {
		return "", time.Time{}, false, err
	}
	req.Header.Set(constants.HeaderGatewayAPIKey, cfg.Node.GatewayAPIKey)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		if !portOpen(port) {
			return "", time.Time{}, false, errServerNotRunning
		}
		return "", time.Time{}, false, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusCreated {
		return "", time.Time{}, false, fmt.Errorf("the server answered %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.Token == "" {
		return "", time.Time{}, false, errors.New("unexpected response from the server")
	}
	return out.Token, out.ExpiresAt, true, nil
}

func portOpen(port string) bool {
	c, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 2*time.Second)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// tokenFromDatabase is used only when the server is down, so no other process has the file open.
func tokenFromDatabase(cfg *config.Config, ttl time.Duration) (string, time.Time, error) {
	if _, err := os.Stat(cfg.DatabasePath); err != nil {
		return "", time.Time{}, fmt.Errorf("no database at %s: has the primary been started?", cfg.DatabasePath)
	}
	database, err := db.Init(cfg.DatabasePath)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()
	token, expires, err := database.CreateJoinToken(ttl)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create token: %w", err)
	}
	return token, expires, nil
}
