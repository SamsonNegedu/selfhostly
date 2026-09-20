package ctl

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// compose runs `docker compose` for one project, env file and set of compose files
type compose struct {
	a       *App
	project string
	env     string
	files   []string
}

func (c compose) args(extra ...string) []string {
	args := []string{"compose"}
	if c.project != "" {
		args = append(args, "-p", c.project)
	}
	if c.env != "" && (EnvFile{c.env}).Exists() {
		args = append(args, "--env-file", c.env)
	}
	for _, f := range c.files {
		args = append(args, "-f", f)
	}
	return append(args, extra...)
}

func (c compose) out(ctx context.Context, extra ...string) (string, string, error) {
	return c.a.Run.Output(ctx, "docker", c.args(extra...)...)
}

// outEnv is out with extra environment variables, which take precedence over the settings file the way
// they do for docker compose itself
func (c compose) outEnv(ctx context.Context, env map[string]string, extra ...string) (string, string, error) {
	return c.a.Run.OutputEnv(ctx, env, "docker", c.args(extra...)...)
}

func (c compose) pass(ctx context.Context, extra ...string) error {
	return c.a.Run.Passthrough(ctx, "docker", c.args(extra...)...)
}

// cid is the container id of a service, or "" when it is not running
func (c compose) cid(ctx context.Context, service string) string {
	out, _, err := c.out(ctx, "ps", "-q", service)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(out, "\n", 2)[0])
}

// health is the container's health, or its state when it has no health check, or "missing"
func (a *App) health(ctx context.Context, container string) string {
	out, _, err := a.Run.Output(ctx, "docker", "inspect", "--format",
		"{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}", container)
	if err != nil {
		return "missing"
	}
	return strings.TrimSpace(out)
}

// waitHealthy polls until the container is healthy or the time is up. dots prints progress.
func (a *App) waitHealthy(ctx context.Context, container string, timeout, every time.Duration, dots bool) bool {
	for waited := time.Duration(0); waited < timeout; waited += every {
		if a.health(ctx, container) == "healthy" {
			return true
		}
		if dots {
			fmt.Fprint(a.Out, ".")
		}
		a.sleep(every)
	}
	return false
}

// logs is everything a container has written, standard output and error together
func (a *App) logs(ctx context.Context, container string, tail int) string {
	args := []string{"logs"}
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprint(tail))
	}
	out, errOut, _ := a.Run.Output(ctx, "docker", append(args, container)...)
	return out + errOut
}

func (a *App) primaryCompose() compose {
	return compose{a: a, env: a.envPath(a.EnvFile), files: []string{a.dirPath(primaryCompose)}}
}

func (a *App) nodeCompose(direct bool) compose {
	files := []string{a.dirPath(nodeCompose)}
	if direct {
		files = append(files, a.dirPath(nodeDirectExtra))
	}
	for _, f := range strings.Fields(a.NodeExtraCompose) {
		files = append(files, f)
	}
	return compose{a: a, project: nodeProject, env: a.dirPath(nodeEnvFile), files: files}
}
