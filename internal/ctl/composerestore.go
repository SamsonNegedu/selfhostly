package ctl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const composeSaveDir = "compose"

// saveComposeFiles copies the compose files this run manages into the rollback point, so a rollback can put back
// the file that worked. The list keeps the original paths; the copies are numbered.
func (u *upgrader) saveComposeFiles(dir string) error {
	saved := filepath.Join(dir, composeSaveDir)
	if err := os.MkdirAll(saved, 0o700); err != nil {
		return err
	}
	var list []string
	for i, f := range u.dc.files {
		if !fileExists(f) {
			continue
		}
		if err := copyFile(f, filepath.Join(saved, fmt.Sprint(i))); err != nil {
			return err
		}
		list = append(list, fmt.Sprintf("%d|%s", i, f))
	}
	return os.WriteFile(filepath.Join(dir, "compose-files"), []byte(strings.Join(list, "\n")+"\n"), 0o600)
}

// restoreCompose puts back every saved compose file whose content changed since the rollback point.
func (u *upgrader) restoreCompose(dir string) {
	b, err := os.ReadFile(filepath.Join(dir, "compose-files"))
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		idx, path, ok := strings.Cut(line, "|")
		if !ok {
			continue
		}
		want, err := os.ReadFile(filepath.Join(dir, composeSaveDir, idx))
		if err != nil {
			continue
		}
		if cur, err := os.ReadFile(path); err == nil && bytes.Equal(cur, want) {
			continue
		}
		if err := writeFileKeepingMode(path, want); err != nil {
			u.a.say("  could not restore %s: %v", path, err)
			continue
		}
		u.a.say("  restored %s", path)
	}
}

// writeFileKeepingMode replaces a file atomically and keeps its permissions, so a compose file is never seen half written.
func writeFileKeepingMode(path string, content []byte) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".compose-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Chmod(name, mode); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Rename(name, path)
}
