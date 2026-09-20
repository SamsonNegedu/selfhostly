package ctl

import "github.com/spf13/cobra"

func (a *App) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the selfhostlyctl version",
		Args:  cobra.NoArgs,
		Run:   func(c *cobra.Command, _ []string) { c.Println(Version) },
	}
}
