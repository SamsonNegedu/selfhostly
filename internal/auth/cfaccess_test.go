package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
)

const testTeam = "team.cloudflareaccess.com"

func setup(t *testing.T) (*rsa.PrivateKey, *CFAccessVerifier) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"keys": []map[string]string{{
			"kid": "k1", "kty": "RSA",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	t.Cleanup(srv.Close)
	return key, NewCFAccessVerifier(testTeam, "aud-1", srv.URL)
}

func sign(t *testing.T, key *rsa.PrivateKey, method jwt.SigningMethod, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(method, claims)
	tok.Header["kid"] = "k1"
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func goodClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss": "https://" + testTeam, "aud": []string{"aud-1"},
		"email": "me@example.com", "exp": time.Now().Add(time.Hour).Unix(),
	}
}

func TestVerifyAcceptsValidToken(t *testing.T) {
	key, v := setup(t)
	email, err := v.Verify(context.Background(), sign(t, key, jwt.SigningMethodRS256, goodClaims()))
	if err != nil || email != "me@example.com" {
		t.Fatalf("got %q %v", email, err)
	}
}

func TestVerifyRejects(t *testing.T) {
	key, v := setup(t)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)

	expired := goodClaims()
	expired["exp"] = time.Now().Add(-time.Hour).Unix()
	wrongAud := goodClaims()
	wrongAud["aud"] = []string{"someone-else"}
	wrongIss := goodClaims()
	wrongIss["iss"] = "https://evil.example.com"

	cases := map[string]string{
		"empty":     "",
		"expired":   sign(t, key, jwt.SigningMethodRS256, expired),
		"wrong aud": sign(t, key, jwt.SigningMethodRS256, wrongAud),
		"wrong iss": sign(t, key, jwt.SigningMethodRS256, wrongIss),
		"wrong key": sign(t, other, jwt.SigningMethodRS256, goodClaims()),
		"wrong alg": sign(t, key, jwt.SigningMethodRS384, goodClaims()),
		"not a jwt": "abc.def.ghi",
	}
	for name, tok := range cases {
		if _, err := v.Verify(context.Background(), tok); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}
