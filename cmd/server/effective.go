package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/secrets"
)

// setting is one line of the effective configuration: what a start will actually use, and where
// that value came from. Secrets are never included, only where they come from.
type setting struct{ name, value, source string }

func authMode(cfg *config.Config) string {
	switch {
	case cfg.Auth.Enabled:
		return "github login"
	case cfg.Security.CloudflareAccess.Enabled():
		return "cloudflare access (" + cfg.Security.CloudflareAccess.TeamDomain + ")"
	default:
		return "none"
	}
}

func envOrDefault(name string) string {
	if os.Getenv(name) != "" {
		return "environment (" + name + ")"
	}
	return "default"
}

// secretsKeySource says where the encryption key comes from without reading its value
func secretsKeySource(cfg *config.Config) string {
	if cfg.Security.SettingsEncryptionKey != "" {
		return "environment (SETTINGS_ENCRYPTION_KEY)"
	}
	if _, ok := secrets.FindKey("", filepath.Dir(cfg.DatabasePath)); ok {
		return "saved file " + filepath.Join(filepath.Dir(cfg.DatabasePath), constants.SecretsKeyFileName)
	}
	return "none yet (created on first start if needed)"
}

// effectiveSettings lists everything that decides how the next start behaves
func effectiveSettings(cfg *config.Config) []setting {
	role := "primary"
	if !cfg.Node.IsPrimary {
		role = "secondary"
	}
	hostDir := cfg.Security.HostAppsDir
	hostSrc := "environment (HOST_APPS_DIR)"
	if hostDir == "" {
		hostDir, hostSrc = "(detected at startup)", "detected from this container's mounts"
	}
	return []setting{
		{"security mode", cfg.Security.Mode, cfg.Sources["security_mode"]},
		{"encrypt secrets at rest", fmt.Sprint(cfg.Security.EncryptSecretsAtRest), envOrDefault("ENCRYPT_SECRETS_AT_REST")},
		{"encryption key", "", secretsKeySource(cfg)},
		{"authentication", authMode(cfg), "AUTH_ENABLED / CF_ACCESS_*"},
		{"allowed github users", fmt.Sprint(len(cfg.Auth.GitHub.AllowedUsers)), "GITHUB_ALLOWED_USERS"},
		{"secure cookies", fmt.Sprint(cfg.Auth.SecureCookie), envOrDefault("AUTH_SECURE_COOKIE") + " / derived from AUTH_BASE_URL"},
		{"session hours", fmt.Sprint(cfg.Security.SessionHours), envOrDefault("AUTH_SESSION_HOURS")},
		{"node role", role, "NODE_IS_PRIMARY"},
		{"node id", cfg.Node.ID, cfg.Sources["node_id"]},
		{"node api key", "", cfg.Sources["node_api_key"]},
		{"registration token", "", orNone(cfg.Sources["registration_token"])},
		{"apps directory", cfg.AppsDir, envOrDefault("APPS_DIR")},
		{"host apps directory", hostDir, hostSrc},
		{"extra allowed volume paths", fmt.Sprint(len(cfg.Security.AllowedVolumePaths)), envOrDefault("ALLOWED_VOLUME_PATHS")},
		{"database", cfg.DatabasePath, envOrDefault("DATABASE_PATH")},
	}
}

func orNone(s string) string {
	if s == "" {
		return "environment (REGISTRATION_TOKEN) or not used on this node"
	}
	return s
}

// logEffectiveConfig writes one startup record showing exactly what this start is running with,
// so any restart can be understood from its log alone.
func logEffectiveConfig(cfg *config.Config) {
	attrs := []any{}
	for _, s := range effectiveSettings(cfg) {
		v := s.value
		if v == "" {
			v = "(secret, not shown)"
		}
		attrs = append(attrs, strings.ReplaceAll(s.name, " ", "_"), v+"  <- "+s.source)
	}
	slog.Info("effective configuration", attrs...)
}
