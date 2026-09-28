package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/selfhostly/internal/constants"
)

// ErrDeployHookNotFound is returned by RevokeDeployHook when the id does not belong to the app.
var ErrDeployHookNotFound = errors.New("deploy hook not found")

// CreateDeployHook issues a new deploy hook for an app. Only its hash is stored, so the caller must
// keep the returned plaintext token: it cannot be recovered later, the same as a join token.
func (db *DB) CreateDeployHook(appID, name, sourceKind string) (plain string, hook *DeployHook, err error) {
	raw := make([]byte, constants.DeployHookTokenByteSize)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	plain = constants.DeployHookTokenPrefix + base64.RawURLEncoding.EncodeToString(raw)

	hook = &DeployHook{
		ID:         uuid.New().String(),
		AppID:      appID,
		Name:       name,
		SourceKind: sourceKind,
		CreatedAt:  time.Now().UTC(),
	}
	_, err = db.Exec(`INSERT INTO app_deploy_hooks (id, app_id, name, source_kind, token_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		hook.ID, hook.AppID, hook.Name, hook.SourceKind, hashToken(plain), hook.CreatedAt)
	if err != nil {
		return "", nil, err
	}
	return plain, hook, nil
}

// ListDeployHooks returns an app's hooks, newest first, never nil. Token hashes are still scanned
// (they are part of the row) but the type never marshals them to JSON.
func (db *DB) ListDeployHooks(appID string) ([]*DeployHook, error) {
	rows, err := db.Query(`SELECT id, app_id, name, source_kind, token_hash, created_at, last_used_at, COALESCE(last_used_ip, '')
		FROM app_deploy_hooks WHERE app_id = ? ORDER BY created_at DESC`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*DeployHook{}
	for rows.Next() {
		var h DeployHook
		if err := rows.Scan(&h.ID, &h.AppID, &h.Name, &h.SourceKind, &h.TokenHash, &h.CreatedAt, &h.LastUsedAt, &h.LastUsedIP); err != nil {
			return nil, err
		}
		out = append(out, &h)
	}
	return out, rows.Err()
}

// ConsumeDeployHookToken looks up the hook a presented token belongs to, scoped to one app so a
// token for a different app never matches, and records the calling address. It reports (nil, nil)
// for a token that does not match any hook on this app: not found is not an error here, since a
// wrong or revoked token is an expected caller mistake, not a database fault.
//
// The update and the read run in one transaction, so a concurrent RevokeDeployHook cannot delete the
// row between the two: without that, a revoke landing in that gap would make this record a use
// against a row that then reads back as gone, rejecting a token that was still valid when presented.
func (db *DB) ConsumeDeployHookToken(appID, presented, callerIP string) (*DeployHook, error) {
	if presented == "" {
		return nil, nil
	}
	hash := hashToken(presented)
	now := time.Now().UTC()

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed

	res, err := tx.Exec(`UPDATE app_deploy_hooks SET last_used_at = ?, last_used_ip = ?
		WHERE app_id = ? AND token_hash = ?`, now, callerIP, appID, hash)
	if err != nil {
		return nil, err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return nil, err
	}

	var h DeployHook
	err = tx.QueryRow(`SELECT id, app_id, name, source_kind, token_hash, created_at, last_used_at, COALESCE(last_used_ip, '')
		FROM app_deploy_hooks WHERE app_id = ? AND token_hash = ?`, appID, hash).
		Scan(&h.ID, &h.AppID, &h.Name, &h.SourceKind, &h.TokenHash, &h.CreatedAt, &h.LastUsedAt, &h.LastUsedIP)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &h, nil
}

// RevokeDeployHook deletes one hook. It stops working immediately; other hooks on the app, if any,
// are unaffected. Returns ErrDeployHookNotFound when the id does not belong to appID.
func (db *DB) RevokeDeployHook(appID, hookID string) error {
	res, err := db.Exec(`DELETE FROM app_deploy_hooks WHERE id = ? AND app_id = ?`, hookID, appID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrDeployHookNotFound
	}
	return nil
}
