package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func compose(svcBody string, extra string) string {
	return "services:\n  a:\n    image: x\n" + svcBody + extra
}

func TestPolicyBlocksHostEscapesInAnyMode(t *testing.T) {
	cases := map[string]string{
		"run exposes docker.sock": compose("    volumes:\n      - /run:/h\n", ""),
		"var exposes var/run":     compose("    volumes:\n      - /var:/h\n", ""),
		"usr":                     compose("    volumes:\n      - /usr:/h\n", ""),
		"long form root":          compose("    volumes:\n      - type: bind\n        source: /\n        target: /h\n", ""),
		"long form run":           compose("    volumes:\n      - type: bind\n        source: /run\n        target: /h\n", ""),
		"relative escape":         compose("    volumes:\n      - ./../../data:/h\n", ""),
		"driver_opts bind":        compose("    volumes:\n      - v:/h\n", "volumes:\n  v:\n    driver_opts:\n      type: none\n      o: bind\n      device: /\n"),
		"userns host":             compose("    userns_mode: host\n", ""),
		"cgroup host":             compose("    cgroup: host\n", ""),
		"uts host":                compose("    uts: host\n", ""),
		"group_add docker":        compose("    group_add: [docker]\n", ""),
		"device cgroup rules":     compose("    device_cgroup_rules:\n      - 'c 1:3 rwm'\n", ""),
		"cap bpf":                 compose("    cap_add: [BPF]\n", ""),
		"systempaths unconfined":  compose("    security_opt:\n      - systempaths=unconfined\n", ""),
		"env_file absolute":       compose("    env_file: /etc/shadow\n", ""),
		"env_file climbs":         compose("    env_file:\n      - ../other/.env\n", ""),
		"build context absolute":  compose("    build: /etc\n", ""),
		"secret file /etc":        compose("", "secrets:\n  s:\n    file: /etc/shadow\n"),
		"extends":                 compose("    extends:\n      file: /x.yml\n      service: y\n", ""),
	}
	for name, body := range cases {
		for _, enforce := range []bool{false, true} {
			cfg := &SecurityConfig{Enforce: enforce, AppName: "a", HostAppsDir: "/srv/apps"}
			if err := ValidateComposeContentWithConfig(body, cfg); err == nil {
				t.Errorf("%s (enforce=%v): expected rejection", name, enforce)
			}
		}
	}
}

func TestIncludeIsRejectedBeforeAnyFileRead(t *testing.T) {
	err := ValidateComposeContentWithConfig("include:\n  - /etc/x.yml\nservices:\n  a:\n    image: x\n", nil)
	if err == nil || !strings.Contains(err.Error(), "include") {
		t.Fatalf("expected include rejection, got %v", err)
	}
}

func TestPolicyAllowListBehaviourByMode(t *testing.T) {
	body := compose("    volumes:\n      - /mnt/media:/media\n", "")
	warn := &SecurityConfig{Enforce: false, AppName: "a", HostAppsDir: "/srv/apps"}
	if err := ValidateComposeContentWithConfig(body, warn); err != nil {
		t.Fatalf("warn mode must tolerate an unlisted path: %v", err)
	}
	enforce := &SecurityConfig{Enforce: true, AppName: "a", HostAppsDir: "/srv/apps"}
	if err := ValidateComposeContentWithConfig(body, enforce); err == nil {
		t.Fatal("enforce mode must reject an unlisted path")
	}
	enforce.AllowedVolumePaths = []string{"/mnt/media"}
	if err := ValidateComposeContentWithConfig(body, enforce); err != nil {
		t.Fatalf("listed path must pass: %v", err)
	}
}

func TestPolicyAllowsAppDirectoryAndNamedVolumes(t *testing.T) {
	body := compose("    volumes:\n      - ./data:/data\n      - cache:/cache\n      - /srv/apps/a/config:/config\n", "volumes:\n  cache: {}\n")
	cfg := &SecurityConfig{Enforce: true, AppName: "a", HostAppsDir: "/srv/apps"}
	if err := ValidateComposeContentWithConfig(body, cfg); err != nil {
		t.Fatalf("app-local paths must pass: %v", err)
	}
	files := compose("    env_file: .env\n    build:\n      context: ./src\n", "")
	if res := CheckComposePolicy([]byte(files), cfg); len(res.Hard)+len(res.Soft) != 0 {
		t.Fatalf("relative env_file and build context must pass: %+v", res)
	}
}

func TestPolicyInterpolatedBindIsSoft(t *testing.T) {
	body := compose("    volumes:\n      - ${DATA}/x:/x\n", "")
	if err := ValidateComposeContentWithConfig(body, &SecurityConfig{Enforce: true, AppName: "a", HostAppsDir: "/srv/apps"}); err == nil {
		t.Fatal("enforce mode must reject unverifiable interpolated bind sources")
	}
}

func TestPolicyContainerNamespaceIsSoft(t *testing.T) {
	body := compose("    network_mode: container:vpn\n", "")
	if err := ValidateComposeContentWithConfig(body, &SecurityConfig{Enforce: false}); err != nil {
		t.Fatalf("warn mode tolerates container: joins: %v", err)
	}
	if err := ValidateComposeContentWithConfig(body, &SecurityConfig{Enforce: true}); err == nil {
		t.Fatal("enforce mode rejects container: joins")
	}
}

func TestPolicySymlinkEscapeIsBlocked(t *testing.T) {
	appsDir := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(appsDir, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(appsDir, "a", "link")); err != nil {
		t.Skip("symlinks unavailable")
	}
	real, _ := filepath.EvalSymlinks(appsDir)
	cfg := &SecurityConfig{Enforce: true, AppName: "a", HostAppsDir: real, AppsDir: appsDir}
	body := compose("    volumes:\n      - ./link:/x\n", "")
	if err := ValidateComposeContentWithConfig(body, cfg); err == nil {
		t.Fatal("a symlink leaving the apps directory must be rejected")
	}
}

func TestAuditComposeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(path, []byte(compose("    volumes:\n      - /mnt/x:/x\n", "")), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := AuditComposeFile(path, &SecurityConfig{AppName: "a", HostAppsDir: "/srv/apps"})
	if err != nil || len(res.Soft) != 1 || len(res.Hard) != 0 {
		t.Fatalf("unexpected audit result %+v %v", res, err)
	}
}
