package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/cleanup"
	"github.com/selfhostly/internal/cloudflare"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
)

const (
	statusStopped  = "stopped"
	statusRunning  = "running"
	statusUpdating = "updating"
	statusError    = "error"
)

// CreateAppRequest represents a create app request
type CreateAppRequest struct {
	Name           string                   `json:"name" binding:"required"`
	Description    string                   `json:"description"`
	ComposeContent string                   `json:"compose_content" binding:"required"`
	IngressRules   []cloudflare.IngressRule `json:"ingress_rules,omitempty"`
}

// UpdateAppRequest represents an update app request
type UpdateAppRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	ComposeContent string `json:"compose_content"`
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// createApp creates a new app
func (s *Server) createApp(c *gin.Context) {
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.WarnContext(c.Request.Context(), "invalid create app request", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request format"})
		return
	}

	// Get Cloudflare settings
	settings, err := s.database.GetSettings()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get settings", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve configuration"})
		return
	}

	// Parse compose to extract networks
	compose, err := docker.ParseCompose([]byte(req.ComposeContent))
	if err != nil {
		slog.WarnContext(c.Request.Context(), "invalid compose file", "app", req.Name, "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid compose file format", Details: err.Error()})
		return
	}

	// var tunnel *db.CloudflareTunnel // TODO: Implement tunnel creation logic
	var tunnelID, tunnelToken, publicURL string

	// Create Cloudflare tunnel if API token is configured
	if settings.CloudflareAPIToken != nil && settings.CloudflareAccountID != nil &&
		*settings.CloudflareAPIToken != "" && *settings.CloudflareAccountID != "" {
		tunnelManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)

		// Create Cloudflare tunnel
		tunnelID, tunnelToken, err = tunnelManager.ApiManager.CreateTunnel(req.Name)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to create cloudflare tunnel", "app", req.Name, "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create Cloudflare tunnel", Details: err.Error()})
			return
		}

		// Log tunnel details
		slog.InfoContext(c.Request.Context(), "cloudflare tunnel created", "app", req.Name, "tunnelID", tunnelID, "tunnelTokenLength", len(tunnelToken))

		// Create public route
		publicURL, err = tunnelManager.ApiManager.CreatePublicRoute(tunnelID, "http://localhost:8080")
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to create public route", "app", req.Name, "tunnelID", tunnelID, "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create public route", Details: err.Error()})
			return
		}

		// Inject cloudflared service
		networks := docker.ExtractNetworks(compose)
		network := ""
		if len(networks) > 0 {
			network = networks[0]
		}

		slog.InfoContext(c.Request.Context(), "injecting cloudflared into compose", "app", req.Name, "tunnelTokenLength", len(tunnelToken))
		if err := docker.InjectCloudflared(compose, req.Name, tunnelToken, network); err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to inject cloudflared service", "app", req.Name, "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to inject cloudflared service", Details: err.Error()})
			return
		}

		// Log cloudflared service after injection
		if cloudflaredService, ok := compose.Services["cloudflared"]; ok {
			slog.InfoContext(c.Request.Context(), "cloudflared service in compose", "app", req.Name, "command", cloudflaredService.Command)
		}

		composeBytes, err := docker.MarshalComposeFile(compose)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to marshal compose file", "app", req.Name, "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to marshal compose file", Details: err.Error()})
			return
		}
		req.ComposeContent = string(composeBytes)

		slog.InfoContext(c.Request.Context(), "compose content after injection", "app", req.Name, "composeLength", len(req.ComposeContent))
	}

	// Debug logging before app creation
	slog.InfoContext(c.Request.Context(), "about to create app with tunnel info", "tunnelID", tunnelID, "tunnelTokenLength", len(tunnelToken), "publicURL", publicURL)

	// Create app in database
	app := db.NewApp(req.Name, req.Description, req.ComposeContent)
	app.TunnelToken = tunnelToken
	app.TunnelID = tunnelID
	app.TunnelDomain = publicURL
	app.PublicURL = publicURL
	app.Status = statusStopped
	app.ErrorMessage = nil // Initialize with nil for optional field
	app.UpdatedAt = time.Now()

	// Debug logging after app creation
	slog.InfoContext(c.Request.Context(), "app creation with tunnel info", "appID", app.ID, "tunnelID", app.TunnelID, "tunnelTokenLength", len(app.TunnelToken))

	if err := s.database.CreateApp(app); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to create app in database", "app", req.Name, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create app", Details: err.Error()})
		return
	}

	// Create initial compose version (version 1)
	user, userExists := getUserFromContext(c)
	var changedBy *string
	if userExists && user.Name != "" {
		changedBy = &user.Name
	}
	initialReason := "Initial version"
	initialVersion := db.NewComposeVersion(app.ID, 1, app.ComposeContent, &initialReason, changedBy)
	if err := s.database.CreateComposeVersion(initialVersion); err != nil {
		slog.WarnContext(c.Request.Context(), "failed to create initial compose version", "appID", app.ID, "error", err)
		// Don't fail the app creation if version tracking fails
	}

	// Create app directory and write compose file
	if err := s.dockerManager.CreateAppDirectory(app.Name, app.ComposeContent); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to create app directory", "app", req.Name, "error", err)
		// Rollback database entry
		if deleteErr := s.database.DeleteApp(app.ID); deleteErr != nil {
			slog.ErrorContext(c.Request.Context(), "failed to rollback app creation", "appID", app.ID, "error", deleteErr)
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create app directory", Details: err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "app created successfully", "app", req.Name, "appID", app.ID)

	// Update tunnel metadata with correct appID if Cloudflare tunnel was created
	slog.InfoContext(c.Request.Context(), "checking tunnel metadata creation", "tunnelID", tunnelID, "tunnelTokenLength", len(tunnelToken))
	if tunnelID != "" && tunnelToken != "" {
		tunnel := db.NewCloudflareTunnel(app.ID, tunnelID, req.Name, tunnelToken, *settings.CloudflareAccountID)

		if err := s.database.CreateCloudflareTunnel(tunnel); err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to create tunnel metadata in database", "appID", app.ID, "error", err)
			// Continue despite the error
		} else {
			slog.InfoContext(c.Request.Context(), "tunnel metadata created successfully", "appID", app.ID, "tunnelID", tunnelID)
		}
	} else {
		slog.WarnContext(c.Request.Context(), "tunnel metadata not created - missing tunnel info", "tunnelID", tunnelID, "tunnelTokenLength", len(tunnelToken))
	}

	// Auto-start app if configured
	if settings.AutoStartApps {
		slog.InfoContext(c.Request.Context(), "auto-starting app", "app", req.Name, "appID", app.ID)
		if err := s.dockerManager.StartApp(app.Name); err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to auto-start app", "app", app.Name, "appID", app.ID, "error", err)

			// Transition to error state but continue with app creation
			app.Status = statusError
			errorMessage := err.Error()
			app.ErrorMessage = &errorMessage
			app.UpdatedAt = time.Now()
			if err := s.database.UpdateApp(app); err != nil {
				slog.ErrorContext(c.Request.Context(), "failed to update app status to error after auto-start failure", "app", app.Name, "appID", app.ID, "error", err)
			}
		} else {
			// Update status in database
			app.Status = statusRunning
			app.ErrorMessage = nil // Clear any previous error message
			app.UpdatedAt = time.Now()
			if err := s.database.UpdateApp(app); err != nil {
				slog.ErrorContext(c.Request.Context(), "failed to update app status after auto-start", "app", app.Name, "appID", app.ID, "error", err)
			}

			// Apply ingress rules if provided
			if len(req.IngressRules) > 0 && tunnelID != "" && settings.CloudflareAPIToken != nil && settings.CloudflareAccountID != nil {
				slog.InfoContext(c.Request.Context(), "applying ingress rules after app start", "app", req.Name, "appID", app.ID, "ruleCount", len(req.IngressRules))

				// Ensure there's a catch-all rule at the end if not provided
				if req.IngressRules[len(req.IngressRules)-1].Service != "http_status:404" {
					req.IngressRules = append(req.IngressRules, cloudflare.IngressRule{
						Service: "http_status:404",
					})
				}

				// Use the shared helper to apply ingress rules (with DNS creation)
				if err := s.applyIngressRulesInternal(app, tunnelID, req.IngressRules, settings, true); err != nil {
					slog.ErrorContext(c.Request.Context(), "failed to apply ingress rules", "app", req.Name, "appID", app.ID, "error", err)
					// Don't fail app creation if ingress update fails
				}
			}
		}
	}

	c.JSON(http.StatusCreated, app)
}

// getApp returns a single app
func (s *Server) getApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(id)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found", "appID", id)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	c.JSON(http.StatusOK, app)
}

// updateApp updates an app
func (s *Server) updateApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	var req UpdateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.WarnContext(c.Request.Context(), "invalid update app request", "appID", id, "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request format"})
		return
	}

	app, err := s.database.GetApp(id)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found for update", "appID", id)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	// Get Cloudflare settings
	settings, err := s.database.GetSettings()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get settings", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve configuration"})
		return
	}

	// Parse compose content if provided
	var compose *docker.ComposeFile
	composeContent := app.ComposeContent // Default to existing
	if req.ComposeContent != "" {
		composeContent = req.ComposeContent
	}

	// If Cloudflare is configured and app has a tunnel token, ensure cloudflared is in the compose
	if settings.CloudflareAPIToken != nil && settings.CloudflareAccountID != nil &&
		*settings.CloudflareAPIToken != "" && *settings.CloudflareAccountID != "" &&
		app.TunnelToken != "" {
		compose, err = docker.ParseCompose([]byte(composeContent))
		if err != nil {
			slog.WarnContext(c.Request.Context(), "invalid compose file", "appID", id, "error", err)
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid compose file format", Details: err.Error()})
			return
		}

		// Extract network and inject cloudflared
		networks := docker.ExtractNetworks(compose)
		network := ""
		if len(networks) > 0 {
			network = networks[0]
		}

		// Log tunnel token before injection
		slog.InfoContext(c.Request.Context(), "injecting cloudflared", "appID", id, "tunnelToken", app.TunnelToken, "tunnelTokenLength", len(app.TunnelToken))

		if err := docker.InjectCloudflared(compose, app.Name, app.TunnelToken, network); err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to inject cloudflared service", "appID", id, "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to inject cloudflared service", Details: err.Error()})
			return
		}

		// Log cloudflared service after injection
		if cloudflaredService, ok := compose.Services["cloudflared"]; ok {
			slog.InfoContext(c.Request.Context(), "cloudflared service injected", "appID", id, "command", cloudflaredService.Command)
		}

		composeBytes, err := docker.MarshalComposeFile(compose)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to marshal compose file", "appID", id, "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to marshal compose file", Details: err.Error()})
			return
		}
		composeContent = string(composeBytes)

		// Log compose content after marshaling
		slog.InfoContext(c.Request.Context(), "compose file after injection", "appID", id, "composeContent", composeContent)
	}

	if req.Name != "" {
		app.Name = req.Name
	}
	if req.Description != "" {
		app.Description = req.Description
	}

	// Check if compose content has changed
	composeChanged := composeContent != app.ComposeContent
	app.ComposeContent = composeContent
	app.UpdatedAt = time.Now()

	if err := s.database.UpdateApp(app); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update app in database", "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app", Details: err.Error()})
		return
	}

	// Create a new compose version if content changed
	if composeChanged {
		latestVersion, err := s.database.GetLatestVersionNumber(id)
		if err != nil {
			slog.WarnContext(c.Request.Context(), "failed to get latest version number", "appID", id, "error", err)
			latestVersion = 0
		}

		// Get authenticated user info
		user, userExists := getUserFromContext(c)
		var changedBy *string
		if userExists && user.Name != "" {
			changedBy = &user.Name
		}

		// Mark all versions as not current
		if err := s.database.MarkAllVersionsAsNotCurrent(id); err != nil {
			slog.WarnContext(c.Request.Context(), "failed to mark versions as not current", "appID", id, "error", err)
		}

		// Create new version
		updateReason := "Compose file updated"
		newVersion := db.NewComposeVersion(id, latestVersion+1, app.ComposeContent, &updateReason, changedBy)
		if err := s.database.CreateComposeVersion(newVersion); err != nil {
			slog.WarnContext(c.Request.Context(), "failed to create compose version", "appID", id, "error", err)
			// Don't fail the update if version tracking fails
		} else {
			slog.InfoContext(c.Request.Context(), "created compose version", "appID", id, "version", latestVersion+1)
		}
	}

	// Update compose file on disk
	dockerManager := docker.NewManager(s.config.AppsDir)
	if err := dockerManager.WriteComposeFile(app.Name, app.ComposeContent); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update compose file", "app", app.Name, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update compose file", Details: err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "app updated successfully", "app", app.Name, "appID", id)
	c.JSON(http.StatusOK, app)
}

// deleteApp deletes an app using the comprehensive cleanup system
func (s *Server) deleteApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(id)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found for deletion", "appID", id)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	// Get Cloudflare settings
	settings, err := s.database.GetSettings()
	if err != nil {
		slog.WarnContext(c.Request.Context(), "failed to get settings for cleanup", "app", app.Name, "error", err)
		// Continue with basic cleanup even if settings fail
	}

	var tunnelManager *cloudflare.TunnelManager
	if settings != nil && settings.CloudflareAPIToken != nil && settings.CloudflareAccountID != nil &&
		*settings.CloudflareAPIToken != "" && *settings.CloudflareAccountID != "" {
		tunnelManager = cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)
	} else {
		slog.WarnContext(c.Request.Context(), "Cloudflare not configured, will skip Cloudflare cleanup", "app", app.Name)
	}

	// Create cleanup manager
	dockerManager := docker.NewManager(s.config.AppsDir)
	cleanupManager := cleanup.NewCleanupManager(dockerManager, s.database, settings, tunnelManager)

	// Perform comprehensive cleanup
	results, err := cleanupManager.CleanupApp(app)

	// Calculate summary
	successCount, failedCount, totalDuration := cleanupManager.GetSummary()

	// Log comprehensive results
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "App cleanup completed with errors",
			"app", app.Name,
			"appID", app.ID,
			"successCount", successCount,
			"failedCount", failedCount,
			"totalDuration", totalDuration,
			"error", err)
	} else {
		slog.InfoContext(c.Request.Context(), "App cleanup completed successfully",
			"app", app.Name,
			"appID", app.ID,
			"successCount", successCount,
			"failedCount", failedCount,
			"totalDuration", totalDuration)
	}

	// Return appropriate response based on cleanup results
	if failedCount > 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":       "App deleted successfully, but some cleanup errors occurred",
			"appID":         app.ID,
			"successCount":  successCount,
			"failedCount":   failedCount,
			"totalDuration": totalDuration.String(),
			"errors":        failedCount,
			"details":       results,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message":       "App deleted successfully",
			"appID":         app.ID,
			"successCount":  successCount,
			"failedCount":   failedCount,
			"totalDuration": totalDuration.String(),
			"details":       results,
		})
	}
}

// startApp starts an app
func (s *Server) startApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(id)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found for start", "appID", id)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	dockerManager := docker.NewManager(s.config.AppsDir)
	if err := dockerManager.StartApp(app.Name); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to start app", "app", app.Name, "appID", id, "error", err)

		// Transition to error state
		app.Status = statusError
		errorMessage := err.Error()
		app.ErrorMessage = &errorMessage
		app.UpdatedAt = time.Now()
		if err := s.database.UpdateApp(app); err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to update app status to error", "app", app.Name, "appID", id, "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app status"})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to start app", Details: err.Error()})
		return
	}

	// Update status in database
	app.Status = statusRunning
	app.ErrorMessage = nil // Clear any previous error message
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update app status", "app", app.Name, "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app status"})
		return
	}

	slog.InfoContext(c.Request.Context(), "app started successfully", "app", app.Name, "appID", id)
	c.JSON(http.StatusOK, app)
}

// stopApp stops an app
func (s *Server) stopApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(id)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found for stop", "appID", id)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	dockerManager := docker.NewManager(s.config.AppsDir)
	if err := dockerManager.StopApp(app.Name); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to stop app", "app", app.Name, "appID", id, "error", err)

		// Transition to error state
		app.Status = statusError
		errorMessage := err.Error()
		app.ErrorMessage = &errorMessage
		app.UpdatedAt = time.Now()
		if err := s.database.UpdateApp(app); err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to update app status to error", "app", app.Name, "appID", id, "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app status"})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to stop app", Details: err.Error()})
		return
	}

	// Update status in database
	app.Status = statusStopped
	app.ErrorMessage = nil // Clear any previous error message
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update app status", "app", app.Name, "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app status"})
		return
	}

	slog.InfoContext(c.Request.Context(), "app stopped successfully", "app", app.Name, "appID", id)
	c.JSON(http.StatusOK, app)
}

// updateAppContainers updates app containers with zero downtime
func (s *Server) updateAppContainers(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(id)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found for update", "appID", id)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	slog.InfoContext(c.Request.Context(), "initiating app container update", 
		"app", app.Name, 
		"appID", id, 
		"currentStatus", app.Status,
		"appsDir", s.config.AppsDir)

	// Update status to updating
	app.Status = statusUpdating
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update app status to updating", "app", app.Name, "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app status"})
		return
	}

	slog.InfoContext(c.Request.Context(), "app status set to updating, starting docker compose update", 
		"app", app.Name, 
		"appID", id)

	dockerManager := docker.NewManager(s.config.AppsDir)
	if err := dockerManager.UpdateApp(app.Name); err != nil {
		// Update status to error
		slog.ErrorContext(c.Request.Context(), "docker compose update failed", 
			"app", app.Name, 
			"appID", id, 
			"error", err,
			"errorDetails", err.Error())
		
		app.Status = statusError
		errorMessage := err.Error()
		app.ErrorMessage = &errorMessage
		app.UpdatedAt = time.Now()
		if dbErr := s.database.UpdateApp(app); dbErr != nil {
			slog.ErrorContext(c.Request.Context(), "failed to update app status to error after update failure", 
				"app", app.Name, 
				"appID", id, 
				"updateError", err,
				"dbError", dbErr)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app status"})
			return
		}
		
		slog.ErrorContext(c.Request.Context(), "app status updated to error", 
			"app", app.Name, 
			"appID", id, 
			"errorMessage", errorMessage)
		
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app containers", Details: err.Error()})
		return
	}

	// Update status to running
	slog.InfoContext(c.Request.Context(), "docker compose update completed successfully, updating app status to running", 
		"app", app.Name, 
		"appID", id)
	
	app.Status = statusRunning
	app.ErrorMessage = nil // Clear any previous error message
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update app status to running after successful update", 
			"app", app.Name, 
			"appID", id, 
			"error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app status"})
		return
	}

	slog.InfoContext(c.Request.Context(), "app containers updated successfully", 
		"app", app.Name, 
		"appID", id,
		"finalStatus", app.Status)
	c.JSON(http.StatusOK, app)
}

// getAppLogs returns app logs
func (s *Server) getAppLogs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(id)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found for logs", "appID", id)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	dockerManager := docker.NewManager(s.config.AppsDir)
	logs, err := dockerManager.GetAppLogs(app.Name)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get app logs", "app", app.Name, "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve logs", Details: err.Error()})
		return
	}

	c.Header("Content-Type", "text/plain")
	c.Data(http.StatusOK, "text/plain", logs)
}

// getAppStats returns real-time resource statistics for an app
func (s *Server) getAppStats(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(id)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found for stats", "appID", id)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	// Only fetch stats if app is running
	if app.Status != statusRunning {
		// Return empty stats for non-running apps
		c.JSON(http.StatusOK, gin.H{
			"app_name":              app.Name,
			"total_cpu_percent":     0,
			"total_memory_bytes":    0,
			"memory_limit_bytes":    0,
			"containers":            []interface{}{},
			"timestamp":             time.Now(),
			"status":                app.Status,
			"message":               fmt.Sprintf("App is %s", app.Status),
		})
		return
	}

	dockerManager := docker.NewManager(s.config.AppsDir)
	stats, err := dockerManager.GetAppStats(app.Name)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get app stats", "app", app.Name, "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve stats", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// repairApp repairs an app's compose file if needed (e.g., adds missing cloudflared token)
func (s *Server) repairApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	app, err := s.database.GetApp(id)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found for repair", "appID", id)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	// Get Cloudflare settings
	settings, err := s.database.GetSettings()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get settings", "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve configuration"})
		return
	}

	// Only repair if Cloudflare is configured and app has tunnel ID but missing token
	if settings.CloudflareAPIToken == nil || settings.CloudflareAccountID == nil ||
		*settings.CloudflareAPIToken == "" || *settings.CloudflareAccountID == "" ||
		app.TunnelID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Cannot repair: Cloudflare not configured or no tunnel ID"})
		return
	}

	// If tunnel token is already in database, no repair needed
	if app.TunnelToken != "" {
		c.JSON(http.StatusOK, gin.H{
			"message": "App already repaired - tunnel token exists",
			"app":     app,
		})
		return
	}

	// Fetch tunnel token from Cloudflare
	cfManager := cloudflare.NewManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID)
	tunnelToken, err := cfManager.GetTunnelToken(app.TunnelID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get tunnel token from Cloudflare", "appID", id, "tunnelID", app.TunnelID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get tunnel token from Cloudflare", Details: err.Error()})
		return
	}

	// Update database with tunnel token
	app.TunnelToken = tunnelToken
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update app with tunnel token", "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app", Details: err.Error()})
		return
	}

	// Parse existing compose content
	compose, err := docker.ParseCompose([]byte(app.ComposeContent))
	if err != nil {
		slog.WarnContext(c.Request.Context(), "invalid compose file", "appID", id, "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid compose file format", Details: err.Error()})
		return
	}

	// Extract network and inject cloudflared with proper token
	networks := docker.ExtractNetworks(compose)
	network := ""
	if len(networks) > 0 {
		network = networks[0]
	}

	if err := docker.InjectCloudflared(compose, app.Name, tunnelToken, network); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to inject cloudflared service", "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to inject cloudflared service", Details: err.Error()})
		return
	}

	composeBytes, err := docker.MarshalComposeFile(compose)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to marshal compose file", "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to marshal compose file", Details: err.Error()})
		return
	}

	app.ComposeContent = string(composeBytes)
	if err := s.database.UpdateApp(app); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update compose content", "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update compose content", Details: err.Error()})
		return
	}

	// Update compose file on disk
	dockerManager := docker.NewManager(s.config.AppsDir)
	if err := dockerManager.WriteComposeFile(app.Name, app.ComposeContent); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update compose file", "app", app.Name, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update compose file", Details: err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "app repaired successfully", "app", app.Name, "appID", id)
	c.JSON(http.StatusOK, gin.H{
		"message": "App repaired successfully",
		"app":     app,
	})
}

// listApps returns all apps
func (s *Server) listApps(c *gin.Context) {
	apps, err := s.database.GetAllApps()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to retrieve apps", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve apps"})
		return
	}

	c.JSON(http.StatusOK, apps)
}

// RollbackRequest represents a rollback request with optional metadata
type RollbackRequest struct {
	ChangeReason *string `json:"change_reason"`
}

// getComposeVersions returns all compose versions for an app
func (s *Server) getComposeVersions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}

	// Verify app exists
	_, err := s.database.GetApp(id)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found", "appID", id)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	versions, err := s.database.GetComposeVersionsByAppID(id)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to retrieve compose versions", "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve compose versions"})
		return
	}

	// Return empty array instead of null if no versions
	if versions == nil {
		versions = []*db.ComposeVersion{}
	}

	c.JSON(http.StatusOK, versions)
}

// getComposeVersion returns a specific compose version
func (s *Server) getComposeVersion(c *gin.Context) {
	id := c.Param("id")
	versionParam := c.Param("version")
	if id == "" || versionParam == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID or version"})
		return
	}

	// Parse version number
	var version int
	if _, err := fmt.Sscanf(versionParam, "%d", &version); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid version number"})
		return
	}

	// Verify app exists
	_, err := s.database.GetApp(id)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found", "appID", id)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	composeVersion, err := s.database.GetComposeVersion(id, version)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "compose version not found", "appID", id, "version", version)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Compose version not found"})
		return
	}

	c.JSON(http.StatusOK, composeVersion)
}

// rollbackToVersion rolls back to a specific compose version
func (s *Server) rollbackToVersion(c *gin.Context) {
	id := c.Param("id")
	versionParam := c.Param("version")
	if id == "" || versionParam == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID or version"})
		return
	}

	// Parse version number
	var targetVersion int
	if _, err := fmt.Sscanf(versionParam, "%d", &targetVersion); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid version number"})
		return
	}

	// Get optional rollback request body
	var req RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Body is optional, so just use empty request if binding fails
		req = RollbackRequest{}
	}

	// Get the app
	app, err := s.database.GetApp(id)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "app not found for rollback", "appID", id)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "App not found"})
		return
	}

	// Get the target version
	targetComposeVersion, err := s.database.GetComposeVersion(id, targetVersion)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "target compose version not found", "appID", id, "version", targetVersion)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Target version not found"})
		return
	}

	// Get current version number
	currentVersionNumber, err := s.database.GetLatestVersionNumber(id)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to get current version number", "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get current version"})
		return
	}

	// Get authenticated user info
	user, userExists := getUserFromContext(c)
	var changedBy *string
	if userExists && user.Name != "" {
		changedBy = &user.Name
	}

	// Create a new version with the rolled-back content
	newVersionNumber := currentVersionNumber + 1
	rolledBackFrom := currentVersionNumber
	changeReason := req.ChangeReason
	if changeReason == nil {
		reason := fmt.Sprintf("Rolled back to version %d", targetVersion)
		changeReason = &reason
	}

	newVersion := db.NewComposeVersion(id, newVersionNumber, targetComposeVersion.ComposeContent, changeReason, changedBy)
	newVersion.RolledBackFrom = &rolledBackFrom

	// Mark all versions as not current
	if err := s.database.MarkAllVersionsAsNotCurrent(id); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to mark versions as not current", "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to prepare rollback"})
		return
	}

	// Create the new version
	if err := s.database.CreateComposeVersion(newVersion); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to create rolled-back version", "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create rolled-back version"})
		return
	}

	// Update the app with the rolled-back content
	app.ComposeContent = targetComposeVersion.ComposeContent
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update app with rolled-back content", "appID", id, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update app"})
		return
	}

	// Update compose file on disk
	dockerManager := docker.NewManager(s.config.AppsDir)
	if err := dockerManager.WriteComposeFile(app.Name, app.ComposeContent); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update compose file on disk", "app", app.Name, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update compose file", Details: err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "rolled back compose version", "app", app.Name, "appID", id, "fromVersion", targetVersion, "toVersion", newVersionNumber)
	c.JSON(http.StatusOK, gin.H{
		"message":      "Rolled back successfully",
		"app":          app,
		"new_version":  newVersion,
		"from_version": targetVersion,
	})
}
