package docker

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Manager handles Docker operations
type Manager struct {
	appsDir         string
	commandExecutor CommandExecutor
}

// NewManager creates a new Docker manager with default command executor
func NewManager(appsDir string) *Manager {
	return &Manager{
		appsDir:         appsDir,
		commandExecutor: NewRealCommandExecutor(),
	}
}

// NewManagerWithExecutor creates a new Docker manager with a custom command executor (for testing)
func NewManagerWithExecutor(appsDir string, executor CommandExecutor) *Manager {
	return &Manager{
		appsDir:         appsDir,
		commandExecutor: executor,
	}
}

// GetCommandExecutor returns the command executor (for debugging purposes)
func (m *Manager) GetCommandExecutor() CommandExecutor {
	return m.commandExecutor
}

// CreateAppDirectory creates an app directory and writes compose file
func (m *Manager) CreateAppDirectory(name, composeContent string) error {
	appPath := filepath.Join(m.appsDir, name)
	composePath := filepath.Join(appPath, "docker-compose.yml")

	slog.Info("creating app directory", "app", name, "appPath", appPath, "composePath", composePath)

	// Create app directory
	if err := os.MkdirAll(appPath, 0755); err != nil {
		slog.Error("failed to create app directory", "app", name, "appPath", appPath, "error", err)
		return fmt.Errorf("failed to create app directory: %w", err)
	}

	// Write docker-compose.yml
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		slog.Error("failed to write compose file", "app", name, "composePath", composePath, "error", err)
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	slog.Info("app directory created successfully", "app", name, "appPath", appPath, "composeSize", len(composeContent))
	return nil
}

// WriteComposeFile writes the compose file content to the app directory
func (m *Manager) WriteComposeFile(name, content string) error {
	composePath := filepath.Join(m.appsDir, name, "docker-compose.yml")
	
	slog.Info("writing compose file", "app", name, "composePath", composePath, "composeSize", len(content))
	
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		slog.Error("failed to write compose file", "app", name, "composePath", composePath, "error", err)
		return fmt.Errorf("failed to write compose file: %w", err)
	}
	
	slog.Info("compose file written successfully", "app", name, "composePath", composePath)
	return nil
}

// StartApp starts the app using docker compose
func (m *Manager) StartApp(name string) error {
	appPath := filepath.Join(m.appsDir, name)
	
	slog.Info("starting app", "app", name, "appPath", appPath, "command", "docker compose up -d")
	
	output, err := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", "docker-compose.yml", "up", "-d")
	if err != nil {
		slog.Error("failed to start app", "app", name, "error", err, "output", string(output))
		return fmt.Errorf("failed to start app: %w\nOutput: %s", err, string(output))
	}
	
	slog.Info("app started successfully", "app", name, "output", string(output))
	return nil
}

// StopApp stops the app using docker compose
func (m *Manager) StopApp(name string) error {
	appPath := filepath.Join(m.appsDir, name)
	
	slog.Info("stopping app", "app", name, "appPath", appPath, "command", "docker compose down")
	
	output, err := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", "docker-compose.yml", "down")
	if err != nil {
		slog.Error("failed to stop app", "app", name, "error", err, "output", string(output))
		return fmt.Errorf("failed to stop app: %w\nOutput: %s", err, string(output))
	}
	
	slog.Info("app stopped successfully", "app", name, "output", string(output))
	return nil
}

// UpdateApp performs zero-downtime update
func (m *Manager) UpdateApp(name string) error {
	appPath := filepath.Join(m.appsDir, name)
	composeFile := "docker-compose.yml"
	composePath := filepath.Join(appPath, composeFile)

	slog.Info("starting app update", "app", name, "appPath", appPath, "composeFile", composePath)

	// Verify compose file exists
	if _, err := os.Stat(composePath); err != nil {
		slog.Error("compose file not found", "app", name, "composePath", composePath, "error", err)
		return fmt.Errorf("compose file not found at %s: %w", composePath, err)
	}

	// Step 1: Pull latest images (ignoring services with build configurations)
	slog.Info("pulling latest images", "app", name, "command", "docker compose pull --ignore-buildable")
	pullOutput, pullErr := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", composeFile, "pull", "--ignore-buildable")
	if pullErr != nil {
		// If pull fails (e.g., older docker compose version, or all services use build),
		// log but continue - the 'up' command will handle building if needed
		slog.Warn("failed to pull images, continuing with update", 
			"app", name, 
			"error", pullErr, 
			"output", string(pullOutput),
			"note", "this is expected for services using 'build:' directives or older docker compose versions")
	} else {
		slog.Info("images pulled successfully", "app", name, "output", string(pullOutput))
	}

	// Step 2: Update app services with --build flag
	slog.Info("updating app services", "app", name, "command", "docker compose up -d --build")
	upOutput, upErr := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", composeFile, "up", "-d", "--build")
	if upErr != nil {
		slog.Error("failed to update app services", 
			"app", name, 
			"error", upErr, 
			"output", string(upOutput),
			"exitCode", upErr.Error())
		return fmt.Errorf("failed to update app: %w\nCommand: docker compose -f %s up -d --build\nOutput: %s", upErr, composeFile, string(upOutput))
	}

	slog.Info("app updated successfully", "app", name, "output", string(upOutput))
	return nil
}

// GetAppStatus checks the status of app containers
func (m *Manager) GetAppStatus(name string) (string, error) {
	appPath := filepath.Join(m.appsDir, name)
	
	slog.Debug("getting app status", "app", name, "appPath", appPath)
	
	output, err := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", "docker-compose.yml", "ps")
	if err != nil {
		slog.Error("failed to get app status", "app", name, "error", err, "output", string(output))
		return "unknown", fmt.Errorf("failed to get status: %w\nOutput: %s", err, string(output))
	}

	// Simple status detection (in production, parse the output properly)
	statusStr := string(output)
	if len(statusStr) > 0 {
		slog.Debug("app status retrieved", "app", name, "status", "running")
		return "running", nil
	}
	slog.Debug("app status retrieved", "app", name, "status", "stopped")
	return "stopped", nil
}

// GetAppLogs fetches logs from the app
func (m *Manager) GetAppLogs(name string) ([]byte, error) {
	appPath := filepath.Join(m.appsDir, name)
	
	slog.Debug("fetching app logs", "app", name, "appPath", appPath, "command", "docker compose logs --tail=100")
	
	output, err := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", "docker-compose.yml", "logs", "--tail=100")
	if err != nil {
		slog.Error("failed to get app logs", "app", name, "error", err, "output", string(output))
		return nil, fmt.Errorf("failed to get logs: %w\nOutput: %s", err, string(output))
	}

	// Reverse the logs so latest appears first
	logsStr := string(output)
	lines := strings.Split(logsStr, "\n")

	// Remove empty lines that might result from splitting
	var nonEmptyLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	// Reverse the order of lines
	for i, j := 0, len(nonEmptyLines)-1; i < j; i, j = i+1, j-1 {
		nonEmptyLines[i], nonEmptyLines[j] = nonEmptyLines[j], nonEmptyLines[i]
	}

	slog.Debug("app logs retrieved", "app", name, "lineCount", len(nonEmptyLines))
	return []byte(strings.Join(nonEmptyLines, "\n")), nil
}

// DeleteAppDirectory removes the app directory
func (m *Manager) DeleteAppDirectory(name string) error {
	appPath := filepath.Join(m.appsDir, name)
	
	slog.Info("deleting app directory", "app", name, "appPath", appPath)
	
	if err := os.RemoveAll(appPath); err != nil {
		slog.Error("failed to delete app directory", "app", name, "appPath", appPath, "error", err)
		return fmt.Errorf("failed to delete app directory: %w", err)
	}
	
	slog.Info("app directory deleted successfully", "app", name, "appPath", appPath)
	return nil
}

// RestartCloudflared restarts the cloudflared service to pick up new ingress configuration
func (m *Manager) RestartCloudflared(name string) error {
	appPath := filepath.Join(m.appsDir, name)
	composeFile := "docker-compose.yml"

	slog.Info("restarting cloudflared service", "app", name, "appPath", appPath, "command", "docker compose restart cloudflared")

	output, err := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", composeFile, "restart", "cloudflared")
	if err != nil {
		slog.Error("failed to restart cloudflared", "app", name, "error", err, "output", string(output))
		return fmt.Errorf("failed to restart cloudflared: %w\nOutput: %s", err, string(output))
	}

	slog.Info("cloudflared service restarted successfully", "app", name, "output", string(output))
	return nil
}

// RestartContainer restarts a specific container by ID
func (m *Manager) RestartContainer(containerID string) error {
	slog.Info("restarting container", "containerID", containerID)

	output, err := m.commandExecutor.ExecuteCommand("docker", "restart", containerID)
	if err != nil {
		slog.Error("failed to restart container", "containerID", containerID, "error", err, "output", string(output))
		return fmt.Errorf("failed to restart container: %w\nOutput: %s", err, string(output))
	}

	slog.Info("container restarted successfully", "containerID", containerID, "output", string(output))
	return nil
}

// StopContainer stops a specific container by ID
func (m *Manager) StopContainer(containerID string) error {
	slog.Info("stopping container", "containerID", containerID)

	output, err := m.commandExecutor.ExecuteCommand("docker", "stop", containerID)
	if err != nil {
		slog.Error("failed to stop container", "containerID", containerID, "error", err, "output", string(output))
		return fmt.Errorf("failed to stop container: %w\nOutput: %s", err, string(output))
	}

	slog.Info("container stopped successfully", "containerID", containerID, "output", string(output))
	return nil
}

// DeleteContainer removes a container by ID (force remove)
func (m *Manager) DeleteContainer(containerID string) error {
	slog.Info("deleting container", "containerID", containerID)

	output, err := m.commandExecutor.ExecuteCommand("docker", "rm", "-f", containerID)
	if err != nil {
		slog.Error("failed to delete container", "containerID", containerID, "error", err, "output", string(output))
		return fmt.Errorf("failed to delete container: %w\nOutput: %s", err, string(output))
	}

	slog.Info("container deleted successfully", "containerID", containerID, "output", string(output))
	return nil
}
