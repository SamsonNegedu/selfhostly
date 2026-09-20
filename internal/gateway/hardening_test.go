package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
)

func signWith(t *testing.T, method jwt.SigningMethod, secret string, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestValidTokenRules(t *testing.T) {
	cfg := &Config{AuthEnabled: true, JWTSecret: "secret"}
	good := jwt.MapClaims{"iss": "selfhostly", "exp": time.Now().Add(time.Hour).Unix()}
	if !cfg.validToken(signWith(t, jwt.SigningMethodHS256, "secret", good)) {
		t.Fatal("a well-formed token must be accepted")
	}

	cases := map[string]string{
		"wrong secret":    signWith(t, jwt.SigningMethodHS256, "other", good),
		"other algorithm": signWith(t, jwt.SigningMethodHS384, "secret", good),
		"missing expiry":  signWith(t, jwt.SigningMethodHS256, "secret", jwt.MapClaims{"iss": "selfhostly"}),
		"expired":         signWith(t, jwt.SigningMethodHS256, "secret", jwt.MapClaims{"iss": "selfhostly", "exp": time.Now().Add(-time.Hour).Unix()}),
		"wrong issuer":    signWith(t, jwt.SigningMethodHS256, "secret", jwt.MapClaims{"iss": "someone", "exp": time.Now().Add(time.Hour).Unix()}),
		"missing issuer":  signWith(t, jwt.SigningMethodHS256, "secret", jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()}),
		"alg none":        "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJpc3MiOiJzZWxmaG9zdGx5IiwiZXhwIjo5OTk5OTk5OTk5fQ.",
	}
	for name, tok := range cases {
		if cfg.validToken(tok) {
			t.Errorf("%s: must be rejected", name)
		}
	}
}

// captureTransport records the request the proxy would send upstream
type captureTransport struct{ got *http.Request }

func (c *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.got = r
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))}, nil
}

func TestProxyStripsInboundCredentialHeaders(t *testing.T) {
	proxy, _, _ := setupTestProxy(t)
	capture := &captureTransport{}
	proxy.transport = capture

	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	req.Header.Set("X-Gateway-API-Key", "guessed")
	req.Header.Set("X-Node-ID", "n1")
	req.Header.Set("X-Node-API-Key", "guessed")
	proxy.ServeHTTP(httptest.NewRecorder(), req)

	if capture.got == nil {
		t.Fatal("request was not forwarded")
	}
	for _, h := range []string{"X-Gateway-API-Key", "X-Node-ID", "X-Node-API-Key"} {
		if v := capture.got.Header.Get(h); v != "" {
			t.Errorf("%s must not reach the backend from a client, got %q", h, v)
		}
	}
}

func TestProxyAddsGatewayKeyOnlyForNodeManagement(t *testing.T) {
	proxy, _, _ := setupTestProxy(t)
	capture := &captureTransport{}
	proxy.transport = capture

	proxy.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/nodes", nil))
	if capture.got.Header.Get("X-Gateway-API-Key") != "test-api-key" {
		t.Fatal("gateway key must be added for /api/nodes")
	}
	proxy.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/apps", nil))
	if capture.got.Header.Get("X-Gateway-API-Key") != "" {
		t.Fatal("gateway key must not be added for user endpoints")
	}
}

func TestPublicHostPinning(t *testing.T) {
	proxy, _, cfg := setupTestProxy(t)
	cfg.PublicHosts = []string{"selfhostly.example.com"}
	capture := &captureTransport{}
	proxy.transport = capture

	forged := httptest.NewRequest(http.MethodGet, "/auth/github/login", nil)
	forged.Header.Set("X-Forwarded-Host", "evil.example.net")
	forged.Header.Set("Referer", "https://evil.example.net/x")
	proxy.ServeHTTP(httptest.NewRecorder(), forged)
	if got := capture.got.Header.Get("X-Forwarded-Host"); got != "selfhostly.example.com" {
		t.Fatalf("a forged host must be replaced, got %q", got)
	}
	if capture.got.Host != "selfhostly.example.com" {
		t.Fatalf("outgoing Host must be pinned, got %q", capture.got.Host)
	}

	legit := httptest.NewRequest(http.MethodGet, "/auth/github/login", nil)
	legit.Header.Set("X-Forwarded-Host", "SelfHostly.Example.com")
	proxy.ServeHTTP(httptest.NewRecorder(), legit)
	if got := capture.got.Header.Get("X-Forwarded-Host"); !strings.EqualFold(got, "selfhostly.example.com") {
		t.Fatalf("a listed host must be kept, got %q", got)
	}
}

func TestPublicHostsUnsetKeepsLegacyBehaviour(t *testing.T) {
	cfg := &Config{}
	if h, ok := cfg.pinPublicHost("anything.example.org"); !ok || h != "anything.example.org" {
		t.Fatal("no PUBLIC_HOSTS means no pinning")
	}
}

func TestLinkedNodesAreRoutedToThePrimary(t *testing.T) {
	proxy, registry, _ := setupTestProxy(t)
	registry.mu.Lock()
	registry.nodes = map[string]NodeEntry{
		"direct": {ID: "direct", APIEndpoint: "http://192.168.1.50:8082", Status: "online"},
		"linked": {ID: "linked", APIEndpoint: "tunnel://linked", Status: "online"},
		"gone":   {ID: "gone", APIEndpoint: "tunnel://gone", Status: "offline"},
	}
	registry.mu.Unlock()

	if got := registry.Get("direct"); got != "http://192.168.1.50:8082" {
		t.Fatalf("a directly reached node keeps its own address, got %q", got)
	}
	if got := registry.Get("linked"); got != proxy.config.PrimaryBackendURL {
		t.Fatalf("a linked node has no reachable address: it must go via the primary, got %q", got)
	}
	if registry.Get("gone") != "" {
		t.Fatal("an offline linked node must not be routed to")
	}
}

func TestLinkEndpointSkipsUserLoginAndGetsNoGatewayKey(t *testing.T) {
	proxy, _, cfg := setupTestProxy(t)
	cfg.AuthEnabled = true
	cfg.JWTSecret = "secret"
	capture := &captureTransport{}
	proxy.transport = capture

	// no session, no JWT: a secondary's link must still reach the primary, which checks its credentials
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/nodes/connect", nil)
	req.Header.Set("X-Selfhostly-Node-Key", "node-key")
	proxy.ServeHTTP(w, req)
	if capture.got == nil {
		t.Fatalf("the link endpoint must be forwarded without a user login, got %d", w.Code)
	}
	if capture.got.Header.Get("X-Gateway-API-Key") != "" {
		t.Fatal("the gateway key must not be attached to a node's link request")
	}
	if capture.got.Header.Get("X-Selfhostly-Node-Key") != "node-key" {
		t.Fatal("the node's link credentials must reach the primary untouched")
	}

	// while every other API path still needs a login
	capture.got = nil
	w = httptest.NewRecorder()
	proxy.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/apps", nil))
	if w.Code != http.StatusUnauthorized || capture.got != nil {
		t.Fatal("other paths must still require a login")
	}
}
