package ctl

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// EnvFile is a KEY=VALUE settings file. Edits keep every other line and comment, and the file stays 0600
// because it holds secrets.
type EnvFile struct{ Path string }

func lineRE(key string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `=.*$`)
}

// Get returns the last value of key, or "" when the file or key is missing
func (e EnvFile) Get(key string) string {
	b, err := os.ReadFile(e.Path)
	if err != nil {
		return ""
	}
	val := ""
	for _, line := range strings.Split(string(b), "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok && strings.TrimSpace(k) == key {
			val = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return val
}

// All returns every key with its last value
func (e EnvFile) All() map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile(e.Path)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return out
}

// Exists reports whether the file is there
func (e EnvFile) Exists() bool { _, err := os.Stat(e.Path); return err == nil }

// Set replaces the line for key or appends one
func (e EnvFile) Set(key, value string) error {
	b, _ := os.ReadFile(e.Path)
	s, line := string(b), key+"="+value
	if re := lineRE(key); re.MatchString(s) {
		s = re.ReplaceAllLiteralString(s, line)
	} else {
		if s != "" && !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		s += line + "\n"
	}
	return e.write(s)
}

// SetIfMissing writes key only when it has no value yet. It reports whether it wrote.
func (e EnvFile) SetIfMissing(key, value string) (bool, error) {
	if value == "" || e.Get(key) != "" {
		return false, nil
	}
	b, _ := os.ReadFile(e.Path)
	s := regexp.MustCompile(`(?m)^`+regexp.QuoteMeta(key)+`=\s*\n?`).ReplaceAllString(string(b), "")
	if s != "" && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	return true, e.write(s + fmt.Sprintf("%s=%s\n", key, value))
}

// Touch makes sure the file exists with mode 0600
func (e EnvFile) Touch() error {
	f, err := os.OpenFile(e.Path, os.O_CREATE|os.O_RDONLY, 0o600)
	if err != nil {
		return err
	}
	_ = f.Close()
	return os.Chmod(e.Path, 0o600)
}

// readEnv is kept for callers that only need a snapshot
func readEnv(path string) map[string]string { return EnvFile{path}.All() }

// write replaces the file and makes sure it is private, even if it existed with looser permissions
func (e EnvFile) write(content string) error {
	if err := os.WriteFile(e.Path, []byte(content), 0o600); err != nil {
		return err
	}
	return os.Chmod(e.Path, 0o600)
}
