package ctl

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
)

// serverBinaries are where the server binary lives in a container: the published image, and the
// output of the dev stack's live-reload build. A container is a Selfhostly backend because one of
// these is in it, not because of what it is called.
var serverBinaries = []string{"./selfhostly", "./tmp/main"}

// probeScript succeeds only inside a Selfhostly backend, and prints the binary and the role
var probeScript = func() string {
	var b strings.Builder
	b.WriteString("for b in")
	for _, bin := range serverBinaries {
		b.WriteString(" " + bin)
	}
	b.WriteString(`; do if test -x "$b"; then echo "$b ${NODE_IS_PRIMARY:-true}"; exit 0; fi; done; exit 1`)
	return b.String()
}()

// backend is a running Selfhostly backend and where its server binary is
type backend struct {
	name    string
	bin     string
	primary bool
}

// backends returns the running containers that are Selfhostly backends, by probing what is in them.
// primaryOnly leaves out secondary nodes. When a container was named, only that one is considered.
func (a *App) backends(ctx context.Context, primaryOnly bool) ([]backend, error) {
	if name := firstNonEmpty(a.Container, os.Getenv("SFH_PRIMARY_CONTAINER")); name != "" {
		if !a.isRunning(ctx, name) {
			return nil, fmt.Errorf("container %q is not running (see: selfhostlyctl status)", name)
		}
		b, err := a.requireBackend(ctx, name)
		if err != nil {
			return nil, err
		}
		return []backend{b}, nil
	}
	var found []backend
	for _, n := range a.psNames(ctx) {
		if b, ok := a.probe(ctx, n); ok && (b.primary || !primaryOnly) {
			found = append(found, b)
		}
	}
	return found, nil
}

// chooseBackend picks one of several: it asks when it can, and otherwise lists them and names the flag
func (a *App) chooseBackend(found []backend, role string) (backend, error) {
	names := make([]string, len(found))
	for i, b := range found {
		names[i] = b.name
	}
	if !a.interactive() {
		return backend{}, fmt.Errorf("more than one Selfhostly %s is running: %s. Use --container NAME", role, strings.Join(names, ", "))
	}
	name, err := a.prompter().Select("Which Selfhostly "+role+"?", names)
	if err != nil {
		return backend{}, err
	}
	for _, b := range found {
		if b.name == name {
			return b, nil
		}
	}
	return backend{}, fmt.Errorf("no such container: %s", name)
}

// findBackend returns a running Selfhostly backend.
//  1. --container (or SFH_PRIMARY_CONTAINER) wins
//  2. otherwise every running container is probed for what it is. One match is used, several are
//     asked about, none falls back to choosing from every running container
//
// Without a terminal the ambiguous cases fail with the candidates listed and the flag to use.
// primaryOnly leaves out secondary nodes, for commands that only make sense on the primary.
func (a *App) findBackend(ctx context.Context, primaryOnly bool) (backend, error) {
	found, err := a.backends(ctx, primaryOnly)
	if err != nil {
		return backend{}, err
	}
	role := map[bool]string{true: "primary", false: "backend"}[primaryOnly]
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		all := a.psNames(ctx)
		if len(all) == 0 {
			return backend{}, fmt.Errorf("no containers are running. Start Selfhostly first, or check that Docker is running")
		}
		if !a.interactive() {
			return backend{}, fmt.Errorf("no running container looks like a Selfhostly %s. Running containers: %s. Use --container NAME", role, strings.Join(all, ", "))
		}
		a.say("None of the running containers looks like a Selfhostly %s. Which one is it?", role)
		name, err := a.prompter().Select("Container", all)
		if err != nil {
			return backend{}, err
		}
		return a.requireBackend(ctx, name)
	}
	return a.chooseBackend(found, role)
}

// requireBackend probes a container the user named or picked, and explains if it has no server in it
func (a *App) requireBackend(ctx context.Context, name string) (backend, error) {
	if b, ok := a.probe(ctx, name); ok {
		return b, nil
	}
	return backend{}, fmt.Errorf("container %q has no Selfhostly server binary (looked for %s in its working directory), so it cannot run this command. "+
		"Is it the right container, and is it built from the Selfhostly backend image or the dev stack?", name, strings.Join(serverBinaries, ", "))
}

// probe reports whether a container is a Selfhostly backend, where its binary is, and whether it is a primary
func (a *App) probe(ctx context.Context, name string) (backend, bool) {
	out, _, err := a.Run.Output(ctx, "docker", "exec", name, "sh", "-c", probeScript)
	fields := strings.Fields(out)
	if err != nil || len(fields) < 2 {
		return backend{}, false
	}
	return backend{name: name, bin: fields[0], primary: fields[1] != "false"}, true
}

// psNames lists running container names, optionally filtered
func (a *App) psNames(ctx context.Context, filters ...string) []string {
	args := []string{"ps", "--format", "{{.Names}}"}
	for _, f := range filters {
		args = append(args, "--filter", f)
	}
	out, _, err := a.Run.Output(ctx, "docker", args...)
	if err != nil {
		return nil
	}
	var names []string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			names = append(names, l)
		}
	}
	sort.Strings(names)
	return names
}

func (a *App) isRunning(ctx context.Context, name string) bool {
	for _, n := range a.psNames(ctx, "name=^"+name+"$") {
		if n == name {
			return true
		}
	}
	return false
}
