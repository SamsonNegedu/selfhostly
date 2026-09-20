package ctl

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeRelease serves the given paths (a path maps to its body); anything else is a 404
func fakeRelease(t *testing.T, files map[string]string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func sumLine(body, name string) string {
	s := sha256.Sum256([]byte(body))
	return hex.EncodeToString(s[:]) + "  " + name + "\n"
}

func selfUpdateApp(t *testing.T, base string) (*App, string) {
	a, _ := newApp(t, &fakeRunner{})
	exe := filepath.Join(t.TempDir(), "selfhostlyctl")
	os.WriteFile(exe, []byte("old binary"), 0o755)
	a.ReleaseBase, a.ExecPath = base, exe
	return a, exe
}

func TestSelfUpdateReplacesTheBinary(t *testing.T) {
	name := fmt.Sprintf("selfhostlyctl_%s_%s", runtime.GOOS, runtime.GOARCH)
	base := fakeRelease(t, map[string]string{
		"/latest/download/checksums.txt": sumLine("new binary", name),
		"/latest/download/" + name:       "new binary",
	})
	a, exe := selfUpdateApp(t, base)
	if err := runCLI(a, "self-update"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "new binary" {
		t.Fatalf("binary not replaced: %q", got)
	}
	if st, _ := os.Stat(exe); st.Mode().Perm() != 0o755 {
		t.Fatalf("mode %v", st.Mode())
	}
	if err := runCLI(a, "self-update"); err != nil {
		t.Fatal(err)
	}
}

func TestSelfUpdateRefusesABadChecksum(t *testing.T) {
	name := fmt.Sprintf("selfhostlyctl_%s_%s", runtime.GOOS, runtime.GOARCH)
	base := fakeRelease(t, map[string]string{
		"/latest/download/checksums.txt": sumLine("the real one", name),
		"/latest/download/" + name:       "tampered",
	})
	a, exe := selfUpdateApp(t, base)
	if err := runCLI(a, "self-update"); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("want checksum mismatch, got %v", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatal("the binary was changed despite the bad checksum")
	}
}

func TestSelfUpdateFallsBackToEdgeAndHonoursCheckAndDryRun(t *testing.T) {
	name := fmt.Sprintf("selfhostlyctl_%s_%s", runtime.GOOS, runtime.GOARCH)
	base := fakeRelease(t, map[string]string{
		"/download/edge/checksums.txt": sumLine("edge binary", name),
		"/download/edge/" + name:       "edge binary",
	})
	a, exe := selfUpdateApp(t, base)
	if err := runCLI(a, "self-update", "--check"); err != nil {
		t.Fatal(err)
	}
	a.DryRun = true
	if err := runCLI(a, "self-update"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatal("--check and --dry-run must not change the binary")
	}
	a.DryRun = false
	if err := runCLI(a, "self-update", "--version", "edge"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "edge binary" {
		t.Fatalf("got %q", got)
	}
}

func TestSelfUpdateUnknownVersion(t *testing.T) {
	a, _ := selfUpdateApp(t, fakeRelease(t, nil))
	if err := runCLI(a, "self-update", "--version", "v9.9.9"); err == nil || !strings.Contains(err.Error(), "v9.9.9") {
		t.Fatalf("got %v", err)
	}
}

func TestNoteNewerToolOnlyWhenNewerAndNeverFails(t *testing.T) {
	name := fmt.Sprintf("selfhostlyctl_%s_%s", runtime.GOOS, runtime.GOARCH)
	base := fakeRelease(t, map[string]string{"/latest/download/checksums.txt": sumLine("new binary", name)})
	old := Version
	defer func() { Version = old }()

	a, out := newApp(t, &fakeRunner{})
	a.ReleaseBase, a.ExecPath = base, filepath.Join(t.TempDir(), "ctl")
	os.WriteFile(a.ExecPath, []byte("old binary"), 0o755)

	Version = "dev"
	a.noteNewerTool()
	if out.Len() != 0 {
		t.Fatalf("a dev build must stay quiet, got %q", out)
	}
	Version = "v1.0.0"
	a.noteNewerTool()
	if !strings.Contains(out.String(), "self-update") {
		t.Fatalf("want a hint, got %q", out)
	}
	out.Reset()
	os.WriteFile(a.ExecPath, []byte("new binary"), 0o755)
	a.noteNewerTool()
	if out.Len() != 0 {
		t.Fatalf("an up to date tool must stay quiet, got %q", out)
	}
	a.ReleaseBase = "http://127.0.0.1:1" // unreachable
	os.WriteFile(a.ExecPath, []byte("old binary"), 0o755)
	a.noteNewerTool()
	if out.Len() != 0 {
		t.Fatalf("a failed check must stay quiet, got %q", out)
	}
	t.Setenv("SELFHOSTLYCTL_NO_UPDATE_CHECK", "1")
	a.ReleaseBase = base
	a.noteNewerTool()
	if out.Len() != 0 {
		t.Fatalf("opt-out ignored, got %q", out)
	}
}
