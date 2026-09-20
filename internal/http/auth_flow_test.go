package http

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-pkgz/auth/token"
	"github.com/golang-jwt/jwt"
	selfauth "github.com/selfhostly/internal/auth"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
)

// These tests drive the real go-pkgz/auth middleware, so they prove the allow-list and revocation
// behave correctly with the library's actual token handling, not just with our own helpers.

func newAuthFixture(t *testing.T) (*db.DB, *config.Config, http.Handler, *token.Service) {
	t.Helper()
	database, err := db.Init(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true, JWTSecret: "a-long-enough-secret-for-the-auth-tests-0123456789",
			BaseURL: "https://app.example.com", SecureCookie: false,
			GitHub: config.GitHubOAuthConfig{ClientID: "id", ClientSecret: "secret", AllowedUsers: []string{"alice"}},
		},
		Security: config.SecurityConfig{Mode: "enforce", SessionHours: 24},
	}
	svc := initAuthService(cfg, selfauth.NewAllowList(cfg.Auth.GitHub.AllowedUsers), database)
	mw := svc.Middleware()
	protected := mw.Auth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	return database, cfg, protected, svc.TokenService()
}

func claimsFor(id, displayName string, issuedAt time.Time, expires time.Time) token.Claims {
	return token.Claims{
		User: &token.User{ID: id, Name: displayName},
		StandardClaims: jwt.StandardClaims{
			Issuer: "selfhostly", Id: "token-" + id,
			IssuedAt: issuedAt.Unix(), ExpiresAt: expires.Unix(),
		},
	}
}

// requestWith issues claims through the real token service and replays the resulting cookie
func requestWith(t *testing.T, ts *token.Service, protected http.Handler, c token.Claims) *httptest.ResponseRecorder {
	t.Helper()
	issue := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://app.example.com/api/apps", nil)
	if _, err := ts.Set(issue, c); err != nil {
		t.Fatal(err)
	}
	for _, ck := range issue.Result().Cookies() {
		req.AddCookie(ck)
	}
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	return rec
}

func TestAllowListUsesIDNotDisplayName(t *testing.T) {
	_, _, protected, ts := newAuthFixture(t)
	now := time.Now()

	legit := requestWith(t, ts, protected, claimsFor(selfauth.UserIDForLogin("alice"), "Alice Example", now, now.Add(time.Hour)))
	if legit.Code != http.StatusOK {
		t.Fatalf("the allow-listed login must get in, got %d", legit.Code)
	}

	// The attack: a different GitHub account whose display name is set to an allowed login.
	// go-pkgz puts that display name in User.Name, which used to be what was compared.
	spoof := requestWith(t, ts, protected, claimsFor(selfauth.UserIDForLogin("mallory"), "alice", now, now.Add(time.Hour)))
	if spoof.Code == http.StatusOK {
		t.Fatal("a stranger whose display name is 'alice' must be rejected")
	}
}

func TestSessionRevocationRejectsOlderTokens(t *testing.T) {
	database, _, protected, ts := newAuthFixture(t)
	id := selfauth.UserIDForLogin("alice")
	login := func() token.Claims {
		now := time.Now()
		return claimsFor(id, "alice", now, now.Add(time.Hour))
	}

	// The token service stamps the issue time itself, so build the cookie once and replay it
	issue := httptest.NewRecorder()
	if _, err := ts.Set(issue, login()); err != nil {
		t.Fatal(err)
	}
	replay := func() int {
		req := httptest.NewRequest("GET", "https://app.example.com/api/apps", nil)
		for _, ck := range issue.Result().Cookies() {
			req.AddCookie(ck)
		}
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := replay(); code != http.StatusOK {
		t.Fatalf("a valid session must work before revocation, got %d", code)
	}
	if _, err := database.RevokeSessions(); err != nil {
		t.Fatal(err)
	}
	if code := replay(); code == http.StatusOK {
		t.Fatal("the existing session must be rejected after revocation")
	}

	time.Sleep(1100 * time.Millisecond) // a new login lands in a later second than the revocation
	if rec := requestWith(t, ts, protected, login()); rec.Code != http.StatusOK {
		t.Fatalf("logging in again after revocation must work, got %d", rec.Code)
	}
}

func TestExpiredTokenIsRefreshedWhileSessionCookieLives(t *testing.T) {
	_, _, protected, ts := newAuthFixture(t)
	id := selfauth.UserIDForLogin("alice")
	issued := time.Now().Add(-2 * time.Hour)
	expired := time.Now().Add(-time.Hour)

	rec := requestWith(t, ts, protected, claimsFor(id, "alice", issued, expired))
	if rec.Code != http.StatusOK {
		t.Fatalf("a short token lifetime must not log users out while their session cookie is valid, got %d", rec.Code)
	}
	refreshed := false
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == "JWT" && ck.Value != "" {
			refreshed = true
		}
	}
	if !refreshed {
		t.Fatal("the expired token must be replaced by a fresh cookie")
	}
}
