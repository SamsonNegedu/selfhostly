package update

// Plan states.
const (
	PlanPreparing = "preparing"
	PlanReady     = "ready"
	PlanBlocked   = "blocked"
	PlanFailed    = "failed"
)

// Compose states: how the running compose file relates to the one the release ships.
const (
	ComposeCurrent    = "current"    // identical: nothing to swap
	ComposeBehind     = "behind"     // swapping loses nothing; carry-over lines go to .env; needs approval
	ComposeCustomized = "customized" // swapping would lose hand edits: merge by hand
)

// Blocker codes.
const (
	BlockJobRunning       = "job_running"
	BlockComposeCustom    = "compose_customized"
	BlockMissingRequired  = "missing_required"
	BlockUnsupportedFrom  = "unsupported_from"
	BlockSocketProxy      = "socket_proxy"
	BlockDiskLow          = "disk_low"
	BlockDoctorFailed     = "doctor_failed"
	BlockNotAdmin         = "not_admin"
	BlockInProgress       = "update_in_progress"
	BlockReleaseMismatch  = "release_mismatch"
	BlockNoRunningInstall = "no_running_install"
)

// Blocker is a reason an update cannot start yet.
type Blocker struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ComposePlan is what the review found about the compose file.
type ComposePlan struct {
	State         string `json:"state"`
	Diff          string `json:"diff"`
	Report        string `json:"report"`
	ApprovalToken string `json:"approval_token"`
}

// SettingNeed is a setting the operator has to provide.
type SettingNeed struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Secret      bool   `json:"secret"`
}

// SettingsPlan sorts the release's new settings by what the update does about them.
type SettingsPlan struct {
	RequiredMissing []SettingNeed `json:"required_missing"`
	Generated       []string      `json:"generated"`
	Optional        []string      `json:"optional"`
}

// Plan is the review of one update: what blocks it, what changes and what it needs.
type Plan struct {
	Version  string       `json:"version"`
	State    string       `json:"state"`
	Error    string       `json:"error"`
	Blockers []Blocker    `json:"blockers"`
	Compose  ComposePlan  `json:"compose"`
	Settings SettingsPlan `json:"settings"`
}

// NewPlan returns a plan with empty lists, so it serialises as [] and not null.
func NewPlan(version string) *Plan {
	return &Plan{
		Version: version, Blockers: []Blocker{},
		Settings: SettingsPlan{RequiredMissing: []SettingNeed{}, Generated: []string{}, Optional: []string{}},
	}
}
