package update

import (
	"bytes"
	"context"
	"os/exec"
)

// Docker runs the docker CLI. Stdout and stderr stay apart because the plan container prints JSON on stdout.
type Docker interface {
	Run(ctx context.Context, args ...string) (stdout, stderr string, err error)
}

// ExecDocker runs the real docker binary.
type ExecDocker struct{}

// Run implements Docker.
func (ExecDocker) Run(ctx context.Context, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	return so.String(), se.String(), err
}
