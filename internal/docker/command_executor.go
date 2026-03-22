package docker

import (
	"bufio"
	"context"
	"io"
	"os/exec"

	"golang.org/x/sync/errgroup"
)

// CommandExecutor defines the interface for executing system commands
type CommandExecutor interface {
	// ExecuteCommand executes a command and returns the combined output
	ExecuteCommand(name string, args ...string) ([]byte, error)

	// ExecuteCommandInDir executes a command in a specific directory
	ExecuteCommandInDir(dir, name string, args ...string) ([]byte, error)

	// ExecuteCommandInDirStream runs a command in dir and invokes onLine for each line of stdout and stderr.
	// onLine may be nil. Lines are split on '\n' (carriage returns are stripped).
	ExecuteCommandInDirStream(ctx context.Context, dir, name string, args []string, onLine func(string)) error
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

// ExecuteCommandInDirStream implements CommandExecutor.
func (r *RealCommandExecutor) ExecuteCommandInDirStream(ctx context.Context, dir, name string, args []string, onLine func(string)) error {
	if onLine == nil {
		_, err := r.ExecuteCommandInDir(dir, name, args...)
		return err
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	streamPipe := func(r io.Reader) error {
		sc := bufio.NewScanner(r)
		buf := make([]byte, 0, 64*1024)
		sc.Buffer(buf, 1024*1024)
		for sc.Scan() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			onLine(sc.Text())
		}
		return sc.Err()
	}

	var eg errgroup.Group
	eg.Go(func() error { return streamPipe(stdout) })
	eg.Go(func() error { return streamPipe(stderr) })

	pipeErr := eg.Wait()
	waitErr := cmd.Wait()
	if pipeErr != nil {
		return pipeErr
	}
	return waitErr
}
