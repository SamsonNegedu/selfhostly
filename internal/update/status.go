package update

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// Run states.
const (
	RunPending     = "pending"
	RunRunning     = "running"
	RunSucceeded   = "succeeded"
	RunRolledBack  = "rolled_back"
	RunFailed      = "failed"
	RunInterrupted = "interrupted"
)

// Step states.
const (
	StepPending = "pending"
	StepRunning = "running"
	StepDone    = "done"
	StepFailed  = "failed"
)

// Phases of an update run, in order. They are also the step names the UI shows.
const (
	PhaseVerify        = "verify"
	PhaseRollbackPoint = "rollback_point"
	PhaseConfigure     = "configure"
	PhasePull          = "pull"
	PhaseDryStart      = "dry_start"
	PhaseDatabase      = "database"
	PhasePrimary       = "primary"
	PhaseGateway       = "gateway"
	PhaseFrontend      = "frontend"
	PhaseApps          = "apps"
	PhaseDone          = "done"
)

// Phases lists the steps of a run in order.
var Phases = []string{
	PhaseVerify, PhaseRollbackPoint, PhaseConfigure, PhasePull, PhaseDryStart,
	PhaseDatabase, PhasePrimary, PhaseGateway, PhaseFrontend, PhaseApps,
}

// Run kinds.
const (
	KindUpdate   = "update"
	KindRollback = "rollback"
)

// Where the run files live under the data directory.
const (
	updatesDirName = "updates"
	statusFileName = "status.json"
	statusFileMode = 0o600
	updatesDirMode = 0o700
)

// RunStep is one step of a run.
type RunStep struct {
	Name    string `json:"name"`
	State   string `json:"state"`
	Message string `json:"message"`
}

// Run is the status file: what an update run has done so far.
type Run struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	State       string    `json:"state"`
	Phase       string    `json:"phase"`
	FromVersion string    `json:"from_version"`
	ToVersion   string    `json:"to_version"`
	Message     string    `json:"message"`
	Steps       []RunStep `json:"steps"`
	Warnings    []string  `json:"warnings"`
	StartedAt   string    `json:"started_at"`
	FinishedAt  string    `json:"finished_at"`
}

// NewRun starts a run record with every step pending.
func NewRun(id, from, to string) *Run {
	steps := make([]RunStep, len(Phases))
	for i, p := range Phases {
		steps[i] = RunStep{Name: p, State: StepPending}
	}
	return &Run{
		ID: id, Kind: KindUpdate, State: RunPending, Phase: PhaseVerify, FromVersion: from, ToVersion: to,
		Steps: steps, Warnings: []string{}, StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// rollbackPhases are the steps of a rollback: each service is recreated from the saved state.
var rollbackPhases = []string{PhasePrimary, PhaseGateway, PhaseFrontend}

// NewRollbackRun starts the record of a rollback to a version.
func NewRollbackRun(id, from, to string) *Run {
	r := NewRun(id, from, to)
	r.Kind, r.Phase = KindRollback, PhasePrimary
	r.Steps = make([]RunStep, len(rollbackPhases))
	for i, p := range rollbackPhases {
		r.Steps[i] = RunStep{Name: p, State: StepPending}
	}
	return r
}

// Active reports whether the run has not finished.
func (r *Run) Active() bool { return r.State == RunPending || r.State == RunRunning }

// UpdatesDir is where run files live.
func UpdatesDir(dataDir string) string { return filepath.Join(dataDir, updatesDirName) }

// RunDir is the folder for one run's files (manifest copy, inputs).
func RunDir(dataDir, id string) string { return filepath.Join(UpdatesDir(dataDir), id) }

// StatusPath is the status file.
func StatusPath(dataDir string) string { return filepath.Join(UpdatesDir(dataDir), statusFileName) }

// ReadRun loads the status file. It returns nil, nil when there is none.
func ReadRun(path string) (*Run, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var r Run
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// WriteRun replaces the status file atomically, so a reader never sees half of it.
func WriteRun(path string, r *Run) error {
	if err := os.MkdirAll(filepath.Dir(path), updatesDirMode); err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".status-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Chmod(name, statusFileMode); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Rename(name, path)
}

// Advance marks the current phase done and starts the next one.
func (r *Run) Advance(phase, message string) {
	for i := range r.Steps {
		s := &r.Steps[i]
		switch {
		case s.Name == phase:
			s.State, s.Message = StepRunning, message
		case s.State == StepRunning:
			s.State = StepDone
		}
	}
	r.State, r.Phase = RunRunning, phase
}

// FailStep marks the running step failed.
func (r *Run) FailStep(message string) {
	for i := range r.Steps {
		if r.Steps[i].State == StepRunning {
			r.Steps[i].State, r.Steps[i].Message = StepFailed, message
		}
	}
}

// Finish records how the run ended.
func (r *Run) Finish(state, message string) {
	// a rollback that worked ends as "rolled back", and its steps are done all the same
	worked := state == RunSucceeded || (state == RunRolledBack && r.Kind == KindRollback)
	for i := range r.Steps {
		if r.Steps[i].State == StepRunning && worked {
			r.Steps[i].State = StepDone
		}
	}
	if state == RunSucceeded {
		r.Phase = PhaseDone
	}
	r.State, r.Message = state, message
	r.FinishedAt = time.Now().UTC().Format(time.RFC3339)
}
