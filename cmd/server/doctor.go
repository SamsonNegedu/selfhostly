package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	selfauth "github.com/selfhostly/internal/auth"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/secrets"
	"github.com/selfhostly/internal/validation"
	_ "modernc.org/sqlite"
)

// finding levels. FAIL means it will not work, WARN means you should act, NOTE means it is expected or optional.
const (
	levelFail = "FAIL"
	levelWarn = "WARN"
	levelNote = "NOTE"
)

type finding struct{ level, message, fix string }

// doctorReport collects check results and prints them as they are added. Only FAIL changes the exit
// code. Every WARN comes with the step that resolves it, and the end of the report lists them again, so
// nobody has to scroll to find what needs doing.
type doctorReport struct {
	fails, warns, notes int
	items               []finding
}

func (r *doctorReport) pass(format string, a ...interface{}) {
	fmt.Printf("  PASS  %s\n", fmt.Sprintf(format, a...))
}

func (r *doctorReport) add(level, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	r.items = append(r.items, finding{level: level, message: msg})
	fmt.Printf("  %s  %s\n", level, msg)
}

// warn is something you should act on. Follow it with fix() saying how.
func (r *doctorReport) warn(format string, a ...interface{}) {
	r.warns++
	r.add(levelWarn, format, a...)
}

// note is expected for how this install is set up, or optional. Nothing needs doing.
func (r *doctorReport) note(format string, a ...interface{}) {
	r.notes++
	r.add(levelNote, format, a...)
}

func (r *doctorReport) fail(format string, a ...interface{}) {
	r.fails++
	r.add(levelFail, format, a...)
}

// fix attaches the step that resolves the last finding
func (r *doctorReport) fix(format string, a ...interface{}) {
	hint := fmt.Sprintf(format, a...)
	if n := len(r.items); n > 0 {
		r.items[n-1].fix = hint
	}
	fmt.Printf("          fix: %s\n", hint)
}

// summary prints the verdict and repeats what needs attention
func (r *doctorReport) summary() {
	fmt.Println()
	switch {
	case r.fails > 0:
		fmt.Printf("Result: NOT READY. %d must be fixed, %d should be looked at, %d notes.\n", r.fails, r.warns, r.notes)
	case r.warns > 0:
		fmt.Printf("Result: works, but %d thing(s) need your attention. %d notes need nothing.\n", r.warns, r.notes)
	default:
		fmt.Printf("Result: healthy. Nothing needs action (%d notes are expected or optional).\n", r.notes)
	}
	n := 0
	for _, level := range []string{levelFail, levelWarn} {
		for _, it := range r.items {
			if it.level != level {
				continue
			}
			if n == 0 {
				fmt.Println("\nTo do:")
			}
			n++
			fmt.Printf("  %d. %s\n", n, it.message)
			if it.fix != "" {
				fmt.Printf("     -> %s\n", it.fix)
			}
		}
	}
}

func section(name string) { fmt.Printf("\n%s\n", name) }

// runDoctor is the preflight for new installs and upgrades. It never changes state: the database
// is opened read-only and no containers are touched. Exit code is 1 when any check fails.
func runDoctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	auditApps := fs.Bool("audit-apps", false, "check every deployed app's compose file against the current policy")
	offline := fs.Bool("offline", false, "skip checks that need the internet (GitHub, Cloudflare)")
	skipIntegrity := fs.Bool("skip-db-integrity", false, "skip the database integrity check (used when the database is being written by a running server in another container)")
	_ = fs.Parse(args)

	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	_ = godotenv.Load(envFile)

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("  FAIL  configuration: %v\n", err)
		return 1
	}

	r := &doctorReport{}
	fmt.Printf("selfhostly doctor (security mode: %s)\n", cfg.Security.Mode)

	// Mirror what the next start does, without changing anything
	if cfg.Node.IsPrimary {
		if id, ok := readPrimaryID(cfg.DatabasePath); ok {
			cfg.AdoptPrimaryIdentity(id, false)
		}
	}

	section("What the next start will use")
	for _, st := range effectiveSettings(cfg) {
		val := st.value
		if val == "" {
			val = "-"
		}
		fmt.Printf("  %-27s %-34s %s\n", st.name, val, st.source)
	}

	section("Configuration")
	// What this install looks like decides what is a problem: no login is normal in development and on
	// a secondary, and a mistake on something that has login or a public https address.
	dev := cfg.Environment == constants.EnvDevelopment
	prodLike := cfg.Auth.Enabled || cfg.Security.CloudflareAccess.Enabled() || strings.HasPrefix(strings.ToLower(cfg.Auth.BaseURL), "https://")
	issues := cfg.ValidateStartup()
	advisory := false
	for _, i := range issues {
		switch {
		case i.Fatal:
			r.fail("%s", i.Message)
		case i.Advisory:
			advisory = true
			r.note("%s", i.Message)
		default:
			r.warn("%s", i.Message)
		}
	}
	if len(issues) == 0 {
		r.pass("no configuration problems")
	}
	if dev && prodLike {
		r.warn("APP_ENV=development on an install that looks like production (login or an https address is set): debug endpoints are exposed")
		r.fix("selfhostlyctl upgrade --set APP_ENV=production")
	} else if dev {
		r.note("development mode: debug endpoints are on. Expected while developing; use APP_ENV=production on a real server")
	}

	section("Docker")
	checkDockerAccess(r)
	dm := docker.NewManager(cfg.AppsDir)
	ex := dm.GetCommandExecutor()
	if out, err := ex.ExecuteCommand(docker.DockerCommand, "version", "--format", "{{.Server.Version}}"); err != nil {
		r.fail("cannot reach the Docker daemon: %v", err)
	} else {
		r.pass("Docker daemon reachable (server %s)", strings.TrimSpace(string(out)))
	}
	if _, err := ex.ExecuteCommand(docker.DockerCommand, docker.ComposeCommand, "version"); err != nil {
		r.fail("docker compose plugin not available: %v", err)
	} else {
		r.pass("docker compose available")
	}

	section("Apps directory")
	abs, _ := filepath.Abs(cfg.AppsDir)
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		r.fail("APPS_DIR %s does not exist", abs)
	} else {
		probe := filepath.Join(abs, ".doctor-write-test")
		if err := os.WriteFile(probe, []byte("x"), 0o600); err != nil {
			r.fail("APPS_DIR %s is not writable: %v", abs, err)
		} else {
			_ = os.Remove(probe)
			r.pass("APPS_DIR %s is writable", abs)
		}
	}
	hostDir := cfg.Security.HostAppsDir
	source := "HOST_APPS_DIR"
	if hostDir == "" {
		hostDir = docker.DetectHostAppsDir(cfg.AppsDir, ex)
		source = "detected"
	}
	if _, err := os.Stat("/.dockerenv"); err == nil && hostDir == abs && cfg.Security.HostAppsDir == "" {
		r.warn("host path for apps could not be detected (running in a container)")
		r.fix("set HOST_APPS_DIR to the host path of %s, then restart", abs)
	} else {
		r.pass("host apps directory: %s (%s)", hostDir, source)
	}
	cfg.Security.HostAppsDir = hostDir

	section("Database")
	checkDatabase(r, cfg, *skipIntegrity)

	section("Authentication")
	if cfg.Auth.Enabled {
		r.pass("GitHub OAuth enabled, %d allowed user(s) configured", len(cfg.Auth.GitHub.AllowedUsers))
		if !*offline {
			al := selfauth.NewAllowList(cfg.Auth.GitHub.AllowedUsers)
			before := al.Size()
			al.Resolve(context.Background(), nil, "", cfg.Auth.GitHub.AllowedUsers)
			if al.Size() > before {
				r.pass("allow-list logins resolved on GitHub")
			} else {
				r.note("could not resolve logins on GitHub (offline or rate limited); typed casing will be used")
			}
		}
	} else if cfg.Security.CloudflareAccess.Enabled() {
		r.pass("Cloudflare Access verification configured for %s", cfg.Security.CloudflareAccess.TeamDomain)
		if !*offline {
			checkURL(r, "https://"+cfg.Security.CloudflareAccess.TeamDomain+constants.CFAccessCertsPath, "Cloudflare Access signing keys")
		}
	} else {
		switch {
		case !cfg.Node.IsPrimary:
			if !advisory {
				r.note("no user login on this secondary node: the primary and gateway authenticate users, and reach this node with its key")
			}
		case dev:
			r.note("no login configured. Fine for local development; not for a server other people can reach")
		default:
			r.warn("no login configured: the API is open to anything that can reach it")
			r.fix("enable GitHub login (AUTH_ENABLED=true, GITHUB_CLIENT_ID/SECRET, GITHUB_ALLOWED_USERS) or Cloudflare Access (CF_ACCESS_TEAM_DOMAIN, CF_ACCESS_AUD): see docs/security/overview.md")
		}
	}

	if !cfg.Node.IsPrimary && cfg.Node.PrimaryNodeURL != "" {
		section("Cluster")
		checkURL(r, strings.TrimSuffix(cfg.Node.PrimaryNodeURL, "/")+"/api/health", "primary node")
	}

	if *auditApps {
		section("Deployed apps vs. current compose policy")
		auditDeployedApps(r, cfg)
	}

	r.summary()
	if r.fails > 0 {
		return 1
	}
	return 0
}

func checkDatabase(r *doctorReport, cfg *config.Config, skipIntegrity bool) {
	if _, err := os.Stat(cfg.DatabasePath); err != nil {
		r.pass("no database yet: this will be a fresh install (security mode defaults to enforce)")
		return
	}
	conn, err := sql.Open("sqlite", "file:"+cfg.DatabasePath+"?mode=ro")
	if err != nil {
		r.fail("cannot open database read-only: %v", err)
		return
	}
	defer conn.Close()
	if skipIntegrity {
		r.pass("database integrity check skipped (the server verifies it on every start)")
	} else if err := quickCheck(conn); err != nil {
		r.fail("database integrity check failed: %v", err)
		return
	} else {
		r.pass("database integrity ok")
	}
	var nodes, apps int
	_ = conn.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&nodes)
	_ = conn.QueryRow("SELECT COUNT(*) FROM apps").Scan(&apps)
	r.pass("%d node(s), %d app(s)", nodes, apps)
	var version sql.NullInt64
	_ = conn.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	if int(version.Int64) < db.LatestSchemaVersion() {
		r.note("schema version %d is behind %d: the next start migrates it (a backup is taken first). Nothing to do", version.Int64, db.LatestSchemaVersion())
	} else {
		r.pass("schema is current (version %d)", version.Int64)
	}
	checkIdentity(r, conn, cfg)
	checkSecrets(r, conn, cfg)

	// Restarting mid-deployment interrupts the job (it is marked failed on the next start)
	if jrows, err := conn.Query(`SELECT type, status FROM jobs WHERE status IN ('pending', 'running')`); err == nil {
		var active []string
		for jrows.Next() {
			var typ, status string
			if jrows.Scan(&typ, &status) == nil {
				active = append(active, typ+" ("+status+")")
			}
		}
		jrows.Close()
		if len(active) > 0 {
			r.warn("%d job(s) in progress: %s", len(active), strings.Join(active, ", "))
			r.fix("wait for them to finish before restarting")
		} else {
			r.pass("no deployment jobs in progress")
		}
	}
	if matches, _ := filepath.Glob(cfg.DatabasePath + ".bak-*"); len(matches) > 0 {
		r.pass("%d database backup(s) present", len(matches))
	}
}

func checkURL(r *doctorReport, url, what string) {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		r.fail("%s unreachable: %v", what, err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		r.fail("%s returned %d", what, resp.StatusCode)
		return
	}
	r.pass("%s reachable", what)
}

// auditDeployedApps reports which existing apps the current policy would block, so an install can
// be moved to enforce mode without breaking anything that is already running.
func auditDeployedApps(r *doctorReport, cfg *config.Config) {
	entries, err := os.ReadDir(cfg.AppsDir)
	if err != nil {
		r.fail("cannot read APPS_DIR: %v", err)
		return
	}
	checked, clean := 0, 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(cfg.AppsDir, e.Name(), docker.ComposeFileName)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		checked++
		res, err := validation.AuditComposeFile(path, validation.NewSecurityConfig(cfg, e.Name()))
		if err != nil {
			r.warn("%s: cannot read compose file: %v", e.Name(), err)
			continue
		}
		if len(res.Hard) == 0 && len(res.Soft) == 0 {
			clean++
			continue
		}
		for _, h := range res.Hard {
			r.fail("%s: BLOCKED in every mode: %s", e.Name(), h)
		}
		for _, s := range res.Soft {
			r.warn("%s: blocked once SECURITY_MODE=enforce: %s (fix the compose file, or add the path to ALLOWED_VOLUME_PATHS)", e.Name(), s)
		}
	}
	if checked == 0 {
		r.pass("no deployed apps found")
		return
	}
	r.pass("%d of %d app(s) already comply", clean, checked)
}

// checkSecrets reports how stored credentials are held and whether every encrypted value can
// actually be read with the key that is available. It never creates a key.
func checkSecrets(r *doctorReport, conn *sql.DB, cfg *config.Config) {
	sealed, plain := 0, 0
	var sealedValues []string
	for _, col := range db.SecretColumns {
		rows, err := conn.Query(`SELECT ` + col.Column + ` FROM ` + col.Table +
			` WHERE ` + col.Column + ` IS NOT NULL AND ` + col.Column + ` != ''`)
		if err != nil {
			continue
		}
		for rows.Next() {
			var v string
			if rows.Scan(&v) != nil {
				continue
			}
			if secrets.IsSealed(v) {
				sealed++
				sealedValues = append(sealedValues, v)
			} else {
				plain++
			}
		}
		rows.Close()
	}

	want := "plaintext (ENCRYPT_SECRETS_AT_REST is off)"
	if cfg.Security.EncryptSecretsAtRest {
		want = "encrypted (ENCRYPT_SECRETS_AT_REST is on)"
	}
	r.pass("%d stored secret(s) encrypted, %d plaintext; configured state: %s", sealed, plain, want)
	if plain > 0 && !cfg.Security.EncryptSecretsAtRest {
		r.note("stored secrets are not encrypted at rest. Optional: it protects a copied database file, not a running server")
		r.fix("save SETTINGS_ENCRYPTION_KEY somewhere off this machine, then: selfhostlyctl upgrade --set ENCRYPT_SECRETS_AT_REST=true")
	}
	if sealed != 0 && plain != 0 || (sealed > 0) != cfg.Security.EncryptSecretsAtRest && sealed+plain > 0 {
		r.note("stored secrets do not match ENCRYPT_SECRETS_AT_REST: the next start converts them by itself")
	}
	if sealed == 0 {
		return
	}

	key := cfg.Security.SettingsEncryptionKey
	if key == "" {
		b, err := os.ReadFile(filepath.Join(filepath.Dir(cfg.DatabasePath), constants.SecretsKeyFileName))
		if err != nil {
			r.fail("encrypted secrets exist but no key is available: restore secrets.key from your backup or set SETTINGS_ENCRYPTION_KEY")
			return
		}
		key = strings.TrimSpace(string(b))
	}
	box, err := secrets.NewBox(key, false)
	if err != nil {
		r.fail("cannot use the encryption key: %v", err)
		return
	}
	unreadable := 0
	for _, v := range sealedValues {
		if _, err := box.Open(v); err != nil {
			unreadable++
		}
	}
	if unreadable > 0 {
		r.fail("%d encrypted secret(s) cannot be decrypted with the available key", unreadable)
	} else {
		r.pass("all encrypted secrets are readable with the available key")
	}
}

// checkIdentity catches the failure that is hardest to spot after a restart: the node ID this
// process will use no longer matches the one recorded in the database. That happens when NODE_ID is
// dropped from .env, or a saved identity file is lost, and it would orphan every app.
func checkIdentity(r *doctorReport, conn *sql.DB, cfg *config.Config) {
	var total int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM nodes`).Scan(&total); err != nil || total == 0 {
		return
	}
	if cfg.Node.IsPrimary {
		var primaryID string
		if err := conn.QueryRow(`SELECT id FROM nodes WHERE is_primary = 1 LIMIT 1`).Scan(&primaryID); err != nil {
			return
		}
		if primaryID == cfg.Node.ID {
			r.pass("node identity matches the database (%s)", primaryID)
		} else {
			r.fail("this start would use node id %s but the database's primary is %s (%s). "+
				"Set NODE_ID=%s in .env before restarting, or every app will be orphaned",
				cfg.Node.ID, primaryID, cfg.Sources["node_id"], primaryID)
		}
		return
	}
	var n int
	_ = conn.QueryRow(`SELECT COUNT(*) FROM nodes WHERE id = ?`, cfg.Node.ID).Scan(&n)
	if n == 0 {
		r.warn("this secondary's node id %s is not in its database: it will register as a new node (%s)", cfg.Node.ID, cfg.Sources["node_id"])
	} else {
		r.pass("node identity matches the database")
	}
}

// quickCheck runs SQLite's quick_check. Against a database a running server is writing in WAL mode
// a single read can land mid-write and report a false problem, so a failure is retried before it is
// believed.
func quickCheck(conn *sql.DB) error {
	var last string
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Second)
		}
		var check string
		if err := conn.QueryRow("PRAGMA quick_check").Scan(&check); err != nil {
			last, lastErr = "", err
			continue
		}
		if check == "ok" {
			return nil
		}
		last, lastErr = check, nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("%s (failed on 3 attempts)", last)
}

// readPrimaryID reads the primary node's ID from the database without modifying it
func readPrimaryID(dbPath string) (string, bool) {
	if _, err := os.Stat(dbPath); err != nil {
		return "", false
	}
	conn, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return "", false
	}
	defer conn.Close()
	var id string
	if err := conn.QueryRow(`SELECT id FROM nodes WHERE is_primary = 1 LIMIT 1`).Scan(&id); err != nil {
		return "", false
	}
	return id, true
}

// checkDockerAccess states the facts behind the most common host setup problem: the user this
// process runs as must be in the group that owns the Docker socket. Compose cannot detect that
// group, so the check names the exact value to configure.
func checkDockerAccess(r *doctorReport) {
	host := os.Getenv("DOCKER_HOST")
	if host != "" && !strings.HasPrefix(host, "unix://") {
		r.pass("Docker is reached over %s (not the local socket)", host)
		return
	}
	sock := strings.TrimPrefix(host, "unix://")
	if sock == "" {
		sock = "/var/run/docker.sock"
	}
	st, err := os.Stat(sock)
	if err != nil {
		r.fail("Docker socket %s not found: mount it into this container, or set DOCKER_HOST", sock)
		return
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	uid, gid := os.Getuid(), os.Getgid()
	groups, _ := os.Getgroups()
	r.pass("running as uid %d gid %d; %s is owned by group %d", uid, gid, sock, sys.Gid)
	member := uid == 0 || uint32(gid) == sys.Gid
	for _, g := range groups {
		if uint32(g) == sys.Gid {
			member = true
		}
	}
	if !member && st.Mode().Perm()&0o006 == 0 {
		r.fail("this process is not in the group that owns the Docker socket (%d), so it cannot use Docker. "+
			"Set DOCKER_GID=%d in .env and recreate the primary", sys.Gid, sys.Gid)
	}
}
