package http

import (
	"testing"
	"time"

	"github.com/selfhostly/internal/db"
)

func TestAuditRoutesPointAtRealRoutes(t *testing.T) {
	// Every audited route must exist with the same method and pattern, or the log silently loses its action.
	s, _ := newTestServer(t, "standard")
	registered := map[string]bool{}
	for _, r := range s.engine.Routes() {
		registered[r.Method+" "+r.Path] = true
	}
	for key := range auditRoutes {
		if !registered[key] {
			t.Errorf("audit route %q is not a registered route", key)
		}
	}
}

func TestAuditEntryKeepsActionAndTarget(t *testing.T) {
	_, database := newTestServer(t, "standard")
	entry := db.AuditEntry{
		Time: time.Now(), Actor: "u1", Method: "POST", Path: "/api/apps/a1/stop", Status: 200,
		Action: "app.stop", TargetType: "app", TargetID: "a1", TargetName: "nextcloud",
	}
	if err := database.InsertAudit(entry); err != nil {
		t.Fatal(err)
	}
	got, err := database.ListAudit(10)
	if err != nil || len(got) == 0 {
		t.Fatalf("list: %v %d", err, len(got))
	}
	if got[0].Action != "app.stop" || got[0].TargetName != "nextcloud" || got[0].TargetID != "a1" {
		t.Errorf("entry lost its action or target: %+v", got[0])
	}
}

func TestEmptyAuditLogIsAnEmptyList(t *testing.T) {
	_, database := newTestServer(t, "standard")
	got, err := database.ListAudit(10)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("an empty log must be an empty list, not nil, so the API returns [] and not null")
	}
}
