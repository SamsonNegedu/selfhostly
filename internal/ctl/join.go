package ctl

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/selfhostly/internal/ctl/hostcheck"
	"github.com/spf13/cobra"
)

type joinOpts struct {
	primaryURL, token, name, endpoint string
	port                              int
	direct, skipReach                 bool
}

func (a *App) joinCmd() *cobra.Command {
	var o joinOpts
	cmd := &cobra.Command{
		Use:   "join",
		Short: "Connect this machine to a Selfhostly primary as a secondary node",
		Args:  cobra.NoArgs,
		RunE:  func(c *cobra.Command, _ []string) error { return a.join(c.Context(), o) },
	}
	f := cmd.Flags()
	f.StringVar(&o.primaryURL, "primary-url", "", "address of the primary (printed by join-token)")
	f.StringVar(&o.token, "token", "", "one-time join token (starts with sfj_)")
	f.StringVar(&o.name, "name", "", "name for this machine (default: its hostname)")
	f.StringVar(&o.endpoint, "endpoint", "", "direct mode: the address the primary uses to reach this machine")
	f.IntVar(&o.port, "port", 0, "direct mode: this node's API port (default 8082)")
	f.BoolVar(&o.direct, "direct", false, "older mode: the primary calls this machine")
	f.BoolVar(&o.skipReach, "skip-reach-check", false, "do not check that the primary is reachable")
	return cmd
}

func (a *App) join(ctx context.Context, o joinOpts) error {
	env := EnvFile{a.dirPath(nodeEnvFile)}
	a.say("Join this machine to a Selfhostly cluster")
	if o.direct {
		a.say("Direct mode: the primary will call this machine, so it needs an open port. Four steps: connection details, machine check, start, confirm.")
	} else {
		a.say("This machine will connect OUT to the primary. Nothing needs to reach it: no port, no firewall rule.")
		a.say("Four steps: connection details, check the machine, start the node, confirm it connected.")
	}

	a.step("1 of 4: connection details")
	if err := a.ask(&o.primaryURL, "Address of the primary (printed by 'join-token' on the primary)", "For example https://selfhostly.example.com", "", false); err != nil {
		return err
	}
	if err := a.ask(&o.token, "One-time join token (starts with sfj_)", "", "", true); err != nil {
		return err
	}
	if o.token == "" {
		return fmt.Errorf("a join token is required: run 'selfhostlyctl join-token' on the primary")
	}
	o.primaryURL = strings.TrimRight(o.primaryURL, "/")
	host, _ := os.Hostname()
	if host == "" {
		host = "node"
	}
	if err := a.ask(&o.name, "A name for this machine", "Shown in the UI.", host, false); err != nil {
		return err
	}
	if o.port == 0 {
		o.port, _ = strconv.Atoi(env.Get("NODE_PORT"))
	}
	if o.port == 0 {
		o.port = 8082
	}
	if o.direct {
		def := fmt.Sprintf("http://%s:%d", lanIP(), o.port)
		if err := a.ask(&o.endpoint, "Address the PRIMARY will use to reach this machine",
			"Must work from the primary: a LAN or VPN address of this machine. Not localhost.", def, false); err != nil {
			return err
		}
		if strings.Contains(o.endpoint, "localhost") || strings.Contains(o.endpoint, "127.0.0.1") {
			return fmt.Errorf("the endpoint cannot be localhost: the primary could never reach it")
		}
	}

	a.why("Checking that the primary is reachable from here...")
	switch {
	case o.skipReach:
		a.say("  (skipped: --skip-reach-check)")
	default:
		if err := a.httpGet(o.primaryURL + "/api/health"); err != nil {
			return fmt.Errorf("cannot reach %s/api/health from this machine (%v).\n"+
				"  - is the address right, and does this machine have internet access?\n"+
				"  - in direct mode, is the primary's port published? On the primary run: selfhostlyctl join-token --direct", o.primaryURL, err)
		}
		a.say("  ✓ reached %s", o.primaryURL)
	}
	if strings.HasPrefix(o.primaryURL, "http://") {
		a.say("  ! this address is plain http: the node's key crosses the network unencrypted. Prefer an https address.")
	}

	a.step("2 of 4: is this machine ready?")
	role := hostcheck.SecondaryLink
	if o.direct {
		role = hostcheck.SecondaryDirect
		if !a.DryRun {
			if err := env.Set("NODE_PORT", strconv.Itoa(o.port)); err != nil {
				return err
			}
		}
	}
	rep, err := a.ensureHostReady(ctx, role, env, o.port)
	if err != nil {
		return err
	}
	if a.DryRun {
		a.say("\n(dry run) stopping before any change to the configuration")
		return nil
	}

	a.step("3 of 4: start the node")
	data := firstNonEmpty(env.Get("DATA_DIR"), a.absDir("node-data"))
	apps := firstNonEmpty(env.Get("APPS_DIR"), a.absDir("node-apps"))
	for _, d := range []string{data, apps} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	gid := rep.SocketGID
	if gid == 0 {
		gid = 984
	}
	for k, v := range map[string]string{
		"PRIMARY_NODE_URL": o.primaryURL, "REGISTRATION_TOKEN": o.token, "NODE_NAME": o.name,
		"DATA_DIR": data, "APPS_DIR": apps, "HOST_APPS_DIR": apps,
	} {
		if err := env.Set(k, v); err != nil {
			return err
		}
	}
	transport := "tunnel"
	if o.direct {
		transport = "direct"
		if err := env.Set("NODE_API_ENDPOINT", o.endpoint); err != nil {
			return err
		}
	}
	if err := env.Set("NODE_TRANSPORT", transport); err != nil {
		return err
	}
	_, _ = env.SetIfMissing("APP_UID", strconv.Itoa(os.Getuid()))
	_, _ = env.SetIfMissing("DOCKER_GID", strconv.Itoa(gid))
	a.say("  wrote %s (mode 600)", env.Path)

	if !a.confirm("Download the image and start the node?", true) {
		a.say("Not starting. Run this command again when ready.")
		return nil
	}
	dc := a.nodeCompose(o.direct)
	if err := dc.pass(ctx, "pull"); err != nil {
		a.say("  (pull failed: continuing with the image already on this machine)")
	}
	if err := dc.pass(ctx, "up", "-d"); err != nil {
		return fmt.Errorf("could not start the node: %w", err)
	}
	nc := dc.cid(ctx, "node")
	fmt.Fprint(a.Out, "  waiting for the node to become healthy")
	if !a.waitHealthy(ctx, nc, 120*time.Second, 3*time.Second, true) {
		fmt.Fprintln(a.Out)
		return fmt.Errorf("the node did not become healthy. Look at: docker logs %s", nodeContainer)
	}
	fmt.Fprintln(a.Out, " healthy")

	a.step("4 of 4: confirm it connected to the primary")
	if o.direct {
		return a.confirmDirect(ctx, nc, o)
	}
	return a.confirmLink(ctx, nc)
}

var errField = regexp.MustCompile(`"error":"([^"]*)`)

func lastError(logs, msg string) string {
	last := ""
	for _, l := range strings.Split(logs, "\n") {
		if strings.Contains(l, `"msg":"`+msg+`"`) {
			if m := errField.FindStringSubmatch(l); m != nil {
				last = m[1]
			}
		}
	}
	return last
}

// confirmLink: the node logs "node link up" once the primary has accepted its connection
func (a *App) confirmLink(ctx context.Context, nc string) error {
	a.why("The node connects out to the primary and keeps that connection open. It retries by itself if it drops.")
	var logs string
	for s := 0; s < 100; s += 3 {
		logs = a.logs(ctx, nc, 0)
		if strings.Contains(logs, `"msg":"node link up"`) {
			fmt.Fprintln(a.Out)
			a.say("  ✓ connected to the primary")
			a.say("\nDone. This machine now appears in the Nodes page. It reconnects on its own after a restart or a network drop.")
			a.say("The token has been used up.")
			return nil
		}
		fmt.Fprint(a.Out, ".")
		a.sleep(3 * time.Second)
	}
	fmt.Fprintln(a.Out)
	last := lastError(logs, "node link down, reconnecting")
	a.say("The node did not connect. %s", ifSet("Last error: ", last))
	a.say(joinAdvice(last, false))
	a.say("  Logs: docker logs %s", nodeContainer)
	return fmt.Errorf("the node did not connect")
}

// confirmDirect: the node registers over HTTP, and the primary reports whether it can call the node back
func (a *App) confirmDirect(ctx context.Context, nc string, o joinOpts) error {
	a.why("Registration retries for about a minute and a half.")
	var logs string
	for s := 0; s < 110; s += 4 {
		logs = a.logs(ctx, nc, 0)
		if strings.Contains(logs, `"msg":"auto-registration successful"`) {
			status := ""
			for _, l := range strings.Split(logs, "\n") {
				if strings.Contains(l, `"msg":"auto-registration response"`) {
					if m := regexp.MustCompile(`"status":"([a-z]*)"`).FindStringSubmatch(l); m != nil {
						status = m[1]
					}
				}
			}
			fmt.Fprintln(a.Out)
			a.say("  ✓ registered with the primary")
			if status == "unreachable" {
				a.say("  ! but the primary could not reach this machine at %s", o.endpoint)
				a.say("    Check that address is reachable FROM the primary and that port %d is open in this machine's firewall.", o.port)
			} else {
				a.say("  ✓ the primary can reach it (%s)", status)
			}
			a.say("\nDone. This machine now appears in the Nodes page. The token has been used up.")
			return nil
		}
		fmt.Fprint(a.Out, ".")
		a.sleep(4 * time.Second)
	}
	fmt.Fprintln(a.Out)
	last := lastError(logs, "auto-registration failed")
	a.say("Registration did not complete. %s", ifSet("Last error: ", last))
	a.say(joinAdvice(last, true))
	a.say("  Logs: docker logs %s", nodeContainer)
	return fmt.Errorf("registration did not complete")
}

func joinAdvice(last string, direct bool) string {
	flag := ""
	if direct {
		flag = " --direct"
	}
	switch {
	case containsAny(last, "401", "token", "credentials"):
		return "  The token is wrong, expired or already used. Run 'selfhostlyctl join-token" + flag + "' on the primary and join again."
	case strings.Contains(last, "409"):
		return "  A node with that name already exists. Join again with --name <another name>."
	case strings.Contains(last, "429"):
		return "  Too many attempts in a minute. Wait a minute and check again: it keeps retrying by itself."
	case containsAny(last, "refused", "timeout", "route", "resolve", "dial"):
		return "  This machine cannot reach the primary. Check the address and this machine's internet access."
	}
	return ""
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

func ifSet(prefix, v string) string {
	if v == "" {
		return ""
	}
	return prefix + v
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

// absDir is a folder under the Selfhostly directory, as an absolute path
func (a *App) absDir(name string) string {
	p, err := filepath.Abs(a.dirPath(name))
	if err != nil {
		return a.dirPath(name)
	}
	return p
}

// healthMarker is what a Selfhostly primary's /api/health says about itself
const healthMarker = `"service":"selfhostly"`

// httpGet checks that url answers like a Selfhostly primary. Reaching *something* is not enough: a login
// page from Cloudflare Access or a captive portal answers 200 too, and the node would then never connect.
func (a *App) httpGet(url string) error {
	if a.HTTPGet != nil {
		return a.HTTPGet(url)
	}
	c := &http.Client{Timeout: 8 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if !strings.Contains(strings.ReplaceAll(string(body), " ", ""), healthMarker) {
		return fmt.Errorf("it answered, but not like a Selfhostly primary (is a login page, such as Cloudflare Access, in front of it? " +
			"see docs/operations/cloudflare-zero-trust.md#adding-a-secondary-machine)")
	}
	return nil
}

// lanIP is the address this machine uses to reach the network. Nothing is sent: a UDP "connection" only
// picks the outgoing interface.
func lanIP() string {
	c, err := net.Dial("udp", "1.1.1.1:80")
	if err != nil {
		return "<address-of-this-machine>"
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).IP.String()
}
