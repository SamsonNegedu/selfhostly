package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
)

const (
	testRegToken = "registration-token-for-tests"
	testGWKey    = "gateway-key-for-tests-0123456789"
	testNodeKey  = "primary-node-key-for-tests-0123"
)

func newTestServer(t *testing.T, mode string) (*Server, *db.DB) {
	t.Helper()
	dir := t.TempDir()
	database, err := db.Init(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := &config.Config{
		ServerAddress: ":8082",
		DatabasePath:  filepath.Join(dir, "t.db"),
		AppsDir:       filepath.Join(dir, "apps"),
		Environment:   constants.EnvProduction,
		CORS:          config.CORSConfig{AllowedOrigins: []string{"https://app.example.com"}},
		Auth:          config.AuthConfig{BaseURL: "https://app.example.com"},
		Node: config.NodeConfig{
			ID: "primary-id", Name: "primary", IsPrimary: true,
			APIEndpoint: "http://127.0.0.1:1", APIKey: testNodeKey,
			RegistrationToken: testRegToken, GatewayAPIKey: testGWKey,
		},
		Security: config.SecurityConfig{
			Mode: mode, HostAppsDir: filepath.Join(dir, "apps"),
			NodeEndpointAllowLoopback: true, SessionHours: 24,
		},
	}
	if err := database.InitNode(cfg); err != nil {
		t.Fatal(err)
	}
	return NewServer(cfg, database), database
}

func do(s *Server, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)
	return w
}

func TestSecretsEqual(t *testing.T) {
	if !secretsEqual("abc", "abc") || secretsEqual("abc", "abd") || secretsEqual("", "") || secretsEqual("a", "") {
		t.Fatal("secretsEqual must match equal non-empty secrets only")
	}
}

func TestOriginGuard(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	auth := map[string]string{constants.HeaderGatewayAPIKey: testGWKey}
	withOrigin := func(o string) map[string]string {
		return map[string]string{constants.HeaderGatewayAPIKey: testGWKey, "Origin": o}
	}

	if w := do(s, "POST", "/api/nodes/join-tokens", nil, withOrigin("https://evil.example.net")); w.Code != http.StatusForbidden {
		t.Errorf("cross-origin POST must be rejected, got %d", w.Code)
	}
	if w := do(s, "POST", "/api/nodes/join-tokens", nil, withOrigin("https://app.example.com")); w.Code != http.StatusCreated {
		t.Errorf("configured origin must pass, got %d: %s", w.Code, w.Body)
	}
	if w := do(s, "POST", "/api/nodes/join-tokens", nil, auth); w.Code != http.StatusCreated {
		t.Errorf("no Origin (non-browser) must pass, got %d", w.Code)
	}

	// A cross-site HTML form cannot send JSON, so a form body is refused
	req := httptest.NewRequest("POST", "/api/nodes/join-tokens", strings.NewReader("a=b"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set(constants.HeaderGatewayAPIKey, testGWKey)
	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("non-JSON body must be rejected, got %d", w.Code)
	}
}

func TestOriginGuardOnlyWarnsInWarnMode(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeWarn)
	w := do(s, "POST", "/api/nodes/join-tokens", nil, map[string]string{
		constants.HeaderGatewayAPIKey: testGWKey, "Origin": "https://evil.example.net"})
	if w.Code != http.StatusCreated {
		t.Fatalf("warn mode must not block, got %d", w.Code)
	}
}

func TestUnauthenticatedAPIRequiresCredentials(t *testing.T) {
	// Auth is off in this fixture, so the credential paths are exercised through node keys
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	if w := do(s, "GET", "/api/nodes", nil, map[string]string{constants.HeaderGatewayAPIKey: "wrong"}); w.Code != http.StatusUnauthorized {
		t.Errorf("wrong gateway key must be rejected, got %d", w.Code)
	}
	if w := do(s, "GET", "/api/nodes", nil, map[string]string{constants.HeaderNodeID: "primary-id", constants.HeaderNodeAPIKey: "wrong"}); w.Code != http.StatusUnauthorized {
		t.Errorf("wrong node key must be rejected, got %d", w.Code)
	}
	if w := do(s, "GET", "/api/nodes", nil, map[string]string{constants.HeaderNodeID: "primary-id", constants.HeaderNodeAPIKey: testNodeKey}); w.Code != http.StatusOK {
		t.Errorf("correct node key must be accepted, got %d", w.Code)
	}
}

func TestNodeCredentialsCannotAdministerSecurity(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	nodeAuth := map[string]string{constants.HeaderNodeID: "primary-id", constants.HeaderNodeAPIKey: testNodeKey}
	for _, tc := range []struct{ method, path string }{
		{"POST", "/api/nodes/join-tokens"},
		{"POST", "/api/security/revoke-sessions"},
		{"GET", "/api/security/audit"},
		{"GET", "/api/events"},
	} {
		if w := do(s, tc.method, tc.path, nil, nodeAuth); w.Code != http.StatusForbidden {
			t.Errorf("%s %s with node credentials must be 403, got %d", tc.method, tc.path, w.Code)
		}
	}
}

func registerBody(id, key, token string) map[string]string {
	return map[string]string{"id": id, "name": "sec-" + id, "api_endpoint": "http://127.0.0.1:1", "api_key": key, "token": token}
}

func TestAutoRegistrationTokens(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)

	if w := do(s, "POST", "/api/nodes/register", registerBody("n1", "k1", "nope"), nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("bad token must be 401, got %d", w.Code)
	}
	if w := do(s, "POST", "/api/nodes/register", registerBody("n1", "k1", testRegToken), nil); w.Code != http.StatusCreated {
		t.Fatalf("shared token must register, got %d: %s", w.Code, w.Body)
	}

	// Re-registration with the recorded key refreshes; a different key is refused
	if w := do(s, "POST", "/api/nodes/register", registerBody("n1", "k1", testRegToken), nil); w.Code != http.StatusOK {
		t.Fatalf("re-registration with same key must succeed, got %d", w.Code)
	}
	if w := do(s, "POST", "/api/nodes/register", registerBody("n1", "hijack", testRegToken), nil); w.Code != http.StatusConflict {
		t.Fatalf("a different key must not replace an existing node's key, got %d", w.Code)
	}
	if n, _ := database.GetNode("n1"); n == nil || n.APIKey != "k1" {
		t.Fatal("the recorded key must be unchanged")
	}

	// Join tokens work exactly once
	join, _, _ := database.CreateJoinToken(time.Hour)
	if w := do(s, "POST", "/api/nodes/register", registerBody("n2", "k2", join), nil); w.Code != http.StatusCreated {
		t.Fatalf("join token must register, got %d: %s", w.Code, w.Body)
	}
	if w := do(s, "POST", "/api/nodes/register", registerBody("n3", "k3", join), nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("a used join token must be refused, got %d", w.Code)
	}
}

func TestAutoRegistrationRejectsMetadataEndpoint(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	body := registerBody("n9", "k9", testRegToken)
	body["api_endpoint"] = "http://169.254.169.254/latest/meta-data"
	if w := do(s, "POST", "/api/nodes/register", body, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("metadata endpoint must be rejected, got %d", w.Code)
	}
}

func TestRegistrationRateLimit(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	var last int
	for i := 0; i < constants.RegisterRateLimitAttempts+2; i++ {
		last = do(s, "POST", "/api/nodes/register", registerBody("nx", "kx", "wrong"), nil).Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after the budget is spent, got %d", last)
	}
}

func TestAttemptLimiterWindow(t *testing.T) {
	l := newAttemptLimiter(2, time.Minute)
	now := time.Now()
	if !l.allow("a", now) || !l.allow("a", now) || l.allow("a", now) {
		t.Fatal("third attempt inside the window must be refused")
	}
	if !l.allow("b", now) {
		t.Fatal("other keys are independent")
	}
	if !l.allow("a", now.Add(2*time.Minute)) {
		t.Fatal("the budget must reset after the window")
	}
}

func TestAuditRecordsMutations(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	do(s, "POST", "/api/nodes/join-tokens", nil, map[string]string{constants.HeaderGatewayAPIKey: testGWKey})
	entries, err := database.ListAudit(10)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected an audit record: %v %v", err, entries)
	}
	if entries[0].Actor != "gateway" || entries[0].Method != "POST" || entries[0].Status != http.StatusCreated {
		t.Fatalf("unexpected audit record %+v", entries[0])
	}
}

func TestSessionRevocationEndpoint(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	if w := do(s, "POST", "/api/security/revoke-sessions", nil, map[string]string{constants.HeaderGatewayAPIKey: testGWKey}); w.Code != http.StatusOK {
		t.Fatalf("revoke: %d %s", w.Code, w.Body)
	}
	if database.SessionsValidAfter().IsZero() {
		t.Fatal("revocation time must be stored")
	}
}

func TestSecurityHeaders(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	w := do(s, "GET", "/api/health", nil, nil)
	h := w.Header()
	if h.Get("Content-Security-Policy") == "" || h.Get("X-Content-Type-Options") != "nosniff" ||
		h.Get("X-Frame-Options") != "DENY" || h.Get("Permissions-Policy") == "" {
		t.Fatalf("missing security headers: %v", h)
	}
	if h.Get("X-XSS-Protection") != "" {
		t.Fatal("the obsolete X-XSS-Protection header must not be sent")
	}
}

func TestBodyLimit(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	req := httptest.NewRequest("POST", "/api/nodes/join-tokens", strings.NewReader("x"))
	req.ContentLength = maxBodySize + 1
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(constants.HeaderGatewayAPIKey, testGWKey)
	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body must be 413, got %d", w.Code)
	}
}

func TestAutoRegistrationReportsTheRealReachability(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)

	// a node that genuinely answers its health endpoint
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer node.Close()

	up := registerBody("up", "k-up", testRegToken)
	up["api_endpoint"] = node.URL
	w := do(s, "POST", "/api/nodes/register", up, nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body)
	}
	var resp struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != constants.NodeStatusOnline {
		t.Fatalf("a reachable node must be reported online at registration, got %q", resp.Status)
	}
	if n, err := database.GetNode("up"); err != nil || n.Status != constants.NodeStatusOnline {
		t.Fatalf("and stored as online: %v %+v", err, n)
	}

	// a node nothing is listening for
	down := registerBody("down", "k-down", testRegToken)
	w = do(s, "POST", "/api/nodes/register", down, nil)
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if w.Code != http.StatusCreated || resp.Status != constants.NodeStatusUnreachable {
		t.Fatalf("an unreachable node must still register, reported unreachable: %d %q", w.Code, resp.Status)
	}
}
