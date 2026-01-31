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
	"github.com/selfhostly/internal/tunnel"
	cloudflareProvider "github.com/selfhostly/internal/tunnel/providers/cloudflare"
	"github.com/selfhostly/internal/validation"
)

// appService implements the AppService interface
type appService struct {
	database         *db.DB
	dockerManager    *docker.Manager
	nodeClient       *node.Client
	config           *config.Config
	logger           *slog.Logger
	router           *routing.NodeRouter
	appsAgg          *routing.AppsAggregator
	settingsManager  *cloudflare.SettingsManager  // DEPRECATED: for backward compatibility
	providerRegistry *tunnel.Registry             // NEW: for multi-provider support
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
	
	// Initialize provider registry
	registry := tunnel.NewRegistry()
	
	// Register Cloudflare provider
	registry.Register("cloudflare", func(config map[string]interface{}) (tunnel.Provider, error) {
		config["database"] = database
		config["logger"] = logger
		return cloudflareProvider.NewProvider(config)
	})
	
	// Future providers can be registered here
	
	return &appService{
		database:         database,
		dockerManager:    dockerManager,
		nodeClient:       nodeClient,
		config:           cfg,
		logger:           logger,
		router:           router,
		appsAgg:          appsAgg,
		settingsManager:  settingsManager,
		providerRegistry: registry,
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

	// Get settings
	settings, err := s.database.GetSettings()
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
	var createdTunnelAppID string // Track the app ID used for tunnel creation

	// Try to create tunnel using provider abstraction
	providerName := settings.GetActiveProviderName()
	providerConfig, err := settings.GetProviderConfig(providerName)
	
	if err == nil && providerConfig != nil {
		s.logger.InfoContext(ctx, "creating tunnel using provider", "provider", providerName, "app", req.Name)
		
		provider, err := s.providerRegistry.GetProvider(providerName, providerConfig)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create provider", "provider", providerName, "error", err)
			return nil, fmt.Errorf("failed to create tunnel provider: %w", err)
		}

		// Generate app ID that will be used (consistent with NewApp below)
		tempApp := db.NewApp(req.Name, req.Description, req.ComposeContent)
		createdTunnelAppID = tempApp.ID

		// Create tunnel via provider (provider handles its own DB storage)
		tunnelResult, err := provider.CreateTunnel(ctx, tunnel.CreateOptions{
			AppID: createdTunnelAppID,
			Name:  req.Name,
		})
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create tunnel", "provider", providerName, "error", err)
			return nil, domain.WrapTunnelCreationFailed(req.Name, err)
		}

		tunnelID = tunnelResult.TunnelID
		tunnelToken = tunnelResult.TunnelToken
		publicURL = tunnelResult.PublicURL

		s.logger.InfoContext(ctx, "tunnel created successfully", "provider", providerName, "tunnel_id", tunnelID, "public_url", publicURL)

		// Check if provider needs container injection
		if containerProvider, ok := provider.(tunnel.ContainerProvider); ok {
			containerConfig := containerProvider.GetContainerConfig(tunnelToken, req.Name)
			if containerConfig != nil {
				networks := docker.ExtractNetworks(compose)
				network := ""
				if len(networks) > 0 {
					network = networks[0]
				}

				s.logger.InfoContext(ctx, "injecting tunnel container into compose", "provider", providerName, "app", req.Name)
				injected, err := docker.InjectTunnelContainer(compose, req.Name, containerConfig, network)
				if err != nil {
					s.logger.ErrorContext(ctx, "failed to inject tunnel container", "app", req.Name, "error", err)
					return nil, fmt.Errorf("failed to inject tunnel container: %w", err)
				}

				if injected {
					composeBytes, err := docker.MarshalComposeFile(compose)
					if err != nil {
						s.logger.ErrorContext(ctx, "failed to marshal compose file", "app", req.Name, "error", err)
						return nil, fmt.Errorf("failed to marshal compose file: %w", err)
					}
					req.ComposeContent = string(composeBytes)
					s.logger.InfoContext(ctx, "tunnel container injected successfully", "provider", providerName)
				}
			}
		} else {
			s.logger.DebugContext(ctx, "provider does not require container injection", "provider", providerName)
		}
	} else {
		s.logger.InfoContext(ctx, "no tunnel provider configured, creating app without tunnel", "app", req.Name)
	}

	// Create app in database (using the same ID if tunnel was created)
	var app *db.App
	if createdTunnelAppID != "" {
		// Use the same ID that was used for tunnel creation
		app = &db.App{
			ID:             createdTunnelAppID,
			Name:           req.Name,
			Description:    req.Description,
			ComposeContent: req.ComposeContent,
			TunnelToken:    tunnelToken,
			TunnelID:       tunnelID,
			TunnelDomain:   publicURL,
			PublicURL:      publicURL,
			Status:         "stopped",
			ErrorMessage:   nil,
			NodeID:         s.config.Node.ID,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
	} else {
		// No tunnel created, use normal flow
		app = db.NewApp(req.Name, req.Description, req.ComposeContent)
		app.TunnelToken = tunnelToken
		app.TunnelID = tunnelID
		app.TunnelDomain = publicURL
		app.PublicURL = publicURL
		app.Status = "stopped"
		app.ErrorMessage = nil
		app.NodeID = s.config.Node.ID
		app.UpdatedAt = time.Now()
	}

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

	// Note: Tunnel metadata is already created by the provider during CreateTunnel() call
	// The provider handles its own database table, so we don't need to create it here
	s.logger.DebugContext(ctx, "tunnel metadata handled by provider", "has_tunnel", tunnelID != "")

	// Apply ingress rules if provided and tunnel was created (before auto-start)
	if len(req.IngressRules) > 0 && createdTunnelAppID != "" {
		s.logger.InfoContext(ctx, "applying ingress rules", "app", req.Name, "ruleCount", len(req.IngressRules))

		// Get the provider again to apply ingress rules
		if providerConfig, err := settings.GetProviderConfig(providerName); err == nil && providerConfig != nil {
			if provider, err := s.providerRegistry.GetProvider(providerName, providerConfig); err == nil {
				// Check if provider supports ingress configuration
				if ingressProvider, ok := provider.(tunnel.IngressProvider); ok {
					if err := ingressProvider.UpdateIngress(ctx, createdTunnelAppID, req.IngressRules); err != nil {
						s.logger.ErrorContext(ctx, "failed to apply ingress rules", "provider", providerName, "app", req.Name, "error", err)
						// Don't fail app creation if ingress update fails
					} else {
						s.logger.InfoContext(ctx, "ingress rules applied successfully", "provider", providerName, "app", req.Name)
						
						// Reload app from database to get updated tunnel_domain and public_url
						// UpdateIngress may have updated these fields via the tunnel provider
						if refreshedApp, err := s.database.GetApp(app.ID); err == nil {
							app = refreshedApp
							s.logger.DebugContext(ctx, "reloaded app after ingress update", "app", req.Name, "public_url", app.PublicURL)
						} else {
							s.logger.WarnContext(ctx, "failed to reload app after ingress update", "app", req.Name, "error", err)
						}
					}
				} else {
					s.logger.WarnContext(ctx, "provider does not support ingress configuration", "provider", providerName)
				}
			}
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

			// Restart tunnel container to pick up ingress configuration if app was auto-started
			if len(req.IngressRules) > 0 && createdTunnelAppID != "" {
				s.logger.InfoContext(ctx, "restarting tunnel container after auto-start", "app", req.Name)
				if err := s.dockerManager.RestartTunnelService(app.Name); err != nil {
					s.logger.WarnContext(ctx, "failed to restart tunnel container, ingress rules updated but container restart may be required", "app", req.Name, "appID", app.ID, "error", err)
				} else {
					s.logger.InfoContext(ctx, "tunnel container restarted successfully after ingress update", "app", req.Name, "appID", app.ID)
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

	app := result.(*db.App)
	// Set the node_id for consistency with ListApps
	app.NodeID = nodeID
	return app, nil
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

	result, err := s.router.RouteToNode(ctx, nodeID,
		func() (interface{}, error) {
			app, err := s.database.GetApp(appID)
			if err != nil {
				return nil, domain.WrapAppNotFound(appID, err)
			}

			// Get settings
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

			// If app has a tunnel token, ensure tunnel container is in the compose using provider abstraction
			if app.TunnelToken != "" {
				providerName := settings.GetActiveProviderName()
				providerConfig, err := settings.GetProviderConfig(providerName)
				if err == nil && providerConfig != nil {
					provider, err := s.providerRegistry.GetProvider(providerName, providerConfig)
					if err == nil {
						// Check if provider needs container injection
						if containerProvider, ok := provider.(tunnel.ContainerProvider); ok {
							compose, err := docker.ParseCompose([]byte(composeContent))
							if err != nil {
								s.logger.WarnContext(ctx, "invalid compose file", "appID", appID, "error", err)
								return nil, domain.WrapComposeInvalid(err)
							}
							containerConfig := containerProvider.GetContainerConfig(app.TunnelToken, app.Name)
							if containerConfig != nil {
								networks := docker.ExtractNetworks(compose)
								network := ""
								if len(networks) > 0 {
									network = networks[0]
								}
								injected, err := docker.InjectTunnelContainer(compose, app.Name, containerConfig, network)
								if err != nil {
									s.logger.ErrorContext(ctx, "failed to inject tunnel container", "appID", appID, "error", err)
									return nil, fmt.Errorf("failed to inject tunnel container: %w", err)
								}
								if injected {
									composeBytes, err := docker.MarshalComposeFile(compose)
									if err != nil {
										s.logger.ErrorContext(ctx, "failed to marshal compose file", "appID", appID, "error", err)
										return nil, fmt.Errorf("failed to marshal compose file: %w", err)
									}
									composeContent = string(composeBytes)
								}
							}
						}
					}
				}
			}

			if req.Name != "" {
				app.Name = req.Name
			}
			if req.Description != "" {
				app.Description = req.Description
			}

			composeChanged := composeContent != app.ComposeContent
			app.ComposeContent = composeContent
			app.UpdatedAt = time.Now()

			if err := s.database.UpdateApp(app); err != nil {
				s.logger.ErrorContext(ctx, "failed to update app in database", "appID", appID, "error", err)
				return nil, domain.WrapDatabaseOperation("update app", err)
			}

			if composeChanged {
				latestVersion, err := s.database.GetLatestVersionNumber(appID)
				if err != nil {
					s.logger.WarnContext(ctx, "failed to get latest version number", "appID", appID, "error", err)
					latestVersion = 0
				}
				if err := s.database.MarkAllVersionsAsNotCurrent(appID); err != nil {
					s.logger.WarnContext(ctx, "failed to mark versions as not current", "appID", appID, "error", err)
				}
				updateReason := "Compose file updated"
				newVersion := db.NewComposeVersion(appID, latestVersion+1, app.ComposeContent, &updateReason, nil)
				if err := s.database.CreateComposeVersion(newVersion); err != nil {
					s.logger.WarnContext(ctx, "failed to create compose version", "appID", appID, "error", err)
				}
			}

			if err := s.dockerManager.WriteComposeFile(app.Name, app.ComposeContent); err != nil {
				s.logger.ErrorContext(ctx, "failed to update compose file", "app", app.Name, "error", err)
				return nil, domain.WrapContainerOperationFailed("write compose file", err)
			}

			s.logger.InfoContext(ctx, "app updated successfully", "app", app.Name, "appID", appID)
			return app, nil
		},
		func(n *db.Node) (interface{}, error) {
			return s.nodeClient.UpdateApp(n, appID, req)
		},
	)
	if err != nil {
		return nil, err
	}
	return result.(*db.App), nil
}

// DeleteApp deletes an app using comprehensive cleanup
func (s *appService) DeleteApp(ctx context.Context, appID string, nodeID string) error {
	s.logger.InfoContext(ctx, "deleting app", "appID", appID, "nodeID", nodeID)

	_, err := s.router.RouteToNode(ctx, nodeID,
		func() (interface{}, error) {
			app, err := s.database.GetApp(appID)
			if err != nil {
				return nil, domain.WrapAppNotFound(appID, err)
			}

			tunnelManager, settings, err := s.settingsManager.GetConfiguredTunnelManager()
			if err != nil {
				s.logger.WarnContext(ctx, "failed to get settings for cleanup", "app", app.Name, "error", err)
				settings = nil
			}

			cleanupManager := cleanup.NewCleanupManager(s.dockerManager, s.database, settings, tunnelManager)
			results, err := cleanupManager.CleanupApp(app)
			successCount, failedCount, totalDuration := cleanupManager.GetSummary()

			if err != nil {
				s.logger.ErrorContext(ctx, "app cleanup completed with errors",
					"app", app.Name, "appID", app.ID,
					"successCount", successCount, "failedCount", failedCount, "totalDuration", totalDuration, "error", err)
			} else {
				s.logger.InfoContext(ctx, "app cleanup completed successfully",
					"app", app.Name, "appID", app.ID,
					"successCount", successCount, "failedCount", failedCount, "totalDuration", totalDuration)
			}
			for _, result := range results {
				if !result.Success {
					s.logger.ErrorContext(ctx, "cleanup step failed", "app", app.Name, "step", result.Step, "error", result.Error, "duration", result.Duration)
				}
			}
			return nil, err
		},
		func(n *db.Node) (interface{}, error) {
			return nil, s.nodeClient.DeleteApp(n, appID)
		},
	)
	return err
}

// StartApp starts an application
func (s *appService) StartApp(ctx context.Context, appID string, nodeID string) (*db.App, error) {
	s.logger.InfoContext(ctx, "starting app", "appID", appID, "nodeID", nodeID)

	result, err := s.router.RouteToNode(ctx, nodeID,
		func() (interface{}, error) {
			app, err := s.database.GetApp(appID)
			if err != nil {
				return nil, domain.WrapAppNotFound(appID, err)
			}
			if err := s.dockerManager.StartApp(app.Name); err != nil {
				app.Status = "error"
				em := err.Error()
				app.ErrorMessage = &em
				app.UpdatedAt = time.Now()
				_ = s.database.UpdateApp(app)
				return nil, domain.WrapContainerOperationFailed("start app", err)
			}
			app.Status = "running"
			app.ErrorMessage = nil
			app.UpdatedAt = time.Now()
			if err := s.database.UpdateApp(app); err != nil {
				return nil, domain.WrapDatabaseOperation("update app status", err)
			}
			s.logger.InfoContext(ctx, "app started successfully", "app", app.Name, "appID", appID)
			return app, nil
		},
		func(n *db.Node) (interface{}, error) {
			if err := s.nodeClient.StartApp(n, appID); err != nil {
				return nil, err
			}
			return s.nodeClient.GetApp(n, appID)
		},
	)
	if err != nil {
		return nil, err
	}
	return result.(*db.App), nil
}

// StopApp stops an application
func (s *appService) StopApp(ctx context.Context, appID string, nodeID string) (*db.App, error) {
	s.logger.InfoContext(ctx, "stopping app", "appID", appID, "nodeID", nodeID)

	result, err := s.router.RouteToNode(ctx, nodeID,
		func() (interface{}, error) {
			app, err := s.database.GetApp(appID)
			if err != nil {
				return nil, domain.WrapAppNotFound(appID, err)
			}
			if err := s.dockerManager.StopApp(app.Name); err != nil {
				app.Status = "error"
				em := err.Error()
				app.ErrorMessage = &em
				app.UpdatedAt = time.Now()
				_ = s.database.UpdateApp(app)
				return nil, domain.WrapContainerOperationFailed("stop app", err)
			}
			app.Status = "stopped"
			app.ErrorMessage = nil
			app.UpdatedAt = time.Now()
			if err := s.database.UpdateApp(app); err != nil {
				return nil, domain.WrapDatabaseOperation("update app status", err)
			}
			s.logger.InfoContext(ctx, "app stopped successfully", "app", app.Name, "appID", appID)
			return app, nil
		},
		func(n *db.Node) (interface{}, error) {
			if err := s.nodeClient.StopApp(n, appID); err != nil {
				return nil, err
			}
			return s.nodeClient.GetApp(n, appID)
		},
	)
	if err != nil {
		return nil, err
	}
	return result.(*db.App), nil
}

// UpdateAppContainers updates app containers with zero downtime
func (s *appService) UpdateAppContainers(ctx context.Context, appID string, nodeID string) (*db.App, error) {
	s.logger.InfoContext(ctx, "updating app containers", "appID", appID, "nodeID", nodeID)

	result, err := s.router.RouteToNode(ctx, nodeID,
		func() (interface{}, error) {
			app, err := s.database.GetApp(appID)
			if err != nil {
				return nil, domain.WrapAppNotFound(appID, err)
			}
			app.Status = "updating"
			app.UpdatedAt = time.Now()
			if err := s.database.UpdateApp(app); err != nil {
				return nil, domain.WrapDatabaseOperation("update app status", err)
			}
			if err := s.dockerManager.UpdateApp(app.Name); err != nil {
				app.Status = "error"
				em := err.Error()
				app.ErrorMessage = &em
				app.UpdatedAt = time.Now()
				_ = s.database.UpdateApp(app)
				return nil, domain.WrapContainerOperationFailed("update app", err)
			}
			app.Status = "running"
			app.ErrorMessage = nil
			app.UpdatedAt = time.Now()
			if err := s.database.UpdateApp(app); err != nil {
				return nil, domain.WrapDatabaseOperation("update app status", err)
			}
			s.logger.InfoContext(ctx, "app containers updated successfully", "app", app.Name, "appID", appID)
			return app, nil
		},
		func(n *db.Node) (interface{}, error) {
			return s.nodeClient.UpdateAppContainers(n, appID)
		},
	)
	if err != nil {
		return nil, err
	}
	return result.(*db.App), nil
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
