package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/selfhostly/internal/cleanup"
	"github.com/selfhostly/internal/cloudflare"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/node"
	"github.com/selfhostly/internal/routing"
	"github.com/selfhostly/internal/validation"
)

// appService implements the AppService interface
type appService struct {
	database        *db.DB
	dockerManager   *docker.Manager
	nodeClient      *node.Client
	config          *config.Config
	logger          *slog.Logger
	router          *routing.NodeRouter
	appsAgg         *routing.AppsAggregator
	settingsManager *cloudflare.SettingsManager
}

// NewAppService creates a new app service
func NewAppService(
	database *db.DB,
	dockerManager *docker.Manager,
	cfg *config.Config,
	logger *slog.Logger,
) domain.AppService {
	nodeClient := node.NewClient()
	router := routing.NewNodeRouter(database, nodeClient, cfg.Node.ID, logger)
	appsAgg := routing.NewAppsAggregator(router, logger)
	settingsManager := cloudflare.NewSettingsManager(database, logger)
	
	return &appService{
		database:        database,
		dockerManager:   dockerManager,
nodeClient:      nodeClient,
		config:          cfg,
		logger:          logger,
		router:          router,
		appsAgg:         appsAgg,
		settingsManager: settingsManager,
	}
}

// CreateApp creates a new application
func (s *appService) CreateApp(ctx context.Context, req domain.CreateAppRequest) (*db.App, error) {
	s.logger.InfoContext(ctx, "creating app", "name", req.Name, "targetNode", req.NodeID)

	// If node_id is specified and it's not this node, forward to that node
	if req.NodeID != "" && req.NodeID != s.config.Node.ID {
		s.logger.InfoContext(ctx, "forwarding app creation to remote node", "name", req.Name, "nodeID", req.NodeID)

		targetNode, err := s.database.GetNode(req.NodeID)
		if err != nil {
			return nil, fmt.Errorf("target node not found: %w", err)
		}

		// Forward request to target node
		app, err := s.nodeClient.CreateApp(targetNode, req)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create app on remote node", "name", req.Name, "nodeID", req.NodeID, "error", err)
			return nil, err
		}

		s.logger.InfoContext(ctx, "app created on remote node", "name", req.Name, "nodeID", req.NodeID, "appID", app.ID)
		return app, nil
	}

	// Validate app name
	if err := validation.ValidateAppName(req.Name); err != nil {
		s.logger.WarnContext(ctx, "invalid app name", "name", req.Name, "error", err)
		return nil, domain.WrapValidationError("app name", err)
	}

	// Validate compose content
	if err := validation.ValidateComposeContent(req.ComposeContent); err != nil {
		s.logger.WarnContext(ctx, "invalid compose content", "error", err)
		return nil, domain.WrapValidationError("compose content", err)
	}

	// Validate description if provided
	if req.Description != "" {
		if err := validation.ValidateDescription(req.Description); err != nil {
			s.logger.WarnContext(ctx, "invalid description", "error", err)
			return nil, domain.WrapValidationError("description", err)
		}
	}

	// Get Cloudflare settings
	settings, err := s.settingsManager.GetSettings()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get settings", "error", err)
		return nil, err
	}

	// Parse compose to extract networks
	compose, err := docker.ParseCompose([]byte(req.ComposeContent))
	if err != nil {
		s.logger.WarnContext(ctx, "invalid compose file", "app", req.Name, "error", err)
		return nil, domain.WrapComposeInvalid(err)
	}

	var tunnelID, tunnelToken, publicURL string

	// Create Cloudflare tunnel if API token is configured
	if settings.CloudflareAPIToken != nil && settings.CloudflareAccountID != nil &&
		*settings.CloudflareAPIToken != "" && *settings.CloudflareAccountID != "" {
		tunnelManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)

		// Create Cloudflare tunnel
		tunnelID, tunnelToken, err = tunnelManager.ApiManager.CreateTunnel(req.Name)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create cloudflare tunnel", "app", req.Name, "error", err)
			return nil, domain.WrapTunnelCreationFailed(req.Name, err)
		}

		s.logger.InfoContext(ctx, "cloudflare tunnel created", "app", req.Name, "tunnelID", tunnelID)

		// Create public route
		publicURL, err = tunnelManager.ApiManager.CreatePublicRoute(tunnelID, "http://localhost:8080")
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create public route", "app", req.Name, "tunnelID", tunnelID, "error", err)
			return nil, fmt.Errorf("failed to create public route: %w", err)
		}

		// Inject cloudflared service
		networks := docker.ExtractNetworks(compose)
		network := ""
		if len(networks) > 0 {
			network = networks[0]
		}

		s.logger.InfoContext(ctx, "injecting cloudflared into compose", "app", req.Name)
		if err := docker.InjectCloudflared(compose, req.Name, tunnelToken, network); err != nil {
			s.logger.ErrorContext(ctx, "failed to inject cloudflared service", "app", req.Name, "error", err)
			return nil, fmt.Errorf("failed to inject cloudflared service: %w", err)
		}

		composeBytes, err := docker.MarshalComposeFile(compose)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to marshal compose file", "app", req.Name, "error", err)
			return nil, fmt.Errorf("failed to marshal compose file: %w", err)
		}
		req.ComposeContent = string(composeBytes)
	}

	// Create app in database
	app := db.NewApp(req.Name, req.Description, req.ComposeContent)
	app.TunnelToken = tunnelToken
	app.TunnelID = tunnelID
	app.TunnelDomain = publicURL
	app.PublicURL = publicURL
	app.Status = "stopped"
	app.ErrorMessage = nil
	app.NodeID = s.config.Node.ID // Assign to current node
	app.UpdatedAt = time.Now()

	if err := s.database.CreateApp(app); err != nil {
		s.logger.ErrorContext(ctx, "failed to create app in database", "app", req.Name, "error", err)
		return nil, domain.WrapDatabaseOperation("create app", err)
	}

	// Create initial compose version (version 1)
	// Note: changedBy will be set by the caller if user context is available
	initialReason := "Initial version"
	initialVersion := db.NewComposeVersion(app.ID, 1, app.ComposeContent, &initialReason, nil)
	if err := s.database.CreateComposeVersion(initialVersion); err != nil {
		s.logger.WarnContext(ctx, "failed to create initial compose version", "appID", app.ID, "error", err)
		// Don't fail the app creation if version tracking fails
	}

	// Create app directory and write compose file
	if err := s.dockerManager.CreateAppDirectory(app.Name, app.ComposeContent); err != nil {
		s.logger.ErrorContext(ctx, "failed to create app directory", "app", req.Name, "error", err)
		// Rollback database entry
		if deleteErr := s.database.DeleteApp(app.ID); deleteErr != nil {
			s.logger.ErrorContext(ctx, "failed to rollback app creation", "appID", app.ID, "error", deleteErr)
		}
		return nil, domain.WrapContainerOperationFailed("create app directory", err)
	}

	// Update tunnel metadata with correct appID if Cloudflare tunnel was created
	if tunnelID != "" && tunnelToken != "" {
		tunnel := db.NewCloudflareTunnel(app.ID, tunnelID, req.Name, tunnelToken, *settings.CloudflareAccountID)

		if err := s.database.CreateCloudflareTunnel(tunnel); err != nil {
			s.logger.ErrorContext(ctx, "failed to create tunnel metadata in database", "appID", app.ID, "error", err)
			// Continue despite the error
		} else {
			s.logger.InfoContext(ctx, "tunnel metadata created successfully", "appID", app.ID, "tunnelID", tunnelID)
		}
	}

	// Auto-start app if configured
	if settings.AutoStartApps {
		s.logger.InfoContext(ctx, "auto-starting app", "app", req.Name, "appID", app.ID)
		if err := s.dockerManager.StartApp(app.Name); err != nil {
			s.logger.ErrorContext(ctx, "failed to auto-start app", "app", app.Name, "appID", app.ID, "error", err)

			// Transition to error state but continue with app creation
			app.Status = "error"
			errorMessage := err.Error()
			app.ErrorMessage = &errorMessage
			app.UpdatedAt = time.Now()
			if err := s.database.UpdateApp(app); err != nil {
				s.logger.ErrorContext(ctx, "failed to update app status to error", "app", app.Name, "error", err)
			}
		} else {
			// Update status in database
			app.Status = "running"
			app.ErrorMessage = nil
			app.UpdatedAt = time.Now()
			if err := s.database.UpdateApp(app); err != nil {
				s.logger.ErrorContext(ctx, "failed to update app status after auto-start", "app", app.Name, "error", err)
			}

			// Apply ingress rules if provided
			if len(req.IngressRules) > 0 && tunnelID != "" && settings.CloudflareAPIToken != nil && settings.CloudflareAccountID != nil {
				s.logger.InfoContext(ctx, "applying ingress rules after app start", "app", req.Name, "ruleCount", len(req.IngressRules))

				// Convert db.IngressRule to cloudflare.IngressRule and ensure catch-all rule
				cfRules := cloudflare.ConvertToCloudflareRules(req.IngressRules)
				cfRules = cloudflare.EnsureCatchAllRule(cfRules)

				tunnelManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)
				if err := tunnelManager.UpdateTunnelIngress(tunnelID, cfRules, "", ""); err != nil {
					s.logger.ErrorContext(ctx, "failed to apply ingress rules", "app", req.Name, "error", err)
					// Don't fail app creation if ingress update fails
				} else {
					// Restart cloudflared container to pick up new ingress configuration
					// This is a best-effort operation - we don't fail the app creation if it fails
					if err := s.dockerManager.RestartCloudflared(app.Name); err != nil {
						s.logger.WarnContext(ctx, "failed to restart cloudflared container, ingress rules updated but container restart may be required", "app", req.Name, "appID", app.ID, "error", err)
					} else {
						s.logger.InfoContext(ctx, "cloudflared container restarted successfully after ingress update", "app", req.Name, "appID", app.ID)
					}
				}
			}
		}
	}

	s.logger.InfoContext(ctx, "app created successfully", "app", req.Name, "appID", app.ID)
	return app, nil
}

// GetApp retrieves an app by ID, optionally with a node_id hint for optimization
func (s *appService) GetApp(ctx context.Context, appID string, nodeID string) (*db.App, error) {
	s.logger.DebugContext(ctx, "getting app", "appID", appID, "nodeID", nodeID)

	result, err := s.router.RouteToNode(
		ctx,
		nodeID,
		func() (interface{}, error) {
			app, err := s.database.GetApp(appID)
			if err != nil {
				return nil, domain.WrapAppNotFound(appID, err)
			}
			return app, nil
		},
		func(n *db.Node) (interface{}, error) {
			return s.nodeClient.GetApp(n, appID)
		},
	)

	if err != nil {
		return nil, err
	}

	return result.(*db.App), nil
}

// ListApps retrieves all apps
func (s *appService) ListApps(ctx context.Context, nodeIDs []string) ([]*db.App, error) {
	s.logger.DebugContext(ctx, "listing apps", "nodeIDs", nodeIDs)

	// Determine which nodes to fetch from
	targetNodes, err := s.router.DetermineTargetNodes(ctx, nodeIDs)
	if err != nil {
		return nil, err
	}

	// Aggregate apps from all target nodes in parallel
	allApps, err := s.appsAgg.AggregateApps(
		ctx,
		targetNodes,
		func() ([]*db.App, error) {
			return s.database.GetAllApps()
		},
		func(n *db.Node) ([]*db.App, error) {
			return s.nodeClient.GetApps(n)
		},
	)

	return allApps, err
}

// UpdateApp updates an existing app
func (s *appService) UpdateApp(ctx context.Context, appID string, nodeID string, req domain.UpdateAppRequest) (*db.App, error) {
	s.logger.InfoContext(ctx, "updating app", "appID", appID, "nodeID", nodeID)

	// Validate name if provided
	if req.Name != "" {
		if err := validation.ValidateAppName(req.Name); err != nil {
			s.logger.WarnContext(ctx, "invalid app name", "name", req.Name, "error", err)
			return nil, domain.WrapValidationError("app name", err)
		}
	}

	// Validate description if provided
	if req.Description != "" {
		if err := validation.ValidateDescription(req.Description); err != nil {
			s.logger.WarnContext(ctx, "invalid description", "error", err)
			return nil, domain.WrapValidationError("description", err)
		}
	}

	// Validate compose content if provided
	if req.ComposeContent != "" {
		if err := validation.ValidateComposeContent(req.ComposeContent); err != nil {
			s.logger.WarnContext(ctx, "invalid compose content", "error", err)
			return nil, domain.WrapValidationError("compose content", err)
		}
	}

	// If app is on a remote node, forward the request
	if !s.router.IsLocalNode(nodeID) {
		result, err := s.router.RouteToNode(
			ctx,
			nodeID,
			nil, // Not used since we know it's remote
			func(n *db.Node) (interface{}, error) {
				return s.nodeClient.UpdateApp(n, appID, req)
			},
		)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to update app on remote node", "appID", appID, "nodeID", nodeID)
			return nil, err
		}
		s.logger.InfoContext(ctx, "app updated on remote node", "appID", appID, "nodeID", nodeID)
		return result.(*db.App), nil
	}

	// Get the app from local node
	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}

	// Get Cloudflare settings
	settings, err := s.settingsManager.GetSettings()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get settings", "error", err)
		return nil, err
	}

	// Parse compose content if provided
	composeContent := app.ComposeContent // Default to existing
	if req.ComposeContent != "" {
		composeContent = req.ComposeContent
	}

	// If Cloudflare is configured and app has a tunnel token, ensure cloudflared is in the compose
	if settings.CloudflareAPIToken != nil && settings.CloudflareAccountID != nil &&
		*settings.CloudflareAPIToken != "" && *settings.CloudflareAccountID != "" &&
		app.TunnelToken != "" {
		compose, err := docker.ParseCompose([]byte(composeContent))
		if err != nil {
			s.logger.WarnContext(ctx, "invalid compose file", "appID", appID, "error", err)
			return nil, domain.WrapComposeInvalid(err)
		}

		// Extract network and inject cloudflared
		networks := docker.ExtractNetworks(compose)
		network := ""
		if len(networks) > 0 {
			network = networks[0]
		}

		if err := docker.InjectCloudflared(compose, app.Name, app.TunnelToken, network); err != nil {
			s.logger.ErrorContext(ctx, "failed to inject cloudflared service", "appID", appID, "error", err)
			return nil, fmt.Errorf("failed to inject cloudflared service: %w", err)
		}

		composeBytes, err := docker.MarshalComposeFile(compose)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to marshal compose file", "appID", appID, "error", err)
			return nil, fmt.Errorf("failed to marshal compose file: %w", err)
		}
		composeContent = string(composeBytes)
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
		s.logger.ErrorContext(ctx, "failed to update app in database", "appID", appID, "error", err)
		return nil, domain.WrapDatabaseOperation("update app", err)
	}

	// Create a new compose version if content changed
	if composeChanged {
		latestVersion, err := s.database.GetLatestVersionNumber(appID)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to get latest version number", "appID", appID, "error", err)
			latestVersion = 0
		}

		// Mark all versions as not current
		if err := s.database.MarkAllVersionsAsNotCurrent(appID); err != nil {
			s.logger.WarnContext(ctx, "failed to mark versions as not current", "appID", appID, "error", err)
		}

		// Create new version
		updateReason := "Compose file updated"
		newVersion := db.NewComposeVersion(appID, latestVersion+1, app.ComposeContent, &updateReason, nil)
		if err := s.database.CreateComposeVersion(newVersion); err != nil {
			s.logger.WarnContext(ctx, "failed to create compose version", "appID", appID, "error", err)
			// Don't fail the update if version tracking fails
		}
	}

	// Update compose file on disk
	if err := s.dockerManager.WriteComposeFile(app.Name, app.ComposeContent); err != nil {
		s.logger.ErrorContext(ctx, "failed to update compose file", "app", app.Name, "error", err)
		return nil, domain.WrapContainerOperationFailed("write compose file", err)
	}

	s.logger.InfoContext(ctx, "app updated successfully", "app", app.Name, "appID", appID)
	return app, nil
}

// DeleteApp deletes an app using comprehensive cleanup
func (s *appService) DeleteApp(ctx context.Context, appID string, nodeID string) error {
	s.logger.InfoContext(ctx, "deleting app", "appID", appID, "nodeID", nodeID)

	// If app is on a remote node, forward the request
	if !s.router.IsLocalNode(nodeID) {
		_, err := s.router.RouteToNode(
			ctx,
			nodeID,
			nil, // Not used since we know it's remote
			func(n *db.Node) (interface{}, error) {
				return nil, s.nodeClient.DeleteApp(n, appID)
			},
		)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to delete app on remote node", "appID", appID, "nodeID", nodeID)
		}
		return err
	}

	// Get the app from local node
	app, err := s.database.GetApp(appID)
	if err != nil {
		return domain.WrapAppNotFound(appID, err)
	}

	// Get Cloudflare settings and tunnel manager if configured
	tunnelManager, settings, err := s.settingsManager.GetConfiguredTunnelManager()
	if err != nil {
		s.logger.WarnContext(ctx, "failed to get settings for cleanup", "app", app.Name, "error", err)
		// Continue with basic cleanup even if settings fail
		settings = nil
	}

	// Create cleanup manager
	cleanupManager := cleanup.NewCleanupManager(s.dockerManager, s.database, settings, tunnelManager)

	// Perform comprehensive cleanup
	results, err := cleanupManager.CleanupApp(app)

	// Calculate summary
	successCount, failedCount, totalDuration := cleanupManager.GetSummary()

	// Log comprehensive results
	if err != nil {
		s.logger.ErrorContext(ctx, "app cleanup completed with errors",
			"app", app.Name,
			"appID", app.ID,
			"successCount", successCount,
			"failedCount", failedCount,
			"totalDuration", totalDuration,
			"error", err)
	} else {
		s.logger.InfoContext(ctx, "app cleanup completed successfully",
			"app", app.Name,
			"appID", app.ID,
			"successCount", successCount,
			"failedCount", failedCount,
			"totalDuration", totalDuration)
	}

	// Log individual failures
	for _, result := range results {
		if !result.Success {
			s.logger.ErrorContext(ctx, "cleanup step failed",
				"app", app.Name,
				"step", result.Step,
				"error", result.Error,
				"duration", result.Duration)
		}
	}

	return err
}

// StartApp starts an application
func (s *appService) StartApp(ctx context.Context, appID string, nodeID string) (*db.App, error) {
	s.logger.InfoContext(ctx, "starting app", "appID", appID, "nodeID", nodeID)

	// If app is on a remote node, forward the request
	if !s.router.IsLocalNode(nodeID) {
		result, err := s.router.RouteToNode(
			ctx,
			nodeID,
			nil, // Not used since we know it's remote
			func(n *db.Node) (interface{}, error) {
				if err := s.nodeClient.StartApp(n, appID); err != nil {
					return nil, err
				}
				return s.nodeClient.GetApp(n, appID)
			},
		)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to start app on remote node", "appID", appID, "nodeID", nodeID)
			return nil, err
		}
		return result.(*db.App), nil
	}

	// Get the app from local node
	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}

	if err := s.dockerManager.StartApp(app.Name); err != nil {
		s.logger.ErrorContext(ctx, "failed to start app", "app", app.Name, "appID", appID, "error", err)

		// Transition to error state
		app.Status = "error"
		errorMessage := err.Error()
		app.ErrorMessage = &errorMessage
		app.UpdatedAt = time.Now()
		if err := s.database.UpdateApp(app); err != nil {
			s.logger.ErrorContext(ctx, "failed to update app status to error", "app", app.Name, "error", err)
		}

		return nil, domain.WrapContainerOperationFailed("start app", err)
	}

	// Update status in database
	app.Status = "running"
	app.ErrorMessage = nil
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		s.logger.ErrorContext(ctx, "failed to update app status", "app", app.Name, "error", err)
		return nil, domain.WrapDatabaseOperation("update app status", err)
	}

	s.logger.InfoContext(ctx, "app started successfully", "app", app.Name, "appID", appID)
	return app, nil
}

// StopApp stops an application
func (s *appService) StopApp(ctx context.Context, appID string, nodeID string) (*db.App, error) {
	s.logger.InfoContext(ctx, "stopping app", "appID", appID, "nodeID", nodeID)

	// If app is on a remote node, forward the request
	if !s.router.IsLocalNode(nodeID) {
		result, err := s.router.RouteToNode(
			ctx,
			nodeID,
			nil, // Not used since we know it's remote
			func(n *db.Node) (interface{}, error) {
				if err := s.nodeClient.StopApp(n, appID); err != nil {
					return nil, err
				}
				return s.nodeClient.GetApp(n, appID)
			},
		)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to stop app on remote node", "appID", appID, "nodeID", nodeID)
			return nil, err
		}
		return result.(*db.App), nil
	}

	// Get the app from local node
	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}

	if err := s.dockerManager.StopApp(app.Name); err != nil {
		s.logger.ErrorContext(ctx, "failed to stop app", "app", app.Name, "appID", appID, "error", err)

		// Transition to error state
		app.Status = "error"
		errorMessage := err.Error()
		app.ErrorMessage = &errorMessage
		app.UpdatedAt = time.Now()
		if err := s.database.UpdateApp(app); err != nil {
			s.logger.ErrorContext(ctx, "failed to update app status to error", "app", app.Name, "error", err)
		}

		return nil, domain.WrapContainerOperationFailed("stop app", err)
	}

	// Update status in database
	app.Status = "stopped"
	app.ErrorMessage = nil
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		s.logger.ErrorContext(ctx, "failed to update app status", "app", app.Name, "error", err)
		return nil, domain.WrapDatabaseOperation("update app status", err)
	}

	s.logger.InfoContext(ctx, "app stopped successfully", "app", app.Name, "appID", appID)
	return app, nil
}

// UpdateAppContainers updates app containers with zero downtime
func (s *appService) UpdateAppContainers(ctx context.Context, appID string, nodeID string) (*db.App, error) {
	s.logger.InfoContext(ctx, "updating app containers", "appID", appID, "nodeID", nodeID)

	// If app is on a remote node, forward the request
	if !s.router.IsLocalNode(nodeID) {
		result, err := s.router.RouteToNode(
			ctx,
			nodeID,
			nil, // Not used since we know it's remote
			func(n *db.Node) (interface{}, error) {
				return s.nodeClient.UpdateAppContainers(n, appID)
			},
		)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to update app containers on remote node", "appID", appID, "nodeID", nodeID)
			return nil, err
		}
		s.logger.InfoContext(ctx, "app containers updated on remote node", "appID", appID, "nodeID", nodeID)
		return result.(*db.App), nil
	}

	// Get the app from local node
	app, err := s.database.GetApp(appID)
	if err != nil {
		s.logger.DebugContext(ctx, "app not found for update", "appID", appID)
		return nil, domain.WrapAppNotFound(appID, err)
	}

	// Update status to updating
	app.Status = "updating"
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		s.logger.ErrorContext(ctx, "failed to update app status to updating", "app", app.Name, "error", err)
		return nil, domain.WrapDatabaseOperation("update app status", err)
	}

	if err := s.dockerManager.UpdateApp(app.Name); err != nil {
		s.logger.ErrorContext(ctx, "docker compose update failed", "app", app.Name, "error", err)

		// Update status to error
		app.Status = "error"
		errorMessage := err.Error()
		app.ErrorMessage = &errorMessage
		app.UpdatedAt = time.Now()
		if dbErr := s.database.UpdateApp(app); dbErr != nil {
			s.logger.ErrorContext(ctx, "failed to update app status to error", "app", app.Name, "error", dbErr)
		}

		return nil, domain.WrapContainerOperationFailed("update app", err)
	}

	// Update status to running
	app.Status = "running"
	app.ErrorMessage = nil
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		s.logger.ErrorContext(ctx, "failed to update app status to running", "app", app.Name, "error", err)
		return nil, domain.WrapDatabaseOperation("update app status", err)
	}

	s.logger.InfoContext(ctx, "app containers updated successfully", "app", app.Name, "appID", appID)
	return app, nil
}

// RepairApp repairs an app's compose file (e.g., adds missing cloudflared token)
func (s *appService) RepairApp(ctx context.Context, appID string) (*db.App, error) {
	s.logger.InfoContext(ctx, "repairing app", "appID", appID)

	app, err := s.database.GetApp(appID)
	if err != nil {
		s.logger.DebugContext(ctx, "app not found for repair", "appID", appID)
		return nil, domain.WrapAppNotFound(appID, err)
	}

	// Get Cloudflare settings
	settings, err := s.settingsManager.RequireCloudflareSettings()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get settings", "error", err)
		return nil, err
	}

	// Only repair if app has tunnel ID
	if app.TunnelID == "" {
		return nil, fmt.Errorf("cannot repair: no tunnel ID")
	}

	// If tunnel token is already in database, no repair needed
	if app.TunnelToken != "" {
		s.logger.InfoContext(ctx, "app already repaired - tunnel token exists", "appID", appID)
		return app, nil
	}

	// Fetch tunnel token from Cloudflare
	cfManager := cloudflare.NewManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID)
	tunnelToken, err := cfManager.GetTunnelToken(app.TunnelID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get tunnel token from Cloudflare", "appID", appID, "tunnelID", app.TunnelID, "error", err)
		return nil, fmt.Errorf("failed to get tunnel token from Cloudflare: %w", err)
	}

	// Update database with tunnel token
	app.TunnelToken = tunnelToken
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		s.logger.ErrorContext(ctx, "failed to update app with tunnel token", "appID", appID, "error", err)
		return nil, domain.WrapDatabaseOperation("update app", err)
	}

	// Parse existing compose content
	compose, err := docker.ParseCompose([]byte(app.ComposeContent))
	if err != nil {
		s.logger.WarnContext(ctx, "invalid compose file", "appID", appID, "error", err)
		return nil, domain.WrapComposeInvalid(err)
	}

	// Extract network and inject cloudflared with proper token
	networks := docker.ExtractNetworks(compose)
	network := ""
	if len(networks) > 0 {
		network = networks[0]
	}

	if err := docker.InjectCloudflared(compose, app.Name, tunnelToken, network); err != nil {
		s.logger.ErrorContext(ctx, "failed to inject cloudflared service", "appID", appID, "error", err)
		return nil, fmt.Errorf("failed to inject cloudflared service: %w", err)
	}

	composeBytes, err := docker.MarshalComposeFile(compose)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal compose file", "appID", appID, "error", err)
		return nil, fmt.Errorf("failed to marshal compose file: %w", err)
	}

	app.ComposeContent = string(composeBytes)
	if err := s.database.UpdateApp(app); err != nil {
		s.logger.ErrorContext(ctx, "failed to update compose content", "appID", appID, "error", err)
		return nil, domain.WrapDatabaseOperation("update app", err)
	}

	// Update compose file on disk
	if err := s.dockerManager.WriteComposeFile(app.Name, app.ComposeContent); err != nil {
		s.logger.ErrorContext(ctx, "failed to update compose file", "app", app.Name, "error", err)
		return nil, domain.WrapContainerOperationFailed("write compose file", err)
	}

	s.logger.InfoContext(ctx, "app repaired successfully", "app", app.Name, "appID", appID)
	return app, nil
}

// RestartCloudflared restarts the cloudflared container for an app
// This is typically called after updating ingress rules to apply the new configuration
func (s *appService) RestartCloudflared(ctx context.Context, appID string, nodeID string) error {
	s.logger.InfoContext(ctx, "restarting cloudflared container", "appID", appID, "nodeID", nodeID)

	// If app is on a remote node, we can't restart cloudflared remotely
	// The ingress update already happened via Cloudflare API, so this is best-effort
	// In a multi-node setup, cloudflared restart would need to happen on the target node
	// For now, we only restart if the app is on this node
	if !s.router.IsLocalNode(nodeID) {
		s.logger.WarnContext(ctx, "cannot restart cloudflared on remote node, ingress updated but container restart may be required", "appID", appID, "nodeID", nodeID)
		return nil // Don't fail - ingress update already succeeded
	}

	// Get app from local node
	app, err := s.database.GetApp(appID)
	if err != nil {
		s.logger.DebugContext(ctx, "app not found for cloudflared restart", "appID", appID)
		return domain.WrapAppNotFound(appID, err)
	}

	if err := s.dockerManager.RestartCloudflared(app.Name); err != nil {
		s.logger.ErrorContext(ctx, "failed to restart cloudflared container", "app", app.Name, "error", err)
		return domain.WrapContainerOperationFailed("restart cloudflared", err)
	}

	s.logger.InfoContext(ctx, "cloudflared container restarted successfully", "app", app.Name, "appID", appID)
	return nil
}
