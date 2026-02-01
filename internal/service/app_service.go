package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/selfhostly/internal/cleanup"
	"github.com/selfhostly/internal/cloudflare"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
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
	tunnelService    domain.TunnelService          // NEW: for Quick Tunnel operations
}



// NewAppService creates a new app service
func NewAppService(
	database *db.DB,
	dockerManager *docker.Manager,
	cfg *config.Config,
	logger *slog.Logger,
	tunnelService domain.TunnelService,
) domain.AppService {
	nodeClient := node.NewClient()
	router := routing.NewNodeRouter(database, nodeClient, cfg.Node.ID, logger)
	appsAgg := routing.NewAppsAggregator(router, logger)
	settingsManager := cloudflare.NewSettingsManager(database, logger)
	
	// Initialize provider registry
	registry := tunnel.NewRegistry()
	
	// Register Cloudflare provider
	registry.Register(constants.ProviderCloudflare, func(config map[string]interface{}) (tunnel.Provider, error) {
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
		tunnelService:    tunnelService,
	}
}

// CreateApp creates a new application (local only; gateway forwards POST /api/apps to target node)
func (s *appService) CreateApp(ctx context.Context, req domain.CreateAppRequest) (*db.App, error) {
	s.logger.InfoContext(ctx, "creating app", "name", req.Name, "targetNode", req.NodeID)

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

	// Validate Quick Tunnel params when tunnel_mode is "quick"
	if req.TunnelMode == constants.TunnelModeQuick {
		if strings.TrimSpace(req.QuickTunnelService) == "" {
			return nil, domain.WrapValidationError("quick_tunnel_service", fmt.Errorf("quick_tunnel_service is required for Quick Tunnel mode"))
		}
		if req.QuickTunnelPort < constants.MinPort || req.QuickTunnelPort > constants.MaxPort {
			return nil, domain.WrapValidationError("quick_tunnel_port", fmt.Errorf("quick_tunnel_port must be between %d and %d", constants.MinPort, constants.MaxPort))
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
	var tunnelMode string         // "custom" | "quick" | ""

	providerName := settings.GetActiveProviderName()
	providerConfig, providerConfigErr := settings.GetProviderConfig(providerName)

	if req.TunnelMode == constants.TunnelModeQuick {
		// Quick Tunnel: delegate to tunnel service for provider-agnostic handling
		tunnelMode = constants.TunnelModeQuick
		tempApp := db.NewApp(req.Name, req.Description, req.ComposeContent)
		createdTunnelAppID = tempApp.ID

		metricsPort, err := s.tunnelService.NextFreeQuickTunnelMetricsPort()
		if err != nil {
			s.logger.WarnContext(ctx, "failed to allocate Quick Tunnel metrics port, using fallback", "app", req.Name, "error", err)
		}
		containerConfig, err := s.tunnelService.CreateQuickTunnelConfig(strings.TrimSpace(req.QuickTunnelService), req.QuickTunnelPort, metricsPort)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create Quick Tunnel config", "app", req.Name, "error", err)
			return nil, fmt.Errorf("failed to create Quick Tunnel config: %w", err)
		}
		
		networks := docker.ExtractNetworks(compose)
		network := ""
		if len(networks) > 0 {
			network = networks[0]
		}
		s.logger.InfoContext(ctx, "injecting Quick Tunnel container into compose", "app", req.Name, "service", req.QuickTunnelService, "port", req.QuickTunnelPort)
		injected, err := docker.InjectTunnelContainer(compose, req.Name, containerConfig, network)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to inject Quick Tunnel container", "app", req.Name, "error", err)
			return nil, fmt.Errorf("failed to inject Quick Tunnel container: %w", err)
		}
		if injected {
			composeBytes, err := docker.MarshalComposeFile(compose)
			if err != nil {
				s.logger.ErrorContext(ctx, "failed to marshal compose file", "app", req.Name, "error", err)
				return nil, fmt.Errorf("failed to marshal compose file: %w", err)
			}
			req.ComposeContent = string(composeBytes)
			s.logger.InfoContext(ctx, "Quick Tunnel container injected successfully", "app", req.Name)
		}
	} else if req.TunnelMode == constants.TunnelModeCustom && providerConfigErr == nil && providerConfig != nil {
		// Custom (named) tunnel: create via provider API and inject container
		tunnelMode = constants.TunnelModeCustom
		s.logger.InfoContext(ctx, "creating tunnel using provider", "provider", providerName, "app", req.Name)

		provider, err := s.providerRegistry.GetProvider(providerName, providerConfig)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create provider", "provider", providerName, "error", err)
			return nil, fmt.Errorf("failed to create tunnel provider: %w", err)
		}

		tempApp := db.NewApp(req.Name, req.Description, req.ComposeContent)
		createdTunnelAppID = tempApp.ID

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
		if req.TunnelMode == "" {
			s.logger.InfoContext(ctx, "no tunnel selected, creating app without tunnel", "app", req.Name)
		} else {
			s.logger.InfoContext(ctx, "no tunnel provider configured, creating app without tunnel", "app", req.Name)
		}
	}

	// Create app in database (using the same ID if tunnel was created)
	var app *db.App
	if createdTunnelAppID != "" {
		app = &db.App{
			ID:             createdTunnelAppID,
			Name:           req.Name,
			Description:    req.Description,
			ComposeContent: req.ComposeContent,
			TunnelToken:    tunnelToken,
			TunnelID:       tunnelID,
			TunnelDomain:   publicURL,
			PublicURL:      publicURL,
			Status:         constants.AppStatusStopped,
			ErrorMessage:   nil,
			NodeID:         s.config.Node.ID,
			TunnelMode:     tunnelMode,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
	} else {
		app = db.NewApp(req.Name, req.Description, req.ComposeContent)
		app.TunnelToken = tunnelToken
		app.TunnelID = tunnelID
		app.TunnelDomain = publicURL
		app.PublicURL = publicURL
		app.Status = constants.AppStatusStopped
		app.ErrorMessage = nil
		app.NodeID = s.config.Node.ID
		app.TunnelMode = tunnelMode
		app.UpdatedAt = time.Now()
	}

	if err := s.database.CreateApp(app); err != nil {
		s.logger.ErrorContext(ctx, "failed to create app in database", "app", req.Name, "error", err)
		return nil, domain.WrapDatabaseOperation("create app", err)
	}

	// Create initial compose version (version 1)
	// Note: changedBy will be set by the caller if user context is available
	initialReason := constants.ComposeVersionReasonInitial
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

	// Apply ingress rules if provided and named tunnel was created (custom mode only)
	if len(req.IngressRules) > 0 && createdTunnelAppID != "" && tunnelMode == constants.TunnelModeCustom {
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

			// Quick Tunnel: extract URL from metrics endpoint and update app
			if app.TunnelMode == constants.TunnelModeQuick {
				// Delegate URL extraction to tunnel service (which uses QuickTunnelProvider)
				// The extractor has built-in retry logic to handle startup delays
				quickURL, err := s.tunnelService.ExtractQuickTunnelURL(ctx, app.ID, s.config.Node.ID)
				if err != nil {
					s.logger.WarnContext(ctx, "failed to extract Quick Tunnel URL, app may need manual refresh", "app", app.Name, "error", err)
				} else if quickURL != "" {
					app.PublicURL = quickURL
					app.TunnelDomain = strings.TrimPrefix(quickURL, "https://")
					app.UpdatedAt = time.Now()
					if err := s.database.UpdateApp(app); err != nil {
						s.logger.WarnContext(ctx, "failed to save Quick Tunnel URL to app", "app", app.Name, "error", err)
					} else {
						s.logger.InfoContext(ctx, "Quick Tunnel URL captured", "app", app.Name, "public_url", quickURL)
					}
				}
			}

			// Restart tunnel container to pick up ingress configuration if app was auto-started (custom tunnel only)
			if len(req.IngressRules) > 0 && createdTunnelAppID != "" && tunnelMode == constants.TunnelModeCustom {
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

// GetApp retrieves an app by ID (local only; gateway routes to this node)
func (s *appService) GetApp(ctx context.Context, appID string, nodeID string) (*db.App, error) {
	s.logger.DebugContext(ctx, "getting app", "appID", appID, "nodeID", nodeID)
	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}
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

	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}

	settings, err := s.settingsManager.GetSettings()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get settings", "error", err)
		return nil, err
	}

	composeContent := app.ComposeContent
	if req.ComposeContent != "" {
		composeContent = req.ComposeContent
	}

	if app.TunnelMode == constants.TunnelModeQuick && req.ComposeContent != "" {
		if targetService, targetPort, ok := docker.ExtractQuickTunnelTargetFromCompose(app.ComposeContent); ok {
			compose, err := docker.ParseCompose([]byte(composeContent))
			if err == nil {
				metricsPort := constants.QuickTunnelMetricsPort
				if p, ok := docker.ExtractQuickTunnelMetricsHostPort(app.ComposeContent); ok {
					metricsPort = p
				}
				containerConfig, err := s.tunnelService.CreateQuickTunnelConfig(targetService, targetPort, metricsPort)
				if err == nil {
					networks := docker.ExtractNetworks(compose)
					network := ""
					if len(networks) > 0 {
						network = networks[0]
					}
					injected, err := docker.InjectTunnelContainer(compose, app.Name, containerConfig, network)
					if err == nil && injected {
						composeBytes, err := docker.MarshalComposeFile(compose)
						if err == nil {
							composeContent = string(composeBytes)
							s.logger.InfoContext(ctx, "re-injected Quick Tunnel container into updated compose", "appID", appID, "target", targetService+":"+fmt.Sprint(targetPort))
						}
					}
				}
			}
		}
	}

	if app.TunnelToken != "" {
		providerName := settings.GetActiveProviderName()
		providerConfig, err := settings.GetProviderConfig(providerName)
		if err == nil && providerConfig != nil {
			provider, err := s.providerRegistry.GetProvider(providerName, providerConfig)
			if err == nil {
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
		updateReason := constants.ComposeVersionReasonUpdated
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
}

// DeleteApp deletes an app using comprehensive cleanup (local only)
func (s *appService) DeleteApp(ctx context.Context, appID string, nodeID string) error {
	s.logger.InfoContext(ctx, "deleting app", "appID", appID, "nodeID", nodeID)
	app, err := s.database.GetApp(appID)
	if err != nil {
		return domain.WrapAppNotFound(appID, err)
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
	return err
}

// StartApp starts an application (local only)
func (s *appService) StartApp(ctx context.Context, appID string, nodeID string) (*db.App, error) {
	s.logger.InfoContext(ctx, "starting app", "appID", appID, "nodeID", nodeID)
	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}
	if err := s.dockerManager.StartApp(app.Name); err != nil {
		app.Status = constants.AppStatusError
		em := err.Error()
		app.ErrorMessage = &em
		app.UpdatedAt = time.Now()
		_ = s.database.UpdateApp(app)
		return nil, domain.WrapContainerOperationFailed("start app", err)
	}
	app.Status = constants.AppStatusRunning
	app.ErrorMessage = nil
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		return nil, domain.WrapDatabaseOperation("update app status", err)
	}
	s.logger.InfoContext(ctx, "app started successfully", "app", app.Name, "appID", appID)
	return app, nil
}

// StopApp stops an application (local only)
func (s *appService) StopApp(ctx context.Context, appID string, nodeID string) (*db.App, error) {
	s.logger.InfoContext(ctx, "stopping app", "appID", appID, "nodeID", nodeID)
	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}
	if err := s.dockerManager.StopApp(app.Name); err != nil {
		app.Status = constants.AppStatusError
		em := err.Error()
		app.ErrorMessage = &em
		app.UpdatedAt = time.Now()
		_ = s.database.UpdateApp(app)
		return nil, domain.WrapContainerOperationFailed("stop app", err)
	}
	app.Status = constants.AppStatusStopped
	app.ErrorMessage = nil
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		return nil, domain.WrapDatabaseOperation("update app status", err)
	}
	s.logger.InfoContext(ctx, "app stopped successfully", "app", app.Name, "appID", appID)
	return app, nil
}

// UpdateAppContainers updates app containers with zero downtime (local only)
func (s *appService) UpdateAppContainers(ctx context.Context, appID string, nodeID string) (*db.App, error) {
	s.logger.InfoContext(ctx, "updating app containers", "appID", appID, "nodeID", nodeID)
	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}
	app.Status = constants.AppStatusUpdating
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		return nil, domain.WrapDatabaseOperation("update app status", err)
	}
	if err := s.dockerManager.WriteComposeFile(app.Name, app.ComposeContent); err != nil {
		app.Status = constants.AppStatusError
		em := err.Error()
		app.ErrorMessage = &em
		app.UpdatedAt = time.Now()
		_ = s.database.UpdateApp(app)
		return nil, domain.WrapContainerOperationFailed("write compose file", err)
	}
	if err := s.dockerManager.UpdateApp(app.Name); err != nil {
		app.Status = constants.AppStatusError
		em := err.Error()
		app.ErrorMessage = &em
		app.UpdatedAt = time.Now()
		_ = s.database.UpdateApp(app)
		return nil, domain.WrapContainerOperationFailed("update app", err)
	}
	if err := s.dockerManager.ForceRecreateTunnel(app.Name); err != nil {
		s.logger.WarnContext(ctx, "could not force-recreate tunnel (app may have no tunnel)", "app", app.Name, "appID", appID, "error", err)
	}
	app.Status = constants.AppStatusRunning
	app.ErrorMessage = nil
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		return nil, domain.WrapDatabaseOperation("update app status", err)
	}
	s.logger.InfoContext(ctx, "app containers updated successfully", "app", app.Name, "appID", appID)
	return app, nil
}

// CreateTunnelForApp creates a named (custom domain) tunnel for an app that has none (local only).
func (s *appService) CreateTunnelForApp(ctx context.Context, appID string, nodeID string, body interface{}) (*db.App, bool, error) {
	app, err := s.createTunnelForAppLocal(ctx, appID, nodeID)
	if err != nil {
		return nil, false, err
	}
	return app, true, nil
}

// createTunnelForAppLocal runs the create-tunnel logic on this node (DB, provider, compose, UpdateAppContainers).
func (s *appService) createTunnelForAppLocal(ctx context.Context, appID string, nodeID string) (*db.App, error) {
	s.logger.InfoContext(ctx, "creating tunnel for app", "appID", appID, "nodeID", nodeID)

	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}
	if app.NodeID != "" && app.NodeID != nodeID {
		return nil, fmt.Errorf("app belongs to node %s, not %s", app.NodeID, nodeID)
	}

	// App must not already have a named tunnel
	_, err = s.database.GetCloudflareTunnelByAppID(appID)
	if err == nil {
		return nil, fmt.Errorf("app already has a named tunnel")
	}
	if err != sql.ErrNoRows {
		return nil, domain.WrapDatabaseOperation("get tunnel", err)
	}

	settings, err := s.database.GetSettings()
	if err != nil {
		return nil, err
	}
	providerName := settings.GetActiveProviderName()
	providerConfig, err := settings.GetProviderConfig(providerName)
	if err != nil || providerConfig == nil {
		return nil, fmt.Errorf("tunnel provider not configured: %w", err)
	}
	provider, err := s.providerRegistry.GetProvider(providerName, providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create tunnel provider: %w", err)
	}

	tunnelResult, err := provider.CreateTunnel(ctx, tunnel.CreateOptions{AppID: app.ID, Name: app.Name})
	if err != nil {
		return nil, domain.WrapTunnelCreationFailed(app.Name, err)
	}

	app.TunnelID = tunnelResult.TunnelID
	app.TunnelToken = tunnelResult.TunnelToken
	app.TunnelMode = "custom"
	app.PublicURL = tunnelResult.PublicURL
	app.TunnelDomain = strings.TrimPrefix(tunnelResult.PublicURL, "https://")
	app.UpdatedAt = time.Now()

	containerProvider, ok := provider.(tunnel.ContainerProvider)
	if !ok || containerProvider == nil {
		s.database.UpdateApp(app)
		_, _ = s.UpdateAppContainers(ctx, appID, nodeID)
		return app, nil
	}
	containerConfig := containerProvider.GetContainerConfig(tunnelResult.TunnelToken, app.Name)
	if containerConfig == nil {
		if err := s.database.UpdateApp(app); err != nil {
			return nil, err
		}
		_, _ = s.UpdateAppContainers(ctx, appID, nodeID)
		return app, nil
	}

	compose, err := docker.ParseCompose([]byte(app.ComposeContent))
	if err != nil {
		return nil, domain.WrapComposeInvalid(err)
	}
	networks := docker.ExtractNetworks(compose)
	network := ""
	if len(networks) > 0 {
		network = networks[0]
	}
	injected, err := docker.InjectTunnelContainer(compose, app.Name, containerConfig, network)
	if err != nil {
		return nil, fmt.Errorf("failed to inject tunnel container: %w", err)
	}
	if injected {
		composeBytes, err := docker.MarshalComposeFile(compose)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal compose: %w", err)
		}
		app.ComposeContent = string(composeBytes)
	}
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		return nil, domain.WrapDatabaseOperation("update app", err)
	}

	_, err = s.UpdateAppContainers(ctx, appID, nodeID)
	if err != nil {
		s.logger.WarnContext(ctx, "tunnel created but UpdateAppContainers failed", "appID", appID, "nodeID", nodeID, "error", err)
	}
	return app, nil
}

// SwitchAppToCustomTunnel switches an app from Quick Tunnel to a named (custom domain) tunnel (local only).
func (s *appService) SwitchAppToCustomTunnel(ctx context.Context, appID string, nodeID string, body interface{}) (*db.App, error) {
	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}
	if app.TunnelMode != constants.TunnelModeQuick {
		return nil, fmt.Errorf("app is not using Quick Tunnel (tunnel_mode=%q)", app.TunnelMode)
	}
	createdApp, _, err := s.CreateTunnelForApp(ctx, appID, nodeID, body)
	return createdApp, err
}

// CreateQuickTunnelForApp adds a Quick Tunnel (temporary trycloudflare.com URL) to an app that has no tunnel.
func (s *appService) CreateQuickTunnelForApp(ctx context.Context, appID string, nodeID string, service string, port int) (*db.App, error) {
	s.logger.InfoContext(ctx, "creating Quick Tunnel for app", "appID", appID, "nodeID", nodeID, "service", service, "port", port)

	if strings.TrimSpace(service) == "" {
		return nil, domain.WrapValidationError("service", fmt.Errorf("service is required for Quick Tunnel"))
	}
	if port < constants.MinPort || port > constants.MaxPort {
		return nil, domain.WrapValidationError("port", fmt.Errorf("port must be between %d and %d", constants.MinPort, constants.MaxPort))
	}

	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}
	if app.TunnelMode == constants.TunnelModeCustom || app.TunnelToken != "" {
		return nil, domain.WrapValidationError("tunnel", fmt.Errorf("this app uses a custom domain tunnel; delete the tunnel from the Cloudflare tab first if you want a temporary Quick Tunnel URL instead"))
	}

	compose, err := docker.ParseCompose([]byte(app.ComposeContent))
	if err != nil {
		s.logger.WarnContext(ctx, "invalid compose file", "appID", appID, "error", err)
		return nil, domain.WrapComposeInvalid(err)
	}

	metricsPort, err := s.tunnelService.NextFreeQuickTunnelMetricsPort()
	if err != nil {
		s.logger.WarnContext(ctx, "failed to allocate Quick Tunnel metrics port, using fallback", "appID", appID, "error", err)
	}
	containerConfig, err := s.tunnelService.CreateQuickTunnelConfig(strings.TrimSpace(service), port, metricsPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create Quick Tunnel config: %w", err)
	}
	networks := docker.ExtractNetworks(compose)
	network := ""
	if len(networks) > 0 {
		network = networks[0]
	}
	s.logger.InfoContext(ctx, "injecting Quick Tunnel container into compose", "app", app.Name, "service", service, "port", port, "metricsPort", metricsPort)
	injected, err := docker.InjectTunnelContainer(compose, app.Name, containerConfig, network)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to inject Quick Tunnel container", "app", app.Name, "error", err)
		return nil, fmt.Errorf("failed to inject Quick Tunnel container: %w", err)
	}
	if !injected {
		return nil, fmt.Errorf("Quick Tunnel container was not injected (may already exist)")
	}
	composeBytes, err := docker.MarshalComposeFile(compose)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal compose file", "app", app.Name, "error", err)
		return nil, fmt.Errorf("failed to marshal compose file: %w", err)
	}
	app.ComposeContent = string(composeBytes)
	app.TunnelMode = constants.TunnelModeQuick
	app.TunnelID = ""
	app.TunnelToken = ""
	app.TunnelDomain = ""
	app.PublicURL = ""
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		s.logger.ErrorContext(ctx, "failed to update app in database", "appID", appID, "error", err)
		return nil, domain.WrapDatabaseOperation("update app", err)
	}

	latestVersion, err := s.database.GetLatestVersionNumber(appID)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to get latest version number", "appID", appID, "error", err)
		latestVersion = 0
	}
	if err := s.database.MarkAllVersionsAsNotCurrent(appID); err != nil {
		s.logger.WarnContext(ctx, "failed to mark versions as not current", "appID", appID, "error", err)
	}
	updateReason := constants.ComposeVersionReasonQuickTunnel
	newVersion := db.NewComposeVersion(appID, latestVersion+1, app.ComposeContent, &updateReason, nil)
	if err := s.database.CreateComposeVersion(newVersion); err != nil {
		s.logger.WarnContext(ctx, "failed to create compose version", "appID", appID, "error", err)
	}

	if err := s.dockerManager.WriteComposeFile(app.Name, app.ComposeContent); err != nil {
		s.logger.ErrorContext(ctx, "failed to write compose file", "app", app.Name, "error", err)
		return nil, domain.WrapContainerOperationFailed("write compose file", err)
	}

	if err := s.dockerManager.StartApp(app.Name); err != nil {
		s.logger.ErrorContext(ctx, "failed to start app for Quick Tunnel", "app", app.Name, "error", err)
		app.Status = constants.AppStatusError
		em := err.Error()
		app.ErrorMessage = &em
		app.UpdatedAt = time.Now()
		_ = s.database.UpdateApp(app)
		return nil, domain.WrapContainerOperationFailed("start app", err)
	}
	app.Status = constants.AppStatusRunning
	app.ErrorMessage = nil
	app.UpdatedAt = time.Now()
	if err := s.database.UpdateApp(app); err != nil {
		s.logger.WarnContext(ctx, "failed to update app status after start", "app", app.Name, "error", err)
	}

	// Delegate URL extraction to tunnel service (which uses QuickTunnelProvider)
	// The extractor has built-in retry logic to handle startup delays
	quickURL, err := s.tunnelService.ExtractQuickTunnelURL(ctx, appID, nodeID)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to extract Quick Tunnel URL, app may need manual refresh", "app", app.Name, "error", err)
	} else if quickURL != "" {
		app.PublicURL = quickURL
		app.TunnelDomain = strings.TrimPrefix(quickURL, "https://")
		app.UpdatedAt = time.Now()
		if err := s.database.UpdateApp(app); err != nil {
			s.logger.WarnContext(ctx, "failed to save Quick Tunnel URL to app", "app", app.Name, "error", err)
		} else {
			s.logger.InfoContext(ctx, "Quick Tunnel URL captured", "app", app.Name, "public_url", quickURL)
		}
	}
	return app, nil
}

// GetQuickTunnelURL runs Quick Tunnel URL extraction on this node and returns the URL (local only).
// Delegates to tunnel service which uses QuickTunnelProvider.
func (s *appService) GetQuickTunnelURL(ctx context.Context, appID string, nodeID string) (string, error) {
	app, err := s.database.GetApp(appID)
	if err != nil {
		return "", domain.WrapAppNotFound(appID, err)
	}
	if app.TunnelMode != constants.TunnelModeQuick {
		return "", fmt.Errorf("app is not in Quick Tunnel mode (tunnel_mode=%q)", app.TunnelMode)
	}
	
	// Delegate to tunnel service (which uses QuickTunnelProvider)
	return s.tunnelService.ExtractQuickTunnelURL(ctx, appID, nodeID)
}

// RestartCloudflared restarts the cloudflared container for an app (local only; gateway routes to this node)
func (s *appService) RestartCloudflared(ctx context.Context, appID string, nodeID string) error {
	s.logger.InfoContext(ctx, "restarting cloudflared container", "appID", appID, "nodeID", nodeID)
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
