package update

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Why the feature is off, as the UI shows it.
const (
	ReasonNotEnabled  = "not enabled"
	ReasonNoKey       = "no signing key"
	ReasonNoSocket    = "no docker socket"
	ReasonSecondary   = "secondary node"
	ReasonAuthDisable = "auth disabled"
)

const (
	fetchTimeout      = 20 * time.Second
	planTimeout       = 20 * time.Minute
	launchTimeout     = 2 * time.Minute
	minFreeBytes      = 512 << 20
	initialCheckDelay = 15 * time.Second
	keepRunDirs       = 5
	releaseDirName    = "release"
	releaseFile       = "release.json"
	releaseSigFile    = "release.json.sig"
	inputsFile        = "inputs.json"
	runIDLayout       = "20060102T150405Z"
	fileMode          = 0o600
	dirMode           = 0o700
)

// Errors the HTTP layer turns into status codes.
var (
	ErrDisabled       = errors.New("updates from the UI are not available on this install")
	ErrInProgress     = errors.New("an update is already running")
	ErrNoPlan         = errors.New("review the update first")
	ErrUnknownVersion = errors.New("that version is not the one the last check found")
	ErrNotApproved    = errors.New("approve the compose file change first")
	ErrBadInput       = errors.New("a value is not acceptable")
)

// BlockedError carries the reasons an update cannot start.
type BlockedError struct{ Blockers []Blocker }

func (e *BlockedError) Error() string {
	msgs := make([]string, len(e.Blockers))
	for i, b := range e.Blockers {
		msgs[i] = b.Message
	}
	return "the update is blocked: " + strings.Join(msgs, "; ")
}

// Settings is what the manager needs to know about this install.
type Settings struct {
	Enabled        bool
	ManifestURL    string
	PublicKey      string
	ImagePrefix    string
	CheckInterval  time.Duration
	CurrentVersion string
	IsPrimary      bool
	AuthEnabled    bool
	DockerHost     string // DOCKER_HOST: set when a socket proxy stands in for the socket
}

// Options wires the manager to the outside world. Tests replace every field.
type Options struct {
	Settings     Settings
	Docker       Docker
	HTTP         *http.Client
	ActiveJobs   func() (int, error)
	DataDir      string
	Hostname     string
	Now          func() time.Time
	FreeBytes    func(path string) (uint64, error)
	SocketExists func() bool
}

// Manager checks for releases, reviews them and starts updates. It never runs an update itself: the updater
// container does, because the primary is what gets replaced.
type Manager struct {
	opt Options

	// runMu serializes the calls that start work (review, apply, rollback), so two requests at once cannot both
	// pass the "nothing is running" check before either has written the status file.
	runMu sync.Mutex

	mu        sync.Mutex
	latest    *Manifest
	latestRaw []byte
	latestSig []byte
	checkedAt time.Time
	checkErr  string
	plan      *Plan
	planBusy  bool
}

// NewManager fills in defaults for anything left unset.
func NewManager(o Options) *Manager {
	if o.Docker == nil {
		o.Docker = ExecDocker{}
	}
	if o.HTTP == nil {
		o.HTTP = &http.Client{Timeout: fetchTimeout}
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.FreeBytes == nil {
		o.FreeBytes = freeBytes
	}
	if o.SocketExists == nil {
		o.SocketExists = func() bool { _, err := os.Stat(defaultSocket); return err == nil }
	}
	if o.ActiveJobs == nil {
		o.ActiveJobs = func() (int, error) { return 0, nil }
	}
	if o.Settings.ImagePrefix == "" {
		o.Settings.ImagePrefix = DefaultImagePrefix
	}
	return &Manager{opt: o}
}

func freeBytes(path string) (uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, err
	}
	return uint64(st.Bavail) * uint64(st.Bsize), nil
}

// DisabledReason says why updates are unavailable, or "" when they are available.
func (m *Manager) DisabledReason() string {
	s := m.opt.Settings
	switch {
	case !s.Enabled:
		return ReasonNotEnabled
	case !s.IsPrimary:
		return ReasonSecondary
	case !s.AuthEnabled:
		return ReasonAuthDisable
	}
	if _, ok, err := ResolvePublicKey(s.PublicKey); err != nil || !ok {
		return ReasonNoKey
	}
	if s.DockerHost == "" && !m.opt.SocketExists() {
		return ReasonNoSocket
	}
	return ""
}

func (m *Manager) publicKey() (ed25519.PublicKey, error) {
	key, ok, err := ResolvePublicKey(m.opt.Settings.PublicKey)
	if err != nil || !ok {
		return nil, ErrDisabled
	}
	return key, nil
}

// ---------- view --------------------------------------------------------------------------------

// Available describes the release the last check found.
type Available struct {
	Version     string    `json:"version"`
	PublishedAt string    `json:"published_at"`
	Notes       string    `json:"notes"`
	Settings    []Setting `json:"settings"`
}

// View is what GET /api/system/update returns.
type View struct {
	Enabled        bool       `json:"enabled"`
	DisabledReason string     `json:"disabled_reason"`
	CurrentVersion string     `json:"current_version"`
	CheckedAt      string     `json:"checked_at"`
	CheckError     string     `json:"check_error"`
	Available      *Available `json:"available"`
	Plan           *Plan      `json:"plan"`
	Run            *Run       `json:"run"`
}

// View reports the current state. It is cheap and safe to poll.
func (m *Manager) View(ctx context.Context) View {
	reason := m.DisabledReason()
	v := View{Enabled: reason == "", DisabledReason: reason, CurrentVersion: m.opt.Settings.CurrentVersion}
	if reason == ReasonNotEnabled || reason == ReasonSecondary {
		return v
	}
	v.Run = m.currentRun(ctx)

	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.checkedAt.IsZero() {
		v.CheckedAt = m.checkedAt.UTC().Format(time.RFC3339)
	}
	v.CheckError = m.checkErr
	if m.latest != nil && m.newer(m.latest.Version) {
		v.Available = &Available{
			Version: m.latest.Version, PublishedAt: m.latest.PublishedAt, Notes: m.latest.Notes,
			Settings: append([]Setting{}, m.latest.Settings...),
		}
	}
	if m.plan != nil && v.Available != nil && m.plan.Version == v.Available.Version {
		v.Plan = m.liveBlockers(m.plan)
	}
	return v
}

// newer reports whether a version is ahead of the running one. A build with no version is behind every release.
func (m *Manager) newer(version string) bool {
	cur, err := ParseVersion(m.opt.Settings.CurrentVersion)
	if err != nil {
		return true
	}
	next, err := ParseVersion(version)
	return err == nil && next.Compare(cur) > 0
}

// liveBlockers copies a plan with the running-job blocker refreshed, since jobs come and go after the review.
func (m *Manager) liveBlockers(p *Plan) *Plan {
	cp := *p
	cp.Blockers = make([]Blocker, 0, len(p.Blockers)+1)
	for _, b := range p.Blockers {
		if b.Code != BlockJobRunning {
			cp.Blockers = append(cp.Blockers, b)
		}
	}
	if p.State == PlanPreparing || p.State == PlanFailed {
		return &cp
	}
	if b := m.jobBlocker(); b != nil {
		cp.Blockers = append(cp.Blockers, *b)
	}
	cp.State = PlanReady
	if len(cp.Blockers) > 0 {
		cp.State = PlanBlocked
	}
	return &cp
}

func (m *Manager) jobBlocker() *Blocker {
	n, err := m.opt.ActiveJobs()
	if err != nil {
		return &Blocker{Code: BlockJobRunning, Message: "could not tell whether a deployment is running"}
	}
	if n > 0 {
		return &Blocker{Code: BlockJobRunning, Message: fmt.Sprintf("%d deployment job(s) are running: wait for them to finish", n)}
	}
	return nil
}

// ---------- check -------------------------------------------------------------------------------

// Check fetches and verifies the release manifest now.
func (m *Manager) Check(ctx context.Context) error {
	if m.DisabledReason() != "" {
		return ErrDisabled
	}
	pub, err := m.publicKey()
	if err != nil {
		return err
	}
	data, sig, err := m.fetch(ctx)
	if err == nil {
		var mf *Manifest
		if mf, err = ParseVerified(pub, data, string(sig), m.opt.Settings.ImagePrefix); err == nil {
			m.mu.Lock()
			if m.latest == nil || m.latest.Version != mf.Version {
				m.plan = nil
			}
			m.latest, m.latestRaw, m.latestSig = mf, data, sig
			m.checkedAt, m.checkErr = m.opt.Now(), ""
			m.mu.Unlock()
			return nil
		}
	}
	m.mu.Lock()
	m.checkedAt, m.checkErr = m.opt.Now(), err.Error()
	m.mu.Unlock()
	return err
}

func (m *Manager) fetch(ctx context.Context) (data, sig []byte, err error) {
	url := m.opt.Settings.ManifestURL
	if data, err = m.get(ctx, url); err != nil {
		return nil, nil, fmt.Errorf("could not fetch the release manifest: %w", err)
	}
	if sig, err = m.get(ctx, url+".sig"); err != nil {
		return nil, nil, fmt.Errorf("could not fetch the release signature: %w", err)
	}
	return data, sig, nil
}

func (m *Manager) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.opt.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %d", url, resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxManifestBytes {
		return nil, errors.New("the answer is too large")
	}
	return b, nil
}

// Start checks on a schedule until ctx ends.
func (m *Manager) Start(ctx context.Context) {
	if m.DisabledReason() != "" {
		return
	}
	m.Recover(ctx)
	interval := m.opt.Settings.CheckInterval
	if interval <= 0 {
		interval = DefaultCheckIntervalHours * time.Hour
	}
	go func() {
		timer := time.NewTimer(initialCheckDelay)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				if err := m.Check(ctx); err != nil {
					slog.Warn("update check failed", "error", err)
				}
				timer.Reset(interval)
			}
		}
	}()
}

// ---------- review ------------------------------------------------------------------------------

// Review starts the review of a release: pull its backend image and run the plan in it. It returns at once;
// the result appears in the view.
func (m *Manager) Review(version string) error {
	if m.DisabledReason() != "" {
		return ErrDisabled
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.latest == nil || m.latest.Version != version || !m.newer(version) {
		return ErrUnknownVersion
	}
	if run := m.readRun(); run != nil && run.Active() {
		return ErrInProgress
	}
	if m.planBusy {
		return nil
	}
	m.planBusy = true
	m.plan = NewPlan(version)
	m.plan.State = PlanPreparing
	mf, raw, sig := m.latest, m.latestRaw, m.latestSig
	go m.buildPlan(mf, raw, sig)
	return nil
}

func (m *Manager) setPlan(p *Plan) {
	m.mu.Lock()
	m.plan, m.planBusy = p, false
	m.mu.Unlock()
}

func (m *Manager) buildPlan(mf *Manifest, raw, sig []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), planTimeout)
	defer cancel()
	plan := NewPlan(mf.Version)
	fail := func(err error) {
		slog.Warn("update review failed", "version", mf.Version, "error", err)
		plan.State, plan.Error = PlanFailed, err.Error()
		m.setPlan(plan)
	}

	host, err := InspectSelf(ctx, m.opt.Docker, m.opt.Hostname, m.opt.DataDir)
	if err != nil {
		fail(err)
		return
	}
	if m.opt.Settings.DockerHost != "" {
		plan.Blockers = append(plan.Blockers, Blocker{Code: BlockSocketProxy,
			Message: "this install reaches Docker through a socket proxy, which the updater cannot use yet"})
		plan.State = PlanBlocked
		m.setPlan(plan)
		return
	}
	if free, err := m.opt.FreeBytes(m.opt.DataDir); err == nil && free < minFreeBytes {
		plan.Blockers = append(plan.Blockers, Blocker{Code: BlockDiskLow,
			Message: fmt.Sprintf("only %d MiB are free on the data disk; an update needs at least %d MiB", free>>20, minFreeBytes>>20)})
	}
	releaseDir := filepath.Join(UpdatesDir(m.opt.DataDir), releaseDirName)
	if err := writeRelease(releaseDir, raw, sig); err != nil {
		fail(err)
		return
	}
	backend := mf.Images.Backend
	if _, se, err := m.opt.Docker.Run(ctx, "pull", backend); err != nil {
		fail(fmt.Errorf("could not pull the release image: %s", firstLine(se, err)))
		return
	}
	pubKey := m.opt.Settings.PublicKey
	if pubKey == "" {
		pubKey = DefaultPublicKey
	}
	args := dockerRunArgs(RunSpec{
		ReadOnly: true, Image: backend, Host: host, PublicKey: pubKey, ImagePrefix: m.opt.Settings.ImagePrefix,
		Args: []string{"update-plan", "--dir", host.ProjectDir, "--env-file", host.EnvFile, "--container", host.Name,
			"--release", host.HostPath(filepath.Join(releaseDir, releaseFile)),
			"--sig", host.HostPath(filepath.Join(releaseDir, releaseSigFile)),
			"--from-version", m.opt.Settings.CurrentVersion},
	})
	out, se, err := m.opt.Docker.Run(ctx, args...)
	if err != nil {
		fail(fmt.Errorf("the review could not run: %s", firstLine(se, err)))
		return
	}
	var reviewed Plan
	if err := json.Unmarshal([]byte(out), &reviewed); err != nil {
		fail(errors.New("the review answered something unreadable"))
		return
	}
	reviewed.Blockers = append(plan.Blockers, reviewed.Blockers...)
	reviewed.State = PlanReady
	if len(reviewed.Blockers) > 0 {
		reviewed.State = PlanBlocked
	}
	if reviewed.Blockers == nil {
		reviewed.Blockers = []Blocker{}
	}
	m.setPlan(&reviewed)
}

func writeRelease(dir string, raw, sig []byte) error {
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, releaseFile), raw, fileMode); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, releaseSigFile), sig, fileMode)
}

// ---------- apply -------------------------------------------------------------------------------

// ApplyRequest is what the browser sends to start an update.
type ApplyRequest struct {
	Version        string            `json:"version"`
	Inputs         map[string]string `json:"inputs"`
	ApproveCompose string            `json:"approve_compose"`
}

// Apply checks everything again and starts the updater container. The browser chose only the version among
// the ones the signed manifest names; nothing else it sends reaches a command line.
func (m *Manager) Apply(ctx context.Context, req ApplyRequest) (*Run, error) {
	if m.DisabledReason() != "" {
		return nil, ErrDisabled
	}
	m.runMu.Lock()
	defer m.runMu.Unlock()
	m.mu.Lock()
	plan, mf, raw, sig := m.plan, m.latest, m.latestRaw, m.latestSig
	m.mu.Unlock()
	// currentRun, not readRun: a run whose updater died must count as ended, or a retry would be refused
	if run := m.currentRun(ctx); run != nil && run.Active() {
		return nil, ErrInProgress
	}
	if mf == nil || mf.Version != req.Version || !m.newer(req.Version) {
		return nil, ErrUnknownVersion
	}
	if plan == nil || plan.Version != req.Version || plan.State == PlanPreparing || plan.State == PlanFailed {
		return nil, ErrNoPlan
	}
	plan = m.liveBlockers(plan)
	if err := checkApply(plan, mf, req); err != nil {
		return nil, err
	}

	host, err := InspectSelf(ctx, m.opt.Docker, m.opt.Hostname, m.opt.DataDir)
	if err != nil {
		return nil, err
	}
	m.cleanupUpdaters(ctx)
	id := m.opt.Now().UTC().Format(runIDLayout)
	runDir := RunDir(m.opt.DataDir, id)
	if err := writeRelease(runDir, raw, sig); err != nil {
		return nil, err
	}
	run := NewRun(id, m.opt.Settings.CurrentVersion, mf.Version)
	if err := WriteRun(StatusPath(m.opt.DataDir), run); err != nil {
		return nil, err
	}
	args := []string{"update-run", "--dir", host.ProjectDir, "--env-file", host.EnvFile, "--container", host.Name,
		"--id", id, "--yes", "--non-interactive",
		"--release", host.HostPath(filepath.Join(runDir, releaseFile)),
		"--sig", host.HostPath(filepath.Join(runDir, releaseSigFile)),
		"--status", host.HostPath(StatusPath(m.opt.DataDir)),
		"--from-version", m.opt.Settings.CurrentVersion}
	if len(req.Inputs) > 0 {
		b, _ := json.Marshal(req.Inputs)
		p := filepath.Join(runDir, inputsFile)
		if err := os.WriteFile(p, b, fileMode); err != nil {
			return nil, err
		}
		args = append(args, "--inputs", host.HostPath(p))
	}
	// the token is only meaningful, and only passed on, when a compose change was reviewed
	if plan.Compose.State == ComposeBehind {
		args = append(args, "--approve", plan.Compose.ApprovalToken)
	}
	if err := m.launch(ctx, host, mf.Images.Backend, id, args); err != nil {
		run.Finish(RunFailed, err.Error())
		_ = WriteRun(StatusPath(m.opt.DataDir), run)
		_ = os.Remove(filepath.Join(runDir, inputsFile))
		return nil, err
	}
	m.pruneRunDirs()
	return run, nil
}

// checkApply is the server side of what the UI enforces: the browser is not trusted to have obeyed it.
func checkApply(plan *Plan, mf *Manifest, req ApplyRequest) error {
	listed := map[string]Setting{}
	for _, s := range mf.Settings {
		listed[s.Key] = s
	}
	for k, v := range req.Inputs {
		s, ok := listed[k]
		if !ok || s.Kind == SettingGenerated {
			return fmt.Errorf("%w: %s is not a setting this release asks for", ErrBadInput, k)
		}
		if err := ValidateInputValue(v); err != nil {
			return fmt.Errorf("%w: %s %v", ErrBadInput, k, err)
		}
	}
	var blockers []Blocker
	for _, b := range plan.Blockers {
		if b.Code == BlockMissingRequired && missingCovered(plan, req.Inputs) {
			continue
		}
		blockers = append(blockers, b)
	}
	if len(blockers) > 0 {
		return &BlockedError{Blockers: blockers}
	}
	if plan.Compose.State == ComposeBehind && (req.ApproveCompose == "" || req.ApproveCompose != plan.Compose.ApprovalToken) {
		return ErrNotApproved
	}
	return nil
}

func missingCovered(plan *Plan, inputs map[string]string) bool {
	for _, need := range plan.Settings.RequiredMissing {
		if inputs[need.Key] == "" {
			return false
		}
	}
	return true
}

// ValidateInputValue refuses a value that could add a second line to the settings file.
func ValidateInputValue(v string) error {
	switch {
	case v == "":
		return errors.New("is empty")
	case len(v) > 4096:
		return errors.New("is too long")
	case strings.ContainsAny(v, "\n\r\x00"):
		return errors.New("contains a line break or control character")
	}
	return nil
}

func (m *Manager) launch(ctx context.Context, host *HostInfo, image, id string, args []string) error {
	ctx, cancel := context.WithTimeout(ctx, launchTimeout)
	defer cancel()
	pubKey := m.opt.Settings.PublicKey
	if pubKey == "" {
		pubKey = DefaultPublicKey
	}
	run := dockerRunArgs(RunSpec{
		Name: UpdaterName(id), Detach: true, Image: image, Host: host,
		PublicKey: pubKey, ImagePrefix: m.opt.Settings.ImagePrefix, Args: args,
	})
	if _, se, err := m.opt.Docker.Run(ctx, run...); err != nil {
		return fmt.Errorf("could not start the updater: %s", firstLine(se, err))
	}
	return nil
}

// ---------- rollback ----------------------------------------------------------------------------

// Rollback starts an updater that restores the state saved before the last update. It runs from the image this
// primary runs, so it does not depend on anything being pulled.
func (m *Manager) Rollback(ctx context.Context) (*Run, error) {
	if m.DisabledReason() != "" {
		return nil, ErrDisabled
	}
	m.runMu.Lock()
	defer m.runMu.Unlock()
	if run := m.currentRun(ctx); run != nil && run.Active() {
		return nil, ErrInProgress
	}
	host, err := InspectSelf(ctx, m.opt.Docker, m.opt.Hostname, m.opt.DataDir)
	if err != nil {
		return nil, err
	}
	m.cleanupUpdaters(ctx)
	id := m.opt.Now().UTC().Format(runIDLayout)
	// the rollback returns to the version the last run started from, when there was a last run
	back := ""
	if prev := m.readRun(); prev != nil {
		back = prev.FromVersion
	}
	run := NewRollbackRun(id, m.opt.Settings.CurrentVersion, back)
	if err := WriteRun(StatusPath(m.opt.DataDir), run); err != nil {
		return nil, err
	}
	args := []string{"update-run", "--rollback", "--dir", host.ProjectDir, "--env-file", host.EnvFile, "--container", host.Name,
		"--id", id, "--yes", "--non-interactive", "--status", host.HostPath(StatusPath(m.opt.DataDir))}
	if err := m.launch(ctx, host, host.ImageID, id, args); err != nil {
		run.Finish(RunFailed, err.Error())
		_ = WriteRun(StatusPath(m.opt.DataDir), run)
		return nil, err
	}
	return run, nil
}

// ---------- run state and recovery --------------------------------------------------------------

func (m *Manager) readRun() *Run {
	run, err := ReadRun(StatusPath(m.opt.DataDir))
	if err != nil {
		slog.Warn("could not read the update status file", "error", err)
		return nil
	}
	return run
}

// currentRun returns the last run. A run that says it is active while its updater container is gone was
// interrupted (a crash, a reboot, a kill): it is marked so, so the UI offers rollback or retry instead of waiting forever.
func (m *Manager) currentRun(ctx context.Context) *Run {
	run := m.readRun()
	if run == nil || !run.Active() {
		return run
	}
	if m.updaterAlive(ctx, run.ID) {
		return run
	}
	run.FailStep("the updater stopped before it finished")
	run.Finish(RunInterrupted, "the updater stopped before it finished. Roll back to the version you had, or review and retry")
	if err := WriteRun(StatusPath(m.opt.DataDir), run); err != nil {
		slog.Warn("could not record the interrupted update", "error", err)
	}
	return run
}

// updaterAlive reports whether the run's updater container is still running. When Docker cannot say, the run is
// assumed alive: marking a live run interrupted would be worse than waiting.
func (m *Manager) updaterAlive(ctx context.Context, id string) bool {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, se, err := m.opt.Docker.Run(ctx, "inspect", "--format", "{{.State.Running}}", UpdaterName(id))
	if err != nil {
		return !strings.Contains(strings.ToLower(se), "no such")
	}
	return strings.TrimSpace(out) == "true"
}

// Recover marks an unfinished run interrupted at startup, before anyone asks.
func (m *Manager) Recover(ctx context.Context) {
	if m.DisabledReason() != "" {
		return
	}
	m.currentRun(ctx)
}

// cleanupUpdaters removes finished updater containers, so names and disk do not pile up.
func (m *Manager) cleanupUpdaters(ctx context.Context) {
	out, _, err := m.opt.Docker.Run(ctx, "ps", "-a", "--filter", "label="+updaterLabel, "--filter", "status=exited",
		"--filter", "status=created", "--format", "{{.Names}}")
	if err != nil {
		return
	}
	for _, name := range strings.Fields(out) {
		if strings.HasPrefix(name, updaterNamePref) {
			_, _, _ = m.opt.Docker.Run(ctx, "rm", name)
		}
	}
}

// pruneRunDirs keeps the newest few run folders (release copies), removing the rest.
func (m *Manager) pruneRunDirs() {
	entries, err := os.ReadDir(UpdatesDir(m.opt.DataDir))
	if err != nil {
		return
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != releaseDirName {
			ids = append(ids, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))
	for i, id := range ids {
		if i >= keepRunDirs {
			_ = os.RemoveAll(filepath.Join(UpdatesDir(m.opt.DataDir), id))
		}
	}
}
