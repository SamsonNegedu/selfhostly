package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostPathFromMounts(t *testing.T) {
	mounts := []byte(`[{"Source":"/srv/selfhostly/apps","Destination":"/app/apps"},{"Source":"/srv/selfhostly/data","Destination":"/app/data"},{"Source":"/var/run/docker.sock","Destination":"/var/run/docker.sock"}]`)
	got, ok := hostPathFromMounts(mounts, "/app/apps")
	if !ok || got != "/srv/selfhostly/apps" {
		t.Fatalf("got %q %v", got, ok)
	}
	if got, _ := hostPathFromMounts(mounts, "/app/apps/kan"); got != "/srv/selfhostly/apps/kan" {
		t.Fatalf("sub path: %q", got)
	}
	if _, ok := hostPathFromMounts(mounts, "/elsewhere"); ok {
		t.Fatal("unmounted path must not resolve")
	}
	if _, ok := hostPathFromMounts([]byte("garbage"), "/app/apps"); ok {
		t.Fatal("bad json must not resolve")
	}
}

func TestCommandEnvStripsPlatformSecrets(t *testing.T) {
	t.Setenv("JWT_SECRET", "s3cret")
	t.Setenv("CLOUDFLARE_API_TOKEN", "tok")
	t.Setenv("HARMLESS_SETTING", "keep")
	env := strings.Join(commandEnv(), "\n")
	if strings.Contains(env, "s3cret") || strings.Contains(env, "CLOUDFLARE_API_TOKEN") {
		t.Fatal("platform secrets must not reach docker compose")
	}
	if !strings.Contains(env, "HARMLESS_SETTING=keep") || os.Getenv("PATH") != "" && !strings.Contains(env, "PATH=") {
		t.Fatal("ordinary variables must be preserved")
	}
}

func TestComposeGuardBlocksDeploy(t *testing.T) {
	dir := t.TempDir()
	m := NewManagerWithExecutor(dir, NewMockCommandExecutor())
	if err := m.CreateAppDirectory("a", "services: {}"); err != nil {
		t.Fatal(err)
	}
	m.SetComposeGuard(func(string, []byte) error { return os.ErrPermission })
	if err := m.StartApp("a"); err == nil || !strings.Contains(err.Error(), "security policy") {
		t.Fatalf("guard must block StartApp, got %v", err)
	}
	m.SetComposeGuard(func(string, []byte) error { return nil })
	if err := m.checkGuard("a"); err != nil {
		t.Fatal(err)
	}
}

func TestMissingEnvFileBlocksDeploy(t *testing.T) {
	const compose = "services:\n  web:\n    image: nginx\n    env_file: .env\n"
	dir := t.TempDir()
	m := NewManagerWithExecutor(dir, NewMockCommandExecutor())
	if err := m.CreateAppDirectory("a", compose); err != nil {
		t.Fatal(err)
	}

	err := m.StartApp("a")
	if err == nil || !strings.Contains(err.Error(), ".env") || !strings.Contains(err.Error(), filepath.Join(dir, "a")) {
		t.Fatalf("missing env_file must block StartApp and name the file and folder, got %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "a", ".env"), []byte("A=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := m.checkGuard("a"); err != nil {
		t.Fatalf("present env_file must pass, got %v", err)
	}
}

func TestOptionalAndInterpolatedEnvFiles(t *testing.T) {
	dir := t.TempDir()
	m := NewManagerWithExecutor(dir, NewMockCommandExecutor())
	optional := "services:\n  web:\n    image: nginx\n    env_file:\n      - path: .env\n        required: false\n"
	if err := m.CreateAppDirectory("opt", optional); err != nil {
		t.Fatal(err)
	}
	if err := m.checkGuard("opt"); err != nil {
		t.Fatalf("required: false must pass when the file is missing, got %v", err)
	}
	// A path that depends on a variable cannot be checked without Docker, so it is left to Docker.
	interp := "services:\n  web:\n    image: nginx\n    env_file: ${ENV_FILE}\n"
	if err := m.CreateAppDirectory("interp", interp); err != nil {
		t.Fatal(err)
	}
	if err := m.checkGuard("interp"); err != nil {
		t.Fatalf("interpolated env_file must be left to Docker, got %v", err)
	}
}
