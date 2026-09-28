package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
)

// createTestApp inserts an app directly, the same shortcut internal/db's own tests use, so these
// tests do not depend on the create-app HTTP flow.
func createTestApp(t *testing.T, database *db.DB, name string) *db.App {
	t.Helper()
	app := db.NewApp(name, "", "services: {}")
	if err := database.CreateApp(app); err != nil {
		t.Fatal(err)
	}
	return app
}

func createHook(t *testing.T, s *Server, appID, name string) (id, token string) {
	t.Helper()
	w := do(s, "POST", "/api/apps/"+appID+"/deploy-hooks", map[string]string{"name": name}, gatewayAuth)
	if w.Code != http.StatusCreated {
		t.Fatalf("create hook: %d %s", w.Code, w.Body)
	}
	var resp struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp.ID, resp.Token
}

func TestDeployHookCreateListRevoke(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	app := createTestApp(t, database, "hooked")

	id, token := createHook(t, s, app.ID, "GitHub Actions")
	if id == "" || token == "" {
		t.Fatalf("expected an id and a plaintext token, got %q %q", id, token)
	}

	w := do(s, "GET", "/api/apps/"+app.ID+"/deploy-hooks", nil, gatewayAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), token) {
		t.Fatal("the list response must never include the plaintext token")
	}
	// Decode into a plain map, not a Go struct: json.Unmarshal matches a struct field by name
	// case-insensitively when there is no exact tag match, which would silently hide a response
	// missing its snake_case keys (app_id, source_kind, created_at, ...) - exactly the bug this
	// test exists to catch.
	var raw []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) != 1 {
		t.Fatalf("expected one hook, got %+v", raw)
	}
	for _, field := range []string{"id", "app_id", "name", "source_kind", "created_at", "last_used_at", "last_used_ip"} {
		if _, ok := raw[0][field]; !ok {
			t.Errorf("list response is missing %q: %+v", field, raw[0])
		}
	}
	if raw[0]["id"] != id || raw[0]["name"] != "GitHub Actions" || raw[0]["app_id"] != app.ID {
		t.Fatalf("unexpected list: %+v", raw)
	}
	if createdAt, _ := raw[0]["created_at"].(string); createdAt == "" || strings.HasPrefix(createdAt, "0001-01-01") {
		t.Fatalf("created_at must be the real creation time, got %v", raw[0]["created_at"])
	}

	var hooks []db.DeployHook
	if err := json.Unmarshal(w.Body.Bytes(), &hooks); err != nil {
		t.Fatal(err)
	}
	if len(hooks) != 1 || hooks[0].ID != id || hooks[0].Name != "GitHub Actions" || hooks[0].AppID != app.ID || hooks[0].CreatedAt.IsZero() {
		t.Fatalf("unexpected list, decoded into db.DeployHook: %+v", hooks)
	}

	if w := do(s, "DELETE", "/api/apps/"+app.ID+"/deploy-hooks/"+id, nil, gatewayAuth); w.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d %s", w.Code, w.Body)
	}
	if w := do(s, "DELETE", "/api/apps/"+app.ID+"/deploy-hooks/"+id, nil, gatewayAuth); w.Code != http.StatusNotFound {
		t.Fatalf("revoking an already-gone hook must be 404, got %d", w.Code)
	}
}

func TestDeployTriggerAcceptsOnlyItsOwnAppsValidToken(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	appA := createTestApp(t, database, "trigger-a")
	appB := createTestApp(t, database, "trigger-b")
	_, tokenA := createHook(t, s, appA.ID, "hook-a")

	if w := do(s, "POST", "/api/apps/"+appA.ID+"/deploy-trigger", nil, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("missing token must be 401, got %d", w.Code)
	}
	if w := do(s, "POST", "/api/apps/"+appA.ID+"/deploy-trigger", nil, map[string]string{"Authorization": "Bearer wrong"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token must be 401, got %d", w.Code)
	}
	if w := do(s, "POST", "/api/apps/"+appB.ID+"/deploy-trigger", nil, map[string]string{"Authorization": "Bearer " + tokenA}); w.Code != http.StatusUnauthorized {
		t.Fatalf("a token for a different app must be 401, got %d", w.Code)
	}

	w := do(s, "POST", "/api/apps/"+appA.ID+"/deploy-trigger", nil, map[string]string{"Authorization": "Bearer " + tokenA})
	if w.Code != http.StatusAccepted {
		t.Fatalf("a valid token for its own app must be 202, got %d: %s", w.Code, w.Body)
	}
	var resp struct {
		JobID string `json:"job_id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.JobID == "" {
		t.Fatal("expected a job id, the same as the manual Update button would get")
	}

	// The audit trail names the hook, not a user - there is no session on this request.
	entries, err := database.ListAudit(10)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected an audit record: %v %v", err, entries)
	}
	if entries[0].Actor != "deploy-hook:hook-a" || entries[0].Action != "app.deploy_trigger" {
		t.Fatalf("unexpected audit record %+v", entries[0])
	}
}

func TestDeployTriggerStopsWorkingOnceRevoked(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	app := createTestApp(t, database, "revoke-then-trigger")
	id, token := createHook(t, s, app.ID, "temp")

	if w := do(s, "POST", "/api/apps/"+app.ID+"/deploy-trigger", nil, map[string]string{"Authorization": "Bearer " + token}); w.Code != http.StatusAccepted {
		t.Fatalf("first trigger should succeed: %d %s", w.Code, w.Body)
	}
	do(s, "DELETE", "/api/apps/"+app.ID+"/deploy-hooks/"+id, nil, gatewayAuth)
	if w := do(s, "POST", "/api/apps/"+app.ID+"/deploy-trigger", nil, map[string]string{"Authorization": "Bearer " + token}); w.Code != http.StatusUnauthorized {
		t.Fatalf("a revoked token must stop working, got %d", w.Code)
	}
}

func TestDeployTriggerRateLimit(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	app := createTestApp(t, database, "rate-limited")

	var last int
	for i := 0; i < constants.DeployTriggerRateLimitAttempts+2; i++ {
		w := do(s, "POST", "/api/apps/"+app.ID+"/deploy-trigger", nil, map[string]string{"Authorization": "Bearer wrong"})
		last = w.Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after the budget is spent, got %d", last)
	}
}
