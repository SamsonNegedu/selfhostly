package main

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/selfhostly/internal/update"
)

const testPrefix = "ghcr.io/example/selfhostly-"

func ref(service, c string) string {
	return testPrefix + service + "@sha256:" + strings.Repeat(c, 64)
}

// release makes a key pair, a manifest and its signature in a temp dir, the way CI does
func release(t *testing.T, extra ...string) (dir string) {
	t.Helper()
	dir = t.TempDir()
	if err := keygen([]string{"--out", dir}); err != nil {
		t.Fatal(err)
	}
	compose := filepath.Join(dir, "docker-compose.prod.yml")
	os.WriteFile(compose, []byte("services: {}\n"), 0o644)
	inputs := filepath.Join(dir, "inputs.json")
	os.WriteFile(inputs, []byte(`{"min_from_version":"v1.0.0","settings":[{"key":"FROM_FILE","kind":"optional","description":"kept in the repo"}]}`), 0o644)
	args := append([]string{
		"--version", "v1.4.0", "--backend", ref("backend", "a"), "--gateway", ref("gateway", "b"), "--frontend", ref("frontend", "c"),
		"--compose-file", compose, "--inputs", inputs, "--image-prefix", testPrefix, "--out", filepath.Join(dir, "release.json"),
	}, extra...)
	if err := manifest(args); err != nil {
		t.Fatal(err)
	}
	if err := sign([]string{"--key", filepath.Join(dir, privateKeyFile), "--in", filepath.Join(dir, "release.json"), "--out", filepath.Join(dir, "release.json.sig")}); err != nil {
		t.Fatal(err)
	}
	return dir
}

func verifyIn(dir string) error {
	pub, _ := os.ReadFile(filepath.Join(dir, publicKeyFile))
	return verify([]string{"--pub", strings.TrimSpace(string(pub)), "--image-prefix", testPrefix,
		"--in", filepath.Join(dir, "release.json"), "--sig", filepath.Join(dir, "release.json.sig")})
}

func TestTheWholeReleaseCycleVerifies(t *testing.T) {
	dir := release(t, "--setting", "FROM_FLAG:required:true:asked on the command line")
	if err := verifyIn(dir); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "release.json"))
	for _, want := range []string{`"version": "1.4.0"`, `"min_from_version": "1.0.0"`, "FROM_FILE", "FROM_FLAG"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("release.json lacks %q:\n%s", want, b)
		}
	}
}

func TestAChangedManifestNoLongerVerifies(t *testing.T) {
	dir := release(t)
	p := filepath.Join(dir, "release.json")
	b, _ := os.ReadFile(p)
	os.WriteFile(p, []byte(strings.Replace(string(b), "1.4.0", "1.4.1", 1)), 0o644)
	if err := verifyIn(dir); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("got %v", err)
	}
}

func TestTheManifestRefusesImagesOutsideTheTrustedRepository(t *testing.T) {
	dir := t.TempDir()
	compose := filepath.Join(dir, "c.yml")
	os.WriteFile(compose, []byte("x"), 0o644)
	err := manifest([]string{"--version", "1.0.0", "--backend", "evil.example/selfhostly-backend@sha256:" + strings.Repeat("a", 64),
		"--gateway", ref("gateway", "b"), "--frontend", ref("frontend", "c"), "--compose-file", compose,
		"--image-prefix", testPrefix, "--out", filepath.Join(dir, "r.json")})
	if err == nil {
		t.Fatal("an image outside the trusted repository must not make it into a signed manifest")
	}
}

func TestThePublicKeyDerivedFromThePrivateKeyMatchesKeygen(t *testing.T) {
	dir := t.TempDir()
	if err := keygen([]string{"--out", dir}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, privateKeyFile))
	priv, err := update.ParsePrivateKey(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	pub, _ := os.ReadFile(filepath.Join(dir, publicKeyFile))
	if got := update.EncodeKey(priv.Public().(ed25519.PublicKey)); strings.TrimSpace(string(pub)) != got {
		t.Fatalf("CI derives the public key from the secret, so it must equal the one keygen wrote: %q vs %q", got, pub)
	}
}
