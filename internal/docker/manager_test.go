package docker

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	appsDir := "/tmp/apps"
	manager := NewManager(appsDir)

	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}

	if manager.appsDir != appsDir {
		t.Errorf("Expected appsDir to be %s, got %s", appsDir, manager.appsDir)
	}
}

func TestCreateAppDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := ioutil.TempDir("", "docker-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Test creating an app directory
	appName := "test-app"
	composeContent := "version: '3.8'\nservices:\n  test:\n    image: nginx:latest"

	err = manager.CreateAppDirectory(appName, composeContent)
	if err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}

	// Check if the directory was created
	appPath := filepath.Join(tmpDir, appName)
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		t.Errorf("Expected app directory to exist at %s", appPath)
	}

	// Check if the docker-compose.yml file was created
	composePath := filepath.Join(appPath, "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Errorf("Expected docker-compose.yml to exist at %s", composePath)
	}

	// Check if the content was written correctly
	content, err := ioutil.ReadFile(composePath)
	if err != nil {
		t.Fatalf("Failed to read compose file: %v", err)
	}

	if string(content) != composeContent {
		t.Errorf("Expected content to be %s, got %s", composeContent, string(content))
	}
}

func TestWriteComposeFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := ioutil.TempDir("", "docker-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Create the app directory first
	appName := "test-app"
	appPath := filepath.Join(tmpDir, appName)
	err = os.MkdirAll(appPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}

	// Test writing compose file
	composeContent := "version: '3.8'\nservices:\n  test:\n    image: nginx:latest"

	err = manager.WriteComposeFile(appName, composeContent)
	if err != nil {
		t.Fatalf("Failed to write compose file: %v", err)
	}

	// Check if the content was written correctly
	composePath := filepath.Join(appPath, "docker-compose.yml")
	content, err := ioutil.ReadFile(composePath)
	if err != nil {
		t.Fatalf("Failed to read compose file: %v", err)
	}

	if string(content) != composeContent {
		t.Errorf("Expected content to be %s, got %s", composeContent, string(content))
	}
}

func TestDeleteAppDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := ioutil.TempDir("", "docker-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	// Create the app directory first
	appName := "test-app"
	appPath := filepath.Join(tmpDir, appName)
	err = os.MkdirAll(appPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}

	// Verify the directory exists
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		t.Errorf("Expected app directory to exist at %s", appPath)
	}

	// Delete the app directory
	err = manager.DeleteAppDirectory(appName)
	if err != nil {
		t.Fatalf("Failed to delete app directory: %v", err)
	}

	// Verify the directory no longer exists
	if _, err := os.Stat(appPath); !os.IsNotExist(err) {
		t.Errorf("Expected app directory to not exist at %s", appPath)
	}
}

// TestStartApp tests the StartApp function with mock command executor
func TestStartApp(t *testing.T) {
	tmpDir := t.TempDir()
	mockExecutor := NewMockCommandExecutor()
	manager := NewManagerWithExecutor(tmpDir, mockExecutor)

	appName := "test-app"
	appPath := filepath.Join(tmpDir, appName)

	// Create app directory (required for StartApp to execute)
	if err := os.MkdirAll(appPath, 0755); err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}

	// Mock successful command execution (actual command includes -f docker-compose.yml)
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d"}, []byte("success"))

	err := manager.StartApp(appName)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify command was executed correctly
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d"}) {
		t.Error("Expected docker compose up command to be executed")
	}

	// Verify it was executed in the correct directory
	commands := mockExecutor.GetExecutedCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command to be executed, got %d", len(commands))
	}

	if commands[0].Dir != appPath {
		t.Errorf("Expected command to be executed in %s, got %s", appPath, commands[0].Dir)
	}
}

// TestStartAppWithError tests the StartApp function with mock error
func TestStartAppWithError(t *testing.T) {
	tmpDir := t.TempDir()
	mockExecutor := NewMockCommandExecutor()
	manager := NewManagerWithExecutor(tmpDir, mockExecutor)

	appName := "test-app"
	appPath := filepath.Join(tmpDir, appName)

	// Create app directory (required for StartApp to execute)
	if err := os.MkdirAll(appPath, 0755); err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}

	// Mock failed command execution (actual command includes -f docker-compose.yml)
	mockExecutor.SetMockError("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d"},
		fmt.Errorf("docker command failed"))

	err := manager.StartApp(appName)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d"}) {
		t.Error("Expected docker compose up command to be executed")
	}
}

// TestStartAppWithMissingDirectory tests that StartApp fails clearly when directory is missing
func TestStartAppWithMissingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithExecutor(tmpDir, NewMockCommandExecutor())
	appName := "nonexistent-app"

	// Don't create directory - test missing directory scenario
	err := manager.StartApp(appName)
	if err == nil {
		t.Error("Expected error for missing directory, got nil")
	}

	if !strings.Contains(err.Error(), "directory not found") {
		t.Errorf("Expected 'directory not found' error, got: %v", err)
	}
}

// TestStopApp tests the StopApp function with mock command executor
func TestStopApp(t *testing.T) {
	tmpDir := t.TempDir()
	mockExecutor := NewMockCommandExecutor()
	manager := NewManagerWithExecutor(tmpDir, mockExecutor)

	appName := "test-app"
	appPath := filepath.Join(tmpDir, appName)

	// Create app directory (required for StopApp to execute)
	if err := os.MkdirAll(appPath, 0755); err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}

	// Mock successful command execution (check actual command from ComposeDownCommand)
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "down"}, []byte("success"))

	err := manager.StopApp(appName)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify command was executed correctly
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "down"}) {
		t.Error("Expected docker compose down command to be executed")
	}

	// Verify it was executed in the correct directory
	commands := mockExecutor.GetExecutedCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command to be executed, got %d", len(commands))
	}

	if commands[0].Dir != appPath {
		t.Errorf("Expected command to be executed in %s, got %s", appPath, commands[0].Dir)
	}
}

// TestStopAppWithMissingDirectory tests that StopApp handles missing directories gracefully
func TestStopAppWithMissingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithExecutor(tmpDir, NewMockCommandExecutor())
	appName := "nonexistent-app"

	// Don't create directory - test missing directory scenario
	err := manager.StopApp(appName)
	if err != nil {
		t.Fatalf("Expected no error for missing directory, got %v", err)
	}

	// Verify NO commands were executed (directory doesn't exist)
	executor := manager.GetCommandExecutor().(*MockCommandExecutor)
	if len(executor.GetExecutedCommands()) != 0 {
		t.Errorf("Expected 0 commands to be executed for missing directory, got %d", len(executor.GetExecutedCommands()))
	}
}

// TestUpdateApp tests the UpdateApp function with mock command executor
func TestUpdateApp(t *testing.T) {
	mockExecutor := NewMockCommandExecutor()

	// Create temp directory
	tmpDir, err := ioutil.TempDir("", "docker-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManagerWithExecutor(tmpDir, mockExecutor)

	appName := "test-app"
	appPath := filepath.Join(tmpDir, appName)

	// Create app directory and compose file for the test
	if err := os.MkdirAll(appPath, 0755); err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}
	composePath := filepath.Join(appPath, "docker-compose.yml")
	if err := ioutil.WriteFile(composePath, []byte("version: '3'\nservices:\n  test:\n    image: nginx"), 0644); err != nil {
		t.Fatalf("Failed to create compose file: %v", err)
	}

	// Mock successful command execution with new flags
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "pull", "--ignore-buildable"}, []byte("success"))
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d", "--build"}, []byte("success"))

	err = manager.UpdateApp(appName)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify commands were executed correctly
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "pull", "--ignore-buildable"}) {
		t.Error("Expected docker compose pull --ignore-buildable command to be executed")
	}

	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d", "--build"}) {
		t.Error("Expected docker compose up -d --build command to be executed")
	}

	// Verify they were executed in the correct directory
	commands := mockExecutor.GetExecutedCommands()
	if len(commands) != 2 {
		t.Errorf("Expected 2 commands to be executed, got %d", len(commands))
	}

	for _, cmd := range commands {
		if cmd.Dir != appPath {
			t.Errorf("Expected command to be executed in %s, got %s", appPath, cmd.Dir)
		}
	}
}

// TestUpdateAppWithPullFailure tests UpdateApp when pull fails but build succeeds
func TestUpdateAppWithPullFailure(t *testing.T) {
	mockExecutor := NewMockCommandExecutor()

	// Create temp directory
	tmpDir, err := ioutil.TempDir("", "docker-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManagerWithExecutor(tmpDir, mockExecutor)

	appName := "test-app-with-build"
	appPath := filepath.Join(tmpDir, appName)

	// Create app directory and compose file for the test
	if err := os.MkdirAll(appPath, 0755); err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}
	composePath := filepath.Join(appPath, "docker-compose.yml")
	if err := ioutil.WriteFile(composePath, []byte("version: '3'\nservices:\n  test:\n    build: ."), 0644); err != nil {
		t.Fatalf("Failed to create compose file: %v", err)
	}

	// Mock pull failure (e.g., all services use build:) but successful build
	mockExecutor.SetMockError("docker", []string{"compose", "-f", "docker-compose.yml", "pull", "--ignore-buildable"},
		fmt.Errorf("no services to pull"))
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d", "--build"}, []byte("success"))

	// Update should succeed despite pull failure
	err = manager.UpdateApp(appName)
	if err != nil {
		t.Errorf("Expected no error when pull fails but build succeeds, got %v", err)
	}

	// Verify both commands were executed
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "pull", "--ignore-buildable"}) {
		t.Error("Expected docker compose pull --ignore-buildable command to be executed")
	}

	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d", "--build"}) {
		t.Error("Expected docker compose up -d --build command to be executed")
	}

	// Verify they were executed in the correct directory
	commands := mockExecutor.GetExecutedCommands()
	if len(commands) != 2 {
		t.Errorf("Expected 2 commands to be executed, got %d", len(commands))
	}

	for _, cmd := range commands {
		if cmd.Dir != appPath {
			t.Errorf("Expected command to be executed in %s, got %s", appPath, cmd.Dir)
		}
	}
}

// TestGetAppStatus tests the GetAppStatus function with mock command executor
func TestGetAppStatus(t *testing.T) {
	mockExecutor := NewMockCommandExecutor()
	appsDir := "/tmp/apps"
	manager := NewManagerWithExecutor(appsDir, mockExecutor)

	appName := "test-app"
	appPath := filepath.Join(appsDir, appName)

	// Mock successful command execution with output
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "ps"},
		[]byte("NAME      COMMAND   SERVICE   STATUS   PORTS\napp_1    nginx    nginx     running   0.0.0.0:80->80/tcp"))

	status, err := manager.GetAppStatus(appName)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if status != "running" {
		t.Errorf("Expected status 'running', got '%s'", status)
	}

	// Verify command was executed correctly
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "ps"}) {
		t.Error("Expected docker compose ps command to be executed")
	}

	// Verify it was executed in the correct directory
	commands := mockExecutor.GetExecutedCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command to be executed, got %d", len(commands))
	}

	if commands[0].Dir != appPath {
		t.Errorf("Expected command to be executed in %s, got %s", appPath, commands[0].Dir)
	}
}

// TestGetAppStatusEmpty tests the GetAppStatus function with empty output
func TestGetAppStatusEmpty(t *testing.T) {
	mockExecutor := NewMockCommandExecutor()
	appsDir := "/tmp/apps"
	manager := NewManagerWithExecutor(appsDir, mockExecutor)

	appName := "test-app"

	// Mock successful command execution with empty output
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "ps"}, []byte(""))

	status, err := manager.GetAppStatus(appName)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if status != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", status)
	}
}

// TestGetAppLogs tests the GetAppLogs function with mock command executor
func TestGetAppLogs(t *testing.T) {
	mockExecutor := NewMockCommandExecutor()
	appsDir := "/tmp/apps"
	manager := NewManagerWithExecutor(appsDir, mockExecutor)

	appName := "test-app"
	appPath := filepath.Join(appsDir, appName)

	// Mock successful command execution with log output
	logOutput := []byte("First log\nSecond log\nThird log")
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "logs", "--tail=100"},
		logOutput)

	logs, err := manager.GetAppLogs(appName, "")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedLogs := "Third log\nSecond log\nFirst log"
	if string(logs) != expectedLogs {
		t.Errorf("Expected logs '%s', got '%s'", expectedLogs, string(logs))
	}

	// Verify command was executed correctly
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "logs", "--tail=100"}) {
		t.Error("Expected docker compose logs command to be executed")
	}

	// Verify it was executed in the correct directory
	commands := mockExecutor.GetExecutedCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command to be executed, got %d", len(commands))
	}

	if commands[0].Dir != appPath {
		t.Errorf("Expected command to be executed in %s, got %s", appPath, commands[0].Dir)
	}
}

// TestRestartCloudflared tests the RestartCloudflared function with mock command executor
func TestRestartCloudflared(t *testing.T) {
	mockExecutor := NewMockCommandExecutor()
	appsDir := "/tmp/apps"
	manager := NewManagerWithExecutor(appsDir, mockExecutor)

	appName := "test-app"
	appPath := filepath.Join(appsDir, appName)

	// Mock successful command execution
	mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "restart", "cloudflared"},
		[]byte("success"))

	err := manager.RestartCloudflared(appName)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify command was executed correctly
	if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "restart", "cloudflared"}) {
		t.Error("Expected docker compose restart cloudflared command to be executed")
	}

	// Verify it was executed in the correct directory
	commands := mockExecutor.GetExecutedCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command to be executed, got %d", len(commands))
	}

	if commands[0].Dir != appPath {
		t.Errorf("Expected command to be executed in %s, got %s", appPath, commands[0].Dir)
	}
}
