package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/node"
	"github.com/selfhostly/internal/routing"
	"github.com/selfhostly/internal/system"
	"github.com/selfhostly/internal/validation"
)

// systemService implements the SystemService interface
type systemService struct {
	database      *db.DB
	dockerManager *docker.Manager
	nodeClient    *node.Client
	config        *config.Config
	collector     *system.Collector
	logger        *slog.Logger
	router        *routing.NodeRouter
	statsAgg      *routing.StatsAggregator
}

// NewSystemService creates a new system service
func NewSystemService(
	database *db.DB,
	dockerManager *docker.Manager,
	cfg *config.Config,
	logger *slog.Logger,
) domain.SystemService {
	collector := system.NewCollector(cfg.AppsDir, dockerManager, database, cfg.Node.ID, cfg.Node.Name)
	nodeClient := node.NewClient()
	router := routing.NewNodeRouter(database, nodeClient, cfg.Node.ID, logger)
	statsAgg := routing.NewStatsAggregator(router, logger)

	return &systemService{
		database:      database,
		dockerManager: dockerManager,
		nodeClient:    nodeClient,
		config:        cfg,
		collector:     collector,
		logger:        logger,
		router:        router,
		statsAgg:      statsAgg,
	}
}

// GetSystemStats retrieves system-wide statistics from specified nodes
func (s *systemService) GetSystemStats(ctx context.Context, nodeIDs []string) ([]*system.SystemStats, error) {
	s.logger.DebugContext(ctx, "getting system stats", "nodeIDs", nodeIDs)

	// Determine which nodes to fetch from
	targetNodes, err := s.router.DetermineTargetNodes(ctx, nodeIDs)
	if err != nil {
		return nil, err
	}

	// Log resolved nodes for debugging
	if len(nodeIDs) > 0 && !(len(nodeIDs) == 1 && nodeIDs[0] == "all") {
		resolvedIDs := make([]string, len(targetNodes))
		for i, n := range targetNodes {
			resolvedIDs[i] = n.ID
		}
		s.logger.DebugContext(ctx, "resolved target nodes", "count", len(targetNodes), "node_ids", resolvedIDs)
	}

	// Aggregate stats from all target nodes in parallel
	allStats, err := s.statsAgg.AggregateStats(
		ctx,
		targetNodes,
		func() (*system.SystemStats, error) {
			return s.collector.GetSystemStats()
		},
		func(n *db.Node) (map[string]interface{}, error) {
			return s.nodeClient.GetSystemStats(n)
		},
		s.mapToSystemStats,
	)

	return allStats, err
}

// mapToSystemStats converts a map[string]interface{} (from JSON) to SystemStats
func (s *systemService) mapToSystemStats(data map[string]interface{}, nodeID, nodeName string) (*system.SystemStats, error) {
	// Marshal map back to JSON, then unmarshal into SystemStats struct
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stats map: %w", err)
	}

	var stats system.SystemStats
	if err := json.Unmarshal(jsonData, &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats: %w", err)
	}

	// Ensure node ID and name are set correctly
	stats.NodeID = nodeID
	stats.NodeName = nodeName

	return &stats, nil
}

// GetAppStats retrieves resource statistics for a specific app (local only)
func (s *systemService) GetAppStats(ctx context.Context, appID string, nodeID string) (*domain.AppStats, error) {
	s.logger.DebugContext(ctx, "getting app stats", "appID", appID, "nodeID", nodeID)
	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}
	if app.Status != constants.AppStatusRunning {
		return &domain.AppStats{
			AppName:          app.Name,
			TotalCPUPercent:  0,
			TotalMemoryBytes: 0,
			Containers:       []domain.ContainerStats{},
			Status:           app.Status,
			Message:          fmt.Sprintf("App is %s", app.Status),
		}, nil
	}
	dockerStats, err := s.dockerManager.GetAppStats(app.Name)
	if err != nil {
		return nil, domain.WrapContainerOperationFailed("get app stats", err)
	}
	containers := make([]domain.ContainerStats, len(dockerStats.Containers))
	for i, c := range dockerStats.Containers {
		memPercent := float64(0)
		if c.MemoryLimit > 0 {
			memPercent = (float64(c.MemoryUsage) / float64(c.MemoryLimit)) * 100
		}
		containers[i] = domain.ContainerStats{
			ID:            c.ContainerID,
			Name:          c.ContainerName,
			CPUPercent:    c.CPUPercent,
			MemoryBytes:   int64(c.MemoryUsage),
			MemoryLimit:   int64(c.MemoryLimit),
			MemoryPercent: memPercent,
			NetInput:      int64(c.NetworkRx),
			NetOutput:     int64(c.NetworkTx),
			BlockInput:    int64(c.BlockRead),
			BlockOutput:   int64(c.BlockWrite),
		}
	}
	return &domain.AppStats{
		AppName:          dockerStats.AppName,
		TotalCPUPercent:  dockerStats.TotalCPU,
		TotalMemoryBytes: int64(dockerStats.TotalMemory),
		MemoryLimitBytes: int64(dockerStats.MemoryLimit),
		Containers:       containers,
		Timestamp:        dockerStats.Timestamp,
		Status:           app.Status,
		Message:          "",
	}, nil
}

// GetAppLogs retrieves logs for a specific app
func (s *systemService) GetAppLogs(ctx context.Context, appID string, nodeID string) ([]byte, error) {
	s.logger.DebugContext(ctx, "getting app logs", "appID", appID, "nodeID", nodeID)

	app, err := s.database.GetApp(appID)
	if err != nil {
		return nil, domain.WrapAppNotFound(appID, err)
	}

	logs, err := s.dockerManager.GetAppLogs(app.Name)
	if err != nil {
		return nil, domain.WrapContainerOperationFailed("get app logs", err)
	}

	return logs, nil
}

// RestartContainer restarts a specific container
func (s *systemService) RestartContainer(ctx context.Context, containerID, nodeID string) error {
	s.logger.InfoContext(ctx, "restarting container", "containerID", containerID, "nodeID", nodeID)

	// Validate container ID
	if err := validation.ValidateContainerID(containerID); err != nil {
		s.logger.WarnContext(ctx, "invalid container ID", "containerID", containerID, "error", err)
		return domain.WrapValidationError("container ID", err)
	}

	if err := s.dockerManager.RestartContainer(containerID); err != nil {
		s.logger.ErrorContext(ctx, "failed to restart container", "containerID", containerID, "nodeID", nodeID, "error", err)
		return domain.WrapContainerOperationFailed("restart container", err)
	}

	s.logger.InfoContext(ctx, "container restarted successfully", "containerID", containerID, "nodeID", nodeID)
	return nil
}

// StopContainer stops a specific container
func (s *systemService) StopContainer(ctx context.Context, containerID, nodeID string) error {
	s.logger.InfoContext(ctx, "stopping container", "containerID", containerID, "nodeID", nodeID)

	// Validate container ID
	if err := validation.ValidateContainerID(containerID); err != nil {
		s.logger.WarnContext(ctx, "invalid container ID", "containerID", containerID, "error", err)
		return domain.WrapValidationError("container ID", err)
	}

	if err := s.dockerManager.StopContainer(containerID); err != nil {
		s.logger.ErrorContext(ctx, "failed to stop container", "containerID", containerID, "nodeID", nodeID, "error", err)
		return domain.WrapContainerOperationFailed("stop container", err)
	}

	s.logger.InfoContext(ctx, "container stopped successfully", "containerID", containerID, "nodeID", nodeID)
	return nil
}

// DeleteContainer deletes a specific container
func (s *systemService) DeleteContainer(ctx context.Context, containerID, nodeID string) error {
	s.logger.InfoContext(ctx, "deleting container", "containerID", containerID, "nodeID", nodeID)

	// Validate container ID
	if err := validation.ValidateContainerID(containerID); err != nil {
		s.logger.WarnContext(ctx, "invalid container ID", "containerID", containerID, "error", err)
		return domain.WrapValidationError("container ID", err)
	}

	if err := s.dockerManager.DeleteContainer(containerID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete container", "containerID", containerID, "nodeID", nodeID, "error", err)
		return domain.WrapContainerOperationFailed("delete container", err)
	}

	s.logger.InfoContext(ctx, "container deleted successfully", "containerID", containerID, "nodeID", nodeID)
	return nil
}
