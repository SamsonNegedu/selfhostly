package ctl

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	selfhostly "github.com/selfhostly"
	"github.com/selfhostly/internal/update"
)

// updateWorld is an install that runs the release's own compose file, plus a signed release for it
type updateWorld struct {
	a       *App
	sc      *scripted
	out     interface{ String() string }
	live    string
	status  string
	release string
	sig     string
	pub     ed25519.PublicKey
	priv    ed25519.PrivateKey
}

func digestRef(service, c string) string {
	return update.DefaultImagePrefix + service + "@sha256:" + strings.Repeat(c, 64)
}

func newUpdateWorld(t *testing.T, st *stack, settings ...update.Setting) *updateWorld {
	t.Helper()
	a, _, out := upgradeApp(t, st)
	body, err := selfhostly.Files.ReadFile(primaryCompose)
	if err != nil {
		t.Fatal(err)
	}
	live := filepath.Join(a.Dir, primaryCompose)
	os.WriteFile(live, body, 0o644)
	sc := liveInstall(t, live)
	sc.handle = wrapStack(sc.handle, st)
	a.Run = sc

	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	t.Setenv(envUpdatePublicKey, update.EncodeKey(pub))
	sum := sha256.Sum256(body)
	w := &updateWorld{a: a, sc: sc, out: out, live: live, pub: pub, priv: priv}
	w.status = update.StatusPath(a.Dir)
	w.publish(t, update.Manifest{
		Schema: update.SchemaVersion, Version: "1.4.0",
		Images: update.Images{
			Backend: digestRef("backend", "a"), Gateway: digestRef("gateway", "b"), Frontend: digestRef("frontend", "c"),
		},
		Compose:  update.ComposeInfo{SHA256: hex.EncodeToString(sum[:])},
		Settings: settings,
	})
	return w
}

// wrapStack lets liveInstall keep answering for the stack's health while the test flips it
func wrapStack(h func(string) (string, bool), st *stack) func(string) (string, bool) {
	return func(cmd string) (string, bool) {
		if strings.Contains(cmd, "State.Health") && strings.HasSuffix(cmd, " pc") {
			if st.primaryHealthy {
				return "healthy\n", false
			}
			return "starting\n", false
		}
		return h(cmd)
	}
}

func (w *updateWorld) publish(t *testing.T, m update.Manifest) {
	t.Helper()
	dir := t.TempDir()
	data, _ := json.Marshal(m)
	w.release, w.sig = filepath.Join(dir, "release.json"), filepath.Join(dir, "release.json.sig")
	os.WriteFile(w.release, data, 0o644)
	os.WriteFile(w.sig, []byte(update.Sign(w.priv, data)), 0o644)
}

func (w *updateWorld) start(t *testing.T, from, to string) {
	t.Helper()
	if err := update.WriteRun(w.status, update.NewRun("run1", from, to)); err != nil {
		t.Fatal(err)
	}
}

func (w *updateWorld) run(extra ...string) error {
	args := append([]string{"update-run", "--id", "run1", "--release", w.release, "--sig", w.sig, "--status", w.status, "--from-version", "1.3.0"}, extra...)
	return runCLI(w.a, args...)
}

func (w *updateWorld) result(t *testing.T) *update.Run {
	t.Helper()
	r, err := update.ReadRun(w.status)
	if err != nil || r == nil {
		t.Fatalf("no status file: %v", err)
	}
	return r
}

func (w *updateWorld) env(t *testing.T) string {
	t.Helper()
	b, _ := os.ReadFile(filepath.Join(w.a.Dir, ".env"))
	return string(b)
}

func TestUpdateRunAppliesTheReleaseAndReportsEveryStep(t *testing.T) {
	w := newUpdateWorld(t, &stack{primaryHealthy: true}, update.Setting{Key: "NEW_TOKEN", Kind: update.SettingGenerated})
	w.start(t, "1.3.0", "1.4.0")
	if err := w.run(); err != nil {
		t.Fatalf("%v\n%s", err, w.out)
	}
	r := w.result(t)
	if r.State != update.RunSucceeded || r.Phase != update.PhaseDone {
		t.Fatalf("got %+v", r)
	}
	for _, s := range r.Steps {
		if s.State != update.StepDone {
			t.Errorf("step %s is %s, want done", s.Name, s.State)
		}
	}
	env := w.env(t)
	for _, want := range []string{
		"BACKEND_IMAGE=" + digestRef("backend", "a"), "GATEWAY_IMAGE=" + digestRef("gateway", "b"),
		"FRONTEND_IMAGE=" + digestRef("frontend", "c"), "SELFHOSTLY_VERSION=1.4.0", "NEW_TOKEN=", "KEEP=1",
	} {
		if !strings.Contains(env, want) {
			t.Errorf("settings file lacks %q:\n%s", want, env)
		}
	}
	if w.sc.index(" pull ") < 0 {
		t.Fatal("an update must pull the new images")
	}
}

func TestUpdateRunRollsBackAndSaysSoWhenTheNewVersionIsNotHealthy(t *testing.T) {
	st := &stack{primaryHealthy: false}
	w := newUpdateWorld(t, st)
	w.start(t, "1.3.0", "1.4.0")
	rolling := false
	orig := w.sc.handle
	w.sc.handle = func(cmd string) (string, bool) {
		if strings.Contains(cmd, "--force-recreate") {
			st.primaryHealthy, rolling = true, true
		}
		_ = rolling
		return orig(cmd)
	}
	if err := w.run(); err == nil {
		t.Fatalf("expected a failure\n%s", w.out)
	}
	r := w.result(t)
	if r.State != update.RunRolledBack || !strings.Contains(r.Message, "did not become healthy") {
		t.Fatalf("got %+v", r)
	}
	if strings.Contains(w.env(t), "BACKEND_IMAGE") || strings.Contains(w.env(t), "SELFHOSTLY_VERSION") {
		t.Fatalf("the settings file must be back as it was:\n%s", w.env(t))
	}
}

func TestUpdateRunBlocksOnAMissingRequiredValueBeforeTouchingAnything(t *testing.T) {
	w := newUpdateWorld(t, &stack{primaryHealthy: true}, update.Setting{Key: "API_KEY", Kind: update.SettingRequired, Secret: true})
	w.start(t, "1.3.0", "1.4.0")
	if err := w.run(); err == nil || !strings.Contains(err.Error(), "API_KEY") {
		t.Fatalf("got %v", err)
	}
	r := w.result(t)
	if r.State != update.RunFailed || !strings.Contains(r.Message, "nothing was changed") {
		t.Fatalf("got %+v", r)
	}
	for _, sub := range []string{"stop primary", " pull ", "--force-recreate"} {
		if w.sc.index(sub) >= 0 {
			t.Fatalf("nothing may run before the values are there, but ran %q", sub)
		}
	}
	if strings.Contains(w.env(t), "BACKEND_IMAGE") {
		t.Fatalf("the settings must be restored:\n%s", w.env(t))
	}
}

func TestUpdateRunWritesGivenValuesAndDeletesTheInputsFile(t *testing.T) {
	w := newUpdateWorld(t, &stack{primaryHealthy: true}, update.Setting{Key: "API_KEY", Kind: update.SettingRequired, Secret: true})
	w.start(t, "1.3.0", "1.4.0")
	inputs := filepath.Join(t.TempDir(), "inputs.json")
	os.WriteFile(inputs, []byte(`{"API_KEY":"s3cret"}`), 0o600)
	if err := w.run("--inputs", inputs); err != nil {
		t.Fatalf("%v\n%s", err, w.out)
	}
	if !strings.Contains(w.env(t), "API_KEY=s3cret") {
		t.Fatalf("the value was not written:\n%s", w.env(t))
	}
	if _, err := os.Stat(inputs); err == nil {
		t.Fatal("the inputs file holds secrets and must be deleted once read")
	}
}

func TestUpdateRunRefusesSettingsTheReleaseDoesNotAskFor(t *testing.T) {
	w := newUpdateWorld(t, &stack{primaryHealthy: true})
	w.start(t, "1.3.0", "1.4.0")
	inputs := filepath.Join(t.TempDir(), "inputs.json")
	os.WriteFile(inputs, []byte(`{"JWT_SECRET":"attacker"}`), 0o600)
	if err := w.run("--inputs", inputs); err == nil || !strings.Contains(err.Error(), "not a setting this release asks for") {
		t.Fatalf("got %v", err)
	}
	if strings.Contains(w.env(t), "attacker") {
		t.Fatal("an arbitrary variable must never be written from the UI")
	}
}

func TestSettingValuesCannotSmuggleAnotherLine(t *testing.T) {
	for _, v := range []string{"", "a\nJWT_SECRET=x", "a\rb", "a\x00b", strings.Repeat("x", maxInputBytes+1)} {
		if checkSettingValue("KEY", v) == nil {
			t.Errorf("%q must be refused", v)
		}
	}
	if checkSettingValue("KEY", "a normal value = with # symbols") != nil {
		t.Error("an ordinary value must be accepted")
	}
}

func TestUpdateRunRefusesAManifestThatIsNotSignedByTheTrustedKey(t *testing.T) {
	w := newUpdateWorld(t, &stack{primaryHealthy: true})
	w.start(t, "1.3.0", "1.4.0")
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	data, _ := os.ReadFile(w.release)
	os.WriteFile(w.sig, []byte(update.Sign(other, data)), 0o644)
	if err := w.run(); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("got %v", err)
	}
	if r := w.result(t); r.State != update.RunFailed {
		t.Fatalf("got %+v", r)
	}
	if len(w.sc.calls) != 0 {
		t.Fatalf("nothing may run before the signature is verified, ran %v", w.sc.calls)
	}
}

func TestUpdateRunRefusesAVersionThatIsNotTheOneRequested(t *testing.T) {
	w := newUpdateWorld(t, &stack{primaryHealthy: true})
	w.start(t, "1.3.0", "1.5.0")
	if err := w.run(); err == nil || !strings.Contains(err.Error(), "started for") {
		t.Fatalf("got %v", err)
	}
	if len(w.sc.calls) != 0 {
		t.Fatalf("nothing may run, ran %v", w.sc.calls)
	}
}

func TestUpdateRunBlocksAnUpdateFromAVersionTheReleaseDoesNotSupport(t *testing.T) {
	w := newUpdateWorld(t, &stack{primaryHealthy: true})
	m := update.Manifest{}
	data, _ := os.ReadFile(w.release)
	json.Unmarshal(data, &m)
	m.MinFromVersion = "1.3.5"
	w.publish(t, m)
	w.start(t, "1.3.0", "1.4.0")
	if err := w.run(); err == nil || !strings.Contains(err.Error(), "1.3.5") {
		t.Fatalf("got %v", err)
	}
}

func TestUpdatePlanSortsSettingsAndReportsBlockers(t *testing.T) {
	w := newUpdateWorld(t, &stack{primaryHealthy: true},
		update.Setting{Key: "MUST", Kind: update.SettingRequired, Description: "needed", Secret: true},
		update.Setting{Key: "GEN", Kind: update.SettingGenerated},
		update.Setting{Key: "OPT", Kind: update.SettingOptional},
		update.Setting{Key: "KEEP", Kind: update.SettingRequired}, // already in .env
	)
	plan, err := w.a.updatePlan(t.Context(), updateOpts{release: w.release, sig: w.sig, fromVersion: "1.3.0"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.State != update.PlanBlocked || len(plan.Settings.RequiredMissing) != 1 || plan.Settings.RequiredMissing[0].Key != "MUST" ||
		!plan.Settings.RequiredMissing[0].Secret {
		t.Fatalf("got %+v", plan)
	}
	if len(plan.Settings.Generated) != 1 || len(plan.Settings.Optional) != 1 {
		t.Fatalf("got %+v", plan.Settings)
	}
	if plan.Compose.State != update.ComposeCurrent {
		t.Fatalf("the install runs the release's own file, got %q", plan.Compose.State)
	}
	if len(plan.Blockers) != 1 || plan.Blockers[0].Code != update.BlockMissingRequired {
		t.Fatalf("got %+v", plan.Blockers)
	}
}

func TestUpdatePlanIsReadyWhenNothingBlocks(t *testing.T) {
	w := newUpdateWorld(t, &stack{primaryHealthy: true})
	plan, err := w.a.updatePlan(t.Context(), updateOpts{release: w.release, sig: w.sig, fromVersion: "dev"})
	if err != nil || plan.State != update.PlanReady || len(plan.Blockers) != 0 {
		t.Fatalf("got %+v %v", plan, err)
	}
	if w.sc.index("stop") >= 0 || w.sc.index("up -d") >= 0 || w.sc.index(" pull ") >= 0 {
		t.Fatalf("a plan must not change anything: %v", w.sc.calls)
	}
}

func TestUpdatePlanRefusesAReleaseWhoseComposeFileIsNotTheOneInTheImage(t *testing.T) {
	w := newUpdateWorld(t, &stack{primaryHealthy: true})
	m := update.Manifest{}
	data, _ := os.ReadFile(w.release)
	json.Unmarshal(data, &m)
	m.Compose.SHA256 = strings.Repeat("0", 64)
	w.publish(t, m)
	plan, err := w.a.updatePlan(t.Context(), updateOpts{release: w.release, sig: w.sig})
	if err != nil || len(plan.Blockers) != 1 || plan.Blockers[0].Code != update.BlockReleaseMismatch {
		t.Fatalf("got %+v %v", plan, err)
	}
}

func TestStaleLockFromADeadUpdaterIsRemovedButALiveOneIsKept(t *testing.T) {
	a, sc, _ := upgradeApp(t, &stack{primaryHealthy: true})
	lock := filepath.Join(a.Dir, ".upgrade", lockName)
	os.MkdirAll(lock, 0o700)
	os.WriteFile(filepath.Join(lock, lockOwnerFile), []byte("container:deadbox\n"), 0o600)
	orig := sc.handle
	running := "false\n"
	sc.handle = func(cmd string) (string, bool) {
		if strings.Contains(cmd, "docker inspect --format {{.State.Running}} deadbox") {
			return running, false
		}
		return orig(cmd)
	}
	if err := a.acquireLock(t.Context(), lock); err != nil {
		t.Fatalf("a lock whose updater container is gone must be taken over: %v", err)
	}
	os.RemoveAll(lock)

	os.MkdirAll(lock, 0o700)
	os.WriteFile(filepath.Join(lock, lockOwnerFile), []byte("container:deadbox\n"), 0o600)
	running = "true\n"
	if err := a.acquireLock(t.Context(), lock); err == nil {
		t.Fatal("a lock held by a running updater must be respected")
	}
	os.RemoveAll(lock)

	os.MkdirAll(lock, 0o700) // an older lock with no owner: never guess
	if err := a.acquireLock(t.Context(), lock); err == nil {
		t.Fatal("a lock without an owner must stay")
	}
}

func TestComposeApprovalTokenIsTheSameInEveryRun(t *testing.T) {
	// The review and the updater are two processes, each rendering its own report with its own temporary file. The
	// token the browser sends back must still match, or an approved update is refused.
	w := newUpdateWorld(t, &stack{primaryHealthy: true})
	os.WriteFile(w.live, []byte("services: {}\n"), 0o644) // an older file than the release's
	sc := w.sc
	base := sc.handle
	sc.handle = func(cmd string) (string, bool) {
		if strings.Contains(cmd, "config --format json") {
			return `{"services":{"primary":{"environment":{"A":"1"}}}}`, false
		}
		return base(cmd)
	}
	m, err := loadRelease(w.release, w.sig)
	if err != nil {
		t.Fatal(err)
	}
	first, blockers, err := w.a.planCompose(t.Context(), m)
	if err != nil || len(blockers) != 0 || first.State != update.ComposeBehind {
		t.Fatalf("got %+v %v %v", first, blockers, err)
	}
	second, _, _ := w.a.planCompose(t.Context(), m)
	if first.ApprovalToken == "" || first.ApprovalToken != second.ApprovalToken {
		t.Fatalf("the token must not change between runs: %q vs %q", first.ApprovalToken, second.ApprovalToken)
	}
	if strings.Contains(first.Report, "docker-compose.release.") {
		t.Fatalf("the report must not show a temporary path:\n%s", first.Report)
	}
}

// customizedWorld is an install whose compose file has an edit the release file does not carry
func customizedWorld(t *testing.T) *updateWorld {
	t.Helper()
	w := newUpdateWorld(t, &stack{primaryHealthy: true})
	os.WriteFile(w.live, []byte("services: {}\n# hand edited\n"), 0o644)
	base := w.sc.handle
	w.sc.handle = func(cmd string) (string, bool) {
		if strings.Contains(cmd, "config --format json") {
			if strings.Contains(cmd, "-f "+w.live) {
				return `{"services":{"primary":{"environment":{"MY_HAND_SET_VALUE":"keep-me"}}}}`, false
			}
			return `{"services":{"primary":{"environment":{}}}}`, false
		}
		return base(cmd)
	}
	return w
}

func TestPlanBlocksAComposeFileWithEditsTheReleaseWouldDrop(t *testing.T) {
	w := customizedWorld(t)
	plan, err := w.a.updatePlan(t.Context(), updateOpts{release: w.release, sig: w.sig, fromVersion: "1.3.0"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.State != update.PlanBlocked || plan.Compose.State != update.ComposeCustomized || plan.Compose.ApprovalToken != "" {
		t.Fatalf("got %+v", plan)
	}
	if len(plan.Blockers) != 1 || plan.Blockers[0].Code != update.BlockComposeCustom {
		t.Fatalf("got %+v", plan.Blockers)
	}
	if !strings.Contains(plan.Compose.Report, "MY_HAND_SET_VALUE") || plan.Compose.Diff == "" {
		t.Fatalf("the report must name what would be lost:\n%s", plan.Compose.Report)
	}
}

func TestUpdateRunNeverOverwritesACustomizedComposeFile(t *testing.T) {
	w := customizedWorld(t)
	w.start(t, "1.3.0", "1.4.0")
	before, _ := os.ReadFile(w.live)
	if err := w.run("--approve", "anything"); err == nil {
		t.Fatalf("expected a refusal\n%s", w.out)
	}
	if after, _ := os.ReadFile(w.live); string(after) != string(before) {
		t.Fatalf("the hand-edited compose file must be untouched, got %q", after)
	}
	for _, sub := range []string{"stop primary", " pull ", "--force-recreate"} {
		if w.sc.index(sub) >= 0 {
			t.Fatalf("nothing may run, but ran %q", sub)
		}
	}
	if r := w.result(t); r.State != update.RunFailed || !strings.Contains(r.Message, "nothing was changed") {
		t.Fatalf("got %+v", r)
	}
}

func TestUpdateRunRefusesABehindComposeFileWithTheWrongApproval(t *testing.T) {
	w := newUpdateWorld(t, &stack{primaryHealthy: true})
	os.WriteFile(w.live, []byte("services: {}\n"), 0o644) // older than the release's, and it loses nothing
	base := w.sc.handle
	w.sc.handle = func(cmd string) (string, bool) {
		if strings.Contains(cmd, "config --format json") {
			return `{"services":{"primary":{"environment":{"A":"1"}}}}`, false
		}
		return base(cmd)
	}
	w.start(t, "1.3.0", "1.4.0")
	if err := w.run("--approve", "not-the-token"); err == nil || !strings.Contains(err.Error(), "not approved") {
		t.Fatalf("got %v", err)
	}
	if got, _ := os.ReadFile(w.live); string(got) != "services: {}\n" {
		t.Fatalf("the compose file must be untouched, got %q", got)
	}
	// the right token from a fresh review is accepted, and the file is swapped
	m, _ := loadRelease(w.release, w.sig)
	cc, _, _ := w.a.planCompose(t.Context(), m)
	w.start(t, "1.3.0", "1.4.0")
	if err := w.run("--approve", cc.ApprovalToken); err != nil {
		t.Fatalf("%v\n%s", err, w.out)
	}
	want, _ := selfhostly.Files.ReadFile(primaryCompose)
	if got, _ := os.ReadFile(w.live); string(got) != string(want) {
		t.Fatal("an approved swap must write exactly the release's compose file")
	}
}
