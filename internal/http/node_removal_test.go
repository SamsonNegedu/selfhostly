package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
)

func addNode(t *testing.T, database *db.DB, id, status string, primary bool) {
	t.Helper()
	n := db.NewNodeWithID(id, "node-"+id, "http://127.0.0.1:1", "key", primary)
	n.Status = status
	if err := database.CreateNode(n); err != nil {
		t.Fatal(err)
	}
}

func addAppOn(t *testing.T, database *db.DB, nodeID string) *db.App {
	t.Helper()
	app := db.NewApp("app-on-"+nodeID, "", "services: {}")
	app.NodeID = nodeID
	if err := database.CreateApp(app); err != nil {
		t.Fatal(err)
	}
	return app
}

func errorCode(t *testing.T, body []byte) string {
	t.Helper()
	var e struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(body, &e)
	return e.Code
}

func TestStalePrimaryRecordCanBeRemoved(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	addNode(t, database, "old-primary", "unreachable", true)

	w := do(s, "DELETE", "/api/nodes/old-primary", nil, map[string]string{constants.HeaderGatewayAPIKey: testGWKey})
	if w.Code != http.StatusOK {
		t.Fatalf("a primary record of another machine must be removable, got %d: %s", w.Code, w.Body)
	}
}

func TestCurrentNodeCannotBeRemovedFromItself(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	w := do(s, "DELETE", "/api/nodes/primary-id", nil, map[string]string{constants.HeaderGatewayAPIKey: testGWKey})
	if w.Code != http.StatusConflict || errorCode(t, w.Body.Bytes()) != "CURRENT_NODE" {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
}

func TestNodeWithAppsNeedsForceAndOnlyWhenItIsDown(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	auth := map[string]string{constants.HeaderGatewayAPIKey: testGWKey}
	addNode(t, database, "down", "unreachable", false)
	app := addAppOn(t, database, "down")

	if w := do(s, "DELETE", "/api/nodes/down", nil, auth); w.Code != http.StatusConflict || errorCode(t, w.Body.Bytes()) != "NODE_HAS_APPS" {
		t.Fatalf("without force it must explain the apps: %d %s", w.Code, w.Body)
	}
	if w := do(s, "DELETE", "/api/nodes/down?force=true", nil, auth); w.Code != http.StatusOK {
		t.Fatalf("force must remove an unreachable node: %d %s", w.Code, w.Body)
	}
	if got, err := database.GetApp(app.ID); err == nil && got != nil {
		t.Error("the app record on the removed node should be gone")
	}
}

func TestForceIsRefusedForANodeThatAnswers(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	addNode(t, database, "up", "online", false)
	addAppOn(t, database, "up")

	w := do(s, "DELETE", "/api/nodes/up?force=true", nil, map[string]string{constants.HeaderGatewayAPIKey: testGWKey})
	if w.Code != http.StatusConflict || errorCode(t, w.Body.Bytes()) != "NODE_ONLINE" {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
}
