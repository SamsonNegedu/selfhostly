package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStatusFileRoundTripAndAtomicReplace(t *testing.T) {
	path := StatusPath(t.TempDir())
	if r, err := ReadRun(path); r != nil || err != nil {
		t.Fatalf("no file yet: %v %v", r, err)
	}
	run := NewRun("id1", "1.3.0", "1.4.0")
	if err := WriteRun(path, run); err != nil {
		t.Fatal(err)
	}
	run.Advance(PhasePull, "pulling")
	if err := WriteRun(path, run); err != nil {
		t.Fatal(err)
	}
	got, err := ReadRun(path)
	if err != nil || got.Phase != PhasePull || got.State != RunRunning {
		t.Fatalf("got %+v %v", got, err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatalf("no temp files may be left behind, got %d entries", len(entries))
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != statusFileMode {
		t.Fatalf("status must be owner only, got %v", info.Mode())
	}
}

func TestRunStepsAdvanceAndFinish(t *testing.T) {
	r := NewRun("id", "1.0.0", "1.1.0")
	r.Advance(PhaseVerify, "")
	r.Advance(PhaseRollbackPoint, "")
	if r.Steps[0].State != StepDone || r.Steps[1].State != StepRunning {
		t.Fatalf("steps: %+v", r.Steps)
	}
	r.FailStep("boom")
	if r.Steps[1].State != StepFailed || r.Steps[1].Message != "boom" {
		t.Fatalf("steps: %+v", r.Steps)
	}
	r.Finish(RunRolledBack, "rolled back")
	if r.Active() || r.FinishedAt == "" {
		t.Fatalf("finished run: %+v", r)
	}
}

func TestARollbackRunHasItsOwnStepsAndItsStepsFinishDone(t *testing.T) {
	r := NewRollbackRun("r", "1.1.0", "1.0.0")
	if r.Kind != KindRollback || len(r.Steps) != 3 || r.ToVersion != "1.0.0" {
		t.Fatalf("got %+v", r)
	}
	r.Advance(PhasePrimary, "")
	r.Advance(PhaseGateway, "")
	r.Advance(PhaseFrontend, "")
	r.Finish(RunRolledBack, "restored")
	for _, s := range r.Steps {
		if s.State != StepDone {
			t.Fatalf("a rollback that worked must not leave a step running: %+v", r.Steps)
		}
	}
}

func TestAnUpdateThatRolledBackKeepsItsFailedStep(t *testing.T) {
	r := NewRun("r", "1.0.0", "1.1.0")
	r.Advance(PhasePrimary, "")
	r.FailStep("did not become healthy")
	r.Finish(RunRolledBack, "did not become healthy")
	if r.Kind != KindUpdate || r.Steps[6].State != StepFailed {
		t.Fatalf("got %+v", r.Steps)
	}
}
