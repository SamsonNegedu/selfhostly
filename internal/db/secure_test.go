package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/selfhostly/internal/secrets"
)

func newTestDB(t *testing.T) (*DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	d, err := Init(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d, path
}

func TestVersionedMigrationsApplyOnce(t *testing.T) {
	d, _ := newTestDB(t)
	if d.SchemaVersion() != LatestSchemaVersion() {
		t.Fatalf("schema version %d, want %d", d.SchemaVersion(), LatestSchemaVersion())
	}
	if err := d.runVersionedMigrations(); err != nil {
		t.Fatalf("re-running must be a no-op: %v", err)
	}
}

func TestMigrationBacksUpExistingData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	d, err := Init(path)
	if err != nil {
		t.Fatal(err)
	}
	n := NewNode("primary", "http://primary:8082", "plain-key", true)
	if err := d.CreateNode(n); err != nil {
		t.Fatal(err)
	}
	// Simulate a database from before the versioned migrations existed
	if _, err := d.Exec(`DROP TABLE schema_migrations`); err != nil {
		t.Fatal(err)
	}
	d.Close()

	d2, err := Init(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	matches, _ := filepath.Glob(path + ".bak-pre-migration-*")
	if len(matches) != 1 {
		t.Fatalf("expected one pre-migration backup, got %v", matches)
	}
	got, err := d2.GetNode(n.ID)
	if err != nil || got.APIKey != "plain-key" {
		t.Fatalf("existing data must survive: %v %+v", err, got)
	}
}

func TestFreshDatabaseIsNotBackedUp(t *testing.T) {
	_, path := newTestDB(t)
	if matches, _ := filepath.Glob(path + ".bak-*"); len(matches) != 0 {
		t.Fatalf("no backup expected for an empty database, got %v", matches)
	}
}

func TestJobsSurviveReinit(t *testing.T) {
	d, path := newTestDB(t)
	if _, err := d.Exec(`INSERT INTO jobs (id, type, app_id, status, job_hash) VALUES ('j1','app_start','app1','pending','h')`); err != nil {
		// jobs.app_id has a foreign key; create the parent first
		if _, err := d.Exec(`INSERT INTO apps (id, name, compose_content) VALUES ('app1','a','x')`); err != nil {
			t.Fatal(err)
		}
		if _, err := d.Exec(`INSERT INTO jobs (id, type, app_id, status, job_hash) VALUES ('j1','app_start','app1','pending','h')`); err != nil {
			t.Fatal(err)
		}
	}
	d.Close()
	d2, err := Init(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	var n int
	_ = d2.QueryRow(`SELECT COUNT(*) FROM jobs WHERE id = 'j1'`).Scan(&n)
	if n != 1 {
		t.Fatal("pending jobs must survive a restart")
	}
}

func TestNodeKeyEncryptionAndLegacyPlaintext(t *testing.T) {
	d, _ := newTestDB(t)
	// Legacy plaintext row written before encryption was enabled
	legacy := NewNode("legacy", "http://l:1", "legacy-key", false)
	if err := d.CreateNode(legacy); err != nil {
		t.Fatal(err)
	}

	box, _ := secrets.NewBox("test-key", true)
	d.SetSecretBox(box)
	fresh := NewNode("fresh", "http://f:1", "fresh-key", false)
	if err := d.CreateNode(fresh); err != nil {
		t.Fatal(err)
	}
	var raw string
	_ = d.QueryRow(`SELECT api_key FROM nodes WHERE id = ?`, fresh.ID).Scan(&raw)
	if !secrets.IsSealed(raw) || strings.Contains(raw, "fresh-key") {
		t.Fatalf("new keys must be stored encrypted, got %q", raw)
	}

	if got, _ := d.GetNode(legacy.ID); got == nil || got.APIKey != "legacy-key" {
		t.Fatal("legacy plaintext must still read correctly")
	}
	if err := d.MigrateSecrets(); err != nil {
		t.Fatal(err)
	}
	_ = d.QueryRow(`SELECT api_key FROM nodes WHERE id = ?`, legacy.ID).Scan(&raw)
	if !secrets.IsSealed(raw) {
		t.Fatal("MigrateSecrets must encrypt legacy rows")
	}
	if got, _ := d.GetNode(legacy.ID); got.APIKey != "legacy-key" {
		t.Fatal("migrated key must decrypt to the original")
	}
	all, err := d.GetAllNodes()
	if err != nil || len(all) != 2 {
		t.Fatalf("list must decrypt all: %v %d", err, len(all))
	}
}

func TestSettingsSecretsRoundTrip(t *testing.T) {
	d, _ := newTestDB(t)
	box, _ := secrets.NewBox("k", true)
	d.SetSecretBox(box)
	s, err := d.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"cloudflare":{"api_token":"tok"}}`
	s.TunnelProviderConfig = &cfg
	if err := d.UpdateSettings(s); err != nil {
		t.Fatal(err)
	}
	var raw string
	_ = d.QueryRow(`SELECT tunnel_provider_config FROM settings LIMIT 1`).Scan(&raw)
	if strings.Contains(raw, "tok") {
		t.Fatal("provider config must not be stored in plaintext")
	}
	got, _ := d.GetSettings()
	if got.TunnelProviderConfig == nil || *got.TunnelProviderConfig != cfg {
		t.Fatal("provider config must decrypt on read")
	}
}

func TestJoinTokenSingleUseAndExpiry(t *testing.T) {
	d, _ := newTestDB(t)
	tok, _, err := d.CreateJoinToken(time.Hour)
	if err != nil || !strings.HasPrefix(tok, "sfj_") {
		t.Fatal(err, tok)
	}
	if ok, _ := d.ConsumeJoinToken(tok, "node-1"); !ok {
		t.Fatal("first use must succeed")
	}
	if ok, _ := d.ConsumeJoinToken(tok, "node-2"); ok {
		t.Fatal("second use must fail")
	}
	if ok, _ := d.ConsumeJoinToken("sfj_unknown", "n"); ok {
		t.Fatal("unknown token must fail")
	}
	if ok, _ := d.ConsumeJoinToken("not-a-join-token", "n"); ok {
		t.Fatal("shared registration tokens are handled elsewhere")
	}
	expired, _, _ := d.CreateJoinToken(-time.Minute)
	if ok, _ := d.ConsumeJoinToken(expired, "n"); ok {
		t.Fatal("expired token must fail")
	}
	var stored string
	_ = d.QueryRow(`SELECT token_hash FROM join_tokens LIMIT 1`).Scan(&stored)
	if strings.Contains(stored, tok) {
		t.Fatal("token must be stored hashed")
	}
}

func TestSessionRevocationAndAudit(t *testing.T) {
	d, _ := newTestDB(t)
	if !d.SessionsValidAfter().IsZero() {
		t.Fatal("no revocation by default")
	}
	at, err := d.RevokeSessions()
	if err != nil || d.SessionsValidAfter().Before(at.Add(-time.Second)) {
		t.Fatal("revocation timestamp must be stored")
	}
	if err := d.InsertAudit(AuditEntry{Time: time.Now(), Actor: "u", Method: "POST", Path: "/api/x", Status: 200}); err != nil {
		t.Fatal(err)
	}
	list, err := d.ListAudit(10)
	if err != nil || len(list) != 1 || list[0].Actor != "u" {
		t.Fatalf("audit list: %v %+v", err, list)
	}
	if err := d.PruneAudit(time.Hour); err != nil {
		t.Fatal(err)
	}
}

func TestBackupPruning(t *testing.T) {
	d, path := newTestDB(t)
	for i := 0; i < 8; i++ {
		if err := os.WriteFile(path+".bak-old-"+string(rune('a'+i)), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.Backup("manual"); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(path + ".bak-*")
	if len(matches) != 5 {
		t.Fatalf("expected 5 retained backups, got %d", len(matches))
	}
}

func TestMigrateSecretsWorksInBothDirections(t *testing.T) {
	d, _ := newTestDB(t)
	n := NewNode("n", "http://n:1", "the-key", false)
	if err := d.CreateNode(n); err != nil {
		t.Fatal(err)
	}
	raw := func() string {
		var v string
		_ = d.QueryRow(`SELECT api_key FROM nodes WHERE id = ?`, n.ID).Scan(&v)
		return v
	}

	on, _ := secrets.NewBox("k", true)
	d.SetSecretBox(on)
	if err := d.MigrateSecrets(); err != nil || !secrets.IsSealed(raw()) {
		t.Fatalf("flag on must encrypt: %v %q", err, raw())
	}

	off, _ := secrets.NewBox("k", false)
	d.SetSecretBox(off)
	if err := d.MigrateSecrets(); err != nil || raw() != "the-key" {
		t.Fatalf("flag off must decrypt back to plaintext so an older build can read it: %v %q", err, raw())
	}
	if err := d.MigrateSecrets(); err != nil {
		t.Fatalf("must be idempotent: %v", err)
	}
}

func TestMigrateSecretsReportsUnreadableValuesWithoutFailing(t *testing.T) {
	d, _ := newTestDB(t)
	n := NewNode("n", "http://n:1", "the-key", false)
	on, _ := secrets.NewBox("original", true)
	d.SetSecretBox(on)
	if err := d.CreateNode(n); err != nil {
		t.Fatal(err)
	}

	lost, _ := secrets.NewBox("a-different-key", false)
	d.SetSecretBox(lost)
	before := ""
	_ = d.QueryRow(`SELECT api_key FROM nodes`).Scan(&before)
	if err := d.MigrateSecrets(); err != nil {
		t.Fatalf("a lost key must not stop startup: %v", err)
	}
	after := ""
	_ = d.QueryRow(`SELECT api_key FROM nodes`).Scan(&after)
	if before != after {
		t.Fatal("an unreadable value must be left untouched, never overwritten or blanked")
	}
	if _, err := d.GetNode(n.ID); err == nil {
		t.Fatal("reading a node whose key cannot be decrypted must fail rather than return a wrong key")
	}
}

func TestTunnelTokensAreEncryptedForExistingAndNewApps(t *testing.T) {
	d, _ := newTestDB(t)
	raw := func(table, id string) string {
		var v string
		_ = d.QueryRow(`SELECT tunnel_token FROM `+table+` WHERE id = ?`, id).Scan(&v)
		return v
	}
	now := time.Now()

	// An app and a tunnel saved before encryption was turned on hold the token as plain text.
	old := &App{ID: "old", Name: "old", TunnelToken: "plain-token", Status: "running", CreatedAt: now}
	if err := d.CreateApp(old); err != nil {
		t.Fatal(err)
	}
	if err := d.CreateCloudflareTunnel(&CloudflareTunnel{ID: "t-old", AppID: "old", TunnelID: "tid", TunnelName: "n", TunnelToken: "plain-tunnel-token", AccountID: "a", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if raw("apps", "old") != "plain-token" {
		t.Fatalf("setup: expected plain text, got %q", raw("apps", "old"))
	}

	box, _ := secrets.NewBox("k", true)
	d.SetSecretBox(box)

	// Existing rows are readable before they are converted...
	if got, err := d.GetApp("old"); err != nil || got.TunnelToken != "plain-token" {
		t.Fatalf("legacy plain text must still read: %v %q", err, got.TunnelToken)
	}
	// ...and the startup pass converts them, without changing what the app sees.
	if err := d.MigrateSecrets(); err != nil {
		t.Fatal(err)
	}
	if !secrets.IsSealed(raw("apps", "old")) || !secrets.IsSealed(raw("cloudflare_tunnels", "t-old")) {
		t.Fatalf("existing tokens must be encrypted: %q %q", raw("apps", "old"), raw("cloudflare_tunnels", "t-old"))
	}
	if got, err := d.GetApp("old"); err != nil || got.TunnelToken != "plain-token" {
		t.Fatalf("token must read back unchanged: %v %q", err, got.TunnelToken)
	}
	if tun, err := d.GetCloudflareTunnelByAppID("old"); err != nil || tun.TunnelToken != "plain-tunnel-token" {
		t.Fatalf("tunnel token must read back unchanged: %v", err)
	}

	// New and updated apps are encrypted on write, and saving a loaded app does not double-encrypt it.
	fresh := &App{ID: "new", Name: "new", TunnelToken: "fresh-token", Status: "running", CreatedAt: now}
	if err := d.CreateApp(fresh); err != nil {
		t.Fatal(err)
	}
	if !secrets.IsSealed(raw("apps", "new")) || strings.Contains(raw("apps", "new"), "fresh-token") {
		t.Fatalf("a new app's token must be stored encrypted: %q", raw("apps", "new"))
	}
	loaded, _ := d.GetApp("new")
	if err := d.UpdateApp(loaded); err != nil {
		t.Fatal(err)
	}
	if again, _ := d.GetApp("new"); again.TunnelToken != "fresh-token" {
		t.Fatalf("token changed after an update: %q", again.TunnelToken)
	}
	all, err := d.GetAllApps()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range all {
		if secrets.IsSealed(a.TunnelToken) {
			t.Fatalf("GetAllApps returned ciphertext for %s", a.ID)
		}
	}

	// Turning the flag off puts plain text back, so an older build can still read the database.
	off, _ := secrets.NewBox("k", false)
	d.SetSecretBox(off)
	if err := d.MigrateSecrets(); err != nil || raw("apps", "old") != "plain-token" {
		t.Fatalf("flag off must restore plain text: %v %q", err, raw("apps", "old"))
	}
}

func TestUnreadableAppTokenIsAnErrorNotAnEmptyToken(t *testing.T) {
	d, _ := newTestDB(t)
	on, _ := secrets.NewBox("original", true)
	d.SetSecretBox(on)
	if err := d.CreateApp(&App{ID: "a", Name: "a", TunnelToken: "secret", Status: "running", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}

	// With the key lost, returning an empty token would let the next save wipe the stored one.
	wrong, _ := secrets.NewBox("different", false)
	d.SetSecretBox(wrong)
	if _, err := d.GetApp("a"); err == nil {
		t.Fatal("expected an error for a token that cannot be decrypted")
	}
}

func TestCountActiveJobsCountsPendingAndRunningOnly(t *testing.T) {
	database, _ := newTestDB(t)
	if n, err := database.CountActiveJobs(); err != nil || n != 0 {
		t.Fatalf("no jobs: %d %v", n, err)
	}
	app := NewApp("demo", "", "services: {}")
	if err := database.CreateApp(app); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"pending", "running", "completed", "failed"} {
		job := NewJob("app_update", app.ID, nil)
		if err := database.CreateJob(job); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`UPDATE jobs SET status = ? WHERE id = ?`, status, job.ID); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := database.CountActiveJobs(); err != nil || n != 2 {
		t.Fatalf("want 2 (pending and running), got %d %v", n, err)
	}
}
