package update

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"
)

const testPrefix = "ghcr.io/example/selfhostly-"

func digest(c string) string { return "@sha256:" + strings.Repeat(c, 64) }

func validManifest() Manifest {
	return Manifest{
		Schema: SchemaVersion, Version: "1.4.0", MinFromVersion: "1.0.0",
		Images: Images{
			Backend:  testPrefix + "backend" + digest("a"),
			Gateway:  testPrefix + "gateway" + digest("b"),
			Frontend: testPrefix + "frontend" + digest("c"),
		},
		Compose:  ComposeInfo{SHA256: strings.Repeat("d", 64)},
		Settings: []Setting{{Key: "NEW_KEY", Kind: SettingRequired, Secret: true}},
	}
}

func signed(t *testing.T, m Manifest) (ed25519.PublicKey, []byte, string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(m)
	return pub, data, Sign(priv, data)
}

func TestParseVerifiedAcceptsASignedManifest(t *testing.T) {
	pub, data, sig := signed(t, validManifest())
	m, err := ParseVerified(pub, data, sig, testPrefix)
	if err != nil || m.Version != "1.4.0" {
		t.Fatalf("got %v %v", m, err)
	}
}

func TestParseVerifiedRejectsATamperedManifest(t *testing.T) {
	pub, data, sig := signed(t, validManifest())
	tampered := []byte(strings.Replace(string(data), "1.4.0", "1.4.1", 1))
	if _, err := ParseVerified(pub, tampered, sig, testPrefix); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("a changed manifest must fail the signature check, got %v", err)
	}
}

func TestParseVerifiedRejectsAnotherKeysSignature(t *testing.T) {
	_, data, sig := signed(t, validManifest())
	other, _, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := ParseVerified(other, data, sig, testPrefix); err == nil {
		t.Fatal("a signature from a different key must be refused")
	}
}

func TestParseVerifiedChecksTheSignatureBeforeReadingTheContent(t *testing.T) {
	pub, _, _ := signed(t, validManifest())
	if _, err := ParseVerified(pub, []byte("not json"), "AAAA", testPrefix); err == nil || strings.Contains(err.Error(), "JSON") {
		t.Fatalf("an unsigned body must be refused as unsigned, not parsed, got %v", err)
	}
}

func TestValidateRejectsImagesOutsideTheTrustedRepository(t *testing.T) {
	cases := map[string]func(*Manifest){
		"a tag instead of a digest": func(m *Manifest) { m.Images.Backend = testPrefix + "backend:latest" },
		"another registry":          func(m *Manifest) { m.Images.Gateway = "evil.example/selfhostly-gateway" + digest("b") },
		"the wrong service":         func(m *Manifest) { m.Images.Frontend = testPrefix + "backend" + digest("c") },
		"a short digest":            func(m *Manifest) { m.Images.Backend = testPrefix + "backend@sha256:abc" },
		"trailing text":             func(m *Manifest) { m.Images.Backend += " --privileged" },
	}
	for name, mutate := range cases {
		m := validManifest()
		mutate(&m)
		if err := m.Validate(testPrefix); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}

func TestValidateRejectsBadSettings(t *testing.T) {
	for _, s := range []Setting{
		{Key: "lower", Kind: SettingOptional},
		{Key: "A;B", Kind: SettingOptional},
		{Key: "OK", Kind: "surprise"},
	} {
		m := validManifest()
		m.Settings = []Setting{s}
		if err := m.Validate(testPrefix); err == nil {
			t.Errorf("setting %+v must be rejected", s)
		}
	}
	m := validManifest()
	m.Settings = []Setting{{Key: "A", Kind: SettingOptional}, {Key: "A", Kind: SettingOptional}}
	if err := m.Validate(testPrefix); err == nil {
		t.Error("a duplicated key must be rejected")
	}
}

// noBuiltInKey clears the key built into the release, so a test says what it needs and not what the release ships
func noBuiltInKey(t *testing.T) {
	t.Helper()
	saved := DefaultPublicKey
	DefaultPublicKey = ""
	t.Cleanup(func() { DefaultPublicKey = saved })
}

func TestResolvePublicKeyReportsMissingKeyWithoutError(t *testing.T) {
	noBuiltInKey(t)
	if _, ok, err := ResolvePublicKey(""); ok || err != nil {
		t.Fatalf("no key configured: ok=%v err=%v", ok, err)
	}
	if _, _, err := ResolvePublicKey("not a key"); err == nil {
		t.Fatal("a malformed key must be an error")
	}
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	key, ok, err := ResolvePublicKey(EncodeKey(pub))
	if !ok || err != nil || !key.Equal(pub) {
		t.Fatalf("got ok=%v err=%v", ok, err)
	}
}

func TestVersionOrdering(t *testing.T) {
	order := []string{"0.9.0", "1.0.0-rc1", "1.0.0", "1.0.1", "1.2.0", "2.0.0"}
	for i := range order {
		for j := range order {
			a, _ := ParseVersion(order[i])
			b, _ := ParseVersion(order[j])
			want := 0
			if i < j {
				want = -1
			} else if i > j {
				want = 1
			}
			if got := a.Compare(b); got != want {
				t.Errorf("%s vs %s: got %d want %d", order[i], order[j], got, want)
			}
		}
	}
	for _, bad := range []string{"", "1.2", "a.b.c", "1.2.3.4", "dev"} {
		if _, err := ParseVersion(bad); err == nil {
			t.Errorf("%q must not parse", bad)
		}
	}
	if v, err := ParseVersion("v1.2.3"); err != nil || v.String() != "1.2.3" {
		t.Errorf("a leading v is allowed, got %v %v", v, err)
	}
}

func TestTheBuiltInKeyIsUsedWhenNothingIsConfigured(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	saved := DefaultPublicKey
	DefaultPublicKey = EncodeKey(pub)
	t.Cleanup(func() { DefaultPublicKey = saved })
	key, ok, err := ResolvePublicKey("")
	if !ok || err != nil || !key.Equal(pub) {
		t.Fatalf("got ok=%v err=%v", ok, err)
	}
	other, _, _ := ed25519.GenerateKey(rand.Reader)
	if key, _, _ := ResolvePublicKey(EncodeKey(other)); !key.Equal(other) {
		t.Fatal("a configured key must win over the built-in one")
	}
}

func TestTheShippedBuiltInKeyIsAWellFormedKey(t *testing.T) {
	if DefaultPublicKey == "" {
		t.Skip("no key is built in")
	}
	if _, err := ParsePublicKey(DefaultPublicKey); err != nil {
		t.Fatalf("the key committed in pubkey.go must parse, or every install would report the feature unavailable: %v", err)
	}
}
