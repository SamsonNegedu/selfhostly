package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/selfhostly/internal/constants"
)

// CFAccessVerifier validates the Cf-Access-Jwt-Assertion header Cloudflare Access adds to requests
// it has authenticated, so the backend does not have to trust network position alone.
type CFAccessVerifier struct {
	teamDomain string
	audience   string
	certsURL   string
	client     *http.Client

	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

// NewCFAccessVerifier creates a verifier for https://<teamDomain>. certsBase overrides the JWKS
// origin (tests); pass "" for production.
func NewCFAccessVerifier(teamDomain, audience, certsBase string) *CFAccessVerifier {
	if certsBase == "" {
		certsBase = "https://" + teamDomain
	}
	return &CFAccessVerifier{
		teamDomain: teamDomain,
		audience:   audience,
		certsURL:   certsBase + constants.CFAccessCertsPath,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Verify checks signature, expiry, issuer and audience, returning the caller's email claim.
func (v *CFAccessVerifier) Verify(ctx context.Context, tokenStr string) (string, error) {
	if tokenStr == "" {
		return "", errors.New("missing Cloudflare Access token")
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}))
	claims := jwt.MapClaims{}
	_, err := parser.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		kid, _ := t.Header["kid"].(string)
		return v.key(ctx, kid)
	})
	if err != nil {
		return "", fmt.Errorf("invalid Cloudflare Access token: %w", err)
	}
	if iss, _ := claims["iss"].(string); iss != "https://"+v.teamDomain {
		return "", errors.New("unexpected token issuer")
	}
	if !audienceMatches(claims["aud"], v.audience) {
		return "", errors.New("token audience does not match this application")
	}
	email, _ := claims["email"].(string)
	return email, nil
}

func audienceMatches(claim interface{}, want string) bool {
	switch a := claim.(type) {
	case string:
		return a == want
	case []interface{}:
		for _, item := range a {
			if s, _ := item.(string); s == want {
				return true
			}
		}
	}
	return false
}

func (v *CFAccessVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if k, ok := v.keys[kid]; ok && time.Since(v.fetched) < constants.CFAccessJWKSCacheTTL {
		return k, nil
	}
	if err := v.refreshLocked(ctx); err != nil {
		return nil, err
	}
	k, ok := v.keys[kid]
	if !ok {
		return nil, errors.New("signing key not found")
	}
	return k, nil
}

func (v *CFAccessVerifier) refreshLocked(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.certsURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch Cloudflare Access keys: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Cloudflare Access keys endpoint returned %d", resp.StatusCode)
	}
	var doc struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return err
	}
	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		n, err1 := base64.RawURLEncoding.DecodeString(strings.TrimRight(k.N, "="))
		e, err2 := base64.RawURLEncoding.DecodeString(strings.TrimRight(k.E, "="))
		if err1 != nil || err2 != nil {
			continue
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	if len(keys) == 0 {
		return errors.New("no usable signing keys published")
	}
	v.keys = keys
	v.fetched = time.Now()
	return nil
}
