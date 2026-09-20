package ctl

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) joinTokenCmd() *cobra.Command {
	var direct bool
	var primaryURL string
	cmd := &cobra.Command{
		Use:   "join-token",
		Short: "On the primary: create a one-time token and print the command for the new machine",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			target, err := a.findBackend(c.Context(), true)
			if err != nil {
				return fmt.Errorf("no running primary found here (run this on the machine that runs the primary): %w", err)
			}
			out, stderr, err := a.Run.Output(c.Context(), "docker", "exec", target.name, target.bin, "join-token")
			if err != nil {
				return fmt.Errorf("could not create a token: %s", strings.TrimSpace(stderr))
			}
			token := strings.TrimSpace(out)
			if s := strings.TrimSpace(stderr); s != "" {
				fmt.Fprintln(a.Out, "  "+strings.ReplaceAll(s, "\n", "\n  "))
			}

			url, flag := primaryURL, ""
			if direct {
				flag = " --direct"
				if url == "" {
					url = a.publishedURL(c, target.name)
				}
			} else if url == "" {
				url = "https://<your-public-hostname>"
				if host := firstHost(readEnv(filepath.Join(a.Dir, a.EnvFile))["PUBLIC_HOSTS"]); host != "" {
					url = "https://" + host
				}
			}
			fmt.Fprintf(a.Out, "\nOn the new machine, run:\n  selfhostlyctl join%s --primary-url %s --token %s\n\n", flag, url, token)
			fmt.Fprintln(a.Out, "The token works once and expires in an hour. Run this command again for another machine.")
			if !direct {
				fmt.Fprintf(a.Out, "The new machine only needs to reach %s. It opens no port and needs no firewall change.\n", url)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&direct, "direct", false, "older mode: the primary calls the machine (needs a reachable port)")
	cmd.Flags().StringVar(&primaryURL, "primary-url", "", "address the new machine should use for the primary")
	return cmd
}

// publishedURL reads where the primary's API port is published on the host
func (a *App) publishedURL(c *cobra.Command, container string) string {
	out, _, err := a.Run.Output(c.Context(), "docker", "port", container, "8082/tcp")
	line := strings.TrimSpace(strings.SplitN(out, "\n", 2)[0])
	i := strings.LastIndex(line, ":")
	if err != nil || i < 0 {
		return "http://<address-of-this-machine>:8082"
	}
	host, port := line[:i], line[i+1:]
	if host == "127.0.0.1" || host == "localhost" {
		fmt.Fprintf(a.Out, "The primary only listens on this machine (%s). A machine on your network cannot reach it yet.\n"+
			"Publish it on a LAN or VPN address: selfhostlyctl upgrade --set PRIMARY_NODE_BIND=<address>\n", host)
		return "http://<address-of-this-machine>:" + port
	}
	return "http://" + host + ":" + port
}

func firstHost(list string) string {
	return strings.TrimSpace(strings.SplitN(list, ",", 2)[0])
}
