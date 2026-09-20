package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/update"
)

// stubUpdates answers for the update service so the routes can be tested without Docker
type stubUpdates struct {
	err     error
	view    update.View
	applied update.ApplyRequest
}

func (s *stubUpdates) Start(context.Context)            {}
func (s *stubUpdates) View(context.Context) update.View { return s.view }
func (s *stubUpdates) Check(context.Context) error      { return s.err }
func (s *stubUpdates) Review(string) error              { return s.err }
func (s *stubUpdates) Rollback(context.Context) (*update.Run, error) {
	if s.err != nil {
		return nil, s.err
	}
	return update.NewRun("r1", "1.3.0", ""), nil
}
func (s *stubUpdates) Apply(_ context.Context, req update.ApplyRequest) (*update.Run, error) {
	s.applied = req
	if s.err != nil {
		return nil, s.err
	}
	return update.NewRun("r1", "1.3.0", req.Version), nil
}

func TestUpdateRoutesRefuseNodeCredentials(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	nodeAuth := map[string]string{constants.HeaderNodeID: "primary-id", constants.HeaderNodeAPIKey: testNodeKey}
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/system/update"},
		{"POST", "/api/system/update/check"},
		{"POST", "/api/system/update/plan"},
		{"POST", "/api/system/update/apply"},
		{"POST", "/api/system/update/rollback"},
	} {
		if w := do(s, tc.method, tc.path, map[string]string{"version": "1.4.0"}, nodeAuth); w.Code != http.StatusForbidden {
			t.Errorf("%s %s with node credentials must be 403, got %d", tc.method, tc.path, w.Code)
		}
	}
}

func TestUpdateRoutesNeedSomeCredentials(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	if w := do(s, "POST", "/api/system/update/apply", map[string]string{"version": "1.4.0"}, map[string]string{constants.HeaderGatewayAPIKey: "wrong"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("a wrong gateway key must be refused, got %d", w.Code)
	}
}

func TestUpdatesAreOffByDefaultAndCannotBeStarted(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	w := do(s, "GET", "/api/system/update", nil, gatewayAuth)
	var v update.View
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &v) != nil || v.Enabled || v.DisabledReason != update.ReasonNotEnabled {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	for _, path := range []string{"check", "plan", "apply", "rollback"} {
		w := do(s, "POST", "/api/system/update/"+path, map[string]string{"version": "1.4.0"}, gatewayAuth)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s while the feature is off must be 403, got %d %s", path, w.Code, w.Body)
		}
	}
}

func TestUpdateViewSerialisesAbsentPartsAsNull(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	s.updateService = &stubUpdates{view: update.View{Enabled: true, CurrentVersion: "1.3.0"}}
	w := do(s, "GET", "/api/system/update", nil, gatewayAuth)
	var raw map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &raw)
	for _, key := range []string{"available", "plan", "run"} {
		if string(raw[key]) != "null" {
			t.Errorf("%s must be null when absent, got %s", key, raw[key])
		}
	}
}

func TestApplyAnswersWithTheStateAndIsAudited(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	stub := &stubUpdates{view: update.View{Enabled: true, CurrentVersion: "1.3.0"}}
	s.updateService = stub
	w := do(s, "POST", "/api/system/update/apply", map[string]any{
		"version": "1.4.0", "inputs": map[string]string{"KEY": "v"}, "approve_compose": "tok"}, gatewayAuth)
	if w.Code != http.StatusAccepted {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if w.Body.Len() == 0 {
		t.Fatal("a 202 must carry the state: the UI client rejects an empty body")
	}
	if stub.applied.Version != "1.4.0" || stub.applied.Inputs["KEY"] != "v" || stub.applied.ApproveCompose != "tok" {
		t.Fatalf("the request did not reach the service: %+v", stub.applied)
	}
	entries, _ := database.ListAudit(5)
	if len(entries) == 0 || entries[0].Action != "update.apply" || entries[0].TargetName != "1.4.0" {
		t.Fatalf("an update must be audited with its version, got %+v", entries)
	}
	for _, e := range entries {
		if e.Path != "" && len(e.TargetName) > 0 && e.TargetName == "v" {
			t.Fatal("setting values must never reach the audit log")
		}
	}
}

func TestApplyRefusesACrossOriginRequest(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	stub := &stubUpdates{}
	s.updateService = stub
	w := do(s, "POST", "/api/system/update/apply", map[string]string{"version": "1.4.0"},
		map[string]string{constants.HeaderGatewayAPIKey: testGWKey, "Origin": "https://evil.example.net"})
	if w.Code != http.StatusForbidden || stub.applied.Version != "" {
		t.Fatalf("a cross-site page must not be able to start an update: %d", w.Code)
	}
}

func TestUpdateErrorsMapToStableCodes(t *testing.T) {
	blocked := &update.BlockedError{Blockers: []update.Blocker{{Code: update.BlockJobRunning, Message: "wait"}}}
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{blocked, http.StatusConflict, codeUpdateBlocked},
		{update.ErrInProgress, http.StatusConflict, codeUpdateInProgress},
		{update.ErrNoPlan, http.StatusConflict, codeReviewRequired},
		{update.ErrNotApproved, http.StatusConflict, codeComposeNotApproved},
		{update.ErrUnknownVersion, http.StatusConflict, codeUnknownVersion},
		{update.ErrBadInput, http.StatusBadRequest, codeInvalidInput},
		{update.ErrDisabled, http.StatusForbidden, codeUpdatesUnavailable},
		{errors.New("docker exploded"), http.StatusInternalServerError, ""},
	}
	for _, tc := range cases {
		s, _ := newTestServer(t, constants.SecurityModeEnforce)
		s.updateService = &stubUpdates{err: tc.err}
		w := do(s, "POST", "/api/system/update/apply", map[string]string{"version": "1.4.0"}, gatewayAuth)
		var body struct {
			Code     string           `json:"code"`
			Blockers []update.Blocker `json:"blockers"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)
		if w.Code != tc.status || body.Code != tc.code {
			t.Errorf("%v: got %d %q, want %d %q", tc.err, w.Code, body.Code, tc.status, tc.code)
		}
		if tc.err == error(blocked) && len(body.Blockers) != 1 {
			t.Errorf("the blockers must reach the UI, got %+v", body)
		}
	}
}

func TestFailedCheckStillAnswersWithTheView(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	s.updateService = &stubUpdates{err: errors.New("could not fetch"), view: update.View{Enabled: true, CheckError: "could not fetch"}}
	w := do(s, "POST", "/api/system/update/check", nil, gatewayAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("a failed check is reported in the view, not as an error: %d %s", w.Code, w.Body)
	}
}

func TestAnInstallWithNoAppsListsAnEmptyArrayNotNull(t *testing.T) {
	s, _ := newTestServer(t, constants.SecurityModeEnforce)
	w := do(s, "GET", "/api/apps", nil, gatewayAuth)
	if w.Code != http.StatusOK || w.Body.String() != "[]" {
		t.Fatalf("the UI calls .filter on this list, so it must be [] and not null: %d %q", w.Code, w.Body.String())
	}
}
