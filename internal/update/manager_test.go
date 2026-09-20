package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeDocker struct {
	mu     sync.Mutex
	calls  []string
	handle func(args string) (out, stderr string, err error)
}

func (f *fakeDocker) Run(_ context.Context, args ...string) (string, string, error) {
	line := strings.Join(args, " ")
	f.mu.Lock()
	f.calls = append(f.calls, line)
	f.mu.Unlock()
	return f.handle(line)
}

func (f *fakeDocker) called(sub string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if strings.Contains(c, sub) {
			return c
		}
	}
	return ""
}

type env struct {
	m      *Manager
	dk     *fakeDocker
	dir    string
	priv   ed25519.PrivateKey
	server *httptest.Server
	mf     Manifest
	jobs   int
	body   []byte
	sig    string
}

const hostDataDir = "/srv/selfhostly/data"

func inspectJSON(dataMount string) string {
	doc := []map[string]any{{
		"Name": "/selfhostly-primary", "Image": "sha256:current",
		"Config": map[string]any{"User": "1000:984", "Labels": map[string]string{
			labelWorkingDir: "/srv/selfhostly", labelConfigFiles: "/srv/selfhostly/docker-compose.prod.yml",
		}},
		"Mounts": []map[string]string{
			{"Source": hostDataDir, "Destination": dataMount},
			{"Source": "/var/run/docker.sock", "Destination": "/var/run/docker.sock"},
		},
	}}
	b, _ := json.Marshal(doc)
	return string(b)
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	e := &env{priv: priv, dir: t.TempDir(), mf: validManifest()}
	e.mf.Version = "1.4.0"
	e.mf.Settings = nil
	e.mf.Images = Images{
		Backend:  DefaultImagePrefix + "backend" + digest("a"),
		Gateway:  DefaultImagePrefix + "gateway" + digest("b"),
		Frontend: DefaultImagePrefix + "frontend" + digest("c"),
	}
	e.publish()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/release.json":
			w.Write(e.body)
		case "/release.json.sig":
			w.Write([]byte(e.sig))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(e.server.Close)
	e.dk = &fakeDocker{}
	e.dk.handle = e.defaultDocker
	e.m = NewManager(Options{
		Settings: Settings{
			Enabled: true, ManifestURL: e.server.URL + "/release.json", PublicKey: EncodeKey(pub),
			CurrentVersion: "1.3.0", IsPrimary: true, AuthEnabled: true,
		},
		Docker: e.dk, DataDir: e.dir, Hostname: "abc123",
		ActiveJobs:   func() (int, error) { return e.jobs, nil },
		FreeBytes:    func(string) (uint64, error) { return 10 << 30, nil },
		SocketExists: func() bool { return true },
	})
	return e
}

func (e *env) publish() {
	e.body, _ = json.Marshal(e.mf)
	e.sig = Sign(e.priv, e.body)
}

func (e *env) defaultDocker(args string) (string, string, error) {
	switch {
	case strings.HasPrefix(args, "inspect abc123"):
		return inspectJSON(e.dir), "", nil
	case strings.HasPrefix(args, "pull "):
		return "", "", nil
	case strings.HasPrefix(args, "run --rm") && strings.Contains(args, "update-plan"):
		p := NewPlan("1.4.0")
		p.Compose = ComposePlan{State: ComposeCurrent}
		b, _ := json.Marshal(p)
		return string(b), "", nil
	case strings.HasPrefix(args, "run -d"):
		return "containerid\n", "", nil
	case strings.HasPrefix(args, "ps -a"):
		return "", "", nil
	}
	return "", "", nil
}

func (e *env) reviewed(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if err := e.m.Check(ctx); err != nil {
		t.Fatal(err)
	}
	if err := e.m.Review("1.4.0"); err != nil {
		t.Fatal(err)
	}
	e.waitPlan(t, PlanReady, PlanBlocked, PlanFailed)
}

func (e *env) waitPlan(t *testing.T, states ...string) *Plan {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if v := e.m.View(context.Background()); v.Plan != nil {
			for _, s := range states {
				if v.Plan.State == s {
					return v.Plan
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the plan never reached %v: %+v", states, e.m.View(context.Background()).Plan)
	return nil
}

func TestCheckFindsANewerSignedRelease(t *testing.T) {
	e := newEnv(t)
	if err := e.m.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	v := e.m.View(context.Background())
	if !v.Enabled || v.Available == nil || v.Available.Version != "1.4.0" || v.CheckError != "" || v.CurrentVersion != "1.3.0" {
		t.Fatalf("got %+v", v)
	}
}

func TestCheckReportsNothingWhenAlreadyCurrentOrAhead(t *testing.T) {
	for _, cur := range []string{"1.4.0", "1.5.0"} {
		e := newEnv(t)
		e.m.opt.Settings.CurrentVersion = cur
		if err := e.m.Check(context.Background()); err != nil {
			t.Fatal(err)
		}
		if v := e.m.View(context.Background()); v.Available != nil {
			t.Fatalf("running %s: nothing newer than it, got %+v", cur, v.Available)
		}
	}
}

func TestADevelopmentBuildSeesEveryReleaseAsNewer(t *testing.T) {
	e := newEnv(t)
	e.m.opt.Settings.CurrentVersion = "dev"
	e.m.Check(context.Background())
	if v := e.m.View(context.Background()); v.Available == nil {
		t.Fatal("a build with no version must be offered the release")
	}
}

func TestCheckRefusesATamperedManifestAndKeepsWhatItKnew(t *testing.T) {
	e := newEnv(t)
	e.m.Check(context.Background())
	e.mf.Version = "9.9.9"
	e.body, _ = json.Marshal(e.mf) // changed after signing: the old signature no longer matches
	if err := e.m.Check(context.Background()); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("got %v", err)
	}
	v := e.m.View(context.Background())
	if v.CheckError == "" || v.Available == nil || v.Available.Version != "1.4.0" {
		t.Fatalf("a bad manifest must be reported without replacing the good one: %+v", v)
	}
}

func TestCheckReportsAnUnreachableServer(t *testing.T) {
	e := newEnv(t)
	e.server.Close()
	if err := e.m.Check(context.Background()); err == nil {
		t.Fatal("expected an error")
	}
	if v := e.m.View(context.Background()); v.CheckError == "" {
		t.Fatal("the failure must show in the view")
	}
}

func TestDisabledReasons(t *testing.T) {
	noBuiltInKey(t)
	cases := map[string]func(*Settings){
		ReasonNotEnabled:  func(s *Settings) { s.Enabled = false },
		ReasonSecondary:   func(s *Settings) { s.IsPrimary = false },
		ReasonAuthDisable: func(s *Settings) { s.AuthEnabled = false },
		ReasonNoKey:       func(s *Settings) { s.PublicKey = "" },
	}
	for want, mutate := range cases {
		e := newEnv(t)
		mutate(&e.m.opt.Settings)
		if got := e.m.DisabledReason(); got != want {
			t.Errorf("want %q, got %q", want, got)
		}
		if err := e.m.Check(context.Background()); !errors.Is(err, ErrDisabled) {
			t.Errorf("%s: a disabled manager must refuse to check, got %v", want, err)
		}
	}
	e := newEnv(t)
	e.m.opt.SocketExists = func() bool { return false }
	if got := e.m.DisabledReason(); got != ReasonNoSocket {
		t.Errorf("got %q", got)
	}
}

func TestReviewPullsTheImageByDigestAndRunsThePlanInItReadOnly(t *testing.T) {
	e := newEnv(t)
	e.reviewed(t)
	pull := e.dk.called("pull ")
	if !strings.HasSuffix(pull, "selfhostly-backend"+digest("a")) {
		t.Fatalf("the image must be pulled by digest, ran %q", pull)
	}
	run := e.dk.called("update-plan")
	for _, want := range []string{
		"run --rm", "--entrypoint selfhostlyctl", DefaultImagePrefix + "backend" + digest("a"),
		"-v /srv/selfhostly:/srv/selfhostly:ro", "-v " + hostDataDir + ":" + hostDataDir + ":ro",
		"--container selfhostly-primary", "--from-version 1.3.0", "--user 1000:984",
		"--release " + hostDataDir + "/updates/release/release.json",
	} {
		if !strings.Contains(run, want) {
			t.Errorf("the review command lacks %q:\n%s", want, run)
		}
	}
	if v := e.m.View(context.Background()); v.Plan == nil || v.Plan.State != PlanReady {
		t.Fatalf("got %+v", v.Plan)
	}
}

func TestReviewNamesDockerFailuresInThePlan(t *testing.T) {
	e := newEnv(t)
	e.dk.handle = func(args string) (string, string, error) {
		if strings.HasPrefix(args, "pull ") {
			return "", "manifest unknown", errors.New("exit 1")
		}
		return e.defaultDocker(args)
	}
	e.m.Check(context.Background())
	e.m.Review("1.4.0")
	p := e.waitPlan(t, PlanFailed)
	if !strings.Contains(p.Error, "manifest unknown") {
		t.Fatalf("got %+v", p)
	}
}

func TestReviewIsBlockedThroughASocketProxyAndRunsNothing(t *testing.T) {
	e := newEnv(t)
	e.m.opt.Settings.DockerHost = "tcp://socket-proxy:2375"
	e.m.Check(context.Background())
	e.m.Review("1.4.0")
	p := e.waitPlan(t, PlanBlocked)
	if len(p.Blockers) != 1 || p.Blockers[0].Code != BlockSocketProxy {
		t.Fatalf("got %+v", p)
	}
	if e.dk.called("pull ") != "" || e.dk.called("update-plan") != "" {
		t.Fatal("nothing may be pulled or run through a proxy")
	}
}

func TestReviewRefusesAVersionTheCheckDidNotFind(t *testing.T) {
	e := newEnv(t)
	e.m.Check(context.Background())
	if err := e.m.Review("2.0.0"); !errors.Is(err, ErrUnknownVersion) {
		t.Fatalf("got %v", err)
	}
	if err := e.m.Review("1.3.0"); !errors.Is(err, ErrUnknownVersion) {
		t.Fatalf("an older or equal version must be refused, got %v", err)
	}
}

func TestPlanShowsARunningJobAsALiveBlocker(t *testing.T) {
	e := newEnv(t)
	e.reviewed(t)
	e.jobs = 2
	p := e.m.View(context.Background()).Plan
	if p.State != PlanBlocked || len(p.Blockers) != 1 || p.Blockers[0].Code != BlockJobRunning {
		t.Fatalf("got %+v", p)
	}
	e.jobs = 0
	if p := e.m.View(context.Background()).Plan; p.State != PlanReady || len(p.Blockers) != 0 {
		t.Fatalf("the blocker must clear when the job ends, got %+v", p)
	}
	if _, err := func() (*Run, error) {
		e.jobs = 1
		return e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"})
	}(); err == nil {
		t.Fatal("apply must re-check jobs itself")
	} else {
		var be *BlockedError
		if !errors.As(err, &be) || be.Blockers[0].Code != BlockJobRunning {
			t.Fatalf("got %v", err)
		}
	}
}

func TestApplyStartsADetachedUpdaterAndRecordsAPendingRun(t *testing.T) {
	e := newEnv(t)
	e.reviewed(t)
	run, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"})
	if err != nil {
		t.Fatal(err)
	}
	if run.State != RunPending || run.FromVersion != "1.3.0" || run.ToVersion != "1.4.0" {
		t.Fatalf("got %+v", run)
	}
	cmd := e.dk.called("run -d")
	for _, want := range []string{
		"--name " + UpdaterName(run.ID), "--label selfhostly.updater=1", "--restart no", "--user 1000:984",
		"-v /var/run/docker.sock:/var/run/docker.sock", "--entrypoint selfhostlyctl",
		DefaultImagePrefix + "backend" + digest("a") + " update-run", "--id " + run.ID, "--status " + hostDataDir + "/updates/status.json",
		"--release " + hostDataDir + "/updates/" + run.ID + "/release.json", "--container selfhostly-primary",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("the updater command lacks %q:\n%s", want, cmd)
		}
	}
	if got := e.m.readRun(); got == nil || got.ID != run.ID {
		t.Fatalf("the status file must exist, got %+v", got)
	}
}

func TestApplyNeedsAReviewFirst(t *testing.T) {
	e := newEnv(t)
	e.m.Check(context.Background())
	if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"}); !errors.Is(err, ErrNoPlan) {
		t.Fatalf("got %v", err)
	}
	if e.dk.called("run -d") != "" {
		t.Fatal("nothing may start without a plan")
	}
}

func TestApplyOnlyTakesTheVersionTheManifestNames(t *testing.T) {
	e := newEnv(t)
	e.reviewed(t)
	if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "9.9.9"}); !errors.Is(err, ErrUnknownVersion) {
		t.Fatalf("got %v", err)
	}
}

func TestApplyBlocksOnMissingRequiredValuesUntilTheyAreGiven(t *testing.T) {
	e := newEnv(t)
	e.mf.Settings = []Setting{{Key: "API_KEY", Kind: SettingRequired, Secret: true}}
	e.publish()
	e.dk.handle = func(args string) (string, string, error) {
		if strings.Contains(args, "update-plan") {
			p := NewPlan("1.4.0")
			p.Compose.State = ComposeCurrent
			p.Settings.RequiredMissing = []SettingNeed{{Key: "API_KEY", Secret: true}}
			p.Blockers = []Blocker{{Code: BlockMissingRequired, Message: "needs API_KEY"}}
			b, _ := json.Marshal(p)
			return string(b), "", nil
		}
		return e.defaultDocker(args)
	}
	e.reviewed(t)
	if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"}); err == nil {
		t.Fatal("a missing required value must block")
	}
	run, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0", Inputs: map[string]string{"API_KEY": "s3cret"}})
	if err != nil {
		t.Fatal(err)
	}
	inputs := filepath.Join(e.dir, "updates", run.ID, inputsFile)
	if info, err := os.Stat(inputs); err != nil || info.Mode().Perm() != fileMode {
		t.Fatalf("the inputs file must exist and be owner only: %v %v", info, err)
	}
	if cmd := e.dk.called("run -d"); strings.Contains(cmd, "s3cret") {
		t.Fatalf("a value must never be on a command line:\n%s", cmd)
	}
}

func TestApplyRefusesInputsTheReleaseDoesNotAskFor(t *testing.T) {
	e := newEnv(t)
	e.mf.Settings = []Setting{{Key: "NEW_KEY", Kind: SettingOptional}, {Key: "GEN_KEY", Kind: SettingGenerated}}
	e.publish()
	e.reviewed(t)
	cases := map[string]struct {
		inputs map[string]string
		reason string
	}{
		"an unlisted variable": {map[string]string{"JWT_SECRET": "x"}, "not a setting this release asks for"},
		"a generated setting":  {map[string]string{"GEN_KEY": "x"}, "not a setting this release asks for"},
		"a line break":         {map[string]string{"NEW_KEY": "a\nJWT_SECRET=x"}, "line break"},
		"an empty value":       {map[string]string{"NEW_KEY": ""}, "is empty"},
	}
	for name, c := range cases {
		_, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0", Inputs: c.inputs})
		if !errors.Is(err, ErrBadInput) || !strings.Contains(err.Error(), c.reason) {
			t.Errorf("%s: got %v", name, err)
		}
	}
	if e.dk.called("run -d") != "" {
		t.Fatal("nothing may start when an input is refused")
	}
}

func TestApplyNeedsTheComposeApprovalToken(t *testing.T) {
	e := newEnv(t)
	e.dk.handle = func(args string) (string, string, error) {
		if strings.Contains(args, "update-plan") {
			p := NewPlan("1.4.0")
			p.Compose = ComposePlan{State: ComposeBehind, Diff: "--- a", ApprovalToken: "tok123"}
			b, _ := json.Marshal(p)
			return string(b), "", nil
		}
		return e.defaultDocker(args)
	}
	e.reviewed(t)
	for _, wrong := range []string{"", "other"} {
		if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0", ApproveCompose: wrong}); !errors.Is(err, ErrNotApproved) {
			t.Fatalf("token %q: got %v", wrong, err)
		}
	}
	run, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0", ApproveCompose: "tok123"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.dk.called("run -d"), "--approve tok123") || run == nil {
		t.Fatal("the approval must reach the updater")
	}
}

func TestApplyRefusesACustomizedComposeFile(t *testing.T) {
	e := newEnv(t)
	e.dk.handle = func(args string) (string, string, error) {
		if strings.Contains(args, "update-plan") {
			p := NewPlan("1.4.0")
			p.Compose.State = ComposeCustomized
			p.Blockers = []Blocker{{Code: BlockComposeCustom, Message: "merge by hand"}}
			b, _ := json.Marshal(p)
			return string(b), "", nil
		}
		return e.defaultDocker(args)
	}
	e.reviewed(t)
	var be *BlockedError
	if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"}); !errors.As(err, &be) {
		t.Fatalf("got %v", err)
	}
}

func TestOnlyOneUpdateRunsAtATime(t *testing.T) {
	e := newEnv(t)
	e.reviewed(t)
	e.dk.handle = func(args string) (string, string, error) {
		if strings.HasPrefix(args, "inspect --format {{.State.Running}}") {
			return "true\n", "", nil
		}
		return e.defaultDocker(args)
	}
	if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"}); !errors.Is(err, ErrInProgress) {
		t.Fatalf("got %v", err)
	}
	if _, err := e.m.Rollback(context.Background()); !errors.Is(err, ErrInProgress) {
		t.Fatalf("a rollback must not start during an update, got %v", err)
	}
}

func TestARunWhoseUpdaterIsGoneIsMarkedInterrupted(t *testing.T) {
	e := newEnv(t)
	run := NewRun("r1", "1.3.0", "1.4.0")
	run.Advance(PhasePrimary, "")
	WriteRun(StatusPath(e.dir), run)

	e.dk.handle = func(args string) (string, string, error) {
		if strings.HasPrefix(args, "inspect --format") {
			return "", "Error: No such object: selfhostly-updater-r1", errors.New("exit 1")
		}
		return e.defaultDocker(args)
	}
	got := e.m.View(context.Background()).Run
	if got.State != RunInterrupted || got.FinishedAt == "" {
		t.Fatalf("got %+v", got)
	}
	if persisted := e.m.readRun(); persisted.State != RunInterrupted {
		t.Fatal("the interrupted state must be saved, so it survives a restart")
	}
}

func TestARunWithALiveUpdaterIsLeftAlone(t *testing.T) {
	e := newEnv(t)
	run := NewRun("r1", "1.3.0", "1.4.0")
	run.Advance(PhaseGateway, "")
	WriteRun(StatusPath(e.dir), run)
	e.dk.handle = func(args string) (string, string, error) {
		if strings.HasPrefix(args, "inspect --format") {
			return "true\n", "", nil
		}
		return e.defaultDocker(args)
	}
	if got := e.m.View(context.Background()).Run; got.State != RunRunning {
		t.Fatalf("got %+v", got)
	}
}

func TestWhenDockerCannotSayTheRunIsNotWrittenOff(t *testing.T) {
	e := newEnv(t)
	run := NewRun("r1", "1.3.0", "1.4.0")
	run.Advance(PhaseGateway, "")
	WriteRun(StatusPath(e.dir), run)
	e.dk.handle = func(args string) (string, string, error) {
		if strings.HasPrefix(args, "inspect --format") {
			return "", "Cannot connect to the Docker daemon", errors.New("exit 1")
		}
		return e.defaultDocker(args)
	}
	if got := e.m.View(context.Background()).Run; got.State != RunRunning {
		t.Fatalf("an unknown answer must not mark a live run interrupted, got %+v", got)
	}
}

func TestRollbackNamesTheVersionItReturnsTo(t *testing.T) {
	e := newEnv(t)
	prev := NewRun("old", "1.0.0", "1.1.0")
	prev.Finish(RunSucceeded, "")
	WriteRun(StatusPath(e.dir), prev)
	run, err := e.m.Rollback(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if run.Kind != KindRollback || run.ToVersion != "1.0.0" {
		t.Fatalf("got %+v", run)
	}
}

func TestRollbackRunsFromTheImageThisPrimaryRuns(t *testing.T) {
	e := newEnv(t)
	run, err := e.m.Rollback(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cmd := e.dk.called("run -d")
	if !strings.Contains(cmd, "sha256:current update-run --rollback") || !strings.Contains(cmd, "--id "+run.ID) {
		t.Fatalf("got %q", cmd)
	}
	if strings.Contains(cmd, "--release") {
		t.Fatal("a rollback needs no release")
	}
}

func TestFinishedUpdaterContainersAreCleanedUpBeforeANewRun(t *testing.T) {
	e := newEnv(t)
	e.dk.handle = func(args string) (string, string, error) {
		if strings.HasPrefix(args, "ps -a") {
			return "selfhostly-updater-old\nsomeone-elses-container\n", "", nil
		}
		return e.defaultDocker(args)
	}
	if _, err := e.m.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if e.dk.called("rm selfhostly-updater-old") == "" {
		t.Fatal("the old updater must be removed")
	}
	if e.dk.called("rm someone-elses") != "" {
		t.Fatal("only updater containers may be removed")
	}
}

func TestOldRunFoldersArePruned(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 8; i++ {
		os.MkdirAll(filepath.Join(e.dir, "updates", "2020010"+string(rune('0'+i))+"T000000Z"), 0o700)
	}
	os.MkdirAll(filepath.Join(e.dir, "updates", releaseDirName), 0o700)
	e.m.pruneRunDirs()
	entries, _ := os.ReadDir(filepath.Join(e.dir, "updates"))
	if len(entries) != keepRunDirs+1 {
		t.Fatalf("want %d run folders plus release, got %d", keepRunDirs, len(entries))
	}
}

func TestTwoAppliesAtOnceStartOnlyOneUpdater(t *testing.T) {
	e := newEnv(t)
	e.reviewed(t)
	e.dk.handle = func(args string) (string, string, error) {
		if strings.HasPrefix(args, "inspect --format {{.State.Running}}") {
			return "true\n", "", nil // the updater that started is alive
		}
		return e.defaultDocker(args)
	}
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"})
			results <- err
		}()
	}
	var ok, refused int
	for i := 0; i < 2; i++ {
		if err := <-results; err == nil {
			ok++
		} else if errors.Is(err, ErrInProgress) {
			refused++
		}
	}
	started := 0
	e.dk.mu.Lock()
	for _, c := range e.dk.calls {
		if strings.HasPrefix(c, "run -d") {
			started++
		}
	}
	e.dk.mu.Unlock()
	if ok != 1 || refused != 1 || started != 1 {
		t.Fatalf("one must start and one must be refused: ok=%d refused=%d updaters=%d", ok, refused, started)
	}
}

func TestTheApprovalIsOnlyPassedOnWhenAComposeChangeNeedsIt(t *testing.T) {
	e := newEnv(t)
	e.reviewed(t) // the plan's compose file is current
	if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0", ApproveCompose: "--privileged"}); err != nil {
		t.Fatal(err)
	}
	if cmd := e.dk.called("run -d"); strings.Contains(cmd, "--approve") || strings.Contains(cmd, "privileged") {
		t.Fatalf("a value from the browser must not reach the command line when nothing needs approving:\n%s", cmd)
	}
}

func TestARetryIsNotRefusedBecauseOfARunWhoseUpdaterDied(t *testing.T) {
	e := newEnv(t)
	e.reviewed(t)
	dead := NewRun("r0", "1.0.0", "1.4.0")
	dead.Advance(PhasePull, "")
	WriteRun(StatusPath(e.dir), dead)
	e.dk.handle = func(args string) (string, string, error) {
		if strings.HasPrefix(args, "inspect --format {{.State.Running}}") {
			return "", "Error: No such object", errors.New("exit 1")
		}
		return e.defaultDocker(args)
	}
	if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"}); err != nil {
		t.Fatalf("an interrupted run must not block the retry: %v", err)
	}
}

func TestReviewBlocksWhenTheDataDiskIsNearlyFull(t *testing.T) {
	e := newEnv(t)
	e.m.opt.FreeBytes = func(string) (uint64, error) { return 50 << 20, nil }
	e.reviewed(t)
	p := e.m.View(context.Background()).Plan
	found := false
	for _, b := range p.Blockers {
		found = found || b.Code == BlockDiskLow
	}
	if !found || p.State != PlanBlocked {
		t.Fatalf("got %+v", p)
	}
	if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"}); err == nil {
		t.Fatal("apply must refuse while the disk is nearly full")
	}
}

func TestReviewFailsClearlyWhenDockerCannotBeReached(t *testing.T) {
	e := newEnv(t)
	e.dk.handle = func(args string) (string, string, error) {
		return "", "Cannot connect to the Docker daemon at unix:///var/run/docker.sock", errors.New("exit 1")
	}
	e.m.Check(context.Background())
	e.m.Review("1.4.0")
	p := e.waitPlan(t, PlanFailed)
	if !strings.Contains(p.Error, "Cannot connect to the Docker daemon") {
		t.Fatalf("got %+v", p)
	}
	if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"}); !errors.Is(err, ErrNoPlan) {
		t.Fatalf("a failed review must not allow an apply, got %v", err)
	}
}

func TestApplyReportsAnUpdaterThatCouldNotStartAndRecordsTheFailure(t *testing.T) {
	e := newEnv(t)
	e.reviewed(t)
	e.dk.handle = func(args string) (string, string, error) {
		if strings.HasPrefix(args, "run -d") {
			return "", "docker: Error response from daemon: port is already allocated", errors.New("exit 125")
		}
		return e.defaultDocker(args)
	}
	if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"}); err == nil || !strings.Contains(err.Error(), "could not start the updater") {
		t.Fatalf("got %v", err)
	}
	if r := e.m.readRun(); r == nil || r.State != RunFailed {
		t.Fatalf("the run must be recorded as failed, or the UI would wait forever: %+v", r)
	}
	// and the next attempt is not blocked by it
	e.dk.handle = e.defaultDocker
	if _, err := e.m.Apply(context.Background(), ApplyRequest{Version: "1.4.0"}); err != nil {
		t.Fatalf("got %v", err)
	}
}
