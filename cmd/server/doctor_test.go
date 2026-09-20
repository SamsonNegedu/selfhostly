package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// capture returns what f prints to standard output
func capture(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	rd, wr, _ := os.Pipe()
	os.Stdout = wr
	f()
	wr.Close()
	os.Stdout = old
	b, _ := io.ReadAll(rd)
	return string(b)
}

func TestSummaryListsOnlyWhatNeedsAction(t *testing.T) {
	r := &doctorReport{}
	out := capture(t, func() {
		r.pass("fine")
		r.note("expected in development")
		r.warn("no login configured")
		r.fix("turn on GitHub login")
		r.summary()
	})
	if !strings.Contains(out, "works, but 1 thing(s) need your attention. 1 notes need nothing.") {
		t.Fatalf("verdict missing:\n%s", out)
	}
	todo := out[strings.Index(out, "To do:"):]
	if !strings.Contains(todo, "1. no login configured") || !strings.Contains(todo, "-> turn on GitHub login") {
		t.Fatalf("the warning and its fix must be listed:\n%s", todo)
	}
	if strings.Contains(todo, "expected in development") {
		t.Fatalf("a note must never appear in the to-do list:\n%s", todo)
	}
}

func TestSummaryIsHealthyWithOnlyNotes(t *testing.T) {
	r := &doctorReport{}
	out := capture(t, func() { r.note("optional thing"); r.summary() })
	if !strings.Contains(out, "healthy. Nothing needs action") || strings.Contains(out, "To do:") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestSummaryPutsFailuresFirst(t *testing.T) {
	r := &doctorReport{}
	out := capture(t, func() { r.warn("a warning"); r.fail("a failure"); r.summary() })
	if !strings.Contains(out, "NOT READY") || strings.Index(out, "a failure") > strings.LastIndex(out, "a warning") {
		t.Fatalf("got:\n%s", out)
	}
	if r.fails != 1 || r.warns != 1 {
		t.Fatalf("counts %d %d", r.fails, r.warns)
	}
}
