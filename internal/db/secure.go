package db

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/secrets"
)

const stateSessionsValidAfter = "sessions_valid_after"

// SetSecretBox installs the box used to protect node keys, per-app tunnel tokens and provider tokens. Call it before
// any node or settings access.
func (db *DB) SetSecretBox(b *secrets.Box) {
	db.box = b
}

func (db *DB) seal(v string) (string, error) {
	return db.box.Seal(v)
}

func (db *DB) open(v string) (string, error) {
	return db.box.Open(v)
}

// SecretColumn identifies a stored field that holds a credential
type SecretColumn struct{ Table, Column string }

// SecretColumns lists every stored field that holds a credential
var SecretColumns = []SecretColumn{
	{"nodes", "api_key"},
	{"apps", "tunnel_token"},
	{"cloudflare_tunnels", "tunnel_token"},
	{"settings", "cloudflare_api_token"},
	{"settings", "tunnel_provider_config"},
}

// HasSealedSecrets reports whether any stored credential is encrypted
func (db *DB) HasSealedSecrets() bool {
	for _, col := range SecretColumns {
		var n int
		err := db.QueryRow(`SELECT COUNT(*) FROM `+col.Table+` WHERE `+col.Column+` LIKE ?`,
			constants.SecretsCipherPrefix+"%").Scan(&n)
		if err == nil && n > 0 {
			return true
		}
	}
	return false
}

// MigrateSecrets makes stored secrets match the configured state, in both directions: with
// encrypted writes on it encrypts plaintext values, with them off it decrypts ciphertext. Treating
// the flag as the desired state of every field keeps the database uniform, so rolling back to a
// build that cannot read ciphertext is "turn the flag off, restart once, roll back".
//
// It is idempotent. A value that cannot be decrypted (lost or wrong key) is left untouched and
// reported; it never blocks startup.
func (db *DB) MigrateSecrets() error {
	encrypt := db.box.EncryptsWrites()
	changed, unreadable := 0, 0

	for _, col := range SecretColumns {
		rows, err := db.Query(`SELECT id, ` + col.Column + ` FROM ` + col.Table +
			` WHERE ` + col.Column + ` IS NOT NULL AND ` + col.Column + ` != ''`)
		if err != nil {
			return err
		}
		type item struct{ id, val string }
		var todo []item
		for rows.Next() {
			var it item
			if err := rows.Scan(&it.id, &it.val); err != nil {
				rows.Close()
				return err
			}
			if secrets.IsSealed(it.val) != encrypt {
				todo = append(todo, it)
			}
		}
		rows.Close()

		for _, it := range todo {
			var out string
			if encrypt {
				out, err = db.seal(it.val)
			} else {
				out, err = db.open(it.val)
			}
			if err != nil {
				unreadable++
				slog.Error("stored secret cannot be converted; leaving it as is",
					"table", col.Table, "column", col.Column, "id", it.id, "error", err)
				continue
			}
			if _, err := db.Exec(`UPDATE `+col.Table+` SET `+col.Column+` = ? WHERE id = ?`, out, it.id); err != nil {
				return err
			}
			changed++
		}
	}
	if changed > 0 {
		slog.Info("stored secrets updated to match ENCRYPT_SECRETS_AT_REST", "values", changed, "encrypted", encrypt)
	}
	if unreadable > 0 {
		slog.Error("some stored secrets could not be read: check SETTINGS_ENCRYPTION_KEY and the secrets.key file", "values", unreadable)
	}
	return nil
}

// ===========================
// Session revocation
// ===========================

// SessionsValidAfter returns the time before which issued tokens are no longer accepted
func (db *DB) SessionsValidAfter() time.Time {
	var v string
	if err := db.QueryRow(`SELECT value FROM security_state WHERE key = ?`, stateSessionsValidAfter).Scan(&v); err != nil {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return time.Time{}
	}
	return t
}

// RevokeSessions invalidates every token issued before now
func (db *DB) RevokeSessions() (time.Time, error) {
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO security_state (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		stateSessionsValidAfter, now.Format(time.RFC3339Nano), now)
	return now, err
}

// ===========================
// Join tokens
// ===========================

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// CreateJoinToken issues a single-use token. Only its hash is stored, so it cannot be recovered
// later: the caller must show the returned value once.
func (db *DB) CreateJoinToken(ttl time.Duration) (string, time.Time, error) {
	raw := make([]byte, constants.JoinTokenByteSize)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	plain := constants.JoinTokenPrefix + base64.RawURLEncoding.EncodeToString(raw)
	now := time.Now().UTC()
	expires := now.Add(ttl)
	if _, err := db.Exec(`DELETE FROM join_tokens WHERE expires_at < ? OR used_at IS NOT NULL AND used_at < ?`,
		now, now.Add(-24*time.Hour)); err != nil {
		slog.Warn("could not prune join tokens", "error", err)
	}
	_, err := db.Exec(`INSERT INTO join_tokens (id, token_hash, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		uuid.New().String(), hashToken(plain), now, expires)
	return plain, expires, err
}

// ConsumeJoinToken atomically marks a valid, unused token as used. It reports false for unknown,
// expired or already-used tokens.
func (db *DB) ConsumeJoinToken(plain, nodeID string) (bool, error) {
	if len(plain) < len(constants.JoinTokenPrefix) || plain[:len(constants.JoinTokenPrefix)] != constants.JoinTokenPrefix {
		return false, nil
	}
	now := time.Now().UTC()
	res, err := db.Exec(`UPDATE join_tokens SET used_at = ?, used_by_node_id = ?
		WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?`, now, nodeID, hashToken(plain), now)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// ===========================
// Audit log
// ===========================

// AuditEntry records one state-changing API request
type AuditEntry struct {
	ID         int64     `json:"id"`
	Time       time.Time `json:"time"`
	Actor      string    `json:"actor"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Status     int       `json:"status"`
	RemoteAddr string    `json:"remote_addr"`
	// What was done and to what, for example "app.stop" on the app "nextcloud". Empty for requests
	// that are not in the action table. Never holds request bodies or values.
	Action     string `json:"action"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	TargetName string `json:"target_name"`
}

// InsertAudit appends an audit record
func (db *DB) InsertAudit(e AuditEntry) error {
	_, err := db.Exec(`INSERT INTO audit_log (ts, actor, method, path, status, remote_addr, action, target_type, target_id, target_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Time.UTC(), e.Actor, e.Method, e.Path, e.Status, e.RemoteAddr, e.Action, e.TargetType, e.TargetID, e.TargetName)
	return err
}

// ListAudit returns the most recent audit records, newest first
func (db *DB) ListAudit(limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := db.Query(`SELECT id, ts, actor, method, path, status, COALESCE(remote_addr, ''), action, target_type, target_id, target_name
		FROM audit_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// Never nil, so an empty log is [] in JSON and not null.
	out := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Time, &e.Actor, &e.Method, &e.Path, &e.Status, &e.RemoteAddr, &e.Action, &e.TargetType, &e.TargetID, &e.TargetName); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// PruneAudit deletes records older than the retention window
func (db *DB) PruneAudit(retention time.Duration) error {
	_, err := db.Exec(`DELETE FROM audit_log WHERE ts < ?`, time.Now().UTC().Add(-retention))
	return err
}

// openNodeKey decrypts node.APIKey in place, passing through a prior query error
func (db *DB) openNodeKey(node *Node, queryErr error) (*Node, error) {
	if queryErr != nil {
		return node, queryErr
	}
	plain, err := db.open(node.APIKey)
	if err != nil {
		return nil, fmt.Errorf("node %s: %w", node.ID, err)
	}
	node.APIKey = plain
	return node, nil
}

// openAppToken decrypts app.TunnelToken in place. Like node keys, an unreadable value is an error and not an
// empty token, because saving the app afterwards would overwrite the stored value with nothing.
func (db *DB) openAppToken(app *App) error {
	plain, err := db.open(app.TunnelToken)
	if err != nil {
		return fmt.Errorf("app %s tunnel token: %w", app.ID, err)
	}
	app.TunnelToken = plain
	return nil
}

// openTunnelToken decrypts tunnel.TunnelToken in place, for the same reason as openAppToken
func (db *DB) openTunnelToken(tunnel *CloudflareTunnel) error {
	plain, err := db.open(tunnel.TunnelToken)
	if err != nil {
		return fmt.Errorf("tunnel %s token: %w", tunnel.TunnelID, err)
	}
	tunnel.TunnelToken = plain
	return nil
}

// PrimaryNodeID returns the ID of the primary node without touching its (possibly encrypted) key,
// so it works even when the encryption key is unavailable.
func (db *DB) PrimaryNodeID() (string, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM nodes WHERE is_primary = 1 LIMIT 1`).Scan(&id)
	return id, err
}
