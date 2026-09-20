package gateway

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/selfhostly/internal/constants"
)

const jwtCookieName = "JWT"

// ValidateRequest checks JWT from Cookie or Authorization header; returns true if valid or auth not required
func (c *Config) ValidateRequest(req *http.Request) bool {
	if !c.AuthEnabled {
		return true
	}
	if c.pathSkipsAuth(req.URL.Path) {
		return true
	}
	tokenStr := c.extractToken(req)
	if tokenStr == "" {
		return false
	}
	return c.validToken(tokenStr)
}

// validToken accepts only HS256 tokens signed with the shared secret, issued by this platform,
// and carrying an expiry. Pinning the algorithm stops downgrade and key-confusion tricks.
func (c *Config) validToken(tokenStr string) bool {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	claims := jwt.MapClaims{}
	tok, err := parser.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(c.JWTSecret), nil
	})
	if err != nil || !tok.Valid {
		return false
	}
	if _, ok := claims["exp"]; !ok {
		return false
	}
	iss, err := claims.GetIssuer()
	return err == nil && iss == constants.AuthIssuer
}

func (c *Config) pathSkipsAuth(path string) bool {
	// Auth endpoints - no auth needed (these handle authentication)
	if strings.HasPrefix(path, "/auth/") {
		return true
	}

	// Health check - no auth needed
	if path == "/api/health" {
		return true
	}

	// A secondary's outbound link presents its own node credentials, which the primary checks: there
	// is no user session to validate here.
	if path == constants.LinkPath {
		return true
	}

	// /api/me must be accessible to unauthenticated users to determine auth status
	// It will return 401 if not authenticated, but shouldn't be blocked by gateway
	if path == "/api/me" {
		return true
	}

	return false
}

func (c *Config) extractToken(req *http.Request) string {
	if cookie, err := req.Cookie(jwtCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	auth := req.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
