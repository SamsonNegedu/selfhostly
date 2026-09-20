package ctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	selfhostly "github.com/selfhostly"
	"github.com/selfhostly/internal/constants"
)

func TestComposeWriteNeverOverwritesWithoutForce(t *testing.T) {
	a, _ := newApp(t, &fakeRunner{})
	dest := filepath.Join(a.Dir, primaryCompose)
	if err := runCLI(a, "compose", "write"); err != nil {
		t.Fatal(err)
	}
	want, _ := selfhostly.Files.ReadFile(primaryCompose)
	if got, _ := os.ReadFile(dest); string(got) != string(want) {
		t.Fatal("the written file must be exactly the one that ships with the tool")
	}
	os.WriteFile(dest, []byte("hand edited"), 0o644)
	if err := runCLI(a, "compose", "write"); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("an existing file must be protected, got %v", err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "hand edited" {
		t.Fatal("the existing file was changed")
	}
	if err := runCLI(a, "compose", "write", "--force"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dest); string(got) != string(want) {
		t.Fatal("--force must replace it")
	}
}

func TestComposeWriteRejectsUnknownNames(t *testing.T) {
	a, _ := newApp(t, &fakeRunner{})
	if err := runCLI(a, "compose", "write", "nope"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestEveryShippedComposeFileIsEmbedded(t *testing.T) {
	for name, file := range shippedCompose {
		if b, err := selfhostly.Files.ReadFile(file); err != nil || len(b) == 0 {
			t.Fatalf("%s (%s) is not embedded: %v", name, file, err)
		}
	}
}

func TestComposeDiffComparesWithTheShippedFileWhenThereIsNoCloneHere(t *testing.T) {
	a, out := newApp(t, &fakeRunner{})
	path, cleanup, err := a.newComposeFile("")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := selfhostly.Files.ReadFile(primaryCompose)
	if got, _ := os.ReadFile(path); string(got) != string(want) {
		t.Fatal("must compare with the shipped file")
	}
	if !strings.Contains(out.String(), "ships with selfhostlyctl") {
		t.Fatalf("it must say which file it compared with:\n%s", out)
	}
	cleanup()
	if fileExists(path) {
		t.Fatal("the temporary copy must be removed")
	}

	// a file in the folder wins, and an explicit one wins over both
	local := filepath.Join(a.Dir, primaryCompose)
	os.WriteFile(local, []byte("services: {}\n"), 0o644)
	if p, _, _ := a.newComposeFile(""); p != local {
		t.Fatalf("got %s", p)
	}
	if p, _, _ := a.newComposeFile("/x/y.yml"); p != "/x/y.yml" {
		t.Fatalf("got %s", p)
	}
}

// liveInstall fakes one running primary whose labels say which compose file it was started from
func liveInstall(t *testing.T, files string) *scripted {
	t.Helper()
	st := &stack{primaryHealthy: true}
	return &scripted{handle: func(cmd string) (string, bool) {
		switch {
		case cmd == psAll:
			return "whatever-it-is-called\n", false
		case cmd == "docker exec whatever-it-is-called sh -c "+probeScript:
			return "./selfhostly true\n", false
		case strings.Contains(cmd, constants.LabelComposeConfigFiles):
			return files + "\n", false
		case strings.Contains(cmd, constants.LabelComposeProject):
			return "livestack\n", false
		}
		return st.handle(cmd)
	}}
}

func TestUpgradeUsesTheComposeFileTheInstallWasStartedFrom(t *testing.T) {
	a, _, out := upgradeApp(t, &stack{primaryHealthy: true})
	live := filepath.Join(a.Dir, "docker-compose.yml") // the name on a real Pi: not docker-compose.prod.yml
	os.WriteFile(live, []byte("services: {}\n"), 0o644)
	sc := liveInstall(t, live)
	a.Run = sc
	a.DryRun = true
	if err := runCLI(a, "upgrade"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out.String(), "using the compose file your running install was started from: "+live) {
		t.Fatalf("it must say what it detected:\n%s", out)
	}
	if i := sc.index("-p livestack"); i < 0 || sc.index("-f "+live) < 0 {
		t.Fatalf("the detected project and file must be used: %v", sc.calls)
	}
}

func TestAnExplicitComposeFileIsNeverSecondGuessed(t *testing.T) {
	a, _, _ := upgradeApp(t, &stack{primaryHealthy: true})
	live := filepath.Join(a.Dir, "docker-compose.yml")
	os.WriteFile(live, []byte("services: {}\n"), 0o644)
	sc := liveInstall(t, live)
	a.Run = sc
	a.DryRun = true
	if err := runCLI(a, "upgrade", "--project", "p", "--compose", filepath.Join(a.Dir, "docker-compose.prod.yml")); err != nil {
		t.Fatal(err)
	}
	if sc.index("-f "+live) >= 0 {
		t.Fatal("--compose must be used as given, not replaced by the detected file")
	}
	if sc.index(constants.LabelComposeConfigFiles) >= 0 {
		t.Fatal("nothing should be detected when the file is given")
	}
}

func TestDetectionGivesUpQuietlyWhenTheFilesAreNotOnThisDisk(t *testing.T) {
	a, _, _ := upgradeApp(t, &stack{primaryHealthy: true})
	a.Run = liveInstall(t, "/somewhere/else/docker-compose.yml")
	_, _, err := a.detectLiveCompose(t.Context())
	if err == nil || !strings.Contains(err.Error(), "not on this machine") || !strings.Contains(err.Error(), "--compose") {
		t.Fatalf("a file that is not here must be explained, not guessed around: %v", err)
	}
}

func TestDetectionAsksWhenSeveralPrimariesRunAndNeverGuesses(t *testing.T) {
	a, _, _ := upgradeApp(t, &stack{primaryHealthy: true})
	live := filepath.Join(a.Dir, "docker-compose.yml")
	os.WriteFile(live, []byte("services: {}\n"), 0o644)
	sc := liveInstall(t, live)
	base := sc.handle
	sc.handle = func(cmd string) (string, bool) {
		switch {
		case cmd == psAll:
			return "a\nb\n", false
		case strings.HasPrefix(cmd, "docker exec ") && strings.HasSuffix(cmd, probeScript):
			return "./selfhostly true\n", false
		}
		return base(cmd)
	}
	a.Run = sc
	a.NonInteractive = true
	if _, _, err := a.detectLiveCompose(t.Context()); err == nil || !strings.Contains(err.Error(), "a, b") || !strings.Contains(err.Error(), "--container") {
		t.Fatalf("got %v", err)
	}
	a.NonInteractive = false
	a.Prompt = &pickFirst{}
	if files, _, err := a.detectLiveCompose(t.Context()); err != nil || len(files) != 1 {
		t.Fatalf("interactive detection must ask and continue: %v %v", files, err)
	}
}

func TestDetectionWithNothingRunningFallsBackToTheDefaultFile(t *testing.T) {
	a, _, _ := upgradeApp(t, &stack{primaryHealthy: true})
	files, project, err := a.detectLiveCompose(t.Context())
	if err != nil || len(files) != 0 || project != "" {
		t.Fatalf("got %v %q %v", files, project, err)
	}
}
