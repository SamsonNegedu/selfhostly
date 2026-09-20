package docker

import (
	"os"
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
