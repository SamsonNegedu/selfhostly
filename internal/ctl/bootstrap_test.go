package ctl

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBootstrapNewInstallEnforcesAndIsIdempotent(t *testing.T) {
	a, _ := newApp(t, &fakeRunner{})
	a.DockerSock = filepath.Join(a.Dir, "none")
	if err := runCLI(a, "bootstrap", "--auth", "github", "--domain", "h.example.com", "--github-user", "me", "--quiet"); err != nil {
		t.Fatal(err)
	}
	env := EnvFile{filepath.Join(a.Dir, ".env")}
	all := env.All()
	for _, k := range []string{"JWT_SECRET", "GATEWAY_API_KEY", "SETTINGS_ENCRYPTION_KEY", "NODE_ID", "HOST_APPS_DIR"} {
		if all[k] == "" {
			t.Fatalf("%s not set: %v", k, all)
		}
	}
	if all["SECURITY_MODE"] != "enforce" || all["ENCRYPT_SECRETS_AT_REST"] != "true" || all["PUBLIC_HOSTS"] != "h.example.com" || all["GITHUB_ALLOWED_USERS"] != "me" {
		t.Fatalf("unexpected settings: %v", all)
	}
	if fi, _ := os.Stat(env.Path); fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", fi.Mode().Perm())
	}
	before, _ := os.ReadFile(env.Path)
	if err := runCLI(a, "bootstrap", "--auth", "github", "--domain", "other.example.com", "--quiet"); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(env.Path); string(after) != string(before) {
		t.Fatalf("second run changed the file:\n%s\n---\n%s", before, after)
	}
}

func TestBootstrapExistingInstallWarnsAndKeepsEncryptionOffAndNeverInventsNodeID(t *testing.T) {
	a, _ := newApp(t, &fakeRunner{})
	a.DockerSock = filepath.Join(a.Dir, "none")
	os.MkdirAll(filepath.Join(a.Dir, "data"), 0o755)
	os.WriteFile(filepath.Join(a.Dir, "data", "selfhostly.db"), []byte("x"), 0o600)
	if err := runCLI(a, "bootstrap", "--auth", "none", "--quiet"); err != nil {
		t.Fatal(err)
	}
	all := EnvFile{filepath.Join(a.Dir, ".env")}.All()
	if all["SECURITY_MODE"] != "warn" || all["ENCRYPT_SECRETS_AT_REST"] != "false" {
		t.Fatalf("existing install must start in warn with encryption off: %v", all)
	}
	if all["NODE_ID"] != "" {
		t.Fatal("an existing install's node id comes from its database, never invented")
	}
	if _, ok := all["APP_UID"]; ok {
		t.Fatal("an existing install keeps the compose defaults for the user")
	}
}

func TestBootstrapPrintPlanWritesNothing(t *testing.T) {
	a, _ := newApp(t, &fakeRunner{})
	a.DockerSock = filepath.Join(a.Dir, "none")
	if err := runCLI(a, "bootstrap", "--print-plan"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(a.Dir, ".env")); err == nil {
		t.Fatal("--print-plan must not create the settings file")
	}
}

func TestBackupArchiveHoldsSnapshotKeysAndSettingsPrivately(t *testing.T) {
	out := backendsRunning(map[string]string{"whatever-it-is-called": "true"})
	out["docker exec whatever-it-is-called ./selfhostly backup --dir /app/data/backups"] = "note\n/app/data/backups/snap.db\n"
	r := &fakeRunner{out: out}
	a, _ := newApp(t, r)
	a.Now = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	data := filepath.Join(a.Dir, "data")
	os.MkdirAll(filepath.Join(data, "backups"), 0o755)
	os.WriteFile(filepath.Join(data, "backups", "snap.db"), []byte("SNAP"), 0o600)
	os.WriteFile(filepath.Join(data, "secrets.key"), []byte("KEY"), 0o600)
	os.WriteFile(filepath.Join(data, "selfhostly.db"), []byte("LIVE"), 0o600)
	os.WriteFile(filepath.Join(a.Dir, ".env"), []byte("A=1\n"), 0o600)

	dest := filepath.Join(a.Dir, "out")
	if err := runCLI(a, "backup", dest); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dest, "selfhostly-backup-20260102T030405Z.tar.gz")
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("archive missing or not private: %v %v", err, fi)
	}
	f, _ := os.Open(path)
	gz, _ := gzip.NewReader(f)
	tr := tar.NewReader(gz)
	got := map[string]string{}
	for {
		h, err := tr.Next()
		if err != nil {
			break
		}
		b, _ := io.ReadAll(tr)
		got[h.Name] = string(b)
	}
	if got["data/selfhostly.db"] != "SNAP" || got["data/secrets.key"] != "KEY" || got["env"] != "A=1\n" {
		t.Fatalf("archive contents: %v", got)
	}
	for name, body := range got {
		if strings.Contains(body, "LIVE") {
			t.Fatalf("the live database must not be in the archive (%s)", name)
		}
	}
}

func TestSetupConfiguresStartsAndVerifies(t *testing.T) {
	st := &stack{primaryHealthy: true}
	a, sc, out := upgradeApp(t, st)
	os.Remove(filepath.Join(a.Dir, ".env"))
	a.SkipHostChecks, a.DockerSock = true, filepath.Join(a.Dir, "none")
	err := runCLI(a, "setup", "--auth", "github", "--domain", "s.example.com", "--github-user", "me",
		"--github-client-id", "id", "--github-client-secret", "sec", "--tunnel-token", "tok")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	all := EnvFile{filepath.Join(a.Dir, ".env")}.All()
	if all["GITHUB_CLIENT_ID"] != "id" || all["TUNNEL_TOKEN"] != "tok" || all["SECURITY_MODE"] != "enforce" {
		t.Fatalf("settings: %v", all)
	}
	if sc.index("up -d primary gateway frontend cloudflared") < 0 {
		t.Fatalf("expected all four services to start:\n%v", sc.calls)
	}
	if sc.index("doctor --offline") < 0 {
		t.Fatal("setup must finish by running doctor")
	}
}

func TestSetupWithoutTunnelTokenStartsTheRestAndSaysWhatIsMissing(t *testing.T) {
	a, sc, out := upgradeApp(t, &stack{primaryHealthy: true})
	os.Remove(filepath.Join(a.Dir, ".env"))
	a.SkipHostChecks, a.DockerSock = true, filepath.Join(a.Dir, "none")
	if err := runCLI(a, "setup", "--auth", "none"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if sc.index("cloudflared") >= 0 || !strings.Contains(out.String(), "Still missing") {
		t.Fatalf("the tunnel must not start without a token, and the gap must be reported:\n%s", out)
	}
}

func TestSetupDryRunTouchesNothing(t *testing.T) {
	a, sc, _ := upgradeApp(t, &stack{primaryHealthy: true})
	os.Remove(filepath.Join(a.Dir, ".env"))
	a.SkipHostChecks, a.DryRun = true, true
	if err := runCLI(a, "setup", "--auth", "none"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(a.Dir, ".env")); err == nil || len(sc.calls) > 0 {
		t.Fatalf("dry run changed something: %v", sc.calls)
	}
}
