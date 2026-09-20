package ctl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	selfhostly "github.com/selfhostly"
	"github.com/selfhostly/internal/constants"
	"github.com/spf13/cobra"
)

// shippedCompose maps a short name to the compose file that ships inside selfhostlyctl
var shippedCompose = map[string]string{
	"prod":             "docker-compose.prod.yml",
	"secondary":        "docker-compose.secondary.yml",
	"secondary-direct": "docker-compose.secondary-direct.yml",
	"socket-proxy":     "docker-compose.socket-proxy.yml",
}

func (a *App) composeCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "compose",
		Short: "Work with the compose files that ship with this version",
	}
	var out string
	var force bool
	write := &cobra.Command{
		Use:   "write [prod|secondary|secondary-direct|socket-proxy]",
		Short: "Write a shipped compose file to disk (no git clone needed)",
		Long: "Writes the compose file that matches this version of selfhostlyctl. Nothing is overwritten unless you pass\n" +
			"--force, so your hand-edited file is safe. Default: prod, written as docker-compose.prod.yml.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := "prod"
			if len(args) == 1 {
				name = args[0]
			}
			file, ok := shippedCompose[name]
			if !ok {
				return fmt.Errorf("unknown compose file %q (choose prod, secondary, secondary-direct or socket-proxy)", name)
			}
			dest := a.dirPath(firstNonEmpty(out, file))
			if _, err := os.Stat(dest); err == nil && !force {
				return fmt.Errorf("%s already exists: it was not changed. Pass --force to replace it, or --out to write elsewhere", dest)
			}
			b, err := selfhostly.Files.ReadFile(file)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dest, b, 0o644); err != nil {
				return err
			}
			a.say("wrote %s (the %s compose file that ships with selfhostlyctl %s)", dest, name, Version)
			return nil
		},
	}
	write.Flags().StringVar(&out, "out", "", "where to write it (default: the standard name in --dir)")
	write.Flags().BoolVar(&force, "force", false, "replace the file if it exists")
	root.AddCommand(write)
	return root
}

// newComposeFile is the file compose-diff compares against: the one in --dir if there is one, else the
// one that ships with this version. The second value removes a temporary file when done.
func (a *App) newComposeFile(explicit string) (string, func(), error) {
	if explicit != "" {
		return explicit, func() {}, nil
	}
	if p := a.dirPath(primaryCompose); fileExists(p) {
		return p, func() {}, nil
	}
	b, err := selfhostly.Files.ReadFile(primaryCompose)
	if err != nil {
		return "", nil, err
	}
	tmp, err := os.CreateTemp("", "docker-compose.prod.*.yml")
	if err != nil {
		return "", nil, err
	}
	if _, err := tmp.Write(b); err != nil {
		return "", nil, err
	}
	_ = tmp.Close()
	a.say("(no %s here: comparing with the one that ships with selfhostlyctl %s)", primaryCompose, Version)
	return tmp.Name(), func() { _ = os.Remove(tmp.Name()) }, nil
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// detectLiveCompose finds the compose file(s) and project the running install was started from, using
// the labels Docker Compose puts on its containers. With nothing running it returns nothing and the
// default applies. With several candidates it asks, or fails naming --container: it never guesses.
func (a *App) detectLiveCompose(ctx context.Context) (files []string, project string, err error) {
	found, err := a.backends(ctx, true)
	if err != nil {
		return nil, "", err
	}
	var target backend
	switch len(found) {
	case 0:
		return nil, "", nil
	case 1:
		target = found[0]
	default:
		if target, err = a.chooseBackend(found, "primary"); err != nil {
			return nil, "", err
		}
	}
	label := func(key string) string {
		out, _, err := a.Run.Output(ctx, "docker", "inspect", "--format", `{{index .Config.Labels "`+key+`"}}`, target.name)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(out)
	}
	for _, f := range strings.Split(label(constants.LabelComposeConfigFiles), ",") {
		if f = strings.TrimSpace(f); f != "" {
			if !filepath.IsAbs(f) || !fileExists(f) {
				// started on another machine or from a folder that has since moved: say so, do not guess
				return nil, "", fmt.Errorf("%s was started from %s, which is not on this machine. Pass --compose FILE", target.name, f)
			}
			files = append(files, f)
		}
	}
	return files, label(constants.LabelComposeProject), nil
}
