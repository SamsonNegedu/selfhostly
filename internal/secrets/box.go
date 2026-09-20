// Package secrets encrypts sensitive database fields (node API keys, provider tokens) at rest.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/selfhostly/internal/constants"
)

// Box seals and opens field values. Open accepts both sealed and legacy plaintext values so an
// install can upgrade first and turn on encrypted writes later without a flag day.
type Box struct {
	aead          cipher.AEAD
	encryptWrites bool
}

// NewBox builds a Box from key material of any length (it is hashed to 32 bytes).
func NewBox(key string, encryptWrites bool) (*Box, error) {
	if key == "" {
		return nil, errors.New("empty encryption key")
	}
	sum := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead, encryptWrites: encryptWrites}, nil
}

// FindKey returns the key from explicit (env) if set, otherwise from dir/secrets.key. It never
// creates anything, so it is safe to call when a missing key must be treated as an error.
func FindKey(explicit, dir string) (string, bool) {
	if explicit != "" {
		return explicit, true
	}
	if b, err := os.ReadFile(filepath.Join(dir, constants.SecretsKeyFileName)); err == nil {
		if v := strings.TrimSpace(string(b)); v != "" {
			return v, true
		}
	}
	return "", false
}

// CreateKey writes a new random key to dir/secrets.key (mode 0600) and returns it.
func CreateKey(dir string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	key := base64.StdEncoding.EncodeToString(raw)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, constants.SecretsKeyFileName)
	if err := os.WriteFile(path, []byte(key+"\n"), constants.SecretFileMode); err != nil {
		return "", fmt.Errorf("write key file: %w", err)
	}
	return key, nil
}

// LoadKey returns the existing key, creating one only when none exists.
func LoadKey(explicit, dir string) (string, error) {
	if key, ok := FindKey(explicit, dir); ok {
		return key, nil
	}
	return CreateKey(dir)
}

// EncryptsWrites reports whether Seal produces ciphertext
func (b *Box) EncryptsWrites() bool { return b != nil && b.encryptWrites }

// IsSealed reports whether v carries the ciphertext prefix
func IsSealed(v string) bool { return strings.HasPrefix(v, constants.SecretsCipherPrefix) }

// Seal returns the value to store. It is a no-op when writes are not encrypted, when the value is
// empty, or when it is already sealed.
func (b *Box) Seal(plain string) (string, error) {
	if !b.EncryptsWrites() || plain == "" || IsSealed(plain) {
		return plain, nil
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := b.aead.Seal(nonce, nonce, []byte(plain), nil)
	return constants.SecretsCipherPrefix + base64.StdEncoding.EncodeToString(ct), nil
}

// Open returns the plaintext for a stored value. Values without the prefix are legacy plaintext.
func (b *Box) Open(stored string) (string, error) {
	if !IsSealed(stored) {
		return stored, nil
	}
	if b == nil {
		return "", errors.New("value is encrypted but no key is loaded")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, constants.SecretsCipherPrefix))
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	if len(raw) < b.aead.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:b.aead.NonceSize()], raw[b.aead.NonceSize():]
	plain, err := b.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt (wrong SETTINGS_ENCRYPTION_KEY?): %w", err)
	}
	return string(plain), nil
}
