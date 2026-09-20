package ctl

import (
	"strings"
	"testing"
)

func TestUnifiedDiffShowsOnlyTheChangeWithContext(t *testing.T) {
	a := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n"
	b := "1\n2\n3\n4\n5\nsix\n7\n8\n9\n10\n"
	got := unifiedDiff("old", "new", a, b)
	for _, want := range []string{"--- old", "+++ new", "@@ -3,7 +3,7 @@", "-6\n", "+six\n", " 3\n", " 9\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, " 1\n") || strings.Contains(got, " 10\n") {
		t.Fatalf("lines beyond the context must not appear:\n%s", got)
	}
}

func TestUnifiedDiffOfEqualTextIsEmpty(t *testing.T) {
	if got := unifiedDiff("a", "b", "x\ny\n", "x\ny\n"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestUnifiedDiffHandlesAdditionsAndDeletions(t *testing.T) {
	got := unifiedDiff("a", "b", "keep\ngone\n", "keep\nnew1\nnew2\n")
	for _, want := range []string{"-gone\n", "+new1\n", "+new2\n", " keep\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}
