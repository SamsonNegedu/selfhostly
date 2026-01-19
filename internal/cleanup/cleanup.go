package cleanup

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/selfhost-automaton/internal/cloudflare"
	"github.com/selfhost-automaton/internal/db"
	"github.com/selfhost-automaton/internal/docker"
)

// CleanupResult represents the result of a cleanup operation
type CleanupResult struct {
	Step         string    `json:"step"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Duration     time.Duration `json:"duration"`
}

// CleanupOperation represents a single cleanup step
type CleanupOperation struct {
	Name     string
	Executor  func() error
	OnSuccess func()
	OnError   func(error)
}

// CleanupManager handles comprehensive cleanup operations
type CleanupManager struct {
	dockerManager  *docker.Manager
	database       *db.DB
	settings       *db.Settings
	tunnelManager  *cloudflare.TunnelManager
	startTime      time.Time
	results        []CleanupResult
	operationCount int
}

// NewCleanupManager creates a new cleanup manager
func NewCleanupManager(dockerManager *docker.Manager, database *db.DB, settings *db.Settings, tunnelManager *cloudflare.TunnelManager) *CleanupManager {
	return &CleanupManager{
		dockerManager: dockerManager,
		database:      database,
		settings:      settings,
		tunnelManager: tunnelManager,
		startTime:     time.Now(),
		results:       make([]CleanupResult, 0),
	}
}

// CleanupApp performs comprehensive cleanup of an app and its dependencies
func (cm *CleanupManager) CleanupApp(app *db.App) ([]CleanupResult, error) {
	slog.Info("Starting comprehensive app cleanup", "app", app.Name, "appID", app.ID)
	cm.startTime = time.Now()
	cm.results = make([]CleanupResult, 0)
	
	// Define cleanup operations in the correct order (reverse dependency)
	operations := []CleanupOperation{
		{
			Name: "Stop Docker containers",
			Executor: func() error {
				return cm.dockerManager.StopApp(app.Name)
			},
			OnSuccess: func() {
				slog.Info("Successfully stopped Docker containers", "app", app.Name)
			},
			OnError: func(err error) {
				slog.Warn("Failed to stop Docker containers, continuing anyway", "app", app.Name, "error", err)
			},
		},
		{
			Name: "Clean up Cloudflare DNS records",
			Executor: func() error {
				if app.TunnelID == "" {
					return nil // No tunnel to clean up
				}
				return cm.tunnelManager.ApiManager.DeleteDNSRecordsForTunnel(app.TunnelID)
			},
			OnSuccess: func() {
				slog.Info("Successfully cleaned up Cloudflare DNS records", "app", app.Name, "tunnelID", app.TunnelID)
			},
			OnError: func(err error) {
				slog.Warn("Failed to clean up Cloudflare DNS records, continuing anyway", "app", app.Name, "tunnelID", app.TunnelID, "error", err)
			},
		},
		{
			Name: "Delete Cloudflare tunnel from API",
			Executor: func() error {
				if app.TunnelID == "" {
					return nil // No tunnel to delete
				}
				return cm.tunnelManager.ApiManager.DeleteTunnel(app.TunnelID)
			},
			OnSuccess: func() {
				slog.Info("Successfully deleted Cloudflare tunnel from API", "app", app.Name, "tunnelID", app.TunnelID)
			},
			OnError: func(err error) {
				slog.Warn("Failed to delete Cloudflare tunnel from API, continuing anyway", "app", app.Name, "tunnelID", app.TunnelID, "error", err)
			},
		},
		{
			Name: "Delete Cloudflare tunnel from database",
			Executor: func() error {
				if app.TunnelID == "" {
					return nil // No tunnel to delete
				}
				return cm.tunnelManager.DeleteTunnelByAppID(app.ID)
			},
			OnSuccess: func() {
				slog.Info("Successfully deleted Cloudflare tunnel from database", "app", app.Name, "tunnelID", app.TunnelID)
			},
			OnError: func(err error) {
				slog.Warn("Failed to delete Cloudflare tunnel from database, continuing anyway", "app", app.Name, "tunnelID", app.TunnelID, "error", err)
			},
		},
		{
			Name: "Delete app directory",
			Executor: func() error {
				return cm.dockerManager.DeleteAppDirectory(app.Name)
			},
			OnSuccess: func() {
				slog.Info("Successfully deleted app directory", "app", app.Name)
			},
			OnError: func(err error) {
				slog.Error("Failed to delete app directory", "app", app.Name, "error", err)
			},
		},
		{
			Name: "Delete app from database",
			Executor: func() error {
				return cm.database.DeleteApp(app.ID)
			},
			OnSuccess: func() {
				slog.Info("Successfully deleted app from database", "app", app.Name, "appID", app.ID)
			},
			OnError: func(err error) {
				slog.Error("Failed to delete app from database", "app", app.Name, "appID", app.ID, "error", err)
			},
		},
	}
	
	// Execute all cleanup operations
	var lastError error
	for _, operation := range operations {
		start := time.Now()
		result := CleanupResult{
			Step:     operation.Name,
			Duration: 0,
		}
		
		err := operation.Executor()
		result.Duration = time.Since(start)
		result.Success = err == nil
		
		if err != nil {
			result.Error = err.Error()
			result.ErrorMessage = err.Error()
			lastError = err
		}
		
		cm.results = append(cm.results, result)
		cm.operationCount++
		
		// Execute callbacks
		if err == nil {
			if operation.OnSuccess != nil {
				operation.OnSuccess()
			}
		} else {
			if operation.OnError != nil {
				operation.OnError(err)
			}
		}
	}
	
	// Log summary
	successCount := 0
	for _, result := range cm.results {
		if result.Success {
			successCount++
		}
	}
	
	totalDuration := time.Since(cm.startTime)
	slog.Info("App cleanup completed",
		"app", app.Name,
		"appID", app.ID,
		"totalSteps", len(operations),
		"successSteps", successCount,
		"failedSteps", len(operations)-successCount,
		"totalDuration", totalDuration,
	)
	
	// Log details for failed operations
	for _, result := range cm.results {
		if !result.Success {
			slog.Error("Cleanup step failed",
				"app", app.Name,
				"step", result.Step,
				"error", result.Error,
				"duration", result.Duration,
			)
		}
	}
	
	// Return comprehensive results and any critical errors
	if lastError != nil {
		return cm.results, fmt.Errorf("cleanup completed with errors: %w", lastError)
	}
	
	return cm.results, nil
}

// GetResults returns the cleanup results
func (cm *CleanupManager) GetResults() []CleanupResult {
	return cm.results
}

// GetSummary returns a summary of the cleanup operation
func (cm *CleanupManager) GetSummary() (int, int, time.Duration) {
	successCount := 0
	for _, result := range cm.results {
		if result.Success {
			successCount++
		}
	}
	return successCount, len(cm.results) - successCount, time.Since(cm.startTime)
}
