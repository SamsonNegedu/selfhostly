package ctl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/selfhostly/internal/ctl/hostcheck"
	"github.com/spf13/cobra"
)

type setupOpts struct {
	auth, domain, ghID, ghSecret, tunnel, cfTeam, cfAud string
	ghUsers                                             []string
	noStart                                             bool
}

func (a *App) setupCmd() *cobra.Command {
	var o setupOpts
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up a new primary server, step by step",
		Args:  cobra.NoArgs,
		RunE:  func(c *cobra.Command, _ []string) error { return a.setup(c.Context(), o) },
	}
	f := cmd.Flags()
	f.StringVar(&o.auth, "auth", "", "github, cloudflare or none")
	f.StringVar(&o.domain, "domain", "", "public hostname")
	f.StringArrayVar(&o.ghUsers, "github-user", nil, "allowed GitHub login (repeatable)")
	f.StringVar(&o.ghID, "github-client-id", "", "GitHub OAuth client ID")
	f.StringVar(&o.ghSecret, "github-client-secret", "", "GitHub OAuth client secret")
	f.StringVar(&o.tunnel, "tunnel-token", "", "Cloudflare Tunnel token")
	f.StringVar(&o.cfTeam, "cf-team", "", "Cloudflare Access team domain")
	f.StringVar(&o.cfAud, "cf-aud", "", "Cloudflare Access application audience tag")
	f.BoolVar(&o.noStart, "no-start", false, "configure only; do not start anything")
	return cmd
}

func (a *App) setup(ctx context.Context, o setupOpts) error {
	env := EnvFile{a.envPath(a.EnvFile)}
	a.say("Selfhostly setup")
	a.say("This sets up a new primary server in four steps: check the machine, configure, start, verify.")
	a.say("You can stop at any point and run this command again; it picks up where things really are.")

	a.step("1 of 4: is this machine ready?")
	a.why("Docker, permissions, disk and network. Problems come with the exact command that fixes them.")
	if _, err := a.ensureHostReady(ctx, hostcheck.Primary, env, 0); err != nil {
		return err
	}
	if a.DryRun {
		a.say("\n(dry run) stopping before any change to the configuration")
		return nil
	}

	a.step("2 of 4: configure")
	if env.Get("JWT_SECRET") != "" {
		a.say("Found an existing %s. I will only fill in what is missing and never overwrite a value.", env.Path)
	}
	choice := map[string]string{"1": "github", "2": "cloudflare", "3": "none"}[o.auth]
	if choice == "" {
		choice = o.auth
	}
	if choice == "" {
		a.why("How will people sign in?")
		a.say("  1) GitHub login (recommended without Cloudflare Access)")
		a.say("  2) Cloudflare Access (you already put Cloudflare Zero Trust in front)")
		a.say("  3) No login (only if the machine is not reachable by anyone else)")
		pick := ""
		if err := a.ask(&pick, "Choose 1, 2 or 3", "", "1", false); err != nil {
			return err
		}
		choice = map[string]string{"2": "cloudflare", "3": "none"}[pick]
		if choice == "" {
			choice = "github"
		}
	}
	bo := BootstrapOpts{Auth: choice, CFTeam: o.cfTeam, CFAud: o.cfAud, GithubUsers: o.ghUsers, Domain: o.domain}
	if choice != "none" {
		if err := a.ask(&o.domain, "Public hostname people will use (for example selfhostly.example.com)",
			"Used for sign-in redirects and to pin the gateway to your real hostname.", "", false); err != nil {
			return err
		}
		bo.Domain = o.domain
	}
	switch choice {
	case "github":
		if len(bo.GithubUsers) == 0 {
			u := ""
			if err := a.ask(&u, "Your GitHub username (the only account allowed in; add more later in "+env.Path+")",
				"Access is matched on the account itself, not the display name.", "", false); err != nil {
				return err
			}
			bo.GithubUsers = []string{u}
		}
	case "cloudflare":
		if err := a.ask(&bo.CFTeam, "Cloudflare Access team domain (for example myteam.cloudflareaccess.com)", "", "", false); err != nil {
			return err
		}
		if err := a.ask(&bo.CFAud, "Application AUD tag (Zero Trust > Access > Applications > your app)", "", "", false); err != nil {
			return err
		}
	}
	res, err := a.Bootstrap(ctx, env, bo)
	if err != nil {
		return err
	}
	a.printBootstrap(env, res, bo, true)

	if choice == "github" && env.Get("GITHUB_CLIENT_ID") == "" {
		hint := fmt.Sprintf("Create a GitHub OAuth app at https://github.com/settings/developers\n  Homepage URL: https://%s    Authorization callback URL: https://%s/auth/github/callback", o.domain, o.domain)
		if err := a.ask(&o.ghID, "GitHub OAuth Client ID", hint, "", true); err != nil {
			return err
		}
		if err := a.ask(&o.ghSecret, "GitHub OAuth Client Secret", "", "", true); err != nil {
			return err
		}
		if o.ghID != "" {
			_ = env.Set("GITHUB_CLIENT_ID", o.ghID)
		}
		if o.ghSecret != "" {
			_ = env.Set("GITHUB_CLIENT_SECRET", o.ghSecret)
		}
	}
	if env.Get("TUNNEL_TOKEN") == "" {
		if err := a.ask(&o.tunnel, "Cloudflare Tunnel token",
			"The Cloudflare Tunnel connects your domain to this machine without opening any port.\nCreate one at Zero Trust > Networks > Tunnels, choose Docker, and copy the token.", "", true); err != nil {
			return err
		}
		if o.tunnel != "" {
			_ = env.Set("TUNNEL_TOKEN", o.tunnel)
		}
	}
	var missing []string
	if choice == "github" && (env.Get("GITHUB_CLIENT_ID") == "" || env.Get("GITHUB_CLIENT_SECRET") == "") {
		missing = append(missing, "GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET")
	}
	if env.Get("TUNNEL_TOKEN") == "" {
		missing = append(missing, "TUNNEL_TOKEN")
	}
	if len(missing) > 0 {
		a.say("\nStill missing in %s: %s", env.Path, strings.Join(missing, ", "))
		a.say("Add them, then run 'selfhostlyctl setup' again. Nothing else needs repeating.")
	}

	a.step("3 of 4: start")
	if o.noStart {
		a.say("Skipping start (--no-start).")
		return nil
	}
	services := []string{"primary", "gateway", "frontend", "cloudflared"}
	if env.Get("TUNNEL_TOKEN") == "" {
		services = services[:3]
		a.say("No tunnel token yet, so the tunnel is not started. The rest will start.")
	}
	if !a.confirm("Download the images and start "+strings.Join(services, " ")+"?", true) {
		a.say("Not starting. Run this command again when you are ready.")
		return nil
	}
	dc := a.primaryCompose()
	if err := dc.pass(ctx, append([]string{"pull"}, services...)...); err != nil {
		a.say("  (pull failed: continuing with images already on this machine)")
	}
	if err := dc.pass(ctx, append([]string{"up", "-d"}, services...)...); err != nil {
		return fmt.Errorf("could not start: %w", err)
	}
	fmt.Fprint(a.Out, "  waiting for the backend to become healthy")
	pc := dc.cid(ctx, "primary")
	if !a.waitHealthy(ctx, pc, 150*time.Second, 3*time.Second, true) {
		fmt.Fprintln(a.Out)
		return fmt.Errorf("the backend did not become healthy. Look at: docker logs %s", primaryContainer)
	}
	fmt.Fprintln(a.Out, " healthy")

	a.step("4 of 4: verify")
	_ = a.Run.Passthrough(ctx, "docker", "exec", pc, "./selfhostly", "doctor", "--offline")
	a.say("\nSelfhostly is running.")
	a.say("  Open:            https://%s", firstNonEmpty(o.domain, env.Get("PUBLIC_HOSTS")))
	a.say("  Add a machine:   selfhostlyctl join-token   (then run 'selfhostlyctl join' on the other machine)")
	a.say("  Back up:         selfhostlyctl backup")
	a.say("  Upgrade later:   selfhostlyctl upgrade")
	return nil
}
