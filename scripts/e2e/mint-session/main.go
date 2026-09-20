// Command mint-session signs a login cookie for an allowlisted user, using the same token service and the same
// user ID derivation as the server. It stands in for the GitHub OAuth round trip, which a local test cannot do,
// without changing how the server authenticates: the server still verifies a real signature, issuer and
// allowlist entry. The output is a Playwright storage state file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/go-pkgz/auth/token"
	selfauth "github.com/selfhostly/internal/auth"
	"github.com/selfhostly/internal/constants"
)

// sessionLifetime covers a full test run, including the image builds before the browser starts
const sessionLifetime = 12 * time.Hour

type stateCookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

type storageState struct {
	Cookies []stateCookie `json:"cookies"`
	Origins []struct{}    `json:"origins"`
}

func main() {
	secret := flag.String("secret", os.Getenv("JWT_SECRET"), "JWT secret of the install (default: $JWT_SECRET)")
	login := flag.String("login", "", "GitHub login that is in GITHUB_ALLOWED_USERS")
	domain := flag.String("domain", "localhost", "cookie domain (the host the browser opens)")
	out := flag.String("out", "", "storage state file to write")
	flag.Parse()
	if *secret == "" || *login == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: mint-session --secret S --login L --out FILE [--domain localhost]")
		os.Exit(2)
	}

	svc := token.NewService(token.Opts{
		SecretReader:   token.SecretFunc(func(string) (string, error) { return *secret, nil }),
		TokenDuration:  sessionLifetime,
		CookieDuration: sessionLifetime,
		Issuer:         constants.AuthIssuer,
		SameSite:       http.SameSiteLaxMode,
		DisableXSRF:    true,
	})

	claims := token.Claims{User: &token.User{Name: *login, ID: selfauth.UserIDForLogin(*login)}}
	rec := httptest.NewRecorder()
	if _, err := svc.Set(rec, claims); err != nil {
		fmt.Fprintln(os.Stderr, "could not sign the session:", err)
		os.Exit(1)
	}

	state := storageState{Cookies: []stateCookie{}, Origins: []struct{}{}}
	expires := float64(time.Now().Add(sessionLifetime).Unix())
	for _, c := range rec.Result().Cookies() {
		// the XSRF cookie is unused here (the server disables that check), so only the session cookie is stored
		if c.Name != "JWT" {
			continue
		}
		state.Cookies = append(state.Cookies, stateCookie{
			Name: c.Name, Value: c.Value, Domain: *domain, Path: "/",
			Expires: expires, HTTPOnly: true, Secure: false, SameSite: "Lax",
		})
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, b, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
