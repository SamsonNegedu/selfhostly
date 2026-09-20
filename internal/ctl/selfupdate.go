package ctl

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const releaseRepo = "samsonnegedu/selfhostly"

func (a *App) selfUpdateCmd() *cobra.Command {
	var version string
	var checkOnly bool
	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update selfhostlyctl itself to the newest release",
		Long: `Downloads the selfhostlyctl release for this machine, checks its checksum and replaces this file.
With no --version it takes the newest numbered release, or the newest main build if there is no numbered release yet.
It does not touch your install: use "selfhostlyctl upgrade" for that. If the file is in a folder you cannot write to
(for example /usr/local/bin), run it with sudo.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return a.selfUpdate(version, checkOnly) },
	}
	cmd.Flags().StringVar(&version, "version", "latest", "latest, edge (the newest main), or a release such as v1.4.0")
	cmd.Flags().BoolVar(&checkOnly, "check", false, "only say whether a newer build is available")
	return cmd
}

func (a *App) releaseBase() string {
	if a.ReleaseBase != "" {
		return a.ReleaseBase
	}
	return "https://github.com/" + releaseRepo + "/releases"
}

func (a *App) httpGetBody(url string) ([]byte, error) {
	c := a.HTTPClient
	if c == nil {
		c = &http.Client{Timeout: 5 * time.Minute}
	}
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// releaseState says where the wanted release is and whether the running binary already is that build
type releaseState struct {
	base, file, want, exe string
	current               bool
}

func (a *App) releaseState(version string) (releaseState, error) {
	var st releaseState
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return st, fmt.Errorf("no release for %s (linux and macOS only)", runtime.GOOS)
	}
	st.file = fmt.Sprintf("selfhostlyctl_%s_%s", runtime.GOOS, runtime.GOARCH)
	base, sums, err := a.findRelease(version)
	if err != nil {
		return st, err
	}
	st.base = base
	if st.want = checksumFor(string(sums), st.file); st.want == "" {
		return st, fmt.Errorf("no checksum published for %s", st.file)
	}
	if st.exe = a.ExecPath; st.exe == "" {
		if st.exe, err = os.Executable(); err != nil {
			return st, err
		}
	}
	if st.exe, err = filepath.EvalSymlinks(st.exe); err != nil {
		return st, err
	}
	cur, err := fileSHA256(st.exe)
	if err != nil {
		return st, err
	}
	st.current = cur == st.want
	return st, nil
}

func (a *App) selfUpdate(version string, checkOnly bool) error {
	st, err := a.releaseState(version)
	if err != nil {
		return err
	}
	if st.current {
		a.say("selfhostlyctl %s is already the newest build (%s)", Version, version)
		return nil
	}
	if checkOnly {
		a.say("a newer build is available (you have %s). Run: selfhostlyctl self-update", Version)
		return nil
	}
	if a.DryRun {
		a.say("would replace %s with %s from %s", st.exe, st.file, st.base)
		return nil
	}

	a.say("downloading %s", st.file)
	body, err := a.httpGetBody(st.base + "/" + st.file)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	if got := hex.EncodeToString(sum[:]); got != st.want {
		return fmt.Errorf("checksum mismatch: refusing to install (expected %s, got %s)", st.want, got)
	}
	if err := replaceFile(st.exe, body); err != nil {
		return err
	}
	a.say("updated %s (was %s). Run selfhostlyctl version to see the new one.", st.exe, Version)
	return nil
}

// noteNewerTool tells the user when a newer selfhostlyctl exists. It is a hint after the real work is done, so
// it never fails, waits at most a few seconds, and stays quiet for a dev build, a dry run, or
// SELFHOSTLYCTL_NO_UPDATE_CHECK=1.
func (a *App) noteNewerTool() {
	if Version == "dev" || a.DryRun || os.Getenv("SELFHOSTLYCTL_NO_UPDATE_CHECK") != "" {
		return
	}
	if a.HTTPClient == nil {
		a.HTTPClient = &http.Client{Timeout: 5 * time.Second}
		defer func() { a.HTTPClient = nil }()
	}
	if st, err := a.releaseState("latest"); err == nil && !st.current {
		a.say("\nnote: a newer selfhostlyctl is available (you have %s). Update it with: selfhostlyctl self-update", Version)
	}
}

// findRelease returns the download folder and checksums.txt for the wanted release. "latest" falls back to the
// rolling edge build when no numbered release exists yet, like scripts/install.sh.
func (a *App) findRelease(version string) (base string, sums []byte, err error) {
	rel := a.releaseBase()
	if version == "latest" {
		base = rel + "/latest/download"
		if sums, err = a.httpGetBody(base + "/checksums.txt"); err == nil {
			return base, sums, nil
		}
		base = rel + "/download/edge"
	} else {
		base = rel + "/download/" + version
	}
	sums, err = a.httpGetBody(base + "/checksums.txt")
	if err != nil {
		return "", nil, fmt.Errorf("cannot find release %q: %w", version, err)
	}
	return base, sums, nil
}

func checksumFor(sums, file string) string {
	for _, line := range strings.Split(sums, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == file {
			return f[0]
		}
	}
	return ""
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// replaceFile swaps the file in one rename, so an interrupted update never leaves a half-written binary
func replaceFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".selfhostlyctl-update-*")
	if err != nil {
		return fmt.Errorf("cannot write next to %s (try sudo): %w", path, err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
