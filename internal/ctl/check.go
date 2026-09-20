package ctl

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/selfhostly/internal/ctl/hostcheck"
	"github.com/spf13/cobra"
)

func (a *App) checkCmd() *cobra.Command {
	var fix bool
	var role string
	var port int
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Is this machine ready? Says what is wrong and how to fix it",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			r := hostcheck.Role(role)
			switch r {
			case hostcheck.Primary, hostcheck.SecondaryLink, hostcheck.SecondaryDirect:
			default:
				return fmt.Errorf("--role must be primary, secondary or secondary-direct")
			}
			env := a.envFor(r)
			a.say("Checking this machine for running Selfhostly as a %s.", role)
			if fix {
				_, err := a.ensureHostReady(c.Context(), r, env, port)
				return err
			}
			rep := hostcheck.Run(a.hostEnv(), a.hostOptions(r, env, port))
			a.printReport(rep)
			if !rep.Ready() {
				return fmt.Errorf("this machine is not ready")
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&fix, "fix", false, "offer to run the repairs (asks before each one)")
	f.StringVar(&role, "role", string(hostcheck.Primary), "what the machine will run: primary, secondary or secondary-direct")
	f.IntVar(&port, "port", 0, "with secondary-direct: the API port to check is free")
	f.BoolVar(&a.SkipNet, "skip-network", false, "do not probe the internet")
	return cmd
}

func (a *App) printReport(r hostcheck.Report) {
	for _, f := range r.Findings {
		mark := map[hostcheck.Level]string{hostcheck.OK: "ok  ", hostcheck.Warn: "warn", hostcheck.Fail: "FAIL"}[f.Level]
		a.say("  [%s] %s", mark, f.Title)
		if f.Detail != "" {
			a.say("         %s", f.Detail)
		}
		if f.Fix != nil && f.Level != hostcheck.OK {
			a.say("         fix: %s", f.Fix.Command)
		}
	}
	ok, warn, fail := r.Counts()
	a.say("\n%d ok, %d warnings, %d failed", ok, warn, fail)
}

func (a *App) hostOptions(role hostcheck.Role, env EnvFile, port int) hostcheck.Options {
	return hostcheck.Options{Role: role, EnvFile: env.Path, Dir: a.Dir, Port: port, SkipNet: a.SkipNet}
}

func (a *App) hostEnv() hostcheck.Env {
	if a.HostEnv != nil {
		return *a.HostEnv
	}
	return hostcheck.DefaultEnv()
}

// ensureHostReady checks the machine, offers the repairs, and checks again. It returns the last report
// and an error while the machine is not ready.
func (a *App) ensureHostReady(ctx context.Context, role hostcheck.Role, env EnvFile, port int) (hostcheck.Report, error) {
	if a.SkipHostChecks {
		a.say("  (host checks skipped: --skip-host-checks)")
		return hostcheck.Report{SocketGID: 0}, nil
	}
	rep := hostcheck.Run(a.hostEnv(), a.hostOptions(role, env, port))
	a.printReport(rep)
	if len(rep.Fixes()) > 0 {
		switch {
		case a.DryRun:
			for _, f := range rep.Fixes() {
				a.say("would: %s", f.Description)
			}
			return rep, fmt.Errorf("this machine is not ready")
		case a.Yes || a.interactive():
			if a.applyFixes(ctx, rep) {
				a.say("\nChecking again...")
				rep = hostcheck.Run(a.hostEnv(), a.hostOptions(role, env, port))
				a.printReport(rep)
			}
		}
	}
	if !rep.Ready() {
		return rep, fmt.Errorf("this machine is not ready yet: fix the problems above, then run this again")
	}
	return rep, nil
}

// applyFixes runs each repair the user agrees to. It reports whether it ran any.
func (a *App) applyFixes(ctx context.Context, r hostcheck.Report) bool {
	ran := false
	for _, f := range r.Fixes() {
		if !a.confirm(f.Description+"?  ("+f.Command+")", false) {
			continue
		}
		var err error
		if f.Do != nil {
			err = f.Do()
		} else {
			err = a.Run.Passthrough(ctx, "sh", "-c", f.Command)
		}
		if err != nil {
			fmt.Fprintf(a.Err, "  that did not work: %v\n", err)
			continue
		}
		ran = true
	}
	return ran
}

// envFor is the settings file a role reads: the primary's .env, a secondary's .env.node
func (a *App) envFor(r hostcheck.Role) EnvFile {
	if r == hostcheck.Primary {
		return EnvFile{filepath.Join(a.Dir, a.EnvFile)}
	}
	return EnvFile{filepath.Join(a.Dir, nodeEnvFile)}
}
