package service

import (
	"os"
	"path/filepath"

	"github.com/selfhostly/internal/buildinfo"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/update"
)

// NewUpdateService wires the update manager to this install: its version, its data directory, its job table
// and whether Docker is reached through a proxy.
func NewUpdateService(database *db.DB, cfg *config.Config) domain.UpdateService {
	dataDir, err := filepath.Abs(filepath.Dir(cfg.DatabasePath))
	if err != nil {
		dataDir = filepath.Dir(cfg.DatabasePath)
	}
	hostname, _ := os.Hostname()
	return update.NewManager(update.Options{
		Settings: update.Settings{
			Enabled:        cfg.Updates.Enabled,
			ManifestURL:    cfg.Updates.ManifestURL,
			PublicKey:      cfg.Updates.PublicKey,
			ImagePrefix:    cfg.Updates.ImagePrefix,
			CheckInterval:  cfg.Updates.CheckInterval,
			CurrentVersion: buildinfo.Version,
			IsPrimary:      cfg.Node.IsPrimary,
			AuthEnabled:    cfg.Auth.Enabled,
			DockerHost:     os.Getenv("DOCKER_HOST"),
		},
		ActiveJobs: database.CountActiveJobs,
		DataDir:    dataDir,
		Hostname:   hostname,
	})
}
