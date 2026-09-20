// Package update holds what the primary, the updater and the release tooling share: the signed release
// manifest, version ordering and the status file an update run writes.
package update

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// SchemaVersion is the release.json format this code reads.
const SchemaVersion = 1

// DefaultManifestURL is where releases publish release.json.
const DefaultManifestURL = "https://github.com/samsonnegedu/selfhostly/releases/latest/download/release.json"

// DefaultCheckIntervalHours is how often the primary looks for a new release.
const DefaultCheckIntervalHours = 6

// DefaultImagePrefix is the repository prefix release images must live under unless UPDATE_IMAGE_REPO_PREFIX says otherwise.
const DefaultImagePrefix = "ghcr.io/samsonnegedu/selfhostly-"

// Limits on what a manifest may carry, so a hostile or broken one cannot flood the UI.
const (
	maxNotesBytes    = 16 * 1024
	maxSettings      = 64
	maxDescription   = 512
	maxManifestBytes = 1 << 20
)

// Setting kinds: what the updater does about a setting the release introduces.
const (
	SettingOptional  = "optional"  // has a safe default; the update never writes it
	SettingGenerated = "generated" // the updater creates a random value if .env lacks it
	SettingRequired  = "required"  // only the operator knows it; the update is blocked until it is given
)

// Services whose images a release ships, in the order they are recreated.
const (
	ServiceBackend  = "backend"
	ServiceGateway  = "gateway"
	ServiceFrontend = "frontend"
)

// Services lists the shipped services.
var Services = []string{ServiceBackend, ServiceGateway, ServiceFrontend}

var (
	settingKeyRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)
	sha256Re     = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Images are digest-pinned references, one per shipped service.
type Images struct {
	Backend  string `json:"backend"`
	Gateway  string `json:"gateway"`
	Frontend string `json:"frontend"`
}

// For returns the reference for a service name.
func (i Images) For(service string) string {
	switch service {
	case ServiceBackend:
		return i.Backend
	case ServiceGateway:
		return i.Gateway
	case ServiceFrontend:
		return i.Frontend
	}
	return ""
}

// ComposeInfo identifies the compose file this release expects.
type ComposeInfo struct {
	SHA256 string `json:"sha256"`
}

// Setting is a configuration value a release introduces.
type Setting struct {
	Key         string `json:"key"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Secret      bool   `json:"secret"`
}

// Manifest is release.json.
type Manifest struct {
	Schema         int         `json:"schema"`
	Version        string      `json:"version"`
	PublishedAt    string      `json:"published_at"`
	Notes          string      `json:"notes"`
	MinFromVersion string      `json:"min_from_version"`
	Images         Images      `json:"images"`
	Compose        ComposeInfo `json:"compose"`
	Settings       []Setting   `json:"settings"`
}

// ImagePattern is the only shape of image reference a manifest may name: the trusted repository prefix, a
// shipped service and a sha256 digest. A tag would let a registry change what a signed manifest points at.
func ImagePattern(prefix, service string) *regexp.Regexp {
	return regexp.MustCompile("^" + regexp.QuoteMeta(prefix+service) + "@sha256:[0-9a-f]{64}$")
}

// Validate checks a parsed manifest. prefix is the trusted image repository prefix.
func (m *Manifest) Validate(prefix string) error {
	if m.Schema != SchemaVersion {
		return fmt.Errorf("unsupported manifest schema %d (this build reads %d)", m.Schema, SchemaVersion)
	}
	if _, err := ParseVersion(m.Version); err != nil {
		return fmt.Errorf("manifest version: %w", err)
	}
	if m.MinFromVersion != "" {
		if _, err := ParseVersion(m.MinFromVersion); err != nil {
			return fmt.Errorf("manifest min_from_version: %w", err)
		}
	}
	if len(m.Notes) > maxNotesBytes {
		return errors.New("manifest notes are too long")
	}
	for _, svc := range Services {
		if !ImagePattern(prefix, svc).MatchString(m.Images.For(svc)) {
			return fmt.Errorf("manifest image for %s must be %s%s@sha256:<digest>", svc, prefix, svc)
		}
	}
	if !sha256Re.MatchString(m.Compose.SHA256) {
		return errors.New("manifest compose sha256 is not a hex digest")
	}
	if len(m.Settings) > maxSettings {
		return errors.New("manifest lists too many settings")
	}
	seen := map[string]bool{}
	for _, s := range m.Settings {
		if !settingKeyRe.MatchString(s.Key) {
			return fmt.Errorf("manifest setting %q is not a valid variable name", s.Key)
		}
		if seen[s.Key] {
			return fmt.Errorf("manifest setting %s is listed twice", s.Key)
		}
		seen[s.Key] = true
		switch s.Kind {
		case SettingOptional, SettingGenerated, SettingRequired:
		default:
			return fmt.Errorf("manifest setting %s has unknown kind %q", s.Key, s.Kind)
		}
		if len(s.Description) > maxDescription {
			return fmt.Errorf("manifest setting %s description is too long", s.Key)
		}
	}
	return nil
}

// ParsePublicKey decodes a base64 ed25519 public key.
func ParsePublicKey(b64 string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, errors.New("the update public key is not a base64 ed25519 key")
	}
	return ed25519.PublicKey(raw), nil
}

// ResolvePublicKey picks UPDATE_PUBLIC_KEY, else the built-in key. ok is false when neither is set.
func ResolvePublicKey(configured string) (key ed25519.PublicKey, ok bool, err error) {
	src := strings.TrimSpace(configured)
	if src == "" {
		src = DefaultPublicKey
	}
	if src == "" {
		return nil, false, nil
	}
	key, err = ParsePublicKey(src)
	return key, err == nil, err
}

// Verify checks the base64 signature over the exact manifest bytes.
func Verify(pub ed25519.PublicKey, data []byte, sigB64 string) error {
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sigB64))
	if err != nil || len(sig) != ed25519.SignatureSize {
		return errors.New("the manifest signature is malformed")
	}
	if !ed25519.Verify(pub, data, sig) {
		return errors.New("the manifest signature does not match: refusing to trust this release")
	}
	return nil
}

// ParseVerified verifies the signature first, then parses and validates. Nothing in the data is read before
// the signature has been checked.
func ParseVerified(pub ed25519.PublicKey, data []byte, sigB64, imagePrefix string) (*Manifest, error) {
	if len(data) > maxManifestBytes {
		return nil, errors.New("the manifest is too large")
	}
	if err := Verify(pub, data, sigB64); err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("the manifest is not valid JSON: %w", err)
	}
	if err := m.Validate(imagePrefix); err != nil {
		return nil, err
	}
	return &m, nil
}

// Sign returns the base64 signature of data.
func Sign(priv ed25519.PrivateKey, data []byte) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(priv, data))
}

// ParsePrivateKey decodes a base64 ed25519 private key (seed or full form).
func ParsePrivateKey(b64 string) (ed25519.PrivateKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, errors.New("the private key is not base64")
	}
	switch len(raw) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(raw), nil
	}
	return nil, errors.New("the private key has the wrong length")
}

// EncodeKey formats a public key the way UPDATE_PUBLIC_KEY expects.
func EncodeKey(pub ed25519.PublicKey) string { return base64.StdEncoding.EncodeToString(pub) }
