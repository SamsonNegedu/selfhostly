package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt"
)

func TestConfig_ValidateRequest(t *testing.T) {
	// Create a valid JWT token for testing
	validToken := createTestJWT(t, "test-secret")
	invalidToken := "invalid.token.here"

	tests := []struct {
		name        string
		authEnabled  bool
		jwtSecret    string
		path         string
		method       string
		cookieValue  string
		headerValue  string
		want         bool
	}{
		{
			name:        "auth disabled allows all",
			authEnabled: false,
			jwtSecret:   "test-secret",
			path:        "/api/apps",
			method:      http.MethodGet,
			want:        true,
		},
		{
			name:        "auth disabled allows protected path",
			authEnabled: false,
			jwtSecret:   "test-secret",
			path:        "/api/apps/123",
			method:      http.MethodGet,
			want:        true,
		},
		{
			name:        "auth enabled, valid JWT cookie",
			authEnabled: true,
			jwtSecret:   "test-secret",
			path:        "/api/apps",
			method:      http.MethodGet,
			cookieValue: validToken,
			want:        true,
		},
		{
			name:        "auth enabled, valid JWT header",
			authEnabled: true,
			jwtSecret:   "test-secret",
			path:        "/api/apps",
			method:      http.MethodGet,
			headerValue: "Bearer " + validToken,
			want:        true,
		},
		{
			name:        "auth enabled, invalid JWT cookie",
			authEnabled: true,
			jwtSecret:   "test-secret",
			path:        "/api/apps",
			method:      http.MethodGet,
			cookieValue: invalidToken,
			want:        false,
		},
		{
			name:        "auth enabled, invalid JWT header",
			authEnabled: true,
			jwtSecret:   "test-secret",
			path:        "/api/apps",
			method:      http.MethodGet,
			headerValue: "Bearer " + invalidToken,
			want:        false,
		},
		{
			name:        "auth enabled, no token",
			authEnabled: true,
			jwtSecret:   "test-secret",
			path:        "/api/apps",
			method:      http.MethodGet,
			want:        false,
		},
		{
			name:        "auth enabled, wrong secret",
			authEnabled: true,
			jwtSecret:   "wrong-secret",
			path:        "/api/apps",
			method:      http.MethodGet,
			cookieValue: validToken,
			want:        false,
		},
		{
			name:        "auth enabled, skips /auth/",
			authEnabled: true,
			jwtSecret:   "test-secret",
			path:        "/auth/github/login",
			method:      http.MethodGet,
			want:        true,
		},
		{
			name:        "auth enabled, skips /api/health",
			authEnabled: true,
			jwtSecret:   "test-secret",
			path:        "/api/health",
			method:      http.MethodGet,
			want:        true,
		},
		{
			name:        "auth enabled, header without Bearer prefix",
			authEnabled: true,
			jwtSecret:   "test-secret",
			path:        "/api/apps",
			method:      http.MethodGet,
			headerValue: validToken,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				AuthEnabled: tt.authEnabled,
				JWTSecret:   tt.jwtSecret,
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  jwtCookieName,
					Value: tt.cookieValue,
				})
			}
			if tt.headerValue != "" {
				req.Header.Set("Authorization", tt.headerValue)
			}

			got := cfg.ValidateRequest(req)
			if got != tt.want {
				t.Errorf("ValidateRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_pathSkipsAuth(t *testing.T) {
	cfg := &Config{}

	tests := []struct {
		name   string
		path   string
		method string
		want   bool
	}{
		{"auth prefix", "/auth/login", http.MethodGet, true},
		{"auth subpath", "/auth/github/callback", http.MethodGet, true},
		{"health endpoint", "/api/health", http.MethodGet, true},
		{"health POST", "/api/health", http.MethodPost, true},
		{"me endpoint", "/api/me", http.MethodGet, true},
		{"protected path", "/api/apps", http.MethodGet, false},
		{"other path", "/api/other", http.MethodGet, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.pathSkipsAuth(tt.path, tt.method)
			if got != tt.want {
				t.Errorf("pathSkipsAuth(%q, %q) = %v, want %v", tt.path, tt.method, got, tt.want)
			}
		})
	}
}

func TestConfig_extractToken(t *testing.T) {
	cfg := &Config{}
	validToken := "test.token.here"

	tests := []struct {
		name        string
		cookieValue string
		headerValue string
		want        string
	}{
		{"token in cookie", validToken, "", validToken},
		{"token in header", "", "Bearer " + validToken, validToken},
		{"cookie takes precedence", validToken, "Bearer other", validToken},
		{"no token", "", "", ""},
		{"header without Bearer", "", validToken, ""},
		{"empty cookie", "", "", ""},
		{"empty header", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  jwtCookieName,
					Value: tt.cookieValue,
				})
			}
			if tt.headerValue != "" {
				req.Header.Set("Authorization", tt.headerValue)
			}

			got := cfg.extractToken(req)
			if got != tt.want {
				t.Errorf("extractToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

// createTestJWT creates a valid JWT token for testing
func createTestJWT(t *testing.T, secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": "test-user",
	})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to create test JWT: %v", err)
	}
	return tokenString
}
