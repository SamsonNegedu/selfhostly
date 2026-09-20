// Package ctl is the selfhostlyctl command line: guided setup and operation of a Selfhostly install.
// It is stateless: every command looks at the real machine, so a command interrupted halfway is fixed
// by running it again.
package ctl

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/selfhostly/internal/ctl/hostcheck"
	"github.com/spf13/cobra"
)

// Version is set at build time with -ldflags "-X github.com/selfhostly/internal/ctl.Version=..."
var Version = "dev"

const (
	primaryContainer = "selfhostly-primary"
	nodeContainer    = "selfhostly-node"
	nodeEnvFile      = ".env.node"
	primaryCompose   = "docker-compose.prod.yml"
	nodeCompose      = "docker-compose.secondary.yml"
	nodeDirectExtra  = "docker-compose.secondary-direct.yml"
	nodeProject      = "selfhostly-node"
	defaultEnvFile   = ".env"
)

// App carries what every command needs. Tests replace the fields.
type App struct {
	Out, Err io.Writer
	Run      Runner
	Prompt   Prompter

	Sleep func(time.Duration) // nil: time.Sleep
	Now   func() time.Time    // nil: time.Now
	// DockerSock is where the Docker socket is expected (tests point it elsewhere)
	DockerSock string

	HostEnv *hostcheck.Env // nil: the real machine (tests inject a fake)

	Container               string // --container: the backend container to use, instead of discovering it
	SkipNet, SkipHostChecks bool
	NodeExtraCompose        string // extra compose files for the node (tests, unusual networks)
	HTTPGet                 func(url string) error

	Dir            string
	EnvFile        string
	Yes            bool
	NonInteractive bool
	DryRun         bool
}

// New builds an App wired to the real terminal
func New() *App {
	return &App{
		Out: os.Stdout, Err: os.Stderr, Run: execRunner{}, DockerSock: "/var/run/docker.sock",
		NodeExtraCompose: os.Getenv("SFH_NODE_EXTRA_COMPOSE"),
	}
}

// Root builds the command tree
func (a *App) Root() *cobra.Command {
	root := &cobra.Command{
		Use:           "selfhostlyctl",
		Short:         "Set up, join, upgrade and check a Selfhostly install",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(a.Out)
	root.SetErr(a.Err)
	f := root.PersistentFlags()
	if a.Dir == "" {
		a.Dir = "."
	}
	f.StringVar(&a.Dir, "dir", a.Dir, "the Selfhostly folder (where docker-compose.prod.yml and .env are)")
	if a.EnvFile == "" {
		a.EnvFile = defaultEnvFile
	}
	f.StringVar(&a.EnvFile, "env-file", a.EnvFile, "settings file, relative to --dir")
	f.BoolVarP(&a.Yes, "yes", "y", a.Yes, "answer yes to every question")
	f.BoolVar(&a.NonInteractive, "non-interactive", a.NonInteractive, "never ask; use defaults and flags")
	f.StringVar(&a.Container, "container", a.Container, "the Selfhostly container to use (default: found by its compose service label)")
	f.BoolVar(&a.SkipHostChecks, "skip-host-checks", a.SkipHostChecks, "trust that this machine is ready (for automation)")
	f.BoolVar(&a.DryRun, "dry-run", a.DryRun, "show what would change and change nothing")

	root.AddCommand(a.checkCmd(), a.joinTokenCmd(), a.statusCmd(), a.doctorCmd(), a.versionCmd(),
		a.joinCmd(), a.setupCmd(), a.bootstrapCmd(), a.upgradeCmd(), a.backupCmd(), a.composeDiffCmd(), a.pinImagesCmd(), a.composeCmd())
	return root
}

// Execute runs the CLI and returns the process exit code
func Execute(args []string) int {
	a := New()
	root := a.Root()
	root.SetArgs(args)
	if err := root.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(a.Err, "error:", err)
		return 1
	}
	return 0
}

func (a *App) dirPath(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(a.Dir, name)
}

func (a *App) envPath(name string) string { return a.dirPath(name) }
