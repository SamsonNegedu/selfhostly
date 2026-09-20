package ctl

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fault injection for `upgrade`: break one step at a time and check that the old version is what runs afterwards.
// Each test states the invariant it protects. `answer` puts a fault in front of the normal fake stack.

// answer replaces the reply to every command that matches, and leaves the rest to the stack
func answer(sc *scripted, match func(cmd string) bool, out string, fail bool) {
	orig := sc.handle
	sc.handle = func(cmd string) (string, bool) {
		if match(cmd) {
			return out, fail
		}
		return orig(cmd)
	}
}

func contains(sub string) func(string) bool {
	return func(cmd string) bool { return strings.Contains(cmd, sub) }
}

func envText(t *testing.T, a *App) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(a.Dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// assertRolledBack checks the old images were retagged and both services were recreated from them
func assertRolledBack(t *testing.T, sc *scripted, out string) {
	t.Helper()
	for _, sub := range []string{
		"docker tag selfhostly-rollback/primary:",
		"--force-recreate primary",
		"--force-recreate gateway",
	} {
		if sc.index(sub) < 0 {
			t.Fatalf("rollback did not run %q\n%v\n%s", sub, sc.calls, out)
		}
	}
	if !strings.Contains(out, "rollback complete") {
		t.Fatalf("rollback must say it completed:\n%s", out)
	}
}

func TestFaultPullFailureLeavesTheRunningSystemUntouched(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true})
	answer(sc, contains(" pull "), "network unreachable", true)
	err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--pull", "--health-timeout", "2")
	if err == nil || !strings.Contains(err.Error(), "could not pull") {
		t.Fatalf("got %v\n%s", err, out)
	}
	// the error message promises nothing was changed: an offline pull must not restart the primary or the gateway
	for _, sub := range []string{"stop primary", "--force-recreate", " up -d"} {
		if sc.index(sub) >= 0 {
			t.Fatalf("a failed pull must not touch running containers, but ran %q\n%v", sub, sc.calls)
		}
	}
	if !strings.Contains(envText(t, a), "KEEP=1") {
		t.Fatalf("the settings edit must be undone, got %q", envText(t, a))
	}
}

func TestFaultPrimaryCannotBeRecreatedRollsBack(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true})
	answer(sc, contains("up -d --no-deps primary"), "port is already allocated", true)
	if err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--health-timeout", "2"); err == nil {
		t.Fatalf("expected failure\n%s", out)
	}
	assertRolledBack(t, sc, out.String())
	if !strings.Contains(envText(t, a), "KEEP=1") {
		t.Fatalf("the settings must be restored, got %q", envText(t, a))
	}
}

func TestFaultPrimaryRefusesToStartRollsBack(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true})
	answer(sc, contains("docker logs"), `{"issue":"encrypted data but no key"} refusing to start`, false)
	err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--health-timeout", "2")
	if err == nil || !strings.Contains(err.Error(), "refused to start") {
		t.Fatalf("got %v\n%s", err, out)
	}
	assertRolledBack(t, sc, out.String())
}

func TestFaultPrimaryNeverHealthyRollsBackAndKeepsTheDatabase(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: false})
	rolling := false
	orig := sc.handle
	sc.handle = func(cmd string) (string, bool) {
		if strings.Contains(cmd, "--force-recreate") {
			rolling = true
		}
		// the old image is healthy again once the rollback has recreated it
		if rolling && strings.Contains(cmd, "State.Health") {
			return "healthy\n", false
		}
		return orig(cmd)
	}
	db := filepath.Join(a.Dir, "data", "selfhostly.db")
	os.MkdirAll(filepath.Dir(db), 0o700)
	os.WriteFile(db, []byte("precious"), 0o600)

	if err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--health-timeout", "2"); err == nil {
		t.Fatalf("expected failure\n%s", out)
	}
	assertRolledBack(t, sc, out.String())
	// a rollback keeps the migrated database: the older backend runs on it, and a copy would drop newer writes
	if got, _ := os.ReadFile(db); string(got) != "precious" {
		t.Fatalf("the database must not be rewritten by a rollback, got %q", got)
	}
	last, _ := os.ReadFile(filepath.Join(a.Dir, ".upgrade", "last"))
	copyPath := filepath.Join(a.Dir, ".upgrade", strings.TrimSpace(string(last)), "backup", "selfhostly.db")
	if got, _ := os.ReadFile(copyPath); string(got) != "precious" {
		t.Fatalf("the rollback point must hold a copy of the database, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(a.Dir, ".upgrade", ".lock")); err == nil {
		t.Fatal("the lock must be released after a failed upgrade, or the next one is refused")
	}
}

func TestFaultGatewayNeverHealthyRollsBackTheWholeStack(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true})
	rolling := false
	orig := sc.handle
	sc.handle = func(cmd string) (string, bool) {
		if strings.Contains(cmd, "--force-recreate") {
			rolling = true
		}
		if !rolling && strings.Contains(cmd, "State.Health") && strings.HasSuffix(cmd, " gc") {
			return "starting\n", false
		}
		return orig(cmd)
	}
	err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--health-timeout", "2")
	if err == nil || !strings.Contains(err.Error(), "gateway did not become healthy") {
		t.Fatalf("got %v\n%s", err, out)
	}
	// the primary already runs the new image at this point, so it must be rolled back with the gateway
	assertRolledBack(t, sc, out.String())
	if !strings.Contains(envText(t, a), "KEEP=1") {
		t.Fatalf("the settings must be restored, got %q", envText(t, a))
	}
}

func TestFaultGatewayCannotBeRecreatedRollsBackTheWholeStack(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true})
	answer(sc, contains("up -d --no-deps gateway"), "boom", true)
	err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--health-timeout", "2")
	if err == nil || !strings.Contains(err.Error(), "gateway could not be recreated") {
		t.Fatalf("got %v\n%s", err, out)
	}
	assertRolledBack(t, sc, out.String())
}

func TestFaultRollbackDoesNotClaimSuccessWhenTheOldImageCannotBeRestored(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: false})
	rolling := false
	orig := sc.handle
	sc.handle = func(cmd string) (string, bool) {
		if strings.Contains(cmd, "--force-recreate") {
			rolling = true
		}
		// the rollback's retag, not the one that saved the point
		if strings.HasPrefix(cmd, "docker tag selfhostly-rollback/") {
			return "No such image", true
		}
		if rolling && strings.Contains(cmd, "State.Health") {
			return "healthy\n", false
		}
		return orig(cmd)
	}
	err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--health-timeout", "2")
	if err == nil {
		t.Fatalf("expected failure\n%s", out)
	}
	// without the retag the recreate starts the NEW image again, which is healthy here: that is not a rollback
	if strings.Contains(out.String(), "rollback complete") {
		t.Fatalf("a rollback that could not restore the old image must not report success:\n%s", out)
	}
	if !strings.Contains(out.String(), "rollback finished with problems") {
		t.Fatalf("it must say the rollback failed:\n%s", out)
	}
}

func TestFaultRollbackPutsBackTheComposeFileAndSettingsThatConfigureChanged(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: false})
	rolling := false
	orig := sc.handle
	sc.handle = func(cmd string) (string, bool) {
		if strings.Contains(cmd, "--force-recreate") {
			rolling = true
		}
		if rolling && strings.Contains(cmd, "State.Health") {
			return "healthy\n", false
		}
		return orig(cmd)
	}
	compose := filepath.Join(a.Dir, "docker-compose.prod.yml")
	before, _ := os.ReadFile(compose)
	err := a.upgrade(t.Context(), upgradeOpts{
		project: "p", healthTimeout: 2,
		configure: func(_ context.Context, u *upgrader) error {
			if err := writeFileKeepingMode(compose, []byte("services: {changed: {}}\n")); err != nil {
				return err
			}
			return u.env.Set("BACKEND_IMAGE", "new@sha256:x")
		},
	})
	if err == nil {
		t.Fatalf("expected failure\n%s", out)
	}
	if got, _ := os.ReadFile(compose); string(got) != string(before) {
		t.Fatalf("the compose file must be restored, got %q", got)
	}
	if strings.Contains(envText(t, a), "BACKEND_IMAGE") {
		t.Fatalf("the settings must be restored, got %q", envText(t, a))
	}
}

func TestFaultAConfigureFailureChangesNothingAndLeavesNoRollbackPoint(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true})
	compose := filepath.Join(a.Dir, "docker-compose.prod.yml")
	before, _ := os.ReadFile(compose)
	err := a.upgrade(t.Context(), upgradeOpts{
		project: "p", healthTimeout: 2,
		configure: func(_ context.Context, u *upgrader) error {
			_ = writeFileKeepingMode(compose, []byte("half written"))
			_ = u.env.Set("BACKEND_IMAGE", "new@sha256:x")
			return errors.New("the compose change was not approved")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "not approved") {
		t.Fatalf("got %v\n%s", err, out)
	}
	for _, sub := range []string{"stop primary", " pull ", "--force-recreate", " up -d"} {
		if sc.index(sub) >= 0 {
			t.Fatalf("a failed configure must not reach %q", sub)
		}
	}
	if got, _ := os.ReadFile(compose); string(got) != string(before) {
		t.Fatalf("the compose file must be restored, got %q", got)
	}
	if strings.Contains(envText(t, a), "BACKEND_IMAGE") {
		t.Fatalf("the settings must be restored, got %q", envText(t, a))
	}
	if _, err := os.Stat(filepath.Join(a.Dir, ".upgrade", "last")); err == nil {
		t.Fatal("a run that changed nothing must not leave a rollback point")
	}
}

// the new primary is healthy but doctor says it will not work: an update started from the UI has nobody watching, so it goes back
func TestFaultStrictDoctorFailureAfterTheNewPrimaryStartsRollsBack(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true})
	answer(sc, func(cmd string) bool {
		return strings.Contains(cmd, "exec -T primary ./selfhostly doctor --offline") && !strings.Contains(cmd, "--skip-db-integrity")
	}, "  FAIL  the database is not readable\n", true)
	err := a.upgrade(t.Context(), upgradeOpts{project: "p", healthTimeout: 2, strict: true,
		configure: func(_ context.Context, u *upgrader) error { return u.env.Set("BACKEND_IMAGE", "new@sha256:x") }})
	var failure *upgradeFailure
	if !errors.As(err, &failure) || !failure.rolledBack || !strings.Contains(failure.msg, "doctor reports failures") {
		t.Fatalf("got %v\n%s", err, out)
	}
	assertRolledBack(t, sc, out.String())
	if strings.Contains(envText(t, a), "BACKEND_IMAGE") {
		t.Fatalf("the settings must be restored, got %q", envText(t, a))
	}
}

func TestADoctorFailureOnTheCommandLineIsOnlyReported(t *testing.T) {
	// an operator watching `selfhostlyctl upgrade` decides for themselves: only the UI's unattended mode rolls back
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true})
	answer(sc, func(cmd string) bool {
		return strings.Contains(cmd, "exec -T primary ./selfhostly doctor --offline") && !strings.Contains(cmd, "--skip-db-integrity")
	}, "  FAIL  something\n", true)
	if err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--health-timeout", "2"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if sc.index("--force-recreate") >= 0 {
		t.Fatal("a doctor failure must not roll back an attended upgrade")
	}
}

func TestFaultStrictDatabaseCopyFailureRollsBackBeforeAnythingChanges(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true})
	// a directory where the database file should be: it exists, and copying it fails
	os.MkdirAll(filepath.Join(a.Dir, "data", "selfhostly.db"), 0o700)
	err := a.upgrade(t.Context(), upgradeOpts{project: "p", healthTimeout: 2, strict: true,
		configure: func(_ context.Context, u *upgrader) error { return u.env.Set("BACKEND_IMAGE", "new@sha256:x") }})
	var failure *upgradeFailure
	if !errors.As(err, &failure) || !strings.Contains(failure.msg, "could not copy the database") {
		t.Fatalf("got %v\n%s", err, out)
	}
	if sc.index("up -d --no-deps primary") >= 0 && sc.index("up -d --no-deps primary") < sc.index("--force-recreate") {
		t.Fatal("the new primary must never start without a database copy")
	}
	if strings.Contains(envText(t, a), "BACKEND_IMAGE") {
		t.Fatalf("the settings must be restored, got %q", envText(t, a))
	}
}

func TestADatabaseCopyFailureOnTheCommandLineIsOnlyAWarning(t *testing.T) {
	a, _, out := upgradeApp(t, &stack{primaryHealthy: true})
	os.MkdirAll(filepath.Join(a.Dir, "data", "selfhostly.db"), 0o700)
	if err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--health-timeout", "2"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out.String(), "warning: could not copy the database") {
		t.Fatalf("it must still say so:\n%s", out)
	}
}
