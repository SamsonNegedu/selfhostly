package docker

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"

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

// platformSecretEnv lists variables that hold this platform's own credentials. docker compose
// interpolates ${VAR} from the environment of the process that runs it, so leaving these set would
// let any compose file read them (for example `environment: X: ${JWT_SECRET}`).
var platformSecretEnv = map[string]bool{
	"JWT_SECRET":              true,
	"GITHUB_CLIENT_ID":        true,
	"GITHUB_CLIENT_SECRET":    true,
	"CLOUDFLARE_API_TOKEN":    true,
	"CLOUDFLARE_ACCOUNT_ID":   true,
	"GATEWAY_API_KEY":         true,
	"NODE_API_KEY":            true,
	"PRIMARY_NODE_API_KEY":    true,
	"REGISTRATION_TOKEN":      true,
	"SETTINGS_ENCRYPTION_KEY": true,
	"CF_ACCESS_AUD":           true,
	"TUNNEL_TOKEN":            true,
}

// commandEnv returns the current environment without the platform's own secrets
func commandEnv() []string {
	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			name = kv[:i]
		}
		if platformSecretEnv[name] {
			continue
		}
		out = append(out, kv)
	}
	return out
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
	cmd.Env = commandEnv()
	return cmd.CombinedOutput()
}

// ExecuteCommandInDir executes a command in a specific directory
func (r *RealCommandExecutor) ExecuteCommandInDir(dir, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = commandEnv()
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
	cmd.Env = commandEnv()

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
