package docker

import (
	"os/exec"
)

// CommandExecutor defines the interface for executing system commands
type CommandExecutor interface {
	// ExecuteCommand executes a command and returns the combined output
	ExecuteCommand(name string, args ...string) ([]byte, error)
	
	// ExecuteCommandInDir executes a command in a specific directory
	ExecuteCommandInDir(dir, name string, args ...string) ([]byte, error)
}

// RealCommandExecutor is the production implementation that actually executes commands
type RealCommandExecutor struct{}

// NewRealCommandExecutor creates a new real command executor
func NewRealCommandExecutor() *RealCommandExecutor {
	return &RealCommandExecutor{}
}

// ExecuteCommand executes a command and returns the combined output
func (r *RealCommandExecutor) ExecuteCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// ExecuteCommandInDir executes a command in a specific directory
func (r *RealCommandExecutor) ExecuteCommandInDir(dir, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}
