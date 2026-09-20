package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
)

// capture returns what f prints to standard output
func capture(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	rd, wr, _ := os.Pipe()
	os.Stdout = wr
	f()
	wr.Close()
	os.Stdout = old
	b, _ := io.ReadAll(rd)
	return string(b)
}

func TestSummaryListsOnlyWhatNeedsAction(t *testing.T) {
	r := &doctorReport{}
	out := capture(t, func() {
		r.pass("fine")
		r.note("expected in development")
		r.warn("no login configured")
		r.fix("turn on GitHub login")
		r.summary()
	})
	if !strings.Contains(out, "works, but 1 thing(s) need your attention. 1 notes need nothing.") {
		t.Fatalf("verdict missing:\n%s", out)
	}
	todo := out[strings.Index(out, "To do:"):]
	if !strings.Contains(todo, "1. no login configured") || !strings.Contains(todo, "-> turn on GitHub login") {
		t.Fatalf("the warning and its fix must be listed:\n%s", todo)
	}
	if strings.Contains(todo, "expected in development") {
		t.Fatalf("a note must never appear in the to-do list:\n%s", todo)
	}
}

func TestSummaryIsHealthyWithOnlyNotes(t *testing.T) {
	r := &doctorReport{}
	out := capture(t, func() { r.note("optional thing"); r.summary() })
	if !strings.Contains(out, "healthy. Nothing needs action") || strings.Contains(out, "To do:") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestSummaryPutsFailuresFirst(t *testing.T) {
	r := &doctorReport{}
	out := capture(t, func() { r.warn("a warning"); r.fail("a failure"); r.summary() })
	if !strings.Contains(out, "NOT READY") || strings.Index(out, "a failure") > strings.LastIndex(out, "a warning") {
		t.Fatalf("got:\n%s", out)
	}
	if r.fails != 1 || r.warns != 1 {
		t.Fatalf("counts %d %d", r.fails, r.warns)
	}
}

func writeApp(t *testing.T, appsDir, name, volumes string) {
	t.Helper()
	dir := filepath.Join(appsDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "services:\n  s:\n    image: x\n    volumes:\n" + volumes
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func auditCfg(t *testing.T, allowed ...string) (*config.Config, string) {
	t.Helper()
	apps := t.TempDir()
	cfg := &config.Config{AppsDir: apps}
	cfg.Security.Mode = "warn"
	cfg.Security.HostAppsDir = apps
	cfg.Security.AllowedVolumePaths = allowed
	return cfg, apps
}

func TestAuditIgnoresFoldersThatAreNotDeployedApps(t *testing.T) {
	cfg, apps := auditCfg(t)
	writeApp(t, apps, "real", "      - ./data:/d\n")
	writeApp(t, apps, "selfhostly-bk", "      - /var/run/docker.sock:/s\n") // a copy of the platform's own file
	r := &doctorReport{}
	out := capture(t, func() { auditApps(r, cfg, map[string]bool{"real": true}) })
	if r.fails != 0 {
		t.Fatalf("a folder that is not a deployed app must not fail the audit:\n%s", out)
	}
	if !strings.Contains(out, "1 folder(s) in APPS_DIR are not deployed apps") || !strings.Contains(out, "selfhostly-bk") {
		t.Fatalf("it must say what it skipped:\n%s", out)
	}
	if !strings.Contains(out, "1 of 1 app(s) already comply") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestAuditStillFailsARegisteredAppThatMountsTheDockerSocket(t *testing.T) {
	cfg, apps := auditCfg(t)
	writeApp(t, apps, "bad", "      - /var/run/docker.sock:/s\n")
	r := &doctorReport{}
	out := capture(t, func() { auditApps(r, cfg, map[string]bool{"bad": true}) })
	if r.fails != 1 || !strings.Contains(out, "never allowed, in any mode") || strings.Contains(out, "is inside /var/run/docker.sock") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestAuditGroupsFoldersToAllowIntoOneFindingKeepingTheExistingOnes(t *testing.T) {
	cfg, apps := auditCfg(t, "/mnt/media")
	writeApp(t, apps, "n8n", "      - /home/me/.n8n:/data\n")
	writeApp(t, apps, "enclosed", "      - /home/me/.enclosed-data:/d\n")
	writeApp(t, apps, "sealnotes", "      - /home/me/.n8n:/x\n")
	r := &doctorReport{}
	out := capture(t, func() {
		auditApps(r, cfg, map[string]bool{"n8n": true, "enclosed": true, "sealnotes": true})
	})
	if r.warns != 1 {
		t.Fatalf("all of these are one action, so one finding, not %d:\n%s", r.warns, out)
	}
	want := "selfhostlyctl upgrade --set ALLOWED_VOLUME_PATHS=/mnt/media,/home/me/.enclosed-data,/home/me/.n8n"
	if !strings.Contains(out, want) {
		t.Fatalf("the fix must keep the folders already allowed and add the new ones, once each; want %q in:\n%s", want, out)
	}
	if !strings.Contains(out, "/home/me/.n8n  (n8n, sealnotes)") {
		t.Fatalf("it must show which apps use each folder:\n%s", out)
	}
}

func TestAuditTreatsAVariableAsItsOwnFindingNotAnAllowListOne(t *testing.T) {
	cfg, apps := auditCfg(t)
	writeApp(t, apps, "v", "      - ${DATA_DIR:-./data}:/d\n")
	r := &doctorReport{}
	out := capture(t, func() { auditApps(r, cfg, map[string]bool{"v": true}) })
	if r.warns != 1 || !strings.Contains(out, "${DATA_DIR:-./data}") || strings.Contains(out, "ALLOWED_VOLUME_PATHS=") {
		t.Fatalf("allowing a folder cannot fix a variable, and the whole source must be shown:\n%s", out)
	}
}

func TestAuditWithoutADatabaseChecksEveryFolder(t *testing.T) {
	cfg, apps := auditCfg(t)
	writeApp(t, apps, "any", "      - /var/run/docker.sock:/s\n")
	r := &doctorReport{}
	capture(t, func() { auditApps(r, cfg, nil) })
	if r.fails != 1 {
		t.Fatal("with no way to tell what is deployed, every folder is judged")
	}
}

func TestRegisteredAppNamesReadsThisNodesApps(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "s.db")
	database, err := db.Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	mine := db.NewApp("mine", "", "services: {}")
	mine.NodeID = "node-a"
	other := db.NewApp("elsewhere", "", "services: {}")
	other.NodeID = "node-b"
	for _, a := range []*db.App{mine, other} {
		if err := database.CreateApp(a); err != nil {
			t.Fatal(err)
		}
	}
	database.Close()

	cfg := &config.Config{DatabasePath: dbPath}
	cfg.Node.ID = "node-a"
	names, ok := registeredAppNames(cfg)
	if !ok || !names["mine"] || names["elsewhere"] {
		t.Fatalf("got %v %v", names, ok)
	}
	if _, ok := registeredAppNames(&config.Config{DatabasePath: filepath.Join(t.TempDir(), "missing.db")}); ok {
		t.Fatal("a database that is not there cannot say what is deployed")
	}
}
