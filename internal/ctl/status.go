package ctl

import (
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the containers of this Selfhostly install",
		Long: "Finds the running Selfhostly backend by probing containers (never by name) and shows every container of\n" +
			"its compose project. If no backend is found it shows all running containers.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			const format = "table {{.Names}}\t{{.Status}}\t{{.Image}}"
			projects := map[string]bool{}
			for _, n := range a.psNames(c.Context()) {
				if _, ok := a.probe(c.Context(), n); !ok {
					continue
				}
				out, _, _ := a.Run.Output(c.Context(), "docker", "inspect", "--format", `{{index .Config.Labels "com.docker.compose.project"}}`, n)
				if p := strings.TrimSpace(out); p != "" {
					projects[p] = true
				}
			}
			args := []string{"ps", "--format", format}
			for _, p := range sortedKeys(projects) {
				args = append(args, "--filter", "label=com.docker.compose.project="+p)
			}
			return a.Run.Passthrough(c.Context(), "docker", args...)
		},
	}
}

func (a *App) doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "doctor [flags]",
		Short:              "Run the server's own health checks inside the running container",
		DisableFlagParsing: true, // every argument belongs to the server's doctor
		RunE: func(c *cobra.Command, args []string) error {
			args = a.takeOwnFlags(args)
			b, err := a.findBackend(c.Context(), false)
			if err != nil {
				return err
			}
			return a.Run.Passthrough(c.Context(), "docker", append([]string{"exec", b.name, b.bin, "doctor"}, args...)...)
		},
	}
}

// takeOwnFlags pulls selfhostlyctl's own flags out of arguments that otherwise all belong to the
// server's doctor, and returns the rest untouched.
func (a *App) takeOwnFlags(args []string) []string {
	var rest []string
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--container" && i+1 < len(args):
			a.Container = args[i+1]
			i++
		case strings.HasPrefix(arg, "--container="):
			a.Container = strings.TrimPrefix(arg, "--container=")
		case arg == "--non-interactive":
			a.NonInteractive = true
		default:
			rest = append(rest, arg)
		}
	}
	return rest
}
