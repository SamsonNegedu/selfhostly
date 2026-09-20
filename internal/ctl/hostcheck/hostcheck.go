// Package hostcheck decides whether a machine is ready to run Selfhostly and says exactly how to fix
// what is not. It only reads: repairs are returned as Fix values and run by the caller, after asking.
//
// Every input (files, commands, network) goes through Env, so each check can be tested with fixture
// files and fake commands, including the Raspberry Pi ones, without needing a Pi.
package hostcheck

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Level is how serious a finding is
type Level int

const (
	OK Level = iota
	Warn
	Fail
)

// Fix is a repair. Command is what is shown to the user and run through the shell; Do, when set, is
// used instead for changes that need no privileges (editing a file this user owns).
type Fix struct {
	Description string
	Command     string
	Do          func() error
}

// Finding is one line of the report
type Finding struct {
	Level  Level
	Title  string
	Detail string
	Fix    *Fix
}

// Report is the result of a run
type Report struct {
	Findings  []Finding
	SocketGID int // group owning the Docker socket, 0 when there is none
}

// Counts returns how many findings there are at each level
func (r Report) Counts() (ok, warn, fail int) {
	for _, f := range r.Findings {
		switch f.Level {
		case OK:
			ok++
		case Warn:
			warn++
		default:
			fail++
		}
	}
	return
}

// Fixes returns every repair the report offers, in order
func (r Report) Fixes() []Fix {
	var out []Fix
	for _, f := range r.Findings {
		if f.Fix != nil {
			out = append(out, *f.Fix)
		}
	}
	return out
}

// Ready reports whether nothing failed
func (r Report) Ready() bool {
	_, _, fail := r.Counts()
	return fail == 0
}

// Role is what the machine will run
type Role string

const (
	Primary         Role = "primary"
	SecondaryDirect Role = "secondary-direct"
	SecondaryLink   Role = "secondary" // connects out: needs no port
)

// Options say what to check for
type Options struct {
	Role    Role
	EnvFile string // settings file whose DOCKER_GID, APP_UID, DATA_DIR and APPS_DIR are checked
	Dir     string // where Selfhostly lives (free disk is measured here)
	Port    int    // direct-mode API port to check is free; 0 skips
	SkipNet bool   // skip reachability probes
}

// Env is everything the checks read from the machine
type Env struct {
	OS, Arch                                           string
	MemInfo, ModelFile, CgroupControllers, CmdlineFile string
	DockerSock                                         string
	User                                               string
	UID                                                int
	Sudo                                               string // "" when already root, else "sudo"
	LookPath                                           func(string) (string, error)
	Run                                                func(name string, args ...string) (string, error)
	PortFree                                           func(port int) bool
	Reach                                              func(url string) error
}

// DefaultEnv reads the real machine
func DefaultEnv() Env {
	sudo := "sudo"
	if os.Geteuid() == 0 {
		sudo = ""
	}
	user := os.Getenv("USER")
	if user == "" {
		user = "user"
	}
	return Env{
		OS: runtime.GOOS, Arch: machineArch(),
		MemInfo: "/proc/meminfo", ModelFile: "/proc/device-tree/model",
		CgroupControllers: "/sys/fs/cgroup/cgroup.controllers",
		DockerSock:        "/var/run/docker.sock",
		User:              user, UID: os.Getuid(), Sudo: sudo,
		LookPath: exec.LookPath,
		Run: func(name string, args ...string) (string, error) {
			out, err := exec.Command(name, args...).CombinedOutput()
			return string(out), err
		},
		PortFree: func(port int) bool {
			l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				return false
			}
			_ = l.Close()
			return true
		},
		Reach: func(url string) error {
			c := &http.Client{Timeout: 8 * time.Second}
			resp, err := c.Head(url)
			if err != nil {
				return err
			}
			_ = resp.Body.Close()
			return nil
		},
	}
}

// Sizes the checks judge against
const (
	minDiskFail = 2 << 30 // bytes
	minDiskWarn = 8 << 30
	minRAMWarn  = 900 // MB
)

// Run performs every check for the options and returns the report
func Run(env Env, o Options) Report {
	c := &checker{env: env, opt: o}
	c.arch()
	c.memory()
	c.disk()
	c.timeSync()
	c.piMemory()
	daemonOK := c.docker()
	_ = daemonOK
	c.socket()
	c.envGroup()
	c.dirs()
	c.network()
	return Report{Findings: c.findings, SocketGID: c.sockGID}
}

type checker struct {
	env        Env
	opt        Options
	findings   []Finding
	sockGID    int
	haveSock   bool
	dockerInfo string
}

func (c *checker) add(l Level, title, detail string, fix *Fix) {
	c.findings = append(c.findings, Finding{Level: l, Title: title, Detail: detail, Fix: fix})
}

func (c *checker) linux() bool { return c.env.OS == "linux" }

func (c *checker) sudo(cmd string) string {
	if c.env.Sudo == "" {
		return cmd
	}
	return c.env.Sudo + " " + cmd
}

func (c *checker) installer(pkgs ...string) string {
	list := strings.Join(pkgs, " ")
	for _, m := range []struct{ bin, cmd string }{
		{"apt-get", "apt-get install -y "}, {"dnf", "dnf install -y "}, {"apk", "apk add "},
	} {
		if _, err := c.env.LookPath(m.bin); err == nil {
			return c.sudo(m.cmd + list)
		}
	}
	return ""
}

func (c *checker) arch() {
	switch c.env.Arch {
	case "amd64", "arm64":
		c.add(OK, fmt.Sprintf("architecture %s is supported", c.env.Arch), "", nil)
	case "arm", "armv7l", "armv6l", "armhf":
		c.add(Fail, fmt.Sprintf("architecture %s is 32-bit ARM", c.env.Arch),
			"images are published for 64-bit only (amd64, arm64). On a Raspberry Pi install the 64-bit OS", nil)
	default:
		c.add(Warn, fmt.Sprintf("architecture %s is untested", c.env.Arch), "images are published for amd64 and arm64", nil)
	}
}

func (c *checker) memory() {
	f, err := os.Open(c.env.MemInfo)
	if err != nil {
		return
	}
	defer f.Close()
	var totalKB, swapKB int
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			totalKB, _ = strconv.Atoi(fields[1])
		case "SwapTotal:":
			swapKB, _ = strconv.Atoi(fields[1])
		}
	}
	mb, swap := totalKB/1024, swapKB/1024
	if mb < minRAMWarn {
		c.add(Warn, fmt.Sprintf("%d MB RAM (%d MB swap)", mb, swap), "under 1 GB is tight for Docker plus apps; keep apps small or add swap", nil)
		return
	}
	c.add(OK, fmt.Sprintf("%d MB RAM (%d MB swap)", mb, swap), "", nil)
}

func (c *checker) disk() {
	dir := c.opt.Dir
	if dir == "" {
		dir = "."
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		c.add(Warn, "could not read free disk space for "+dir, "", nil)
		return
	}
	free := uint64(st.Bavail) * uint64(st.Bsize)
	gb := free >> 30
	switch {
	case free < minDiskFail:
		c.add(Fail, fmt.Sprintf("only %d GB free where Selfhostly lives (%s)", gb, dir),
			"images and app data need room: at least 2 GB, ideally 8 GB or more", nil)
	case free < minDiskWarn:
		c.add(Warn, fmt.Sprintf("%d GB free where Selfhostly lives (%s)", gb, dir), "enough to start, but images and app data grow", nil)
	default:
		c.add(OK, fmt.Sprintf("%d GB free where Selfhostly lives (%s)", gb, dir), "", nil)
	}
}

func (c *checker) timeSync() {
	if !c.linux() {
		return
	}
	if _, err := c.env.LookPath("timedatectl"); err != nil {
		c.add(Warn, "cannot check clock synchronisation (no timedatectl)", "", nil)
		return
	}
	out, err := c.env.Run("timedatectl", "show", "-p", "NTPSynchronized", "--value")
	v := strings.TrimSpace(out)
	switch {
	case err != nil || v == "":
		c.add(Warn, "cannot tell whether the clock is synchronised", "", nil)
	case v == "yes":
		c.add(OK, "the clock is synchronised", "", nil)
	default:
		c.add(Warn, "the clock is not synchronised", "login tokens and TLS both depend on correct time",
			&Fix{Description: "Turn on network time", Command: c.sudo("timedatectl set-ntp true")})
	}
}

// On a Raspberry Pi the kernel does not account container memory unless asked to, so the Monitoring
// page shows 0 MB for every container.
func (c *checker) piMemory() {
	if !c.linux() {
		return
	}
	model, err := os.ReadFile(c.env.ModelFile)
	if err != nil || !strings.Contains(string(model), "Raspberry Pi") {
		return
	}
	if b, err := os.ReadFile(c.env.CgroupControllers); err == nil && containsWord(string(b), "memory") {
		c.add(OK, "container memory accounting is enabled (Raspberry Pi, cgroup v2)", "", nil)
		return
	}
	cmdline := c.env.CmdlineFile
	if cmdline == "" {
		for _, p := range []string{"/boot/firmware/cmdline.txt", "/boot/cmdline.txt"} {
			if _, err := os.Stat(p); err == nil {
				cmdline = p
				break
			}
		}
	}
	if cmdline != "" {
		if b, err := os.ReadFile(cmdline); err == nil &&
			strings.Contains(string(b), "cgroup_enable=memory") && strings.Contains(string(b), "cgroup_memory=1") {
			c.add(OK, "container memory accounting is configured in "+cmdline, "", nil)
			return
		}
	}
	var fix *Fix
	if cmdline != "" {
		fix = &Fix{
			Description: "Enable memory accounting in " + cmdline + " (then reboot)",
			Command:     c.sudo(fmt.Sprintf("sed -i.bak '1 s/$/ cgroup_enable=memory cgroup_memory=1/' '%s'", cmdline)),
		}
	}
	c.add(Warn, "container memory accounting is off (Raspberry Pi)",
		"the Monitoring page would show 0 MB for containers. Needs a reboot to take effect", fix)
}

func containsWord(s, w string) bool {
	for _, f := range strings.Fields(s) {
		if f == w {
			return true
		}
	}
	return false
}

// docker checks the CLI, the Compose plugin, and that the daemon is running and usable by this user.
func (c *checker) docker() bool {
	if _, err := c.env.LookPath("docker"); err != nil {
		var fix *Fix
		if c.linux() {
			fix = &Fix{
				Description: "Install Docker with Docker's official convenience script (https://get.docker.com)",
				Command:     "curl -fsSL https://get.docker.com | " + strings.TrimSpace(c.env.Sudo+" sh"),
			}
		}
		c.add(Fail, "Docker is not installed", "Selfhostly runs your apps with Docker", fix)
		return false
	}
	ver, _ := c.env.Run("docker", "--version")
	c.add(OK, "Docker is installed ("+strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(ver), "Docker version "), ",")+")", "", nil)

	if out, err := c.env.Run("docker", "compose", "version", "--short"); err == nil {
		c.add(OK, "Docker Compose plugin is available ("+strings.TrimSpace(out)+")", "", nil)
	} else {
		var fix *Fix
		if cmd := c.installer("docker-compose-plugin"); cmd != "" {
			fix = &Fix{Description: "Install the Compose plugin", Command: cmd}
		}
		c.add(Fail, "the Docker Compose plugin is missing", "the deployment files use 'docker compose'", fix)
	}

	info, err := c.env.Run("docker", "info")
	if err == nil {
		c.dockerInfo = info
		c.add(OK, "the Docker daemon is running and you can use it", "", nil)
		return true
	}
	errLine := firstMatching(info, "error", "failed", "cannot", "denied", "not running")
	switch {
	case strings.Contains(info, "permission denied"):
		c.add(Fail, fmt.Sprintf("your user (%s) is not allowed to use Docker", c.env.User),
			"it is not in the group that owns the Docker socket",
			&Fix{
				Description: fmt.Sprintf("Add %s to the docker group (log out and back in afterwards, or run 'newgrp docker')", c.env.User),
				Command:     c.sudo("usermod -aG docker " + c.env.User),
			})
	case containsAny(info, "Cannot connect", "failed to connect to the docker API", "Is the docker daemon running", "no such file or directory"):
		var fix *Fix
		if c.linux() {
			fix = &Fix{Description: "Start Docker now and on every boot", Command: c.sudo("systemctl enable --now docker")}
		}
		c.add(Fail, "the Docker daemon is not running", errLine, fix)
		if !c.linux() {
			c.add(Warn, "start Docker Desktop or OrbStack", "there is no command that does this for you on "+c.env.OS, nil)
		}
	default:
		c.add(Fail, "Docker is installed but 'docker info' failed", errLine, nil)
	}
	return false
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

func firstMatching(s string, subs ...string) string {
	for _, line := range strings.Split(s, "\n") {
		l := strings.ToLower(line)
		for _, x := range subs {
			if strings.Contains(l, x) {
				return truncate(strings.TrimSpace(line), 200)
			}
		}
	}
	return truncate(strings.TrimSpace(strings.SplitN(s, "\n", 2)[0]), 200)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// socket reports which group owns the Docker socket: the backend container must run in that group.
func (c *checker) socket() {
	if !c.linux() {
		return
	}
	st, err := os.Stat(c.env.DockerSock)
	if err != nil || st.Mode()&os.ModeSocket == 0 {
		c.add(Warn, "no Docker socket at "+c.env.DockerSock, "fine for rootless Docker or a remote DOCKER_HOST", nil)
		return
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	c.sockGID, c.haveSock = int(sys.Gid), true
	c.add(OK, fmt.Sprintf("Docker socket is owned by group %d (mode %o): the backend container will run in group %d",
		sys.Gid, st.Mode().Perm(), sys.Gid), "", nil)
	if strings.Contains(c.dockerInfo, "rootless") {
		c.add(Warn, "this is rootless Docker", "the socket lives under $XDG_RUNTIME_DIR, not "+c.env.DockerSock+": mount that path instead", nil)
	}
}

// readEnvFile parses KEY=VALUE lines (no interpolation), enough for the settings the checks read
func readEnvFile(path string) map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			out[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return out
}

var dockerGIDLine = regexp.MustCompile(`(?m)^DOCKER_GID=.*$`)

// setEnvKey sets KEY=VALUE in an env file, replacing the line or appending it
func setEnvKey(path, key, value string) error {
	b, _ := os.ReadFile(path)
	s := string(b)
	line := key + "=" + value
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `=.*$`)
	if re.MatchString(s) {
		s = re.ReplaceAllString(s, line)
	} else {
		if s != "" && !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		s += line + "\n"
	}
	return os.WriteFile(path, []byte(s), 0o600)
}

const composeDefaultDockerGID = 984

// envGroup: DOCKER_GID in the settings file must agree with the socket, or the backend cannot use Docker
func (c *checker) envGroup() {
	if !c.haveSock || c.opt.EnvFile == "" {
		return
	}
	if _, err := os.Stat(c.opt.EnvFile); err != nil {
		return
	}
	env := readEnvFile(c.opt.EnvFile)
	configured, has := env["DOCKER_GID"]
	envFile, gid := c.opt.EnvFile, c.sockGID
	fix := &Fix{
		Description: fmt.Sprintf("Set DOCKER_GID=%d in %s", gid, envFile),
		Command:     fmt.Sprintf("set DOCKER_GID=%d in %s", gid, envFile),
		Do:          func() error { return setEnvKey(envFile, "DOCKER_GID", strconv.Itoa(gid)) },
	}
	switch {
	case !has || configured == "":
		if gid != composeDefaultDockerGID {
			c.add(Warn, fmt.Sprintf("DOCKER_GID is not set in %s but this machine's socket group is %d (the compose default is %d)",
				envFile, gid, composeDefaultDockerGID), "the backend could not use Docker", fix)
		}
	case configured != strconv.Itoa(gid):
		c.add(Fail, fmt.Sprintf("DOCKER_GID=%s in %s but the socket group here is %d", configured, envFile, gid),
			"the backend container could not use Docker", fix)
	default:
		c.add(OK, "DOCKER_GID in "+envFile+" matches the socket group", "", nil)
	}
}

// dirs: the folders the backend writes must belong to the user it runs as
func (c *checker) dirs() {
	env := readEnvFile(c.opt.EnvFile)
	data, apps := env["DATA_DIR"], env["APPS_DIR"]
	if data == "" {
		data = "./data"
		if c.opt.Role != Primary {
			data = "./node-data"
		}
	}
	if apps == "" {
		apps = "./apps"
		if c.opt.Role != Primary {
			apps = "./node-apps"
		}
	}
	uid := c.env.UID
	if v := env["APP_UID"]; v != "" {
		uid, _ = strconv.Atoi(v)
	} else if _, err := os.Stat(c.opt.EnvFile); err == nil {
		uid = 1000 // an existing install with nothing set uses the compose default
	}
	gid := uid
	if v := env["DOCKER_GID"]; v != "" {
		gid, _ = strconv.Atoi(v)
	}
	for _, d := range []string{data, apps} {
		st, err := os.Stat(d)
		if err != nil {
			c.add(OK, d+" does not exist yet (it will be created)", "", nil)
			continue
		}
		sys, ok := st.Sys().(*syscall.Stat_t)
		if !ok || int(sys.Uid) == uid || uid == 0 {
			c.add(OK, fmt.Sprintf("%s is owned by the user the backend runs as (%d)", d, uid), "", nil)
			continue
		}
		c.add(Warn, fmt.Sprintf("%s is owned by uid %d but the backend runs as uid %d", d, sys.Uid, uid),
			"it could not write its database or app files",
			&Fix{Description: fmt.Sprintf("Give %s to uid %d", d, uid), Command: c.sudo(fmt.Sprintf("chown -R %d:%d '%s'", uid, gid, filepath.Clean(d)))})
	}
}

func (c *checker) network() {
	if !c.opt.SkipNet {
		if err := c.env.Reach("https://ghcr.io"); err != nil {
			c.add(Fail, "cannot reach the image registry (ghcr.io)", "check the machine's internet connection or DNS", nil)
		} else {
			c.add(OK, "can reach the image registry (ghcr.io)", "", nil)
		}
		if c.opt.Role == Primary {
			if err := c.env.Reach("https://api.github.com"); err != nil {
				c.add(Warn, "cannot reach GitHub (needed for GitHub login)", "only needed for that feature", nil)
			} else {
				c.add(OK, "can reach GitHub (needed for GitHub login)", "", nil)
			}
		}
	}
	if c.opt.Role == SecondaryDirect && c.opt.Port > 0 {
		if c.env.PortFree(c.opt.Port) {
			c.add(OK, fmt.Sprintf("port %d is free (this node's API port)", c.opt.Port), "", nil)
		} else {
			c.add(Fail, fmt.Sprintf("port %d is already in use (this node's API port)", c.opt.Port), "choose another port or stop what is listening", nil)
		}
	}
}

// machineArch is the CPU of the machine, not of this binary (an amd64 build under emulation on an arm64
// machine would otherwise report the wrong one)
func machineArch() string {
	out, err := exec.Command("uname", "-m").Output()
	if err != nil {
		return runtime.GOARCH
	}
	switch m := strings.TrimSpace(string(out)); m {
	case "x86_64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		return m
	}
}
