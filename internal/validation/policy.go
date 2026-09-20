package validation

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// PolicyResult separates violations that are never acceptable (Hard) from ones that depend on the
// operator's allow-list and are only blocked when enforcing (Soft).
type PolicyResult struct {
	Hard []string
	Soft []string
}

func (r *PolicyResult) hard(format string, a ...interface{}) {
	r.Hard = append(r.Hard, fmt.Sprintf(format, a...))
}

func (r *PolicyResult) soft(format string, a ...interface{}) {
	r.Soft = append(r.Soft, fmt.Sprintf(format, a...))
}

// pathsDeniedWithSubtree cannot be mounted, nor can anything beneath them.
var pathsDeniedWithSubtree = []string{
	"/var/run/docker.sock", "/run/docker.sock", "/var/run/docker", "/run/docker",
	"/root", "/etc", "/boot", "/sys", "/proc", "/dev", "/host",
	"/var/lib/docker", "/var/lib/kubelet", "/var/lib/rancher",
}

// systemRoots cannot be mounted themselves (they contain the paths above or the system itself)
// but sub-directories are judged on their own.
var systemRoots = []string{"/usr", "/bin", "/sbin", "/lib", "/lib64"}

func isWithin(path, root string) bool {
	return path == root || strings.HasPrefix(path, strings.TrimSuffix(root, "/")+"/")
}

// classifyHostPath returns a reason when an absolute, cleaned host path must never be mounted.
func classifyHostPath(p string) string {
	for _, d := range pathsDeniedWithSubtree {
		if isWithin(p, d) {
			return fmt.Sprintf("%s is inside %s", p, d)
		}
		// p is an ancestor of d: mounting it exposes d (for example /run exposes /run/docker.sock)
		if isWithin(d, p) {
			return fmt.Sprintf("%s contains %s", p, d)
		}
	}
	for _, r := range systemRoots {
		if p == r {
			return fmt.Sprintf("%s is a system directory", p)
		}
	}
	return ""
}

// dangerousServiceKeys are rejected whenever present with a non-default value.
var dangerousCapabilities = map[string]bool{
	"BPF": true, "PERFMON": true, "SYSLOG": true, "LINUX_IMMUTABLE": true, "SYS_PACCT": true,
}

type policyCtx struct {
	cfg        *SecurityConfig
	res        *PolicyResult
	appDirHost string
}

// RejectFileReads fails fast on constructs that make the compose loader read arbitrary files.
// It must run before parsing.
func RejectFileReads(content []byte) error {
	var root map[string]interface{}
	if err := yaml.Unmarshal(content, &root); err != nil || root == nil {
		return nil
	}
	if _, ok := root["include"]; ok {
		return fmt.Errorf("top-level 'include' is not allowed (it reads files outside the app)")
	}
	if services, ok := root["services"].(map[string]interface{}); ok {
		for name, v := range services {
			if svc, ok := v.(map[string]interface{}); ok {
				if _, ok := svc["extends"]; ok {
					return fmt.Errorf("service %q: 'extends' is not allowed (it reads files outside the app)", name)
				}
			}
		}
	}
	return nil
}

// CheckComposePolicy inspects the raw compose YAML, which is what `docker compose` will execute.
// The typed struct used elsewhere covers only a subset of fields, so this walks the generic tree
// and normalises short and long syntax before applying the rules.
func CheckComposePolicy(content []byte, cfg *SecurityConfig) PolicyResult {
	var res PolicyResult
	var root map[string]interface{}
	if err := yaml.Unmarshal(content, &root); err != nil || root == nil {
		return res // syntax errors are reported by the parser with better messages
	}
	if cfg == nil {
		cfg = defaultSecurityConfig
	}
	c := &policyCtx{cfg: cfg, res: &res}
	if cfg.HostAppsDir != "" && cfg.AppName != "" {
		c.appDirHost = filepath.Join(cfg.HostAppsDir, cfg.AppName)
	}

	if _, ok := root["include"]; ok {
		res.hard("top-level 'include' is not allowed (it reads files outside the app)")
	}

	if services, ok := root["services"].(map[string]interface{}); ok {
		names := make([]string, 0, len(services))
		for n := range services {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if svc, ok := services[n].(map[string]interface{}); ok {
				c.checkService(n, svc)
			}
		}
	}
	c.checkTopLevelVolumes(root["volumes"])
	c.checkFileRefs("secrets", root["secrets"])
	c.checkFileRefs("configs", root["configs"])
	return res
}

func (c *policyCtx) checkService(name string, svc map[string]interface{}) {
	r := c.res
	prefix := fmt.Sprintf("service %q", name)

	if _, ok := svc["extends"]; ok {
		r.hard("%s: 'extends' is not allowed (it reads files outside the app)", prefix)
	}
	if truthy(svc["privileged"]) {
		r.hard("%s: privileged mode is not allowed", prefix)
	}
	if v, ok := svc["devices"].([]interface{}); ok && len(v) > 0 {
		r.hard("%s: device access is not allowed", prefix)
	}
	if v, ok := svc["device_cgroup_rules"].([]interface{}); ok && len(v) > 0 {
		r.hard("%s: device_cgroup_rules is not allowed", prefix)
	}
	if v := str(svc["cgroup"]); v != "" && v != "private" {
		r.hard("%s: cgroup %q is not allowed", prefix, v)
	}
	if v := str(svc["cgroup_parent"]); v != "" {
		r.hard("%s: cgroup_parent is not allowed", prefix)
	}
	if v := str(svc["userns_mode"]); v != "" {
		r.hard("%s: userns_mode %q is not allowed", prefix, v)
	}
	if v := str(svc["uts"]); v == "host" {
		r.hard("%s: uts 'host' is not allowed", prefix)
	}
	if v := str(svc["runtime"]); v != "" && v != "runc" {
		r.hard("%s: runtime %q is not allowed", prefix, v)
	}
	for _, key := range []string{"network_mode", "pid", "ipc"} {
		v := str(svc[key])
		if v == "host" {
			r.hard("%s: %s %q is not allowed", prefix, key, v)
		} else if strings.HasPrefix(v, "container:") {
			// Joining another container's namespaces is common (VPN sidecars) but can reach
			// containers this app does not own, so it is an enforce-mode decision.
			r.soft("%s: %s %q joins a container outside this app", prefix, key, v)
		}
	}
	for _, g := range asStrings(svc["group_add"]) {
		if g == "0" || g == "root" || g == "docker" {
			r.hard("%s: group_add %q is not allowed (grants access to root-owned resources)", prefix, g)
		}
	}
	for _, v := range asStrings(svc["volumes_from"]) {
		if strings.HasPrefix(v, "container:") {
			r.soft("%s: volumes_from %q reads volumes of a container outside this app", prefix, v)
		}
	}
	for _, cp := range asStrings(svc["cap_add"]) {
		up := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(cp)), "CAP_")
		if dangerousCapabilities[up] {
			r.hard("%s: capability %q is not allowed", prefix, cp)
		}
	}
	for _, opt := range asStrings(svc["security_opt"]) {
		o := strings.ToLower(strings.TrimSpace(opt))
		if strings.HasPrefix(o, "systempaths=unconfined") || strings.HasPrefix(o, "systempaths:unconfined") {
			r.hard("%s: security_opt %q is not allowed", prefix, opt)
		}
	}

	c.checkVolumes(prefix, svc["volumes"])
	c.checkEnvFile(prefix, svc["env_file"])
	c.checkBuild(prefix, svc["build"])
}

func (c *policyCtx) checkVolumes(prefix string, raw interface{}) {
	items, _ := raw.([]interface{})
	for _, it := range items {
		switch v := it.(type) {
		case string:
			parts := strings.Split(v, ":")
			if len(parts) < 2 {
				continue // anonymous volume
			}
			src := strings.TrimSpace(parts[0])
			if looksLikePath(src) {
				c.checkBindSource(prefix+" volume", src)
			}
		case map[string]interface{}:
			if str(v["type"]) == "bind" {
				c.checkBindSource(prefix+" volume", str(v["source"]))
			}
		}
	}
}

// looksLikePath distinguishes a bind source from a named volume in short syntax.
func looksLikePath(src string) bool {
	return strings.HasPrefix(src, "/") || strings.HasPrefix(src, ".") ||
		strings.HasPrefix(src, "~") || strings.Contains(src, "$")
}

func (c *policyCtx) checkTopLevelVolumes(raw interface{}) {
	vols, _ := raw.(map[string]interface{})
	for name, v := range vols {
		def, _ := v.(map[string]interface{})
		opts, _ := def["driver_opts"].(map[string]interface{})
		if opts == nil {
			continue
		}
		device := str(opts["device"])
		o := str(opts["o"])
		if device != "" && (strings.Contains(o, "bind") || str(opts["type"]) == "none") {
			c.checkBindSource(fmt.Sprintf("volume %q driver_opts.device", name), device)
		}
	}
}

func (c *policyCtx) checkFileRefs(kind string, raw interface{}) {
	entries, _ := raw.(map[string]interface{})
	for name, v := range entries {
		def, _ := v.(map[string]interface{})
		if f := str(def["file"]); f != "" {
			c.checkBindSource(fmt.Sprintf("%s %q file", kind, name), f)
		}
	}
}

func (c *policyCtx) checkEnvFile(prefix string, raw interface{}) {
	var paths []string
	switch v := raw.(type) {
	case string:
		paths = []string{v}
	case []interface{}:
		for _, it := range v {
			switch e := it.(type) {
			case string:
				paths = append(paths, e)
			case map[string]interface{}:
				paths = append(paths, str(e["path"]))
			}
		}
	}
	for _, p := range paths {
		c.requireInsideAppDir(prefix+" env_file", p)
	}
}

func (c *policyCtx) checkBuild(prefix string, raw interface{}) {
	switch v := raw.(type) {
	case string:
		c.checkBuildContext(prefix, v)
	case map[string]interface{}:
		c.checkBuildContext(prefix, str(v["context"]))
		if extra, ok := v["additional_contexts"].(map[string]interface{}); ok {
			for _, e := range extra {
				c.checkBuildContext(prefix, str(e))
			}
		}
	}
}

func (c *policyCtx) checkBuildContext(prefix, ctx string) {
	if ctx == "" || strings.Contains(ctx, "://") || strings.HasPrefix(ctx, "git@") ||
		strings.HasPrefix(ctx, "docker-image:") || strings.HasPrefix(ctx, "service:") {
		return // remote contexts do not read host files
	}
	c.requireInsideAppDir(prefix+" build context", ctx)
}

// requireInsideAppDir rejects absolute paths and paths that climb out of the app directory.
func (c *policyCtx) requireInsideAppDir(what, p string) {
	if p == "" {
		return
	}
	if strings.Contains(p, "$") {
		c.res.soft("%s %q contains a variable and cannot be verified", what, p)
		return
	}
	if filepath.IsAbs(p) || strings.HasPrefix(p, "~") {
		c.res.hard("%s %q must be relative to the app directory", what, p)
		return
	}
	clean := filepath.Clean(p)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		c.res.hard("%s %q escapes the app directory", what, p)
	}
}

// checkBindSource applies the host-path rules to one bind mount source.
func (c *policyCtx) checkBindSource(what, src string) {
	src = strings.TrimSpace(src)
	if src == "" {
		return
	}
	if strings.Contains(src, "$") {
		c.res.soft("%s %q contains a variable, so its host path cannot be verified: use a literal path", what, src)
		return
	}
	if strings.HasPrefix(src, "~") {
		c.res.soft("%s %q uses ~, so its host path cannot be verified: use a literal path", what, src)
		return
	}

	var abs string
	inAppDir := false
	if filepath.IsAbs(src) {
		abs = filepath.Clean(src)
	} else {
		clean := filepath.Clean(src)
		if clean == ".." || strings.HasPrefix(clean, "../") {
			c.res.hard("%s %q escapes the app directory", what, src)
			return
		}
		inAppDir = true
		if c.appDirHost != "" {
			abs = filepath.Join(c.appDirHost, clean)
		}
	}

	if abs != "" {
		if reason := classifyHostPath(abs); reason != "" {
			c.res.hard("%s %q is not allowed: %s", what, src, reason)
			return
		}
		if c.symlinkEscapes(abs) {
			c.res.hard("%s %q resolves through a symlink that leaves the apps directory", what, src)
			return
		}
		if c.appDirHost != "" && isWithin(abs, c.appDirHost) {
			inAppDir = true
		}
	}
	if inAppDir {
		return
	}
	for _, root := range c.cfg.AllowedVolumePaths {
		root = filepath.Clean(strings.TrimSpace(root))
		if root != "" && root != "." && isWithin(abs, root) {
			return
		}
	}
	c.res.soft("%s %q is outside the app directory and ALLOWED_VOLUME_PATHS", what, src)
}

// symlinkEscapes reports whether a path under the host apps directory resolves, on this
// filesystem, to somewhere outside it. Paths this process cannot see are not judged.
func (c *policyCtx) symlinkEscapes(hostPath string) bool {
	if c.cfg.HostAppsDir == "" || c.cfg.AppsDir == "" || !isWithin(hostPath, c.cfg.HostAppsDir) {
		return false
	}
	rel := strings.TrimPrefix(hostPath, strings.TrimSuffix(c.cfg.HostAppsDir, "/"))
	local := filepath.Join(c.cfg.AppsDir, rel)
	resolved, err := filepath.EvalSymlinks(local)
	if err != nil {
		return false
	}
	base, err := filepath.EvalSymlinks(c.cfg.AppsDir)
	if err != nil {
		return false
	}
	return !isWithin(resolved, base)
}

// Apply turns a PolicyResult into an error according to the enforcement mode, logging what is
// tolerated in warn mode so the operator can review it before switching to enforce.
func (r PolicyResult) Apply(enforce bool, app string) error {
	if len(r.Hard) > 0 {
		return fmt.Errorf("%s", strings.Join(r.Hard, "; "))
	}
	if len(r.Soft) == 0 {
		return nil
	}
	if enforce {
		return fmt.Errorf("%s", strings.Join(r.Soft, "; "))
	}
	for _, s := range r.Soft {
		slog.Warn("compose policy: would be blocked in enforce mode", "app", app, "violation", s)
	}
	return nil
}

func truthy(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(t, "true") || t == "1"
	}
	return false
}

func str(v interface{}) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func asStrings(v interface{}) []string {
	items, _ := v.([]interface{})
	out := make([]string, 0, len(items))
	for _, it := range items {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// AuditComposeFile reads a compose file and reports every policy violation, for the dry-run audit
// of already-deployed apps.
func AuditComposeFile(path string, cfg *SecurityConfig) (PolicyResult, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return PolicyResult{}, err
	}
	return CheckComposePolicy(b, cfg), nil
}
