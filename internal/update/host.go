package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Labels Docker Compose puts on a container, which say where the install lives on the host.
const (
	labelWorkingDir  = "com.docker.compose.project.working_dir"
	labelConfigFiles = "com.docker.compose.project.config_files"
	labelEnvFile     = "com.docker.compose.project.environment_file"
	defaultSocket    = "/var/run/docker.sock"
	updaterLabel     = "selfhostly.updater"
	updaterNamePref  = "selfhostly-updater-"
	ctlBinary        = "selfhostlyctl"
)

// HostInfo is what the primary learns about itself from Docker: where the install is on the host, so an updater
// container can mount the same paths and run docker compose against the real files.
type HostInfo struct {
	Name        string   // this container's name
	ImageID     string   // the image this container runs
	User        string   // uid:gid the container runs as
	ProjectDir  string   // the compose working directory on the host
	ConfigFiles []string // compose files on the host
	EnvFile     string   // settings file on the host
	DataDir     string   // host path behind the data directory
	Socket      string   // host path of the Docker socket
	dataMount   string   // the same directory as this process sees it
}

type inspectDoc struct {
	Name   string `json:"Name"`
	Image  string `json:"Image"`
	Config struct {
		User   string            `json:"User"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	Mounts []struct {
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
	} `json:"Mounts"`
}

// InspectSelf asks Docker about this container. hostname is the container id Docker gave it, dataMount is the
// data directory as this process sees it.
func InspectSelf(ctx context.Context, d Docker, hostname, dataMount string) (*HostInfo, error) {
	if hostname == "" {
		return nil, errors.New("this process is not running in a container")
	}
	out, se, err := d.Run(ctx, "inspect", hostname)
	if err != nil {
		return nil, fmt.Errorf("could not inspect this container: %s", firstLine(se, err))
	}
	var docs []inspectDoc
	if err := json.Unmarshal([]byte(out), &docs); err != nil || len(docs) != 1 {
		return nil, errors.New("could not read what Docker says about this container")
	}
	return hostInfoFromInspect(docs[0], dataMount)
}

func hostInfoFromInspect(doc inspectDoc, dataMount string) (*HostInfo, error) {
	h := &HostInfo{
		Name: strings.TrimPrefix(doc.Name, "/"), ImageID: doc.Image, User: doc.Config.User,
		ProjectDir: doc.Config.Labels[labelWorkingDir], Socket: defaultSocket, dataMount: filepath.Clean(dataMount),
	}
	if h.ProjectDir == "" {
		return nil, errors.New("this container was not started with docker compose, so the install folder is unknown")
	}
	for _, f := range strings.Split(doc.Config.Labels[labelConfigFiles], ",") {
		if f = strings.TrimSpace(f); f != "" {
			h.ConfigFiles = append(h.ConfigFiles, f)
		}
	}
	h.EnvFile = doc.Config.Labels[labelEnvFile]
	if h.EnvFile == "" {
		h.EnvFile = filepath.Join(h.ProjectDir, ".env")
	}
	for _, m := range doc.Mounts {
		switch filepath.Clean(m.Destination) {
		case h.dataMount:
			h.DataDir = m.Source
		case defaultSocket:
			h.Socket = m.Source
		}
	}
	if h.DataDir == "" {
		return nil, fmt.Errorf("the data directory %s is not a bind mount of this container", h.dataMount)
	}
	return h, nil
}

// HostPath maps a path under the data directory, as this process sees it, to the same file on the host.
func (h *HostInfo) HostPath(p string) string {
	rel, err := filepath.Rel(h.dataMount, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return p
	}
	return filepath.Join(h.DataDir, rel)
}

// mountDirs are the host directories the updater needs, each mounted at its own path so compose's relative
// paths and the labels on running containers stay valid inside it.
func (h *HostInfo) mountDirs() []string {
	set := map[string]bool{h.ProjectDir: true, h.DataDir: true, filepath.Dir(h.EnvFile): true}
	for _, f := range h.ConfigFiles {
		set[filepath.Dir(f)] = true
	}
	dirs := make([]string, 0, len(set))
	for d := range set {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return dirs
}

// RunSpec describes one container the primary starts to do update work.
type RunSpec struct {
	Name        string // empty for a throwaway container
	Detach      bool
	ReadOnly    bool // mount the install read-only (the review changes nothing)
	Image       string
	Host        *HostInfo
	PublicKey   string
	ImagePrefix string
	Args        []string // the selfhostlyctl arguments
}

// dockerRunArgs builds the docker run command line for an update container.
func dockerRunArgs(s RunSpec) []string {
	args := []string{"run"}
	if s.Detach {
		args = append(args, "-d")
	} else {
		args = append(args, "--rm")
	}
	if s.Name != "" {
		args = append(args, "--name", s.Name)
	}
	args = append(args, "--label", updaterLabel+"=1", "--restart", "no")
	if s.Host.User != "" {
		args = append(args, "--user", s.Host.User)
	}
	args = append(args, "-v", s.Host.Socket+":"+defaultSocket)
	suffix := ""
	if s.ReadOnly {
		suffix = ":ro"
	}
	for _, d := range s.Host.mountDirs() {
		args = append(args, "-v", d+":"+d+suffix)
	}
	args = append(args, "-w", s.Host.ProjectDir,
		"-e", "HOME=/tmp", "-e", "DOCKER_CONFIG=/tmp/.docker", "-e", "SELFHOSTLYCTL_NO_UPDATE_CHECK=1",
		"-e", "UPDATE_PUBLIC_KEY="+s.PublicKey, "-e", "UPDATE_IMAGE_REPO_PREFIX="+s.ImagePrefix,
		"--entrypoint", ctlBinary, s.Image)
	return append(args, s.Args...)
}

// UpdaterName is the container name for a run.
func UpdaterName(runID string) string { return updaterNamePref + runID }

func firstLine(s string, err error) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return err.Error()
	}
	line, _, _ := strings.Cut(s, "\n")
	return line
}
