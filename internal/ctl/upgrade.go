package ctl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	stateRoot        = ".upgrade"
	keepRollbackSets = 5
)

type upgradeOpts struct {
	sets                 []string
	pull, noPull         bool
	withFrontend, force  bool
	noRollback, rollback bool
	rollbackTo, project  string
	healthTimeout        int
	compose              []string
}

func (a *App) upgradeCmd() *cobra.Command {
	var o upgradeOpts
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade or restart safely: rollback point, dry start, health checks, automatic rollback",
		Long: `Steps, in order:
  1. check nothing is deploying and that the compose file renders
  2. save the running images under a rollback tag, so a pull cannot lose them
  3. pull new images (unless you only changed settings), then DRY START: run the new image's doctor in a
     throwaway container with your real settings and data, so a start that would fail is caught first
  4. stop the primary briefly and copy the database
  5. recreate the primary, wait until healthy, run doctor
  6. recreate the gateway and wait
  7. confirm every other container that was running still is
If the primary does not become healthy it rolls back by itself. Your apps are never touched.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if o.rollback {
				return a.rollbackCmd(c.Context(), o)
			}
			return a.upgrade(c.Context(), o)
		},
	}
	f := cmd.Flags()
	f.StringArrayVar(&o.sets, "set", nil, "write KEY=VALUE into the settings file and apply it (repeatable)")
	f.BoolVar(&o.pull, "pull", false, "pull images even when --set is used")
	f.BoolVar(&o.noPull, "no-pull", false, "do not pull (use images already present)")
	f.BoolVar(&o.withFrontend, "with-frontend", false, "also recreate the frontend")
	f.BoolVar(&o.force, "force", false, "continue even if a deployment is running or the dry start found problems")
	f.BoolVar(&o.noRollback, "no-rollback", false, "leave a failed upgrade in place for inspection")
	f.BoolVar(&o.rollback, "rollback", false, "restore the state saved by the last run")
	f.StringVar(&o.rollbackTo, "rollback-to", "", "with --rollback: restore this saved state instead of the last")
	f.IntVar(&o.healthTimeout, "health-timeout", 90, "seconds to wait for health")
	f.StringArrayVar(&o.compose, "compose", nil, "compose file (repeatable; default docker-compose.prod.yml)")
	f.StringVar(&o.project, "project", "", "compose project name (default: detected)")
	return cmd
}

type upgrader struct {
	a        *App
	o        upgradeOpts
	dc       compose
	env      EnvFile
	services []string
	timeout  time.Duration
	prefix   string
}

// resolveCompose decides which compose file(s) and project to act on. Explicit flags win. Otherwise the
// files the running install was started from are read from Docker's own labels, so an install whose
// file is called anything at all needs no flag. Only with nothing running does it default to
// docker-compose.prod.yml.
func (a *App) resolveCompose(ctx context.Context, o *upgradeOpts) error {
	if len(o.compose) > 0 {
		return nil
	}
	files, project, err := a.detectLiveCompose(ctx)
	if err != nil {
		return err
	}
	if len(files) > 0 {
		o.compose = files
		if o.project == "" {
			o.project = project
		}
		a.say("using the compose file your running install was started from: %s", strings.Join(files, ", "))
	}
	return nil
}

func (a *App) newUpgrader(o upgradeOpts) *upgrader {
	files := append([]string(nil), o.compose...)
	if len(files) == 0 {
		files = []string{primaryCompose}
	}
	for i := range files {
		files[i] = a.dirPath(files[i])
	}
	env := EnvFile{a.envPath(a.EnvFile)}
	services := []string{"primary", "gateway"}
	if o.withFrontend {
		services = append(services, "frontend")
	}
	if o.healthTimeout <= 0 {
		o.healthTimeout = 90
	}
	return &upgrader{
		a: a, o: o, env: env, services: services, timeout: time.Duration(o.healthTimeout) * time.Second,
		dc:     compose{a: a, project: o.project, env: env.Path, files: files},
		prefix: firstNonEmpty(os.Getenv("SFH_ROLLBACK_PREFIX"), "selfhostly-rollback"),
	}
}

func (a *App) stateRoot() string { return a.dirPath(stateRoot) }

func (u *upgrader) waitHealthy(ctx context.Context, c string) bool {
	return u.a.waitHealthy(ctx, c, u.timeout, 2*time.Second, false)
}

// ---------- rollback ----------------------------------------------------------------------------

func (a *App) rollbackCmd(ctx context.Context, o upgradeOpts) error {
	if err := a.resolveCompose(ctx, &o); err != nil {
		return err
	}
	u := a.newUpgrader(o)
	stamp := o.rollbackTo
	if stamp == "" {
		if b, err := os.ReadFile(filepath.Join(a.stateRoot(), "last")); err == nil {
			stamp = strings.TrimSpace(string(b))
		}
	}
	dir := filepath.Join(a.stateRoot(), stamp)
	if st, err := os.Stat(dir); stamp == "" || err != nil || !st.IsDir() {
		return fmt.Errorf("no saved state to roll back to (looked in %s)", a.stateRoot())
	}
	return u.doRollback(ctx, stamp, dir)
}

func (u *upgrader) doRollback(ctx context.Context, stamp, dir string) error {
	a := u.a
	a.say("rolling back to the state saved as %s", stamp)
	if b, err := os.ReadFile(filepath.Join(dir, "env")); err == nil {
		if err := os.WriteFile(u.env.Path, b, 0o600); err != nil {
			return err
		}
		a.say("  restored %s", u.env.Path)
	}
	if f, err := os.Open(filepath.Join(dir, "images")); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			parts := strings.Split(sc.Text(), "|")
			if len(parts) < 2 || parts[0] == "" {
				continue
			}
			svc, image := parts[0], parts[1]
			if strings.Contains(image, "@sha256:") {
				a.say("  %s is pinned by digest (%s): nothing to retag; change the *_IMAGE line in %s to go back", svc, image, u.env.Path)
				continue
			}
			if _, _, err := a.Run.Output(ctx, "docker", "tag", u.prefix+"/"+svc+":"+stamp, image); err == nil {
				a.say("  %s image restored to its previous build", svc)
			}
		}
		_ = f.Close()
	}
	ok := true
	for _, s := range u.services {
		if _, _, err := u.dc.out(ctx, "up", "-d", "--no-deps", "--force-recreate", s); err != nil {
			ok = false
		}
		if c := u.dc.cid(ctx, s); c != "" && u.waitHealthy(ctx, c) {
			a.say("  %s healthy", s)
		} else {
			a.say("  %s did NOT become healthy", s)
			ok = false
		}
	}
	if !ok {
		a.say("rollback finished with problems: check 'docker compose logs'")
		return errors.New("rollback finished with problems")
	}
	a.say("rollback complete")
	return nil
}

// ---------- upgrade -----------------------------------------------------------------------------

var errAborted = errors.New("aborted: the running system was not changed")

func (a *App) upgrade(ctx context.Context, o upgradeOpts) (err error) {
	if err := a.resolveCompose(ctx, &o); err != nil {
		return err
	}
	u := a.newUpgrader(o)
	pull := !o.noPull && (o.pull || len(o.sets) == 0)
	for _, kv := range o.sets {
		if !strings.Contains(kv, "=") {
			return fmt.Errorf("--set expects KEY=VALUE, got: %s", kv)
		}
	}

	// 1. preflight
	a.step("1/7: checking the install")
	for _, f := range u.dc.files {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("compose file not found: %s", f)
		}
	}
	if _, se, err := u.dc.out(ctx, "config", "-q"); err != nil {
		return fmt.Errorf("the compose file does not render (fix the message below)\n%s", strings.TrimSpace(se))
	}
	primary := u.dc.cid(ctx, "primary")
	if primary == "" {
		return errors.New("no running 'primary' service found in this project. Is it up? (use --project NAME if it uses a custom name)")
	}
	if u.dc.project == "" {
		out, _, _ := a.Run.Output(ctx, "docker", "inspect", "--format", `{{index .Config.Labels "com.docker.compose.project"}}`, primary)
		u.dc.project = strings.TrimSpace(out)
	}
	a.say("  project: %s, services: %s", u.dc.project, strings.Join(u.services, " "))

	dataDir := a.dirPath(firstNonEmpty(u.env.Get("DATA_DIR"), "./data"))
	db := filepath.Join(dataDir, "selfhostly.db")
	_, dbErr := os.Stat(db)
	if dbErr != nil {
		a.say("  note: %s not found on the host; the database copy will be skipped", db)
	}
	if err := u.checkJobs(ctx, db, dbErr == nil); err != nil {
		return err
	}
	if err := u.checkSettingsAreRead(ctx); err != nil {
		return err
	}
	othersBefore := u.otherContainers(ctx)
	a.say("  %d other container(s) running (your apps); they will not be touched", len(othersBefore))

	a.step("plan%s", map[bool]string{true: ": DRY RUN, nothing will change", false: ""}[a.DryRun])
	for _, kv := range o.sets {
		a.say("  set: %s", strings.SplitN(kv, "=", 2)[0])
	}
	a.say("  pull images: %v", pull)
	a.say("  recreate, in order: %s", strings.Join(u.services, " "))
	a.say("  database copy: yes (primary is stopped for a few seconds)")
	a.say("  on failure: %s", map[bool]string{true: "leave as is (--no-rollback)", false: "roll back automatically"}[o.noRollback])
	if a.DryRun {
		a.say("\ndry run complete")
		return nil
	}
	if !a.confirm("Proceed?", false) {
		a.say("cancelled, nothing changed")
		return nil
	}

	if err := os.MkdirAll(a.stateRoot(), 0o700); err != nil {
		return err
	}
	lock := filepath.Join(a.stateRoot(), ".lock")
	if err := os.Mkdir(lock, 0o700); err != nil {
		return fmt.Errorf("another upgrade is running (remove %s if it is stale)", lock)
	}
	stamp := a.now()
	dir := filepath.Join(a.stateRoot(), stamp)
	if err := os.MkdirAll(filepath.Join(dir, "backup"), 0o700); err != nil {
		_ = os.Remove(lock)
		return err
	}
	primaryStopped, finished := false, false
	defer func() {
		_ = os.Remove(lock)
		if !finished && primaryStopped {
			a.say("the primary was stopped before the command finished: starting it again")
			_, _, _ = u.dc.out(context.Background(), "start", "primary")
		}
	}()

	// 2. rollback point
	a.step("2/7: saving a rollback point (%s)", stamp)
	if err := u.saveRollbackPoint(ctx, stamp, dir); err != nil {
		return err
	}
	fail := func(msg string) error {
		a.say("\nFAILED: %s", msg)
		finished = true
		if o.noRollback {
			a.say("left as is. Roll back with: selfhostlyctl upgrade --rollback")
		} else {
			_ = u.doRollback(ctx, stamp, dir)
		}
		return errors.New(msg)
	}
	for _, kv := range o.sets {
		k, v, _ := strings.Cut(kv, "=")
		if err := u.env.Set(k, v); err != nil {
			return err
		}
		a.say("  set %s", k)
	}

	// 3. pull and dry start
	a.step("3/7: images")
	if pull {
		if err := u.dc.pass(ctx, append([]string{"pull"}, u.services...)...); err != nil {
			return fail("could not pull images (nothing was changed on the running system)")
		}
	} else {
		a.say("  not pulling")
	}
	a.say("  dry start: checking that the new image will start with this configuration")
	if _, _, err := u.dc.out(ctx, "run", "--rm", "--no-deps", "-T", "--entrypoint", "sh", "primary", "-c", `grep -q "selfhostly doctor" ./selfhostly`); err != nil {
		a.say("  this image has no 'doctor' command (older build): skipping the dry start")
	} else {
		// the live server writes the database from another container, so its integrity is not re-checked here
		so, se, err := u.dc.out(ctx, "run", "--rm", "--no-deps", "-T", "primary", "./selfhostly", "doctor", "--offline", "--skip-db-integrity")
		if err == nil {
			a.say("  dry start: ok")
		} else {
			a.say("  dry start found problems:")
			for _, l := range findingLines(so + se) {
				a.say("    %s", l)
			}
			if !o.force {
				// nothing has been stopped or recreated yet, so only the settings edits need undoing
				if b, err := os.ReadFile(filepath.Join(dir, "env")); err == nil {
					_ = os.WriteFile(u.env.Path, b, 0o600)
					a.say("  restored %s", u.env.Path)
				}
				finished = true
				a.say("\nABORTED before any change to the running system: fix the problems above, or pass --force")
				return errAborted
			}
			a.say("  continuing anyway (--force)")
		}
	}

	// 4. database copy
	a.step("4/7: database copy")
	_, _, _ = u.dc.out(ctx, "stop", "primary")
	primaryStopped = true
	if dbErr == nil {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			if err := copyFile(db+suffix, filepath.Join(dir, "backup", filepath.Base(db)+suffix)); err != nil && suffix == "" {
				a.say("  warning: could not copy the database: %v", err)
			}
		}
		a.say("  copied to %s (the server also makes its own copy before any schema change)", filepath.Join(dir, "backup"))
	} else {
		a.say("  skipped")
	}

	// 5. primary
	a.step("5/7: primary")
	if _, _, err := u.dc.out(ctx, "up", "-d", "--no-deps", "primary"); err != nil {
		return fail("the primary could not be recreated")
	}
	primaryStopped = false
	pc := u.dc.cid(ctx, "primary")
	a.say("  waiting up to %ds for it to be healthy", o.healthTimeout)
	if !u.waitHealthy(ctx, pc) {
		tail := strings.Join(strings.Fields(a.logs(ctx, pc, 3)), " ")
		return fail("the primary did not become healthy: " + truncate(tail, 300))
	}
	if logs := a.logs(ctx, pc, 0); strings.Contains(logs, "refusing to start") {
		issue := ""
		if m := regexp.MustCompile(`"issue":"[^"]*`).FindString(logs); m != "" {
			issue = m
		}
		return fail("the primary refused to start: " + issue)
	}
	a.say("  healthy")
	so, se, derr := u.dc.out(ctx, "exec", "-T", "primary", "./selfhostly", "doctor", "--offline")
	switch lines := findingLines(so + se); {
	case derr == nil:
		a.say("  doctor: ok")
	case len(lines) > 0:
		a.say("  doctor reported:")
		for _, l := range lines {
			a.say("    %s", l)
		}
	default:
		a.say("  doctor is not available in the previous build (expected on the first upgrade)")
	}

	// 6. gateway (and frontend)
	a.step("6/7: gateway")
	for _, s := range u.services[1:] {
		if _, _, err := u.dc.out(ctx, "up", "-d", "--no-deps", s); err != nil {
			return fail(s + " could not be recreated")
		}
		if !u.waitHealthy(ctx, u.dc.cid(ctx, s)) {
			return fail(s + " did not become healthy")
		}
		a.say("  %s healthy", s)
	}

	// 7. apps
	a.step("7/7: checking your apps were not disturbed")
	after := setOf(u.otherContainers(ctx))
	var missing []string
	for _, n := range othersBefore {
		if !after[n] {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		a.say("  WARNING: these containers were running before and are not now (not caused by this command, which never touches apps):")
		for _, m := range missing {
			a.say("    %s", m)
		}
	} else {
		a.say("  every other container is still running")
	}
	u.prune()

	finished = true
	a.say("\ndone. Rollback point: %s   (selfhostlyctl upgrade --rollback)", stamp)
	a.say("database copy:       %s", filepath.Join(dir, "backup"))
	a.noteNewerTool()
	return nil
}

func (u *upgrader) checkJobs(ctx context.Context, db string, haveDB bool) error {
	a := u.a
	if haveDB {
		out, _, err := a.Run.Output(ctx, "sqlite3", "file:"+db+"?mode=ro",
			"SELECT type || ' (' || status || ')' FROM jobs WHERE status IN ('pending','running')")
		if err == nil {
			if jobs := strings.TrimSpace(out); jobs != "" {
				a.say("  deployment jobs in progress:")
				for _, j := range strings.Split(jobs, "\n") {
					a.say("    %s", j)
				}
				if !u.o.force {
					return errors.New("a restart would interrupt them. Wait for them to finish, or pass --force")
				}
				return nil
			}
			a.say("  no deployment jobs in progress")
			return nil
		}
	}
	a.say("  cannot check for running jobs (sqlite3 not installed): make sure no app shows 'updating' or 'pending' in the UI")
	return nil
}

// otherContainers lists running containers that belong to other compose projects: the user's apps
func (u *upgrader) otherContainers(ctx context.Context) []string {
	out, _, _ := u.a.Run.Output(ctx, "docker", "ps", "--format", `{{.Names}}|{{.Label "com.docker.compose.project"}}`)
	var names []string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		n, p, ok := strings.Cut(l, "|")
		if ok && n != "" && p != u.dc.project {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

func setOf(v []string) map[string]bool {
	m := map[string]bool{}
	for _, s := range v {
		m[s] = true
	}
	return m
}

func (u *upgrader) saveRollbackPoint(ctx context.Context, stamp, dir string) error {
	a := u.a
	var lines []string
	for _, s := range u.services {
		c := u.dc.cid(ctx, s)
		if c == "" {
			a.say("  %s is not running: skipped", s)
			continue
		}
		image, _, _ := a.Run.Output(ctx, "docker", "inspect", "--format", "{{.Config.Image}}", c)
		imgID, _, _ := a.Run.Output(ctx, "docker", "inspect", "--format", "{{.Image}}", c)
		image, imgID = strings.TrimSpace(image), strings.TrimSpace(imgID)
		if _, _, err := a.Run.Output(ctx, "docker", "tag", imgID, u.prefix+"/"+s+":"+stamp); err != nil {
			return fmt.Errorf("could not save the %s image: %w", s, err)
		}
		lines = append(lines, s+"|"+image+"|"+imgID)
		a.say("  %s: %s", s, image)
	}
	if err := os.WriteFile(filepath.Join(dir, "images"), []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		return err
	}
	if u.env.Exists() {
		if err := copyFile(u.env.Path, filepath.Join(dir, "env")); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(a.stateRoot(), "last"), []byte(stamp+"\n"), 0o600)
}

// prune keeps the last few rollback points and their database copies, so disk use stays bounded
func (u *upgrader) prune() {
	entries, err := os.ReadDir(u.a.stateRoot())
	if err != nil {
		return
	}
	var stamps []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "2") {
			stamps = append(stamps, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(stamps)))
	for i, s := range stamps {
		if i >= keepRollbackSets {
			_ = os.RemoveAll(filepath.Join(u.a.stateRoot(), s))
		}
	}
}

func findingLines(out string) []string {
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "FAIL") || strings.Contains(l, "WARN") {
			lines = append(lines, strings.TrimSpace(l))
		}
	}
	return lines
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// checkSettingsAreRead refuses a --set that the compose file never reads. A value written to the settings
// file only reaches the server if the compose file passes it on, and a hand-edited or older file may not:
// the command would then report success and change nothing. Whether the file reads a variable is decided
// the way compose itself would: render it with and without the variable set, and see if anything differs.
func (u *upgrader) checkSettingsAreRead(ctx context.Context) error {
	if len(u.o.sets) == 0 {
		return nil
	}
	base, _, err := u.dc.outEnv(ctx, nil, "config")
	if err != nil {
		return nil // the file already rendered for the check above; do not block on a second rendering
	}
	var unread []string
	for _, kv := range u.o.sets {
		key, _, _ := strings.Cut(kv, "=")
		probed, _, err := u.dc.outEnv(ctx, map[string]string{key: probeValue}, "config")
		if err == nil && probed == base {
			unread = append(unread, key)
		}
	}
	if len(unread) == 0 {
		return nil
	}
	msg := fmt.Sprintf("your compose file never reads %s, so setting it in %s would change nothing", strings.Join(unread, ", "), u.env.Path)
	if u.o.force {
		u.a.say("  warning: %s (continuing: --force)", msg)
		return nil
	}
	return fmt.Errorf("%s.\n"+
		"  Nothing was changed. Either switch to the current compose file, which reads it from the settings file\n"+
		"  (selfhostlyctl compose write, then selfhostlyctl upgrade --compose docker-compose.prod.yml), or add the\n"+
		"  variable to the service's environment in your file, for example:  %s: ${%s:-}\n"+
		"  Pass --force to write it anyway", msg, unread[0], unread[0])
}
