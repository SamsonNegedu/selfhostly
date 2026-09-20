package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/selfhostly/internal/constants"
)

// versionedMigration is an additive, idempotent schema change. Versioned migrations never drop or
// rewrite existing data, so an older binary keeps working against a newer schema (rollback safe).
type versionedMigration struct {
	version int
	name    string
	stmts   []string
}

var versionedMigrations = []versionedMigration{
	{
		version: 1,
		name:    "security state, join tokens, audit log",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS security_state (
				key TEXT PRIMARY KEY,
				value TEXT NOT NULL,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE IF NOT EXISTS join_tokens (
				id TEXT PRIMARY KEY,
				token_hash TEXT NOT NULL UNIQUE,
				created_at DATETIME NOT NULL,
				expires_at DATETIME NOT NULL,
				used_at DATETIME,
				used_by_node_id TEXT
			)`,
			`CREATE TABLE IF NOT EXISTS audit_log (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				ts DATETIME NOT NULL,
				actor TEXT NOT NULL,
				method TEXT NOT NULL,
				path TEXT NOT NULL,
				status INTEGER NOT NULL,
				remote_addr TEXT
			)`,
			`CREATE INDEX IF NOT EXISTS idx_audit_log_ts ON audit_log(ts DESC)`,
		},
	},
	{
		version: 2,
		name:    "audit log: action and target",
		stmts: []string{
			`ALTER TABLE audit_log ADD COLUMN action TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE audit_log ADD COLUMN target_type TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE audit_log ADD COLUMN target_id TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE audit_log ADD COLUMN target_name TEXT NOT NULL DEFAULT ''`,
		},
	},
}

// LatestSchemaVersion is the highest versioned migration this binary knows
func LatestSchemaVersion() int {
	return versionedMigrations[len(versionedMigrations)-1].version
}

func (db *DB) runVersionedMigrations() error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at DATETIME NOT NULL
	)`); err != nil {
		return err
	}

	applied := map[int]bool{}
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	rows.Close()

	var pending []versionedMigration
	for _, m := range versionedMigrations {
		if !applied[m.version] {
			pending = append(pending, m)
		}
	}
	if len(pending) == 0 {
		return nil
	}

	if db.hasUserData() {
		backup, err := db.Backup("pre-migration")
		if err != nil {
			return fmt.Errorf("refusing to migrate without a backup: %w", err)
		}
		slog.Info("database backed up before migration", "backup", backup, "pending", len(pending))
	}

	for _, m := range pending {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		for _, stmt := range m.stmts {
			if _, err := tx.Exec(stmt); err != nil {
				// Migrations must be safe to run again, and SQLite has no ADD COLUMN IF NOT EXISTS.
				if strings.HasPrefix(stmt, "ALTER TABLE") && strings.Contains(err.Error(), "duplicate column name") {
					continue
				}
				_ = tx.Rollback()
				return fmt.Errorf("migration %d (%s): %w", m.version, m.name, err)
			}
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			m.version, m.name, time.Now()); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		slog.Info("applied database migration", "version", m.version, "name", m.name)
	}
	return nil
}

// hasUserData reports whether the database already holds nodes or apps worth protecting
func (db *DB) hasUserData() bool {
	for _, table := range []string{"nodes", "apps"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err == nil && n > 0 {
			return true
		}
	}
	return false
}

// Backup writes a consistent copy of the database next to it and prunes old copies. VACUUM INTO
// produces a valid standalone file even while the database is in WAL mode and in use.
func (db *DB) Backup(label string) (string, error) {
	if db.dbPath == "" || db.dbPath == ":memory:" {
		return "", fmt.Errorf("in-memory database cannot be backed up")
	}
	dest := fmt.Sprintf("%s.bak-%s-%s", db.dbPath, label, time.Now().UTC().Format("20060102T150405Z"))
	if _, err := db.Exec(`VACUUM INTO ?`, dest); err != nil {
		return "", err
	}
	if err := os.Chmod(dest, constants.SecretFileMode); err != nil {
		slog.Warn("could not restrict backup permissions", "backup", dest, "error", err)
	}
	db.pruneBackups()
	return dest, nil
}

func (db *DB) pruneBackups() {
	matches, err := filepath.Glob(db.dbPath + ".bak-*")
	if err != nil || len(matches) <= constants.DBBackupsToKeep {
		return
	}
	sort.Strings(matches) // timestamps sort lexically
	for _, old := range matches[:len(matches)-constants.DBBackupsToKeep] {
		if strings.HasSuffix(old, "-wal") || strings.HasSuffix(old, "-shm") {
			continue
		}
		if err := os.Remove(old); err != nil {
			slog.Warn("could not remove old backup", "backup", old, "error", err)
		}
	}
}

// SchemaVersion returns the highest applied versioned migration (0 when none)
func (db *DB) SchemaVersion() int {
	var v sql.NullInt64
	if err := db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&v); err != nil || !v.Valid {
		return 0
	}
	return int(v.Int64)
}
