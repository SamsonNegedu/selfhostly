package http

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
)

var gatewayAuth = map[string]string{constants.HeaderGatewayAPIKey: testGWKey}

func TestUnknownJobIsNotFoundWithTheMessageTheUIExpects(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	for _, path := range []string{"/api/jobs/nope", "/api/jobs/nope/logs"} {
		w := do(s, "GET", path+"?node_id=primary-id", nil, gatewayAuth)
		var body ErrorResponse
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if w.Code != http.StatusNotFound || body.Error != "Job not found" {
			t.Errorf("%s: got %d %s", path, w.Code, w.Body)
		}
	}
}

func TestJobLogsReturnLinesAndTheNextCursor(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	app := db.NewApp("demo", "", "services: {}")
	app.NodeID = "primary-id"
	if err := database.CreateApp(app); err != nil {
		t.Fatal(err)
	}
	job := db.NewJob("create_app", app.ID, nil)
	if err := database.CreateJob(job); err != nil {
		t.Fatal(err)
	}
	if err := database.AppendJobLogLines(job.ID, []string{"one", "two"}); err != nil {
		t.Fatal(err)
	}

	w := do(s, "GET", "/api/jobs/"+job.ID+"/logs?node_id=primary-id", nil, gatewayAuth)
	var got struct {
		Lines     []db.JobLogLine `json:"lines"`
		NextAfter int64           `json:"next_after"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != http.StatusOK {
		t.Fatalf("%d %s %v", w.Code, w.Body, err)
	}
	if len(got.Lines) != 2 || got.NextAfter != got.Lines[1].Seq {
		t.Fatalf("lines and cursor wrong: %+v", got)
	}

	w = do(s, "GET", "/api/apps/"+app.ID+"/jobs?node_id=primary-id", nil, gatewayAuth)
	var jobs []db.Job
	if err := json.Unmarshal(w.Body.Bytes(), &jobs); err != nil || len(jobs) != 1 || jobs[0].ID != job.ID {
		t.Fatalf("app jobs: %d %s", w.Code, w.Body)
	}
}

func TestAppWithoutJobsListsAnEmptyArray(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	w := do(s, "GET", "/api/apps/none/jobs?node_id=primary-id", nil, gatewayAuth)
	if w.Code != http.StatusOK || w.Body.String() != "[]" {
		t.Fatalf("want [], got %d %s", w.Code, w.Body)
	}
}

func TestUpdateSettingsKeepsProviderFieldsLeftEmpty(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	put := func(body map[string]any) map[string]any {
		w := do(s, "PUT", "/api/settings", body, gatewayAuth)
		if w.Code != http.StatusOK {
			t.Fatalf("%d %s", w.Code, w.Body)
		}
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	put(map[string]any{"auto_start_apps": true, "active_tunnel_provider": "cloudflare"})
	got := put(map[string]any{"auto_start_apps": false})
	if got["active_tunnel_provider"] != "cloudflare" || got["auto_start_apps"] != false {
		t.Fatalf("an empty provider must keep the stored one and the flag must change: %v", got)
	}
}

func TestAuditTargetNameLooksUpAppsAndNodes(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	app := db.NewApp("nextcloud", "", "services: {}")
	app.NodeID = "primary-id"
	if err := database.CreateApp(app); err != nil {
		t.Fatal(err)
	}
	if got := s.securityService.AuditTargetName(context.Background(), "app", app.ID); got != "nextcloud" {
		t.Errorf("app name = %q", got)
	}
	if got := s.securityService.AuditTargetName(context.Background(), "node", "primary-id"); got != "primary" {
		t.Errorf("node name = %q", got)
	}
	if got := s.securityService.AuditTargetName(context.Background(), "app", "missing"); got != "" {
		t.Errorf("an unknown target must have no name, got %q", got)
	}
}
