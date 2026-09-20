package ctl

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// fakeRunner answers docker calls from a table keyed by the joined command line
type fakeRunner struct {
	out   map[string]string
	errs  map[string]string
	calls []string
}

func (f *fakeRunner) Output(_ context.Context, name string, args ...string) (string, string, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, key)
	if e, ok := f.errs[key]; ok {
		return "", e, errors.New("exit 1")
	}
	return f.out[key], "", nil
}

func (f *fakeRunner) OutputEnv(ctx context.Context, _ map[string]string, name string, args ...string) (string, string, error) {
	return f.Output(ctx, name, args...)
}

func (f *fakeRunner) Passthrough(ctx context.Context, name string, args ...string) error {
	_, _, err := f.Output(ctx, name, args...)
	return err
}

func newApp(t *testing.T, r *fakeRunner) (*App, *bytes.Buffer) {
	t.Helper()
	var out bytes.Buffer
	return &App{Out: &out, Err: &out, Run: r, Dir: t.TempDir(), EnvFile: ".env", Yes: true}, &out
}

func runCLI(a *App, args ...string) error {
	root := a.Root()
	root.SetArgs(args)
	return root.Execute()
}

const psAll = "docker ps --format {{.Names}}"

func probeKey(name string) string {
	return "docker exec " + name + " sh -c " + probeScript
}

// backendsRunning fakes a machine where these containers run: the value is what the probe prints
// (the container's role), and "-" marks a container that is not a Selfhostly backend.
func backendsRunning(containers map[string]string) map[string]string {
	out := map[string]string{}
	var names []string
	for n, role := range containers {
		names = append(names, n)
		if role != "-" {
			out[probeKey(n)] = "./selfhostly " + role + "\n"
		}
	}
	sort.Strings(names)
	out[psAll] = strings.Join(names, "\n") + "\n"
	return out
}

// notBackends makes the probe fail for containers marked "-"
func notBackends(r *fakeRunner, containers map[string]string) {
	r.errs = map[string]string{}
	for n, role := range containers {
		if role == "-" {
			r.errs[probeKey(n)] = "no such file"
		}
	}
}

func TestJoinTokenTunnelUsesPublicHost(t *testing.T) {
	world := backendsRunning(map[string]string{"anything-at-all": "true"})
	world["docker exec anything-at-all ./selfhostly join-token"] = "sfj_secret\n"
	r := &fakeRunner{out: world}
	a, out := newApp(t, r)
	os.WriteFile(filepath.Join(a.Dir, ".env"), []byte("PUBLIC_HOSTS=selfhostly.example.com,other.example.com\n"), 0o600)
	if err := runCLI(a, "join-token"); err != nil {
		t.Fatal(err)
	}
	want := "selfhostlyctl join --primary-url https://selfhostly.example.com --token sfj_secret"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("missing %q in:\n%s", want, out)
	}
}

func TestJoinTokenDirectWarnsWhenLoopbackOnly(t *testing.T) {
	world := backendsRunning(map[string]string{"anything-at-all": "true"})
	world["docker exec anything-at-all ./selfhostly join-token"] = "sfj_x\n"
	world["docker port anything-at-all 8082/tcp"] = "127.0.0.1:8082\n"
	r := &fakeRunner{out: world}
	a, out := newApp(t, r)
	if err := runCLI(a, "join-token", "--direct"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "only listens on this machine") || !strings.Contains(out.String(), "join --direct") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestJoinTokenWithoutPrimaryExplainsWhere(t *testing.T) {
	a, _ := newApp(t, &fakeRunner{})
	a.NonInteractive = true
	err := runCLI(a, "join-token")
	if err == nil || !strings.Contains(err.Error(), "machine that runs the primary") {
		t.Fatalf("got %v", err)
	}
}

func TestJoinTokenFailureShowsServerReason(t *testing.T) {
	r := &fakeRunner{
		out:  backendsRunning(map[string]string{"anything-at-all": "true"}),
		errs: map[string]string{"docker exec anything-at-all ./selfhostly join-token": "database is locked"},
	}
	a, _ := newApp(t, r)
	if err := runCLI(a, "join-token"); err == nil || !strings.Contains(err.Error(), "database is locked") {
		t.Fatalf("got %v", err)
	}
}

func TestDoctorPassesArgumentsAndFallsBackToNode(t *testing.T) {
	r := &fakeRunner{out: backendsRunning(map[string]string{"whatever-name": "false"})}
	a, _ := newApp(t, r)
	if err := runCLI(a, "doctor", "--audit-apps"); err != nil {
		t.Fatal(err)
	}
	last := r.calls[len(r.calls)-1]
	if last != "docker exec whatever-name ./selfhostly doctor --audit-apps" {
		t.Fatalf("got %q", last)
	}
}

func TestCheckRejectsUnknownRole(t *testing.T) {
	a, _ := newApp(t, &fakeRunner{})
	if err := runCLI(a, "check", "--role", "bogus"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestConfirmAnswersYesOnlyWithFlagWhenNobodyCanBeAsked(t *testing.T) {
	a, _ := newApp(t, &fakeRunner{})
	a.NonInteractive, a.Yes = true, false
	if a.confirm("x", false) {
		t.Fatal("must not agree without --yes")
	}
	a.Yes = true
	if !a.confirm("x", false) {
		t.Fatal("must agree with --yes")
	}
}

func TestPinImagesResolvesAllBeforeWriting(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	key := func(ref string) string {
		return "docker buildx imagetools inspect " + ref + " --format {{.Manifest.Digest}}"
	}
	r := &fakeRunner{out: map[string]string{
		key("ghcr.io/samsonnegedu/selfhostly-gateway:1.4.0"):  digest + "\n",
		key("ghcr.io/samsonnegedu/selfhostly-backend:1.4.0"):  digest + "\n",
		key("ghcr.io/samsonnegedu/selfhostly-frontend:1.4.0"): "not a digest\n",
		key("cloudflare/cloudflared:latest"):                  digest + "\n",
	}}
	a, _ := newApp(t, r)
	envPath := filepath.Join(a.Dir, ".env")
	os.WriteFile(envPath, []byte("KEEP=1\n"), 0o600)
	if err := runCLI(a, "pin-images", "--version", "1.4.0"); err == nil {
		t.Fatal("an unresolvable image must fail")
	}
	if b, _ := os.ReadFile(envPath); string(b) != "KEEP=1\n" {
		t.Fatalf("the settings file must be untouched after a failure, got %q", b)
	}
	r.out[key("ghcr.io/samsonnegedu/selfhostly-frontend:1.4.0")] = digest + "\n"
	if err := runCLI(a, "pin-images", "--version", "1.4.0"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(envPath)
	if !strings.Contains(string(b), "BACKEND_IMAGE=ghcr.io/samsonnegedu/selfhostly-backend@"+digest) || !strings.Contains(string(b), "KEEP=1") {
		t.Fatalf("got %q", b)
	}
}

// pickFirst is a Prompter that always chooses the first option and remembers what it was shown
type pickFirst struct{ shown []string }

func (p *pickFirst) Confirm(string, bool) (bool, error)             { return true, nil }
func (p *pickFirst) Input(_, _, def string, _ bool) (string, error) { return def, nil }
func (p *pickFirst) Select(_ string, options []string) (string, error) {
	p.shown = options
	return options[0], nil
}

func TestFindsTheBackendByWhatItIsNotWhatItIsCalled(t *testing.T) {
	c := map[string]string{"web": "-", "postgres": "-", "tomorrows-new-name": "true"}
	r := &fakeRunner{out: backendsRunning(c)}
	notBackends(r, c)
	a, _ := newApp(t, r)
	if err := runCLI(a, "doctor", "--audit-apps"); err != nil {
		t.Fatal(err)
	}
	if last := r.calls[len(r.calls)-1]; last != "docker exec tomorrows-new-name ./selfhostly doctor --audit-apps" {
		t.Fatalf("got %q", last)
	}
}

func TestContainerFlagOverridesDiscovery(t *testing.T) {
	out := backendsRunning(map[string]string{"auto-found": "true"})
	out["docker ps --format {{.Names}} --filter name=^mine$"] = "mine\n"
	out[probeKey("mine")] = "./selfhostly true\n"
	r := &fakeRunner{out: out}
	a, _ := newApp(t, r)
	if err := runCLI(a, "doctor", "--container", "mine"); err != nil {
		t.Fatal(err)
	}
	if last := r.calls[len(r.calls)-1]; !strings.HasPrefix(last, "docker exec mine ./selfhostly doctor") || strings.Contains(last, "--container") {
		t.Fatalf("got %q", last)
	}
	if err := runCLI(a, "doctor", "--container", "gone"); err == nil || !strings.Contains(err.Error(), "not running") {
		t.Fatalf("a named container that is not running must say so, got %v", err)
	}
}

func TestSeveralBackendsAreAskedAboutWhenInteractive(t *testing.T) {
	r := &fakeRunner{out: backendsRunning(map[string]string{"a": "true", "b": "false"})}
	a, _ := newApp(t, r)
	pick := &pickFirst{}
	a.Prompt = pick
	if err := runCLI(a, "doctor"); err != nil {
		t.Fatal(err)
	}
	if len(pick.shown) != 2 || !strings.HasPrefix(r.calls[len(r.calls)-1], "docker exec a ") {
		t.Fatalf("shown %v, last call %q", pick.shown, r.calls[len(r.calls)-1])
	}
}

func TestSeveralBackendsWithoutATerminalListThemAndNameTheFlag(t *testing.T) {
	r := &fakeRunner{out: backendsRunning(map[string]string{"a": "true", "b": "false"})}
	a, _ := newApp(t, r)
	a.NonInteractive = true
	err := runCLI(a, "doctor")
	if err == nil || !strings.Contains(err.Error(), "a, b") || !strings.Contains(err.Error(), "--container") {
		t.Fatalf("got %v", err)
	}
}

func TestPrimaryOnlyCommandsSkipSecondaries(t *testing.T) {
	out := backendsRunning(map[string]string{"the-node": "false", "the-primary": "true"})
	out["docker exec the-primary ./selfhostly join-token"] = "sfj_x\n"
	r := &fakeRunner{out: out}
	a, _ := newApp(t, r)
	a.NonInteractive = true
	if err := runCLI(a, "join-token"); err != nil {
		t.Fatalf("a secondary must not make join-token ambiguous: %v", err)
	}
}

func TestNothingLooksLikeABackendOffersEveryRunningContainer(t *testing.T) {
	c := map[string]string{"web": "-", "dev-api": "-"}
	out := backendsRunning(c)
	out["docker exec dev-api ./selfhostly doctor"] = ""
	r := &fakeRunner{out: out}
	notBackends(r, c)
	a, _ := newApp(t, r)
	pick := &pickFirst{}
	a.Prompt = pick
	if err := runCLI(a, "doctor"); err == nil || !strings.Contains(err.Error(), "no Selfhostly server binary") {
		t.Fatalf("a picked container without the server must be explained: %v", err)
	}
	if len(pick.shown) != 2 {
		t.Fatalf("shown %v", pick.shown)
	}
	a2, _ := newApp(t, r)
	a2.NonInteractive = true
	if err := runCLI(a2, "doctor"); err == nil || !strings.Contains(err.Error(), "--container") {
		t.Fatalf("without a terminal it must list them and name the flag: %v", err)
	}
}

func TestUsesTheBinaryWhereverTheContainerKeepsIt(t *testing.T) {
	world := backendsRunning(map[string]string{"dev-api": "true"})
	world[probeKey("dev-api")] = "./tmp/main true\n" // the dev stack's live-reload build
	r := &fakeRunner{out: world}
	a, _ := newApp(t, r)
	if err := runCLI(a, "doctor", "--audit-apps"); err != nil {
		t.Fatal(err)
	}
	if last := r.calls[len(r.calls)-1]; last != "docker exec dev-api ./tmp/main doctor --audit-apps" {
		t.Fatalf("got %q", last)
	}
}

func TestAPickedContainerWithoutAServerSaysSoInsteadOfAnOCIError(t *testing.T) {
	c := map[string]string{"web": "-"}
	r := &fakeRunner{out: backendsRunning(c)}
	notBackends(r, c)
	a, _ := newApp(t, r)
	a.Prompt = &pickFirst{}
	err := runCLI(a, "doctor")
	if err == nil || !strings.Contains(err.Error(), "no Selfhostly server binary") || !strings.Contains(err.Error(), "./tmp/main") {
		t.Fatalf("got %v", err)
	}
	for _, call := range r.calls {
		if strings.Contains(call, "doctor") {
			t.Fatalf("must not run doctor in a container without the server: %v", r.calls)
		}
	}
}
