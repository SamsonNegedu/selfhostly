package secrets

import (
	"os"
	"strings"
	"testing"
)

func TestSealOpenRoundTrip(t *testing.T) {
	b, err := NewBox("k", true)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := b.Seal("node-key")
	if err != nil || !IsSealed(sealed) || strings.Contains(sealed, "node-key") {
		t.Fatalf("expected ciphertext, got %q (%v)", sealed, err)
	}
	if again, _ := b.Seal(sealed); again != sealed {
		t.Fatal("sealing an already sealed value must be a no-op")
	}
	plain, err := b.Open(sealed)
	if err != nil || plain != "node-key" {
		t.Fatalf("open: %q %v", plain, err)
	}
}

func TestLegacyPlaintextAndDisabledWrites(t *testing.T) {
	b, _ := NewBox("k", false)
	if s, _ := b.Seal("x"); s != "x" {
		t.Fatal("writes disabled must keep plaintext")
	}
	if p, err := b.Open("legacy"); err != nil || p != "legacy" {
		t.Fatal("plaintext must pass through Open")
	}
}

func TestWrongKeyFails(t *testing.T) {
	a, _ := NewBox("a", true)
	b, _ := NewBox("b", true)
	s, _ := a.Seal("v")
	if _, err := b.Open(s); err == nil {
		t.Fatal("expected failure with the wrong key")
	}
}

func TestLoadKeyCreatesAndReusesFile(t *testing.T) {
	dir := t.TempDir()
	k1, err := LoadKey("", dir)
	if err != nil || k1 == "" {
		t.Fatal(err)
	}
	k2, _ := LoadKey("", dir)
	if k1 != k2 {
		t.Fatal("key must be stable across loads")
	}
	if e, _ := LoadKey("explicit", dir); e != "explicit" {
		t.Fatal("explicit key wins")
	}
}

func TestFindKeyNeverCreatesAFile(t *testing.T) {
	dir := t.TempDir()
	if _, ok := FindKey("", dir); ok {
		t.Fatal("no key exists yet")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatal("FindKey must not write anything")
	}
	created, err := CreateKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	if found, ok := FindKey("", dir); !ok || found != created {
		t.Fatal("FindKey must return the key CreateKey wrote")
	}
	if k, ok := FindKey("from-env", dir); !ok || k != "from-env" {
		t.Fatal("an explicit key wins")
	}
}
