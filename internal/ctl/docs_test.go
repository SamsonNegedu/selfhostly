package ctl

import (
	"os"
	"strings"
	"testing"
)

// The reference is generated from the commands. If this fails, a command or flag changed without the
// reference being regenerated: run `make docs`.
func TestCommandReferenceIsUpToDate(t *testing.T) {
	want := ReferenceMarkdown(New().Root())
	got, err := os.ReadFile("../../docs/reference/selfhostlyctl.md")
	if err != nil {
		t.Fatalf("docs/reference/selfhostlyctl.md is missing: run `make docs` (%v)", err)
	}
	if string(got) != want {
		t.Fatal("docs/reference/selfhostlyctl.md is out of date: run `make docs`")
	}
}

func TestReferenceListsEveryVisibleCommandAndNoHiddenOne(t *testing.T) {
	md := ReferenceMarkdown(New().Root())
	for _, name := range []string{"setup", "check", "join", "join-token", "upgrade", "backup", "bootstrap", "compose-diff", "compose write", "pin-images", "doctor", "status"} {
		if !strings.Contains(md, "\n## selfhostlyctl "+name+"\n") {
			t.Errorf("no section for %q", name)
		}
	}
	if strings.Contains(md, "selfhostlyctl docs") {
		t.Error("the hidden docs command must not be documented")
	}
}
