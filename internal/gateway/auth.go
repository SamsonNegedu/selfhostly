package gateway

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
)

const jwtCookieName = "JWT"

// ValidateRequest checks JWT from Cookie or Authorization header; returns true if valid or auth not required
func (c *Config) ValidateRequest(req *http.Request) bool {
	if !c.AuthEnabled {
		return true
	}
	if c.pathSkipsAuth(req.URL.Path, req.Method) {
		return true
	}
	tokenStr := c.extractToken(req)
	if tokenStr == "" {
		return false
	}
	_, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(c.JWTSecret), nil
	})
	return err == nil
}

func (c *Config) pathSkipsAuth(path, method string) bool {
	if strings.HasPrefix(path, "/auth/") {
		return true
	}
	if path == "/api/health" {
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
