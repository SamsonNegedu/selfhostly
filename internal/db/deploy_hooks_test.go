package db

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// newTestApp creates an app with a unique name, since apps.name is unique and several tests in
// this file need more than one app.
func newTestApp(t *testing.T, d *DB) *App {
	t.Helper()
	app := NewApp("demo-"+uuid.New().String(), "", "services: {}")
	if err := d.CreateApp(app); err != nil {
		t.Fatal(err)
	}
	return app
}

func TestCreateDeployHookIsStoredHashedAndScopedToItsApp(t *testing.T) {
	d, _ := newTestDB(t)
	appA := newTestApp(t, d)
	appB := newTestApp(t, d)

	plain, hook, err := d.CreateDeployHook(appA.ID, "GitHub Actions", "generic")
	if err != nil || !strings.HasPrefix(plain, "sfd_") {
		t.Fatal(err, plain)
	}
	if hook.AppID != appA.ID || hook.Name != "GitHub Actions" || hook.SourceKind != "generic" {
		t.Fatalf("unexpected hook: %+v", hook)
	}

	var stored string
	if err := d.QueryRow(`SELECT token_hash FROM app_deploy_hooks WHERE id = ?`, hook.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, plain) {
		t.Fatal("token must be stored hashed, not in plaintext")
	}

	// A correct token only matches the app it was made for.
	got, err := d.ConsumeDeployHookToken(appA.ID, plain, "1.2.3.4")
	if err != nil || got == nil || got.ID != hook.ID {
		t.Fatalf("expected a match on its own app: %v %+v", err, got)
	}
	got, err = d.ConsumeDeployHookToken(appB.ID, plain, "1.2.3.4")
	if err != nil || got != nil {
		t.Fatalf("a token must not match a different app: %v %+v", err, got)
	}
}

func TestConsumeDeployHookTokenRecordsLastUse(t *testing.T) {
	d, _ := newTestDB(t)
	app := newTestApp(t, d)
	plain, hook, err := d.CreateDeployHook(app.ID, "CI", "generic")
	if err != nil {
		t.Fatal(err)
	}
	if hook.LastUsedAt != nil {
		t.Fatal("a fresh hook must not have a last-used time")
	}

	before := time.Now().UTC()
	got, err := d.ConsumeDeployHookToken(app.ID, plain, "10.0.0.9")
	if err != nil || got == nil {
		t.Fatalf("expected a match: %v %+v", err, got)
	}
	if got.LastUsedAt == nil || got.LastUsedAt.Before(before.Add(-time.Second)) {
		t.Fatalf("last_used_at must be set to about now, got %v", got.LastUsedAt)
	}
	if got.LastUsedIP != "10.0.0.9" {
		t.Fatalf("last_used_ip must record the caller, got %q", got.LastUsedIP)
	}

	hooks, err := d.ListDeployHooks(app.ID)
	if err != nil || len(hooks) != 1 || hooks[0].LastUsedIP != "10.0.0.9" {
		t.Fatalf("the list must reflect the same usage: %v %+v", err, hooks)
	}
}

func TestConsumeDeployHookTokenRejectsWrongOrMissingTokens(t *testing.T) {
	d, _ := newTestDB(t)
	app := newTestApp(t, d)
	if _, _, err := d.CreateDeployHook(app.ID, "CI", "generic"); err != nil {
		t.Fatal(err)
	}

	if got, err := d.ConsumeDeployHookToken(app.ID, "sfd_wrong", "ip"); err != nil || got != nil {
		t.Fatalf("a wrong token must not match: %v %+v", err, got)
	}
	if got, err := d.ConsumeDeployHookToken(app.ID, "", "ip"); err != nil || got != nil {
		t.Fatalf("an empty token must never match: %v %+v", err, got)
	}
}

func TestRevokeDeployHookStopsItWorkingAndLeavesOthersAlone(t *testing.T) {
	d, _ := newTestDB(t)
	app := newTestApp(t, d)
	plainA, hookA, err := d.CreateDeployHook(app.ID, "A", "generic")
	if err != nil {
		t.Fatal(err)
	}
	plainB, hookB, err := d.CreateDeployHook(app.ID, "B", "generic")
	if err != nil {
		t.Fatal(err)
	}

	if err := d.RevokeDeployHook(app.ID, hookA.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := d.ConsumeDeployHookToken(app.ID, plainA, "ip"); err != nil || got != nil {
		t.Fatalf("a revoked hook must stop matching: %v %+v", err, got)
	}
	if got, err := d.ConsumeDeployHookToken(app.ID, plainB, "ip"); err != nil || got == nil || got.ID != hookB.ID {
		t.Fatalf("revoking one hook must not affect another: %v %+v", err, got)
	}

	if err := d.RevokeDeployHook(app.ID, hookA.ID); !errors.Is(err, ErrDeployHookNotFound) {
		t.Fatalf("revoking an already-gone hook must report not found, got %v", err)
	}
}

func TestListDeployHooksIsScopedPerAppAndNewestFirst(t *testing.T) {
	d, _ := newTestDB(t)
	appA := newTestApp(t, d)
	appB := newTestApp(t, d)
	_, first, err := d.CreateDeployHook(appA.ID, "first", "generic")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := d.CreateDeployHook(appA.ID, "second", "generic"); err != nil {
		t.Fatal(err)
	}
	// created_at can have only second-level precision depending on the platform and driver, so two
	// hooks made in the same test can tie. Backdate the first explicitly instead of sleeping across a
	// second boundary, which is slow and still not guaranteed to land on the right side of it.
	if _, err := d.Exec(`UPDATE app_deploy_hooks SET created_at = ? WHERE id = ?`, first.CreatedAt.Add(-time.Hour), first.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := d.CreateDeployHook(appB.ID, "other app", "generic"); err != nil {
		t.Fatal(err)
	}

	hooks, err := d.ListDeployHooks(appA.ID)
	if err != nil || len(hooks) != 2 || hooks[0].Name != "second" || hooks[1].Name != "first" {
		t.Fatalf("expected [second, first] for appA: %v %+v", err, hooks)
	}

	other, err := d.ListDeployHooks(appB.ID)
	if err != nil || len(other) != 1 || other[0].Name != "other app" {
		t.Fatalf("appB must only see its own hook: %v %+v", err, other)
	}

	none, err := d.ListDeployHooks("does-not-exist")
	if err != nil || len(none) != 0 {
		t.Fatalf("an unknown app has no hooks, never an error: %v %+v", err, none)
	}
}
