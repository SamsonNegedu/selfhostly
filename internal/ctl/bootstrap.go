package ctl

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// BootstrapOpts are the answers that decide which settings a new install gets
type BootstrapOpts struct {
	Auth        string // github | cloudflare | none
	Domain      string
	GithubUsers []string
	CFTeam      string
	CFAud       string
	PrintPlan   bool
}

// BootstrapResult says what was (or would be) added
type BootstrapResult struct {
	Added []string
	Notes []string
	Mode  string
}

func (a *App) bootstrapCmd() *cobra.Command {
	var o BootstrapOpts
	var quiet bool
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Fill in every setting a new install needs (never overwrites a value)",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			env := EnvFile{a.envPath(a.EnvFile)}
			res, err := a.Bootstrap(c.Context(), env, o)
			if err != nil {
				return err
			}
			a.printBootstrap(env, res, o, quiet)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.Auth, "auth", "github", "github, cloudflare or none")
	f.StringVar(&o.Domain, "domain", "", "public hostname (sets AUTH_BASE_URL, PUBLIC_HOSTS, secure cookies)")
	f.StringArrayVar(&o.GithubUsers, "github-user", nil, "allowed GitHub login (repeatable)")
	f.StringVar(&o.CFTeam, "cf-team", "", "Cloudflare Access team domain")
	f.StringVar(&o.CFAud, "cf-aud", "", "Cloudflare Access application audience tag")
	f.BoolVar(&o.PrintPlan, "print-plan", false, "show what would be added without writing")
	f.BoolVar(&quiet, "quiet", false, "print a one-line summary")
	return cmd
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // the system random source failing is not recoverable
	}
	return hex.EncodeToString(b)
}

func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6], b[8] = (b[6]&0x0f)|0x40, (b[8]&0x3f)|0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

// Bootstrap fills in only settings that are missing or empty. An existing value is never changed,
// so it is safe on a live install (for example after an upgrade adds a new setting).
func (a *App) Bootstrap(ctx context.Context, env EnvFile, o BootstrapOpts) (BootstrapResult, error) {
	switch o.Auth {
	case "github", "cloudflare", "none":
	default:
		return BootstrapResult{}, fmt.Errorf("--auth must be github, cloudflare or none")
	}
	var res BootstrapResult
	if !o.PrintPlan {
		if err := env.Touch(); err != nil {
			return res, err
		}
	}
	set := func(key, value string, secret bool) error {
		if value == "" || env.Get(key) != "" {
			return nil
		}
		name := key
		if secret {
			name += " (generated secret)"
		}
		res.Added = append(res.Added, name)
		if o.PrintPlan {
			return nil
		}
		_, err := env.SetIfMissing(key, value)
		return err
	}

	appsVal := firstNonEmpty(env.Get("APPS_DIR"), "./apps")
	dataVal := firstNonEmpty(env.Get("DATA_DIR"), "./data")
	appsAbs, dataAbs := a.dirPath(appsVal), a.dirPath(dataVal)
	if !o.PrintPlan {
		for _, d := range []*string{&appsAbs, &dataAbs} {
			if err := os.MkdirAll(*d, 0o755); err != nil {
				return res, err
			}
			if abs, err := filepath.Abs(*d); err == nil {
				*d = abs
			}
		}
	}

	// An existing database means an upgrade: start in warn mode so nothing already deployed is
	// blocked until `doctor --audit-apps` has been reviewed. A new install enforces.
	_, statErr := os.Stat(filepath.Join(dataAbs, "selfhostly.db"))
	existing := statErr == nil
	res.Mode = "enforce"
	if existing {
		res.Mode = "warn"
	}

	for _, s := range []struct {
		key string
		n   int
	}{{"JWT_SECRET", 32}, {"GATEWAY_API_KEY", 24}, {"REGISTRATION_TOKEN", 24}, {"SETTINGS_ENCRYPTION_KEY", 32}} {
		if err := set(s.key, randHex(s.n), true); err != nil {
			return res, err
		}
	}

	// A new install gets a fresh node ID. An existing one already has its ID in the database and the
	// backend adopts it, so it is only ever copied from there, never invented.
	if !existing {
		if err := set("NODE_ID", newUUID(), false); err != nil {
			return res, err
		}
	} else if env.Get("NODE_ID") == "" {
		out, _, err := a.Run.Output(ctx, "sqlite3", "file:"+filepath.Join(dataAbs, "selfhostly.db")+"?mode=ro",
			"SELECT id FROM nodes WHERE is_primary = 1 LIMIT 1")
		if err == nil {
			if err := set("NODE_ID", strings.TrimSpace(out), false); err != nil {
				return res, err
			}
		}
		if env.Get("NODE_ID") == "" && !o.PrintPlan {
			res.Notes = append(res.Notes, "NODE_ID left unset: the backend uses the ID stored in its database. Do not set a new one.")
		}
	}

	for _, kv := range [][2]string{{"APPS_DIR", appsVal}, {"DATA_DIR", dataVal}, {"HOST_APPS_DIR", appsAbs}, {"SECURITY_MODE", res.Mode}} {
		if err := set(kv[0], kv[1], false); err != nil {
			return res, err
		}
	}
	// Encryption is always an explicit choice, never implied by the mode: on for a new install, off for
	// an existing one until SETTINGS_ENCRYPTION_KEY is saved somewhere off the server.
	if err := set("ENCRYPT_SECRETS_AT_REST", map[bool]string{false: "true", true: "false"}[existing], false); err != nil {
		return res, err
	}

	// The backend user must be in the group that owns the Docker socket. Only for new installs: an
	// existing install's data directory is owned by its current user and the compose defaults match it.
	if !existing {
		if gid, ok := socketGID(a.DockerSock); ok {
			if err := set("DOCKER_GID", strconv.Itoa(gid), false); err != nil {
				return res, err
			}
		}
		if err := set("APP_UID", strconv.Itoa(os.Getuid()), false); err != nil {
			return res, err
		}
	}

	users := strings.Join(o.GithubUsers, ",")
	switch o.Auth {
	case "github":
		_ = set("AUTH_ENABLED", "true", false)
		_ = set("GITHUB_ALLOWED_USERS", users, false)
	case "cloudflare":
		_ = set("AUTH_ENABLED", "false", false)
		_ = set("CF_ACCESS_TEAM_DOMAIN", o.CFTeam, false)
		_ = set("CF_ACCESS_AUD", o.CFAud, false)
	case "none":
		_ = set("AUTH_ENABLED", "false", false)
		_ = set("ALLOW_UNAUTHENTICATED", "true", false)
	}
	if o.Domain != "" {
		for _, kv := range [][2]string{{"AUTH_BASE_URL", "https://" + o.Domain}, {"NODE_API_ENDPOINT", "http://primary:8082"},
			{"PUBLIC_HOSTS", o.Domain}, {"AUTH_SECURE_COOKIE", "true"}} {
			if err := set(kv[0], kv[1], false); err != nil {
				return res, err
			}
		}
	}
	return res, nil
}

func socketGID(path string) (int, bool) {
	var st syscall.Stat_t
	if err := syscall.Stat(path, &st); err != nil || st.Mode&syscall.S_IFMT != syscall.S_IFSOCK {
		return 0, false
	}
	return int(st.Gid), true
}

func (a *App) printBootstrap(env EnvFile, res BootstrapResult, o BootstrapOpts, quiet bool) {
	mode := "?"
	if fi, err := os.Stat(env.Path); err == nil {
		mode = fmt.Sprintf("%o", fi.Mode().Perm())
	}
	if quiet {
		a.say("filled in %d setting(s) in %s (mode %s); existing values were kept", len(res.Added), env.Path, mode)
		return
	}
	a.say("env file: %s (mode %s)", env.Path, mode)
	if len(res.Added) == 0 {
		a.say("nothing to add: every setting is already present")
	} else {
		a.say(map[bool]string{true: "would add:", false: "added:"}[o.PrintPlan])
		for _, x := range res.Added {
			a.say("  %s", x)
		}
	}
	a.say("security mode in %s: %s", env.Path, firstNonEmpty(env.Get("SECURITY_MODE"), "(plan: "+res.Mode+")"))
	for _, n := range res.Notes {
		a.say("\nnote: %s", n)
	}
	var missing []string
	if o.Auth == "github" {
		for _, k := range []string{"GITHUB_CLIENT_ID", "GITHUB_CLIENT_SECRET", "GITHUB_ALLOWED_USERS"} {
			if env.Get(k) == "" {
				missing = append(missing, k)
			}
		}
	}
	if env.Get("TUNNEL_TOKEN") == "" {
		missing = append(missing, "TUNNEL_TOKEN")
	}
	if len(missing) > 0 {
		a.say("\nstill needed (values only you can provide): %s", strings.Join(missing, " "))
	}
	a.say("\nnext: selfhostlyctl setup   (or: docker compose -f docker-compose.prod.yml up -d && selfhostlyctl doctor)")
}
