package docker

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// containerMount is the subset of `docker inspect` mount data needed to translate paths
type containerMount struct {
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
}

// hostPathFromMounts maps a path inside this container to the host path that backs it.
func hostPathFromMounts(mountsJSON []byte, containerPath string) (string, bool) {
	var mounts []containerMount
	if err := json.Unmarshal(mountsJSON, &mounts); err != nil {
		return "", false
	}
	best := ""
	var bestSrc string
	for _, m := range mounts {
		dest := filepath.Clean(m.Destination)
		if containerPath == dest || strings.HasPrefix(containerPath, dest+"/") {
			if len(dest) > len(best) {
				best, bestSrc = dest, m.Source
			}
		}
	}
	if best == "" {
		return "", false
	}
	return filepath.Join(bestSrc, strings.TrimPrefix(containerPath, best)), true
}

// DetectHostAppsDir returns the host path behind appsDir. Inside a container it inspects its own
// mounts through the Docker API; when it is not containerised (or inspection fails) the path is
// returned as-is because container and host views coincide.
func DetectHostAppsDir(appsDir string, exec CommandExecutor) string {
	abs, err := filepath.Abs(appsDir)
	if err != nil {
		abs = appsDir
	}
	if _, err := os.Stat("/.dockerenv"); err != nil {
		return abs
	}
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return abs
	}
	out, err := exec.ExecuteCommand(DockerCommand, "inspect", "--format", "{{json .Mounts}}", hostname)
	if err != nil {
		slog.Warn("could not inspect own container to find the host apps directory; set HOST_APPS_DIR", "error", err)
		return abs
	}
	host, ok := hostPathFromMounts(out, abs)
	if !ok {
		slog.Warn("apps directory is not a bind mount of this container; set HOST_APPS_DIR", "apps_dir", abs)
		return abs
	}
	return host
}

// FormatHostDirMismatch explains why a compose bind mount only works when the host path is known
func FormatHostDirMismatch(containerPath, hostPath string) string {
	return fmt.Sprintf("container path %s is host path %s", containerPath, hostPath)
}
