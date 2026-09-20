package hostcheck

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeEnv is a healthy 64-bit Linux machine with Docker; tests break one thing at a time
func fakeEnv(t *testing.T) Env {
	t.Helper()
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	return Env{
		OS: "linux", Arch: "arm64",
		MemInfo:           write("meminfo", "MemTotal: 4000000 kB\nSwapTotal: 1000000 kB\n"),
		ModelFile:         filepath.Join(dir, "no-model"),
		CgroupControllers: filepath.Join(dir, "no-cgroup"),
		DockerSock:        filepath.Join(dir, "no-sock"),
		User:              "pi", UID: 1000, Sudo: "sudo",
		LookPath: func(n string) (string, error) { return "/usr/bin/" + n, nil },
		Run: func(name string, args ...string) (string, error) {
			switch strings.Join(append([]string{name}, args...), " ") {
			case "docker --version":
				return "Docker version 27.0.1, build abc\n", nil
			case "docker compose version --short":
				return "2.29.0\n", nil
			case "timedatectl show -p NTPSynchronized --value":
				return "yes\n", nil
			}
			return "", nil
		},
		PortFree: func(int) bool { return true },
		Reach:    func(string) error { return nil },
	}
}

func run(env Env, o Options) Report {
	o.Dir = os.TempDir()
	o.SkipNet = o.SkipNet || env.Reach == nil
	return Run(env, o)
}

func find(r Report, substr string) *Finding {
	for i := range r.Findings {
		if strings.Contains(r.Findings[i].Title, substr) {
			return &r.Findings[i]
		}
	}
	return nil
}

func TestHealthyMachineIsReady(t *testing.T) {
	r := run(fakeEnv(t), Options{Role: Primary})
	if !r.Ready() {
		t.Fatalf("expected ready, got %+v", r.Findings)
	}
}

func TestRejects32BitARM(t *testing.T) {
	e := fakeEnv(t)
	e.Arch = "armv7l"
	r := run(e, Options{Role: Primary})
	if f := find(r, "32-bit"); f == nil || f.Level != Fail {
		t.Fatalf("expected 32-bit failure, got %+v", r.Findings)
	}
}

func TestDockerMissingOffersInstall(t *testing.T) {
	e := fakeEnv(t)
	e.LookPath = func(n string) (string, error) {
		if n == "docker" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + n, nil
	}
	f := find(run(e, Options{Role: Primary}), "Docker is not installed")
	if f == nil || f.Fix == nil || !strings.Contains(f.Fix.Command, "get.docker.com") {
		t.Fatalf("expected install fix, got %+v", f)
	}
}

func TestDockerDaemonDiagnosis(t *testing.T) {
	base := fakeEnv(t).Run
	cases := map[string]struct{ info, title, fix string }{
		"permission": {"permission denied while trying to connect to the docker API", "not allowed to use Docker", "usermod -aG docker pi"},
		"stopped":    {"Cannot connect to the Docker daemon. Is the docker daemon running?", "daemon is not running", "systemctl enable --now docker"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := fakeEnv(t)
			e.Run = func(n string, a ...string) (string, error) {
				if n == "docker" && len(a) == 1 && a[0] == "info" {
					return tc.info, errors.New("exit 1")
				}
				return base(n, a...)
			}
			f := find(run(e, Options{Role: Primary}), tc.title)
			if f == nil || f.Level != Fail || f.Fix == nil || !strings.Contains(f.Fix.Command, tc.fix) {
				t.Fatalf("got %+v", f)
			}
		})
	}
}

func TestRaspberryPiMemoryAccounting(t *testing.T) {
	e := fakeEnv(t)
	dir := t.TempDir()
	e.ModelFile = filepath.Join(dir, "model")
	os.WriteFile(e.ModelFile, []byte("Raspberry Pi 4 Model B\x00"), 0o644)
	e.CmdlineFile = filepath.Join(dir, "cmdline.txt")
	os.WriteFile(e.CmdlineFile, []byte("console=serial0 root=/dev/mmcblk0p2\n"), 0o644)
	e.CgroupControllers = filepath.Join(dir, "controllers")
	os.WriteFile(e.CgroupControllers, []byte("cpuset cpu io pids\n"), 0o644)

	f := find(run(e, Options{Role: Primary}), "memory accounting is off")
	if f == nil || f.Fix == nil || !strings.Contains(f.Fix.Command, "cgroup_enable=memory cgroup_memory=1") || !strings.Contains(f.Fix.Command, e.CmdlineFile) {
		t.Fatalf("expected cmdline fix, got %+v", f)
	}

	// once the kernel exposes the controller, the warning goes away
	os.WriteFile(e.CgroupControllers, []byte("cpuset cpu io memory pids\n"), 0o644)
	if find(run(e, Options{Role: Primary}), "memory accounting is off") != nil {
		t.Fatal("warning should be gone when the memory controller is present")
	}
}

func TestNotRootSkipsSudoOnlyWhenRoot(t *testing.T) {
	e := fakeEnv(t)
	e.Sudo = ""
	e.Run = func(n string, a ...string) (string, error) {
		if len(a) > 0 && a[0] == "show" {
			return "no\n", nil
		}
		return fakeEnv(t).Run(n, a...)
	}
	f := find(run(e, Options{Role: Primary}), "clock is not synchronised")
	if f == nil || f.Fix.Command != "timedatectl set-ntp true" {
		t.Fatalf("root should not get sudo, got %+v", f)
	}
}

func TestDockerGIDMismatchFixEditsEnvFile(t *testing.T) {
	// use a real socket so the group can be read: the file's own group stands in for the socket's
	e := fakeEnv(t)
	sock := filepath.Join(os.TempDir(), "sfh-ctl-test.sock")
	os.Remove(sock)
	l, err := netListenUnix(sock)
	if err != nil {
		t.Skip("cannot create unix socket here:", err)
	}
	defer l.Close()
	defer os.Remove(sock)
	e.DockerSock = sock

	envFile := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(envFile, []byte("APP_UID=1000\nDOCKER_GID=99999\n"), 0o600)
	r := run(e, Options{Role: Primary, EnvFile: envFile})
	f := find(r, "DOCKER_GID=99999")
	if f == nil || f.Level != Fail || f.Fix == nil || f.Fix.Do == nil {
		t.Fatalf("expected DOCKER_GID failure with a repair, got %+v", r.Findings)
	}
	if err := f.Fix.Do(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(envFile)
	if strings.Contains(string(b), "99999") || !strings.Contains(string(b), "APP_UID=1000") {
		t.Fatalf("repair must change only DOCKER_GID, got %q", b)
	}
	if !run(e, Options{Role: Primary, EnvFile: envFile}).Ready() {
		t.Fatal("machine should be ready after the repair")
	}
}

func TestDirectModePortInUse(t *testing.T) {
	e := fakeEnv(t)
	e.PortFree = func(int) bool { return false }
	if f := find(run(e, Options{Role: SecondaryDirect, Port: 8082}), "port 8082 is already in use"); f == nil || f.Level != Fail {
		t.Fatalf("got %+v", f)
	}
	// a machine that connects out needs no port at all
	if find(run(e, Options{Role: SecondaryLink, Port: 8082}), "port 8082") != nil {
		t.Fatal("outbound-only role must not check ports")
	}
}

func TestSetEnvKeyReplacesAndAppends(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(p, []byte("A=1\nDOCKER_GID=5"), 0o600)
	if err := setEnvKey(p, "DOCKER_GID", "984"); err != nil {
		t.Fatal(err)
	}
	if err := setEnvKey(p, "NEW", "x"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "A=1\nDOCKER_GID=984\nNEW=x\n" {
		t.Fatalf("got %q", b)
	}
}
