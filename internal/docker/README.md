# Docker Package

This package contains functionality for managing Docker Compose applications and services.

## Components

### Manager (`manager.go`)

The `Manager` struct handles Docker operations for applications:

- Creating and managing app directories
- Writing Docker Compose files
- Starting, stopping, and updating applications
- Getting application status and logs
- Managing Cloudflare tunnel services

### Command Execution Abstraction

For testability and production-grade architecture, the package uses a command execution abstraction:

- `CommandExecutor` interface defines methods for executing commands
- `RealCommandExecutor` implements the interface for production use
- `MockCommandExecutor` implements the interface for testing

This abstraction allows:

1. **Testability**: Tests can use mock implementations without executing real Docker commands
2. **Flexibility**: Different implementations can be used in different environments
3. **Consistency**: All command execution follows the same pattern

### Compose (`compose.go`)

This file contains structures and functions for working with Docker Compose files:

- `ComposeFile`, `Service`, `Network`, and `Volume` structs representing Docker Compose elements
- Parsing and validating Docker Compose YAML files
- Injecting Cloudflare tunnel services into compose files
- Extracting network information from compose files
- Merging compose files

## Test Coverage

The package includes comprehensive unit tests:

### Manager Tests (`manager_test.go`)

- `TestNewManager`: Tests the creation of a new Manager instance
- `TestCreateAppDirectory`: Tests creating an app directory with compose file
- `TestWriteComposeFile`: Tests writing compose file content to an app directory
- `TestDeleteAppDirectory`: Tests deletion of app directories
- `TestStartApp`: Tests starting an app with mock command execution
- `TestStartAppWithError`: Tests error handling when starting an app
- `TestStopApp`: Tests stopping an app with mock command execution
- `TestUpdateApp`: Tests updating an app with mock command execution
- `TestGetAppStatus`: Tests getting app status with mock command execution
- `TestGetAppStatusEmpty`: Tests getting app status with empty output
- `TestGetAppLogs`: Tests getting app logs with mock command execution
- `TestRestartCloudflared`: Tests restarting cloudflared with mock command execution

### Command Executor Tests (`command_executor_test.go`)

- Mock implementation of CommandExecutor interface
- Helper methods for setting up mock responses
- Verification methods to check command execution

### Compose Tests (`compose_test.go`)

- `TestParseCompose`: Tests parsing valid compose files
- `TestParseComposeNoServices`: Tests handling compose files with no services
- `TestParseComposeInvalidYAML`: Tests handling invalid YAML
- `TestInjectCloudflared`: Tests injecting Cloudflare service with existing network
- `TestInjectCloudflaredNoNetwork`: Tests injecting Cloudflare service with no existing network
- `TestExtractNetworks`: Tests extracting network names from compose files
- `TestMarshalComposeFile`: Tests marshaling compose files to YAML
- `TestMergeServices`: Tests merging compose files
- `TestUniqueStrings`: Tests the unique string utility function

## Running Tests

To run all tests for the docker package:

```bash
go test ./internal/docker -v
```

To run with coverage:

```bash
go test ./internal/docker -cover
```

To run a specific test:

```bash
go test ./internal/docker -run TestName
```

## Production-Grade Features

1. **Dependency Injection**: The Manager can be created with a custom CommandExecutor for testing
2. **Error Handling**: All functions properly handle and report errors
3. **Comprehensive Logging**: Operations are logged with appropriate detail
4. **Resource Management**: Proper cleanup of resources after operations

## Usage Example

### Production Code

```go
manager := docker.NewManager("/apps")
err := manager.StartApp("my-app")
if err != nil {
    log.Printf("Failed to start app: %v", err)
}
```

### Test Code

```go
mockExecutor := docker.NewMockCommandExecutor()
manager := docker.NewManagerWithExecutor("/apps", mockExecutor)

// Set up mock response
mockExecutor.SetMockOutput("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d"}, 
    []byte("success"))

err := manager.StartApp("my-app")

// Verify command was executed
if !mockExecutor.AssertCommandExecuted("docker", []string{"compose", "-f", "docker-compose.yml", "up", "-d"}) {
    t.Error("Expected docker compose up command to be executed")
}
```

## Future Improvements

1. Add integration tests that work with real Docker containers
2. Implement command timeouts for long-running operations
3. Add support for streaming command output
4. Implement retry logic for transient failures
5. Add metrics collection for command execution performance
