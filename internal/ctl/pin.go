package ctl

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var digestRE = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type pinTarget struct{ key, ref string }

func (a *App) pinImagesCmd() *cobra.Command {
	var version string
	cmd := &cobra.Command{
		Use:   "pin-images",
		Short: "Pin the image tags to immutable digests in the settings file",
		Long: "Resolves the image tags docker-compose.prod.yml uses to digests and writes them into the settings file\n" +
			"(GATEWAY_IMAGE, BACKEND_IMAGE, FRONTEND_IMAGE, CLOUDFLARED_IMAGE). A new server then pulls exactly the\n" +
			"bytes you tested. To go back to tags, delete the *_IMAGE lines. Use --dry-run to only print them.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return a.pinImages(c.Context(), version) },
	}
	cmd.Flags().StringVar(&version, "version", "", "image version (default: SELFHOSTLY_VERSION from the environment or settings file, else latest)")
	return cmd
}

func (a *App) pinImages(ctx context.Context, version string) error {
	env := EnvFile{a.envPath(a.EnvFile)}
	version = firstNonEmpty(version, os.Getenv("SELFHOSTLY_VERSION"), env.Get("SELFHOSTLY_VERSION"), "latest")
	targets := []pinTarget{
		{"GATEWAY_IMAGE", "ghcr.io/samsonnegedu/selfhostly-gateway:" + version},
		{"BACKEND_IMAGE", "ghcr.io/samsonnegedu/selfhostly-backend:" + version},
		{"FRONTEND_IMAGE", "ghcr.io/samsonnegedu/selfhostly-frontend:" + version},
		{"CLOUDFLARED_IMAGE", firstNonEmpty(os.Getenv("CLOUDFLARED_TAG_REF"), "cloudflare/cloudflared:latest")},
	}
	// resolve everything first, so a failure half way leaves the settings file untouched
	pinned := make([]string, len(targets))
	for i, t := range targets {
		out, _, err := a.Run.Output(ctx, "docker", "buildx", "imagetools", "inspect", t.ref, "--format", "{{.Manifest.Digest}}")
		digest := strings.TrimSpace(out)
		if err != nil || !digestRE.MatchString(digest) {
			return fmt.Errorf("could not resolve a digest for %s (is it published, and are you logged in?)", t.ref)
		}
		// strip only the trailing tag, so a registry host with a port survives
		pinned[i] = t.ref[:strings.LastIndex(t.ref, ":")] + "@" + digest
		a.say("%s=%s", t.key, pinned[i])
	}
	if a.DryRun {
		a.say("(dry run: %s not modified)", env.Path)
		return nil
	}
	for i, t := range targets {
		if err := env.Set(t.key, pinned[i]); err != nil {
			return err
		}
	}
	a.say("pinned in %s. Roll out with: selfhostlyctl upgrade", env.Path)
	return nil
}
