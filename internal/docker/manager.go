package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Manager handles Docker operations
type Manager struct {
	appsDir        string
	commandExecutor CommandExecutor
}

// NewManager creates a new Docker manager with default command executor
func NewManager(appsDir string) *Manager {
	return &Manager{
		appsDir:        appsDir,
		commandExecutor: NewRealCommandExecutor(),
	}
}

// NewManagerWithExecutor creates a new Docker manager with a custom command executor (for testing)
func NewManagerWithExecutor(appsDir string, executor CommandExecutor) *Manager {
	return &Manager{
		appsDir:        appsDir,
		commandExecutor: executor,
	}
}

// CreateAppDirectory creates an app directory and writes compose file
func (m *Manager) CreateAppDirectory(name, composeContent string) error {
	appPath := filepath.Join(m.appsDir, name)

	// Create app directory
	if err := os.MkdirAll(appPath, 0755); err != nil {
		return fmt.Errorf("failed to create app directory: %w", err)
	}

	// Write docker-compose.yml
	composePath := filepath.Join(appPath, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	return nil
}

// WriteComposeFile writes the compose file content to the app directory
func (m *Manager) WriteComposeFile(name, content string) error {
	composePath := filepath.Join(m.appsDir, name, "docker-compose.yml")
	return os.WriteFile(composePath, []byte(content), 0644)
}

// StartApp starts the app using docker compose
func (m *Manager) StartApp(name string) error {
	appPath := filepath.Join(m.appsDir, name)
	output, err := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", "docker-compose.yml", "up", "-d")
	if err != nil {
		return fmt.Errorf("failed to start app: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// StopApp stops the app using docker compose
func (m *Manager) StopApp(name string) error {
	appPath := filepath.Join(m.appsDir, name)
	output, err := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", "docker-compose.yml", "down")
	if err != nil {
		return fmt.Errorf("failed to stop app: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// UpdateApp performs zero-downtime update
func (m *Manager) UpdateApp(name string) error {
	appPath := filepath.Join(m.appsDir, name)
	// Use just the filename since cmd.Dir is set to the app directory
	composeFile := "docker-compose.yml"

	// Pull latest images
	_, err := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", composeFile, "pull")
	if err != nil {
		return fmt.Errorf("failed to pull images: %w", err)
	}

	// Update app services (cloudflared stays running)
	_, err = m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", composeFile, "up", "-d")
	if err != nil {
		return fmt.Errorf("failed to update app: %w", err)
	}

	return nil
}

// GetAppStatus checks the status of app containers
func (m *Manager) GetAppStatus(name string) (string, error) {
	appPath := filepath.Join(m.appsDir, name)
	// Use just the filename since cmd.Dir is set to the app directory
	output, err := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", "docker-compose.yml", "ps")
	if err != nil {
		return "unknown", fmt.Errorf("failed to get status: %w", err)
	}

	// Simple status detection (in production, parse the output properly)
	statusStr := string(output)
	if len(statusStr) > 0 {
		return "running", nil
	}
	return "stopped", nil
}

// GetAppLogs fetches logs from the app
func (m *Manager) GetAppLogs(name string) ([]byte, error) {
	appPath := filepath.Join(m.appsDir, name)
	// Use just the filename since cmd.Dir is set to the app directory
	output, err := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", "docker-compose.yml", "logs", "--tail=100")
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
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

	return []byte(strings.Join(nonEmptyLines, "\n")), nil
}

// DeleteAppDirectory removes the app directory
func (m *Manager) DeleteAppDirectory(name string) error {
	appPath := filepath.Join(m.appsDir, name)
	return os.RemoveAll(appPath)
}

// RestartCloudflared restarts the cloudflared service to pick up new ingress configuration
func (m *Manager) RestartCloudflared(name string) error {
	appPath := filepath.Join(m.appsDir, name)
	composeFile := "docker-compose.yml"

	// Restart only the cloudflared service
	_, err := m.commandExecutor.ExecuteCommandInDir(appPath, "docker", "compose", "-f", composeFile, "restart", "cloudflared")
	if err != nil {
		return fmt.Errorf("failed to restart cloudflared: %w", err)
	}

	return nil
}
