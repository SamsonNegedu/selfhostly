package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/system"
	"github.com/selfhostly/internal/validation"
)

// systemService implements the SystemService interface
type systemService struct {
	database      *db.DB
	dockerManager *docker.Manager
	collector     *system.Collector
	logger        *slog.Logger
}

// NewSystemService creates a new system service
func NewSystemService(
	database *db.DB,
	dockerManager *docker.Manager,
	cfg *config.Config,
	logger *slog.Logger,
) domain.SystemService {
	collector := system.NewCollector(cfg.AppsDir, dockerManager, database)
	return &systemService{
		database:      database,
		dockerManager: dockerManager,
		collector:     collector,
		logger:        logger,
	}
}

// GetSystemStats retrieves system-wide statistics
func (s *systemService) GetSystemStats(ctx context.Context) (*system.SystemStats, error) {
	s.logger.DebugContext(ctx, "getting system stats")

	sysStats, err := s.collector.GetSystemStats()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get system stats", "error", err)
		return nil, domain.WrapContainerOperationFailed("get system stats", err)
	}

	s.logger.DebugContext(ctx, "system stats retrieved successfully",
		"total_containers", sysStats.Docker.TotalContainers,
		"running", sysStats.Docker.Running,
		"cpu_usage", sysStats.CPU.UsagePercent)

	return sysStats, nil
}

// GetAppStats retrieves resource statistics for a specific app
func (s *systemService) GetAppStats(ctx context.Context, appID string) (*domain.AppStats, error) {
	s.logger.DebugContext(ctx, "getting app stats", "appID", appID)

	app, err := s.database.GetApp(appID)
	if err != nil {
		s.logger.DebugContext(ctx, "app not found for stats", "appID", appID)
		return nil, domain.WrapAppNotFound(appID, err)
	}

	// Only fetch stats if app is running
	if app.Status != "running" {
		// Return empty stats for non-running apps
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
		s.logger.ErrorContext(ctx, "failed to get app stats", "app", app.Name, "error", err)
		return nil, domain.WrapContainerOperationFailed("get app stats", err)
	}

	// Convert docker.AppStats to domain.AppStats
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

	domainStats := &domain.AppStats{
		AppName:          dockerStats.AppName,
		TotalCPUPercent:  dockerStats.TotalCPU,
		TotalMemoryBytes: int64(dockerStats.TotalMemory),
		MemoryLimitBytes: int64(dockerStats.MemoryLimit),
		Containers:       containers,
		Timestamp:        dockerStats.Timestamp,
		Status:           app.Status,
		Message:          "",
	}

	return domainStats, nil
}

// GetAppLogs retrieves logs for a specific app
func (s *systemService) GetAppLogs(ctx context.Context, appID string) ([]byte, error) {
	s.logger.DebugContext(ctx, "getting app logs", "appID", appID)

	app, err := s.database.GetApp(appID)
	if err != nil {
		s.logger.DebugContext(ctx, "app not found for logs", "appID", appID)
		return nil, domain.WrapAppNotFound(appID, err)
	}

	logs, err := s.dockerManager.GetAppLogs(app.Name)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get app logs", "app", app.Name, "error", err)
		return nil, domain.WrapContainerOperationFailed("get app logs", err)
	}

	return logs, nil
}

// RestartContainer restarts a specific container
func (s *systemService) RestartContainer(ctx context.Context, containerID string) error {
	s.logger.InfoContext(ctx, "restarting container", "containerID", containerID)

	// Validate container ID
	if err := validation.ValidateContainerID(containerID); err != nil {
		s.logger.WarnContext(ctx, "invalid container ID", "containerID", containerID, "error", err)
		return domain.WrapValidationError("container ID", err)
	}

	if err := s.dockerManager.RestartContainer(containerID); err != nil {
		s.logger.ErrorContext(ctx, "failed to restart container", "containerID", containerID, "error", err)
		return domain.WrapContainerOperationFailed("restart container", err)
	}

	s.logger.InfoContext(ctx, "container restarted successfully", "containerID", containerID)
	return nil
}

// StopContainer stops a specific container
func (s *systemService) StopContainer(ctx context.Context, containerID string) error {
	s.logger.InfoContext(ctx, "stopping container", "containerID", containerID)

	// Validate container ID
	if err := validation.ValidateContainerID(containerID); err != nil {
		s.logger.WarnContext(ctx, "invalid container ID", "containerID", containerID, "error", err)
		return domain.WrapValidationError("container ID", err)
	}

	if err := s.dockerManager.StopContainer(containerID); err != nil {
		s.logger.ErrorContext(ctx, "failed to stop container", "containerID", containerID, "error", err)
		return domain.WrapContainerOperationFailed("stop container", err)
	}

	s.logger.InfoContext(ctx, "container stopped successfully", "containerID", containerID)
	return nil
}

// DeleteContainer deletes a specific container
func (s *systemService) DeleteContainer(ctx context.Context, containerID string) error {
	s.logger.InfoContext(ctx, "deleting container", "containerID", containerID)

	// Validate container ID
	if err := validation.ValidateContainerID(containerID); err != nil {
		s.logger.WarnContext(ctx, "invalid container ID", "containerID", containerID, "error", err)
		return domain.WrapValidationError("container ID", err)
	}

	if err := s.dockerManager.DeleteContainer(containerID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete container", "containerID", containerID, "error", err)
		return domain.WrapContainerOperationFailed("delete container", err)
	}

	s.logger.InfoContext(ctx, "container deleted successfully", "containerID", containerID)
	return nil
}
