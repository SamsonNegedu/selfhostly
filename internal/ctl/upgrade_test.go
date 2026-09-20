package ctl

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// scripted answers docker calls with a function, and records them in order
type scripted struct {
	calls  []string
	handle func(cmd string) (out string, fail bool)
}

func (s *scripted) Output(_ context.Context, name string, args ...string) (string, string, error) {
	cmd := strings.Join(append([]string{name}, args...), " ")
	s.calls = append(s.calls, cmd)
	out, fail := s.handle(cmd)
	if fail {
		return out, out, errors.New("exit 1")
	}
	return out, "", nil
}

// OutputEnv records the extra variables in the call, so a handler can tell a probing render from a plain one
func (s *scripted) OutputEnv(ctx context.Context, env map[string]string, name string, args ...string) (string, string, error) {
	for k, v := range env {
		args = append(args, "ENV:"+k+"="+v)
	}
	return s.Output(ctx, name, args...)
}

func (s *scripted) Passthrough(ctx context.Context, name string, args ...string) error {
	_, _, err := s.Output(ctx, name, args...)
	return err
}

func (s *scripted) index(sub string) int {
	for i, c := range s.calls {
		if strings.Contains(c, sub) {
			return i
		}
	}
	return -1
}

type stack struct {
	primaryHealthy  bool
	dryStartFails   bool
	ignoresSettings bool // the compose file never reads any variable: rendering does not change with them
	unchanged       bool // the running containers already match the images and configuration
}

func (st *stack) handle(cmd string) (string, bool) {
	switch {
	case strings.Contains(cmd, "config --hash"):
		return "primary h1\ngateway h1\n", false
	case strings.Contains(cmd, configHashLabel):
		if st.unchanged {
			return "h1\n", false
		}
		return "old\n", false
	case strings.Contains(cmd, "docker image inspect"):
		if st.unchanged {
			return "sha256:abc\n", false
		}
		return "sha256:new\n", false
	case strings.HasSuffix(strings.Fields(cmd)[len(strings.Fields(cmd))-1], "config") || strings.Contains(cmd, " config ENV:"):
		if strings.Contains(cmd, "ENV:") && !st.ignoresSettings {
			return "rendered-with-the-variable-set", false
		}
		return "rendered", false
	case strings.Contains(cmd, "ps -q primary"):
		return "pc\n", false
	case strings.Contains(cmd, "ps -q gateway"):
		return "gc\n", false
	case strings.Contains(cmd, "State.Health"):
		if strings.HasSuffix(cmd, " pc") && !st.primaryHealthy {
			return "starting\n", false
		}
		return "healthy\n", false
	case strings.Contains(cmd, "{{.Config.Image}}"):
		return "ghcr.io/x/backend:latest\n", false
	case strings.Contains(cmd, "{{.Image}}"):
		return "sha256:abc\n", false
	case strings.Contains(cmd, "doctor --offline --skip-db-integrity"):
		return "  FAIL  something is wrong\n", st.dryStartFails
	}
	return "", false
}

func upgradeApp(t *testing.T, st *stack) (*App, *scripted, *bytes.Buffer) {
	t.Helper()
	sc := &scripted{handle: st.handle}
	out := &bytes.Buffer{}
	a := &App{Out: out, Err: out, Run: sc, Dir: t.TempDir(), EnvFile: ".env", Yes: true, NonInteractive: true, Sleep: func(time.Duration) {}}
	os.WriteFile(filepath.Join(a.Dir, "docker-compose.prod.yml"), []byte("services: {}\n"), 0o600)
	os.WriteFile(filepath.Join(a.Dir, ".env"), []byte("SECURITY_MODE=warn\nKEEP=1\n"), 0o600)
	return a, sc, out
}

func TestUpgradeHappyPathOrderAndState(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true})
	if err := runCLI(a, "upgrade", "--project", "p", "--set", "SECURITY_MODE=enforce", "--health-timeout", "4"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	tag, stop, up, gw := sc.index("docker tag sha256:abc"), sc.index("stop primary"), sc.index("up -d --no-deps primary"), sc.index("up -d --no-deps gateway")
	if !(tag >= 0 && tag < stop && stop < up && up < gw) {
		t.Fatalf("wrong order: tag=%d stop=%d up=%d gateway=%d\n%v", tag, stop, up, gw, sc.calls)
	}
	if sc.index(" pull ") >= 0 {
		t.Fatal("--set alone must not pull images")
	}
	env, _ := os.ReadFile(filepath.Join(a.Dir, ".env"))
	if !strings.Contains(string(env), "SECURITY_MODE=enforce") || !strings.Contains(string(env), "KEEP=1") {
		t.Fatalf("env not updated correctly: %q", env)
	}
	if _, err := os.Stat(filepath.Join(a.Dir, ".upgrade", "last")); err != nil {
		t.Fatal("rollback point not recorded")
	}
	if _, err := os.Stat(filepath.Join(a.Dir, ".upgrade", ".lock")); err == nil {
		t.Fatal("lock must be released")
	}
}

func TestUpgradeRollsBackWhenPrimaryNeverHealthy(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: false})
	err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--health-timeout", "2")
	if err == nil {
		t.Fatal("expected failure")
	}
	if sc.index("--force-recreate primary") < 0 || sc.index("docker tag selfhostly-rollback/primary:") < 0 {
		t.Fatalf("expected image restore and recreate:\n%v\n%s", sc.calls, out)
	}
	if env, _ := os.ReadFile(filepath.Join(a.Dir, ".env")); !strings.Contains(string(env), "KEEP=1") {
		t.Fatalf("env must be restored, got %q", env)
	}
}

func TestUpgradeAbortsBeforeStoppingAnythingWhenDryStartFails(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true, dryStartFails: true})
	err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--health-timeout", "2")
	if err == nil || !strings.Contains(out.String(), "ABORTED before any change") {
		t.Fatalf("err=%v\n%s", err, out)
	}
	if sc.index("stop primary") >= 0 {
		t.Fatal("nothing may be stopped when the dry start fails")
	}
	if env, _ := os.ReadFile(filepath.Join(a.Dir, ".env")); !strings.Contains(string(env), "KEEP=1") {
		t.Fatalf("env must be restored, got %q", env)
	}
}

func TestUpgradeDryRunChangesNothing(t *testing.T) {
	a, sc, _ := upgradeApp(t, &stack{primaryHealthy: true})
	a.DryRun = true
	if err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2"); err != nil {
		t.Fatal(err)
	}
	for _, sub := range []string{"stop primary", "docker tag", " up -d", " pull "} {
		if sc.index(sub) >= 0 {
			t.Fatalf("dry run ran %q", sub)
		}
	}
	if env, _ := os.ReadFile(filepath.Join(a.Dir, ".env")); strings.Contains(string(env), "KEEP=2") {
		t.Fatal("dry run must not edit the settings")
	}
}

func TestUpgradeRefusesWhileAnotherIsRunning(t *testing.T) {
	a, _, _ := upgradeApp(t, &stack{primaryHealthy: true})
	os.MkdirAll(filepath.Join(a.Dir, ".upgrade", ".lock"), 0o700)
	if err := runCLI(a, "upgrade", "--project", "p", "--health-timeout", "2"); err == nil || !strings.Contains(err.Error(), "another upgrade") {
		t.Fatalf("got %v", err)
	}
}

func TestRollbackWithoutSavedStateSaysSo(t *testing.T) {
	a, _, _ := upgradeApp(t, &stack{})
	if err := runCLI(a, "upgrade", "--rollback"); err == nil || !strings.Contains(err.Error(), "no saved state") {
		t.Fatalf("got %v", err)
	}
}

func TestUpgradeRefusesASettingTheComposeFileNeverReadsAndChangesNothing(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true, ignoresSettings: true})
	err := runCLI(a, "upgrade", "--project", "p", "--set", "SECURITY_MODE=enforce", "--health-timeout", "2")
	if err == nil || !strings.Contains(err.Error(), "never reads SECURITY_MODE") || !strings.Contains(err.Error(), "Nothing was changed") {
		t.Fatalf("got %v\n%s", err, out)
	}
	for _, sub := range []string{"stop primary", "docker tag", " up -d"} {
		if sc.index(sub) >= 0 {
			t.Fatalf("it must refuse before touching anything, but ran %q", sub)
		}
	}
	if env, _ := os.ReadFile(filepath.Join(a.Dir, ".env")); strings.Contains(string(env), "enforce") {
		t.Fatal("the settings file must not be edited")
	}
}

func TestUpgradeForceWritesASettingTheFileDoesNotRead(t *testing.T) {
	a, _, out := upgradeApp(t, &stack{primaryHealthy: true, ignoresSettings: true})
	if err := runCLI(a, "upgrade", "--project", "p", "--set", "SECURITY_MODE=enforce", "--force", "--health-timeout", "2"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out.String(), "never reads SECURITY_MODE") {
		t.Fatalf("it must still say so:\n%s", out)
	}
}

func TestUpgradeWithoutSetDoesNotRenderProbes(t *testing.T) {
	a, sc, _ := upgradeApp(t, &stack{primaryHealthy: true, ignoresSettings: true})
	a.DryRun = true
	if err := runCLI(a, "upgrade", "--project", "p"); err != nil {
		t.Fatal(err)
	}
	if sc.index("ENV:") >= 0 {
		t.Fatal("nothing to probe without --set")
	}
}

func TestUpgradeRestartsNothingWhenNothingChanged(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true, unchanged: true})
	if err := runCLI(a, "upgrade", "--project", "p", "--health-timeout", "2"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, sub := range []string{"stop primary", " up -d"} {
		if sc.index(sub) >= 0 {
			t.Fatalf("nothing changed, but it ran %q", sub)
		}
	}
	if !strings.Contains(out.String(), "nothing to upgrade") {
		t.Fatalf("it must say why:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(a.Dir, ".upgrade", "last")); err == nil {
		t.Fatal("an idle run must not leave a rollback point behind")
	}
}

func TestUpgradeRestartFlagRestartsEvenWhenNothingChanged(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true, unchanged: true})
	if err := runCLI(a, "upgrade", "--project", "p", "--restart", "--health-timeout", "2"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if sc.index("stop primary") < 0 || sc.index("up -d --no-deps primary") < 0 {
		t.Fatalf("--restart must recreate the primary:\n%v", sc.calls)
	}
}

func TestUpgradeWithASettingAlwaysApplies(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true, unchanged: true})
	if err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--health-timeout", "2"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if sc.index("up -d --no-deps primary") < 0 {
		t.Fatal("a --set must be applied by recreating")
	}
}

func TestUpgradeKeepProtectsTheRollbackPointFromPruning(t *testing.T) {
	a, _, out := upgradeApp(t, &stack{primaryHealthy: true})
	root := filepath.Join(a.Dir, ".upgrade")
	for _, stamp := range []string{"2020-01-01", "2020-01-02", "2020-01-03", "2020-01-04", "2020-01-05", "2020-01-06", "2020-01-07"} {
		os.MkdirAll(filepath.Join(root, stamp), 0o700)
	}
	os.WriteFile(filepath.Join(root, "2020-01-01", keepMarker), nil, 0o600)
	if err := runCLI(a, "upgrade", "--project", "p", "--set", "KEEP=2", "--keep", "--health-timeout", "2"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(root, "2020-01-01")); err != nil {
		t.Fatal("a protected rollback point must survive pruning")
	}
	// the run's own point is protected too, so five unprotected ones remain: 03 to 07
	if _, err := os.Stat(filepath.Join(root, "2020-01-02")); err == nil {
		t.Fatal("unprotected points beyond the limit must still be pruned")
	}
	last, _ := os.ReadFile(filepath.Join(root, "last"))
	if _, err := os.Stat(filepath.Join(root, strings.TrimSpace(string(last)), keepMarker)); err != nil {
		t.Fatalf("--keep must mark this run's own point: %v", err)
	}
	if !strings.Contains(out.String(), "protected") {
		t.Fatalf("it must say so:\n%s", out)
	}
}

func TestUpgradeListsRecommendedSettingsThatAreMissing(t *testing.T) {
	a, _, out := upgradeApp(t, &stack{primaryHealthy: true})
	if err := runCLI(a, "upgrade", "--project", "p", "--set", "SECURITY_MODE=enforce", "--health-timeout", "2"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	got := out.String()
	if strings.Contains(got, "SECURITY_MODE=enforce   ") || !strings.Contains(got, "ENCRYPT_SECRETS_AT_REST=true") {
		t.Fatalf("only the missing recommendation should be listed:\n%s", got)
	}
	if env, _ := os.ReadFile(filepath.Join(a.Dir, ".env")); strings.Contains(string(env), "ENCRYPT_SECRETS_AT_REST") {
		t.Fatal("recommendations are only reported, never written")
	}
}
