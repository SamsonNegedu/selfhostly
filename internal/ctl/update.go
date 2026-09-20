package ctl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	selfhostly "github.com/selfhostly"
	"github.com/selfhostly/internal/update"
	"github.com/spf13/cobra"
)

const (
	envUpdatePublicKey   = "UPDATE_PUBLIC_KEY"
	envUpdateImagePrefix = "UPDATE_IMAGE_REPO_PREFIX"
	settingVersion       = "SELFHOSTLY_VERSION"
	updateHealthSeconds  = 120
	maxInputBytes        = 4096
	generatedSecretBytes = 32
	approvalTokenBytes   = 16
)

// imageSettings are the settings file keys the compose file reads image references from.
var imageSettings = map[string]string{
	update.ServiceBackend:  "BACKEND_IMAGE",
	update.ServiceGateway:  "GATEWAY_IMAGE",
	update.ServiceFrontend: "FRONTEND_IMAGE",
}

// updateOpts are the flags of the two update commands. They are run by the updater container the UI starts, not by people.
type updateOpts struct {
	id, release, sig, inputs, approve, fromVersion, status string
	rollback                                               bool
}

func (a *App) updatePlanCmd() *cobra.Command {
	var o updateOpts
	cmd := &cobra.Command{
		Use:    "update-plan",
		Short:  "Review a signed release against this install and print the plan as JSON",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			out := a.Out
			quiet := *a
			quiet.Out, quiet.Err = io.Discard, io.Discard
			plan, err := quiet.updatePlan(c.Context(), o)
			if err != nil {
				return err
			}
			return json.NewEncoder(out).Encode(plan)
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.release, "release", "", "release.json")
	f.StringVar(&o.sig, "sig", "", "release.json.sig")
	f.StringVar(&o.fromVersion, "from-version", "", "the running version")
	return cmd
}

func (a *App) updateRunCmd() *cobra.Command {
	var o updateOpts
	cmd := &cobra.Command{
		Use:    "update-run",
		Short:  "Apply a signed release, or roll back, and report progress to the status file",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE:   func(c *cobra.Command, _ []string) error { return a.updateRun(c.Context(), o) },
	}
	f := cmd.Flags()
	f.StringVar(&o.id, "id", "", "run id")
	f.StringVar(&o.release, "release", "", "release.json")
	f.StringVar(&o.sig, "sig", "", "release.json.sig")
	f.StringVar(&o.inputs, "inputs", "", "JSON file of setting values (deleted after it is read)")
	f.StringVar(&o.approve, "approve", "", "the compose approval token from the plan")
	f.StringVar(&o.fromVersion, "from-version", "", "the running version")
	f.StringVar(&o.status, "status", "", "status file to keep up to date")
	f.BoolVar(&o.rollback, "rollback", false, "restore the last rollback point instead of updating")
	return cmd
}

// loadRelease reads a release the primary saved and verifies it again: the updater does not trust what the primary checked.
func loadRelease(releasePath, sigPath string) (*update.Manifest, error) {
	pub, ok, err := update.ResolvePublicKey(os.Getenv(envUpdatePublicKey))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("no update signing key is configured")
	}
	data, err := os.ReadFile(releasePath)
	if err != nil {
		return nil, err
	}
	sig, err := os.ReadFile(sigPath)
	if err != nil {
		return nil, err
	}
	prefix := firstNonEmpty(os.Getenv(envUpdateImagePrefix), update.DefaultImagePrefix)
	return update.ParseVerified(pub, data, string(sig), prefix)
}

// ---------- plan --------------------------------------------------------------------------------

func (a *App) updatePlan(ctx context.Context, o updateOpts) (*update.Plan, error) {
	m, err := loadRelease(o.release, o.sig)
	if err != nil {
		return nil, err
	}
	plan := update.NewPlan(m.Version)
	if b := checkFromVersion(o.fromVersion, m); b != nil {
		plan.Blockers = append(plan.Blockers, *b)
	}
	a.planSettings(m, plan)
	cp, blockers, err := a.planCompose(ctx, m)
	if err != nil {
		return nil, err
	}
	plan.Blockers = append(plan.Blockers, blockers...)
	if cp != nil {
		plan.Compose = cp.ComposePlan
	}
	if len(plan.Blockers) > 0 {
		plan.State = update.PlanBlocked
	} else {
		plan.State = update.PlanReady
	}
	return plan, nil
}

// checkFromVersion blocks an update from a version older than the release supports. An unknown running version
// (a development build) is not blocked: there is nothing to compare.
func checkFromVersion(from string, m *update.Manifest) *update.Blocker {
	if m.MinFromVersion == "" {
		return nil
	}
	cur, err := update.ParseVersion(from)
	if err != nil {
		return nil
	}
	min, _ := update.ParseVersion(m.MinFromVersion)
	if cur.Compare(min) >= 0 {
		return nil
	}
	return &update.Blocker{Code: update.BlockUnsupportedFrom,
		Message: fmt.Sprintf("this release updates from %s or newer, and this install runs %s. Update to an intermediate release first", min, cur)}
}

// planSettings sorts the release's new settings by what the update will do about each one.
func (a *App) planSettings(m *update.Manifest, plan *update.Plan) {
	env := EnvFile{a.envPath(a.EnvFile)}.All()
	for _, s := range m.Settings {
		if env[s.Key] != "" {
			continue
		}
		switch s.Kind {
		case update.SettingRequired:
			plan.Settings.RequiredMissing = append(plan.Settings.RequiredMissing,
				update.SettingNeed{Key: s.Key, Description: s.Description, Secret: s.Secret})
		case update.SettingGenerated:
			plan.Settings.Generated = append(plan.Settings.Generated, s.Key)
		default:
			plan.Settings.Optional = append(plan.Settings.Optional, s.Key)
		}
	}
	if len(plan.Settings.RequiredMissing) > 0 {
		keys := make([]string, len(plan.Settings.RequiredMissing))
		for i, r := range plan.Settings.RequiredMissing {
			keys[i] = r.Key
		}
		plan.Blockers = append(plan.Blockers, update.Blocker{Code: update.BlockMissingRequired,
			Message: "this release needs values for: " + strings.Join(keys, ", ")})
	}
}

// composeChange is the review of the compose file plus what applying it needs
type composeChange struct {
	update.ComposePlan
	livePath string
	newBody  []byte
	envPath  string
}

// planCompose compares the compose file the install runs with the one this release ships. It never changes anything.
func (a *App) planCompose(ctx context.Context, m *update.Manifest) (*composeChange, []update.Blocker, error) {
	var o upgradeOpts
	if err := a.resolveCompose(ctx, &o); err != nil {
		return nil, nil, err
	}
	if len(o.compose) == 0 {
		return nil, []update.Blocker{{Code: update.BlockNoRunningInstall, Message: "no running Selfhostly install was found to update"}}, nil
	}
	if len(o.compose) != 1 {
		return nil, []update.Blocker{{Code: update.BlockComposeCustom,
			Message: "this install starts from several compose files, so the update cannot swap them safely: update by hand"}}, nil
	}
	newBody, err := selfhostly.Files.ReadFile(primaryCompose)
	if err != nil {
		return nil, nil, err
	}
	newSum := sha256.Sum256(newBody)
	if hex.EncodeToString(newSum[:]) != m.Compose.SHA256 {
		return nil, []update.Blocker{{Code: update.BlockReleaseMismatch,
			Message: "the compose file inside the release image is not the one the release was signed with"}}, nil
	}
	live := o.compose[0]
	liveBody, err := os.ReadFile(live)
	if err != nil {
		return nil, nil, err
	}
	cc := &composeChange{livePath: live, newBody: newBody, envPath: a.envPath(a.EnvFile)}
	if string(liveBody) == string(newBody) {
		cc.State = update.ComposeCurrent
		return cc, nil, nil
	}

	tmpName, err := writeTemp(newBody)
	if err != nil {
		return nil, nil, err
	}
	defer os.Remove(tmpName)
	var report strings.Builder
	quiet := *a
	quiet.Out = &report
	var res diffResult
	if err := quiet.composeDiff(ctx, live, diffOpts{newFile: tmpName, envFile: cc.envPath, result: &res}); err != nil && !errors.Is(err, errNeedsAttention) {
		return nil, nil, err
	}
	// the report names the release file by its temporary path, which is different on every run
	cc.Report = strings.ReplaceAll(strings.TrimSpace(report.String()), tmpName, "the compose file of this release")
	cc.Diff = unifiedDiff(live, "docker-compose.prod.yml (release)", string(liveBody), string(newBody))
	// The approval covers exactly the two files being swapped. It must not depend on the report text, which the
	// review and the updater each produce on their own.
	tokenSum := sha256.Sum256([]byte(hex.EncodeToString(newSum[:]) + "|" + string(liveBody)))
	if res.Lost > 0 || res.Conflicts > 0 {
		cc.State = update.ComposeCustomized
		return cc, []update.Blocker{{Code: update.BlockComposeCustom,
			Message: "your compose file has edits the release file would drop: merge them by hand (selfhostlyctl compose-diff shows which)"}}, nil
	}
	cc.State = update.ComposeBehind
	cc.ApprovalToken = hex.EncodeToString(tokenSum[:approvalTokenBytes])
	return cc, nil, nil
}

// ---------- run ---------------------------------------------------------------------------------

// runReporter keeps the status file current. A failed write only costs the UI some detail, so it is logged, not fatal.
type runReporter struct {
	a    *App
	path string
	run  *update.Run
}

func (r *runReporter) save() {
	if r.path == "" {
		return
	}
	if err := update.WriteRun(r.path, r.run); err != nil {
		r.a.say("could not write the status file: %v", err)
	}
}

func (r *runReporter) phase(name, message string) {
	r.run.Advance(name, message)
	r.save()
}

func (r *runReporter) warn(message string) {
	r.run.Warnings = append(r.run.Warnings, message)
	r.save()
}

func (r *runReporter) finish(state, message string) {
	r.run.Finish(state, message)
	r.save()
}

func (a *App) updateRun(ctx context.Context, o updateOpts) error {
	run, err := update.ReadRun(o.status)
	if err != nil {
		return err
	}
	if run == nil {
		run = update.NewRun(firstNonEmpty(o.id, a.now()), o.fromVersion, "")
	}
	rep := &runReporter{a: a, path: o.status, run: run}
	a.Yes, a.NonInteractive = true, true

	if o.rollback {
		return a.updateRollback(ctx, rep)
	}

	rep.phase(update.PhaseVerify, "")
	m, err := loadRelease(o.release, o.sig)
	if err != nil {
		rep.run.FailStep(err.Error())
		rep.finish(update.RunFailed, err.Error())
		return err
	}
	if run.ToVersion != "" && run.ToVersion != m.Version {
		err := fmt.Errorf("the release says %s but this run was started for %s", m.Version, run.ToVersion)
		rep.run.FailStep(err.Error())
		rep.finish(update.RunFailed, err.Error())
		return err
	}
	run.ToVersion = m.Version
	if b := checkFromVersion(o.fromVersion, m); b != nil {
		rep.run.FailStep(b.Message)
		rep.finish(update.RunFailed, b.Message)
		return errors.New(b.Message)
	}
	inputs, err := readInputs(o.inputs)
	if err != nil {
		rep.run.FailStep(err.Error())
		rep.finish(update.RunFailed, err.Error())
		return err
	}

	uo := upgradeOpts{
		withFrontend: true, strict: true, healthTimeout: updateHealthSeconds,
		progress: rep.phase,
		configure: func(ctx context.Context, u *upgrader) error {
			return a.configureUpdate(ctx, u, m, inputs, o.approve)
		},
	}
	err = a.upgrade(ctx, uo)
	var failure *upgradeFailure
	switch {
	case err == nil:
		rep.finish(update.RunSucceeded, fmt.Sprintf("updated to %s", m.Version))
		return nil
	case errors.As(err, &failure) && failure.rolledBack:
		rep.run.FailStep(failure.msg)
		rep.finish(update.RunRolledBack, failure.msg)
	case errors.As(err, &failure):
		msg := failure.msg + ". The rollback did not finish: " + fmt.Sprint(failure.rollbackErr)
		rep.run.FailStep(failure.msg)
		rep.finish(update.RunFailed, msg)
	default:
		rep.run.FailStep(err.Error())
		rep.finish(update.RunFailed, err.Error()+" (nothing was changed)")
	}
	return err
}

func (a *App) updateRollback(ctx context.Context, rep *runReporter) error {
	err := a.rollbackCmd(ctx, upgradeOpts{healthTimeout: updateHealthSeconds, withFrontend: true, progress: rep.phase})
	if err != nil {
		rep.run.FailStep(err.Error())
		rep.finish(update.RunFailed, err.Error())
		return err
	}
	rep.finish(update.RunRolledBack, "restored the state saved before the last update")
	return nil
}

// readInputs loads the values the operator typed in the UI and deletes the file: they can include secrets.
func readInputs(path string) (map[string]string, error) {
	if path == "" {
		return map[string]string{}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	_ = os.Remove(path)
	in := map[string]string{}
	if err := json.Unmarshal(b, &in); err != nil {
		return nil, fmt.Errorf("the inputs file is not a JSON object of strings: %w", err)
	}
	return in, nil
}

// configureUpdate runs after the rollback point exists and before anything is pulled. Everything it changes
// (the settings file and the compose file) is restored if the update fails.
func (a *App) configureUpdate(ctx context.Context, u *upgrader, m *update.Manifest, inputs map[string]string, approve string) error {
	cc, blockers, err := a.planCompose(ctx, m)
	if err != nil {
		return err
	}
	if len(blockers) > 0 {
		return errors.New(blockers[0].Message)
	}
	if cc.State == update.ComposeBehind {
		if approve == "" || approve != cc.ApprovalToken {
			return errors.New("the compose change was not approved, or the compose file changed since it was reviewed: review the update again")
		}
		// carry hand-set values into the settings file first, so the new file keeps running the same way
		tmp, err := writeTemp(cc.newBody)
		if err != nil {
			return err
		}
		defer os.Remove(tmp)
		quiet := *a
		quiet.Out = io.Discard
		if err := quiet.composeDiff(ctx, cc.livePath, diffOpts{newFile: tmp, envFile: cc.envPath, apply: cc.envPath}); err != nil {
			return fmt.Errorf("could not carry your compose settings over: %w", err)
		}
		if err := writeFileKeepingMode(cc.livePath, cc.newBody); err != nil {
			return fmt.Errorf("could not write the new compose file: %w", err)
		}
		a.say("  compose file replaced: %s", cc.livePath)
	}

	if err := applySettings(u.env, m, inputs); err != nil {
		return err
	}
	for _, svc := range update.Services {
		if err := u.env.Set(imageSettings[svc], m.Images.For(svc)); err != nil {
			return err
		}
	}
	return u.env.Set(settingVersion, m.Version)
}

func writeTemp(body []byte) (string, error) {
	f, err := os.CreateTemp("", "docker-compose.release.*.yml")
	if err != nil {
		return "", err
	}
	_, werr := f.Write(body)
	if cerr := f.Close(); werr != nil || cerr != nil {
		_ = os.Remove(f.Name())
		return "", errors.Join(werr, cerr)
	}
	return f.Name(), nil
}

// applySettings writes what the release needs. Only keys the signed release lists are accepted from the browser,
// so the UI cannot be used to set arbitrary variables.
func applySettings(env EnvFile, m *update.Manifest, inputs map[string]string) error {
	listed := map[string]update.Setting{}
	for _, s := range m.Settings {
		listed[s.Key] = s
	}
	for key, value := range inputs {
		s, ok := listed[key]
		if !ok || s.Kind == update.SettingGenerated {
			return fmt.Errorf("%s is not a setting this release asks for", key)
		}
		if err := checkSettingValue(key, value); err != nil {
			return err
		}
		if err := env.Set(key, value); err != nil {
			return err
		}
	}
	have := env.All()
	var missing []string
	for _, s := range m.Settings {
		if have[s.Key] != "" {
			continue
		}
		switch s.Kind {
		case update.SettingRequired:
			missing = append(missing, s.Key)
		case update.SettingGenerated:
			if _, err := env.SetIfMissing(s.Key, randHex(generatedSecretBytes)); err != nil {
				return err
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing values for: %s", strings.Join(missing, ", "))
	}
	return nil
}

// checkSettingValue refuses anything that could add a second line to the settings file or break out of it
func checkSettingValue(key, value string) error {
	if value == "" {
		return fmt.Errorf("%s is empty", key)
	}
	if len(value) > maxInputBytes {
		return fmt.Errorf("%s is too long", key)
	}
	if strings.ContainsAny(value, "\n\r\x00") {
		return fmt.Errorf("%s contains a line break or control character", key)
	}
	return nil
}
