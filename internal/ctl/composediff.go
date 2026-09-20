package ctl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	secretKey = regexp.MustCompile(`(?i)(SECRET|TOKEN|PASSWORD|PASSWD|CREDENTIAL|API_?KEY)`)
	userPair  = regexp.MustCompile(`^(\d+):(\d+)$`)
)

const (
	probeValue = "__SFH_PROBE__"
	probeUID   = "4242"
	probeGID   = "4343"
)

type diffOpts struct {
	newFile, envFile, apply string
	showSecrets             bool
}

// errNeedsAttention gives compose-diff its exit status 1: something would be lost or needs a decision
var errNeedsAttention = fmt.Errorf("something would be lost")

func (a *App) composeDiffCmd() *cobra.Command {
	var o diffOpts
	cmd := &cobra.Command{
		Use:   "compose-diff OLD.yml [NEW.yml]",
		Short: "Show what replacing your compose file would lose, and carry it over through .env",
		Long: "People edit their production compose file by hand (a hard-coded volume path, a user id, a log setting).\n" +
			"Replacing the file drops those edits silently. This renders both files with the same settings and\n" +
			"reports every difference in what would run. Values the new file reads from the env file can be\n" +
			"written there with --apply, which only adds missing lines and never overwrites one.\n" +
			"Secrets are masked. Exit status: 0 nothing lost, 1 needs attention.",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 2 {
				o.newFile = args[1]
			}
			// rendered with your settings file only when --env-file is given, so the report is about the files
			if c.Flags().Changed("env-file") {
				o.envFile = a.envPath(a.EnvFile)
			}
			return a.composeDiff(c.Context(), args[0], o)
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.apply, "apply", "", "add the carry-over lines that are missing to this env file")
	f.BoolVar(&o.showSecrets, "show-secrets", false, "do not mask secret-looking values")
	return cmd
}

type svcMap = map[string]map[string]any

func (a *App) render(ctx context.Context, file, envFile string, extra map[string]string) (svcMap, string, error) {
	cwd, _ := os.Getwd()
	out, se, err := a.Run.OutputEnv(ctx, extra, "docker", "compose", "--project-directory", cwd, "--env-file", envFile,
		"-f", file, "config", "--format", "json")
	if err != nil {
		return nil, se, err
	}
	var doc struct {
		Services svcMap `json:"services"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		return nil, "", err
	}
	return doc.Services, "", nil
}

func mask(key string, v any, show bool) any {
	if show || !secretKey.MatchString(key) {
		return v
	}
	if s, ok := v.(string); ok && s != "" {
		return "***"
	}
	return v
}

// repr formats a value the way people expect to read it: 'text' for strings
func repr(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case string:
		if strings.Contains(x, "'") {
			return fmt.Sprintf("%q", x)
		}
		return "'" + x + "'"
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}

func envOf(s map[string]any) map[string]any {
	m, _ := s["environment"].(map[string]any)
	return m
}

func portSet(s map[string]any) map[[4]string]bool {
	out := map[[4]string]bool{}
	ps, _ := s["ports"].([]any)
	for _, p := range ps {
		m, _ := p.(map[string]any)
		proto := fmt.Sprint(orDefault(m["protocol"], "tcp"))
		out[[4]string{str(m["host_ip"]), str(m["published"]), str(m["target"]), proto}] = true
	}
	return out
}

func volumeSet(s map[string]any) map[[4]string]bool {
	out := map[[4]string]bool{}
	vs, _ := s["volumes"].([]any)
	for _, v := range vs {
		m, _ := v.(map[string]any)
		out[[4]string{str(m["type"]), str(m["source"]), str(m["target"]), fmt.Sprint(m["read_only"] == true)}] = true
	}
	return out
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func orDefault(v, d any) any {
	if v == nil || v == "" {
		return d
	}
	return v
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func setDiff(a, b map[[4]string]bool) [][4]string {
	var out [][4]string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.Join(out[i][:], "|") < strings.Join(out[j][:], "|") })
	return out
}

func (a *App) composeDiff(ctx context.Context, oldFile string, o diffOpts) error {
	newFile, cleanup, err := a.newComposeFile(o.newFile)
	if err != nil {
		return err
	}
	defer cleanup()
	envFile := o.envFile
	if envFile == "" {
		tmp, err := os.CreateTemp("", "*.env")
		if err != nil {
			return err
		}
		_ = tmp.Close()
		defer os.Remove(tmp.Name())
		envFile = tmp.Name()
	}

	old, se, err := a.render(ctx, oldFile, envFile, nil)
	if err != nil {
		return fmt.Errorf("could not render %s:\n%s", oldFile, strings.TrimSpace(se))
	}
	newSvcs, se, err := a.render(ctx, newFile, envFile, nil)
	if err != nil {
		return fmt.Errorf("could not render %s:\n%s", newFile, strings.TrimSpace(se))
	}

	// Which of the new file's settings can be set from .env? Render it again with every environment key
	// that differs set to a probe value: a key whose rendered value becomes the probe is read from the
	// environment, so a line in .env is enough to carry it over. Only differing keys are probed:
	// setting every key would also change variables used elsewhere (APPS_DIR appears in a volume path).
	diffKeys := map[string]bool{}
	for name, oldSvc := range old {
		for k, v := range envOf(oldSvc) {
			if !reflect.DeepEqual(envOf(newSvcs[name])[k], v) {
				diffKeys[k] = true
			}
		}
	}
	probe := func(keys []string) svcMap {
		env := map[string]string{"APP_UID": probeUID, "DOCKER_GID": probeGID}
		for _, k := range keys {
			env[k] = probeValue
		}
		s, _, err := a.render(ctx, newFile, envFile, env)
		if err != nil {
			return nil
		}
		return s
	}
	keys := sortedKeys(diffKeys)
	probed := probe(keys)
	if probed == nil {
		// a probed key collided with something else in the file: judge each key on its own
		probed = svcMap{}
		for name := range newSvcs {
			probed[name] = map[string]any{"environment": map[string]any{}}
		}
		for _, k := range keys {
			for name, s := range probe([]string{k}) {
				if v, ok := envOf(s)[k]; ok {
					envOf(probed[name])[k] = v
				}
			}
		}
		for name, s := range probe(nil) {
			probed[name]["user"] = s["user"]
		}
	}

	var lost, info []string
	carry := map[string]map[string]string{}
	addCarry := func(k, svc, v string) {
		if carry[k] == nil {
			carry[k] = map[string]string{}
		}
		carry[k][svc] = v
	}

	for _, name := range sortedKeys(old) {
		o2, n := old[name], newSvcs[name]
		if n == nil {
			lost = append(lost, fmt.Sprintf("service %s exists in the old file and not in the new one (a custom service?)", repr(name)))
			continue
		}
		oe, ne, pe := envOf(o2), envOf(n), envOf(probed[name])
		for _, k := range sortedKeys(oe) {
			ov := oe[k]
			if nv, ok := ne[k]; ok && reflect.DeepEqual(nv, ov) {
				continue
			}
			_, inNew := ne[k]
			if (ov == nil || ov == "") && !inNew {
				info = append(info, fmt.Sprintf("%s: %s was empty in the old file (no effect unless your .env sets it)", name, k))
				continue
			}
			shown := repr(mask(k, ov, o.showSecrets))
			switch {
			case pe[k] == probeValue:
				addCarry(k, name, str(ov))
				info = append(info, fmt.Sprintf("%s: %s was %s, the new file reads it from .env (carry-over line below)", name, k, shown))
			case inNew:
				lost = append(lost, fmt.Sprintf("%s.%s: old %s, new %s, and the new file does not read it from .env",
					name, k, shown, repr(mask(k, ne[k], o.showSecrets))))
			default:
				lost = append(lost, fmt.Sprintf("%s.%s = %s is not in the new file at all: add `%s: ${%s:-}` under services.%s.environment",
					name, k, shown, k, k, name))
			}
		}

		if !reflect.DeepEqual(o2["user"], n["user"]) {
			ou := str(o2["user"])
			if m := userPair.FindStringSubmatch(ou); m != nil && probed[name]["user"] == probeUID+":"+probeGID {
				addCarry("APP_UID", name, m[1])
				addCarry("DOCKER_GID", name, m[2])
				info = append(info, fmt.Sprintf("%s: user was %s, the new file reads it from APP_UID / DOCKER_GID", name, ou))
			} else {
				lost = append(lost, fmt.Sprintf("%s.user: old %s, new %s", name, repr(o2["user"]), repr(n["user"])))
			}
		}
		for _, p := range setDiff(portSet(o2), portSet(n)) {
			ip := ""
			if p[0] != "" {
				ip = p[0] + ":"
			}
			lost = append(lost, fmt.Sprintf("%s.ports: %s%s -> %s/%s is in the old file and not in the new one", name, ip, p[1], p[2], p[3]))
		}
		for _, v := range setDiff(volumeSet(o2), volumeSet(n)) {
			ro := ""
			if v[3] == "true" {
				ro = " read-only"
			}
			src := firstNonEmpty(v[1], "(anonymous)")
			lost = append(lost, fmt.Sprintf("%s.volumes: %s -> %s%s (%s mount) is in the old file and not in the new one", name, src, v[2], ro, v[0]))
		}
		for _, field := range []string{"command", "entrypoint", "network_mode", "privileged", "cap_add", "devices", "extra_hosts", "labels", "restart"} {
			if ov := o2[field]; ov != nil && ov != "" && ov != false && !reflect.DeepEqual(ov, n[field]) {
				lost = append(lost, fmt.Sprintf("%s.%s: old %s, new %s", name, field, repr(ov), repr(n[field])))
			}
		}
	}
	for _, name := range sortedKeys(newSvcs) {
		if _, ok := old[name]; !ok {
			info = append(info, "new service in the new file: "+name)
		}
	}

	existing := map[string]string{}
	for _, p := range []string{o.apply, o.envFile} {
		if p != "" {
			for k, v := range (EnvFile{p}).All() {
				existing[k] = v
			}
		}
	}

	type kv struct{ k, v string }
	var lines []kv
	var conflicts []string
	for _, k := range sortedKeys(carry) {
		per := carry[k]
		vals := map[string]bool{}
		for _, v := range per {
			vals[v] = true
		}
		if len(vals) != 1 {
			var parts []string
			for _, s := range sortedKeys(per) {
				parts = append(parts, fmt.Sprintf("%s=%s", s, repr(per[s])))
			}
			conflicts = append(conflicts, fmt.Sprintf("%s has different values in different services: %s", k, strings.Join(parts, ", ")))
			continue
		}
		var v string
		for v = range vals {
		}
		if cur, ok := existing[k]; ok && cur != v {
			conflicts = append(conflicts, fmt.Sprintf("%s: your env file has %s but the old compose file ran with %s. Decide which is right; nothing was changed",
				k, repr(mask(k, cur, o.showSecrets)), repr(mask(k, v, o.showSecrets))))
			continue
		}
		lines = append(lines, kv{k, v})
	}

	a.say("Comparing %s (old) with %s (new)\n", oldFile, newFile)
	if len(lost) > 0 {
		a.say("WOULD BE LOST if you replace the file (needs your attention):")
		for _, x := range lost {
			a.say("  - %s", x)
		}
		a.say("")
	}
	if len(conflicts) > 0 {
		a.say("NEEDS A DECISION:")
		for _, x := range conflicts {
			a.say("  - %s", x)
		}
		a.say("")
	}
	var pending []kv
	if len(lines) > 0 {
		a.say("CARRY OVER through the env file (the new file already reads these):")
		for _, l := range lines {
			status := ""
			if _, ok := existing[l.k]; ok {
				status = "  (already in your env file with this value)"
			} else {
				pending = append(pending, l)
			}
			shown := l.v
			if !o.showSecrets && secretKey.MatchString(l.k) && l.v != "" {
				shown = "***"
			}
			a.say("  %s=%s%s", l.k, shown, status)
		}
		a.say("")
	}
	if len(info) > 0 && o.apply == "" {
		a.say("For information:")
		for _, x := range info {
			a.say("  - %s", x)
		}
		a.say("")
	}

	if o.apply != "" && len(pending) > 0 {
		env := EnvFile{o.apply}
		var names []string
		for _, p := range pending {
			if err := env.Set(p.k, p.v); err != nil {
				return err
			}
			names = append(names, p.k)
		}
		a.say("Added %d line(s) to %s: %s", len(pending), o.apply, strings.Join(names, ", "))
		pending = nil
	}

	if len(lost) == 0 && len(conflicts) == 0 && len(pending) == 0 {
		if len(lines) == 0 {
			a.say("Nothing would be lost: the new file runs the same as the old one.")
		} else {
			a.say("Nothing would be lost once the lines above are in the env file.")
		}
		return nil
	}
	if len(lost) == 0 && len(conflicts) == 0 && o.apply == "" {
		a.say("Run again with --apply <env file> to add the carry-over lines.")
	}
	return errNeedsAttention
}
