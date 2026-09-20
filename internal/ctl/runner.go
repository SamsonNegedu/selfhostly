package ctl

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

// Runner runs external programs. Commands use it instead of os/exec so tests can fake docker.
type Runner interface {
	// Output runs a command and returns its stdout and stderr separately
	Output(ctx context.Context, name string, args ...string) (stdout, stderr string, err error)
	// OutputEnv is Output with extra environment variables
	OutputEnv(ctx context.Context, env map[string]string, name string, args ...string) (stdout, stderr string, err error)
	// Passthrough runs a command wired to this process's terminal
	Passthrough(ctx context.Context, name string, args ...string) error
}

type execRunner struct{}

func (r execRunner) Output(ctx context.Context, name string, args ...string) (string, string, error) {
	return r.OutputEnv(ctx, nil, name, args...)
}

func (execRunner) OutputEnv(ctx context.Context, env map[string]string, name string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var so, se strings.Builder
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	return so.String(), se.String(), err
}

func (execRunner) Passthrough(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
