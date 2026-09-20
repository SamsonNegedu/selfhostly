// Package docs holds the documentation and the test that keeps its links honest.
package docs

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	// [text](target), skipping images and links inside code spans handled below
	mdLink = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)
	// a repository path written in code, for example `docs/operations/operate.md`
	codePath = regexp.MustCompile("`((?:docs|scripts|internal|cmd)/[A-Za-z0-9_./-]+\\.(?:md|sh|go|yml))`")
	heading  = regexp.MustCompile(`(?m)^#{1,6}\s+(.+?)\s*$`)
)

// slug mirrors how GitHub turns a heading into an anchor
func slug(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.NewReplacer("`", "", "*", "", "_", "").Replace(h)
	var b strings.Builder
	for _, r := range h {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case r == '-' || r == '_' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || r > 127:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func anchors(t *testing.T, path string) map[string]bool {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	set := map[string]bool{}
	for _, m := range heading.FindAllStringSubmatch(string(b), -1) {
		set[slug(m[1])] = true
	}
	return set
}

// docFiles are the Markdown files whose links are checked
func docFiles(t *testing.T, root string) []string {
	t.Helper()
	files := []string{filepath.Join(root, "README.md"), filepath.Join(root, "AGENTS.md")}
	for _, dir := range []string{"docs", ".agents"} {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, ".md") {
				files = append(files, p)
			}
			return nil
		})
	}
	return files
}

func TestMarkdownLinksAndAnchorsResolve(t *testing.T) {
	root := ".."
	for _, file := range docFiles(t, root) {
		b, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		text := string(b)
		for _, m := range mdLink.FindAllStringSubmatch(text, -1) {
			target := m[1]
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			pathPart, frag, _ := strings.Cut(target, "#")
			dest := file
			if pathPart != "" {
				dest = filepath.Join(filepath.Dir(file), pathPart)
				if _, err := os.Stat(dest); err != nil {
					t.Errorf("%s: link to %q does not exist", file, target)
					continue
				}
			}
			if frag != "" && strings.HasSuffix(dest, ".md") {
				if set := anchors(t, dest); !set[frag] {
					t.Errorf("%s: link %q points at a heading that does not exist in %s", file, target, dest)
				}
			}
		}
		for _, m := range codePath.FindAllStringSubmatch(text, -1) {
			if strings.ContainsAny(m[1], "*<>") {
				continue
			}
			if _, err := os.Stat(filepath.Join(root, m[1])); err != nil {
				t.Errorf("%s: mentions `%s`, which does not exist", file, m[1])
			}
		}
	}
}
