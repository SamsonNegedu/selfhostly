package domain

import (
	"context"
	"errors"
	"time"

	"github.com/selfhostly/internal/db"
)

// ErrRegistrationUnauthorized: the registration token is wrong, expired or was already used
var ErrRegistrationUnauthorized = errors.New("invalid registration token")

// ErrDeployTriggerUnauthorized: the deploy hook token is missing, unknown, or does not belong to this app
var ErrDeployTriggerUnauthorized = errors.New("invalid deploy hook token")

// SecurityService defines the primary port for the administrative security use cases: join tokens,
// ending sessions and the audit log.
type SecurityService interface {
	// CreateJoinToken issues a single-use token a new secondary node can present when it registers
	CreateJoinToken(ctx context.Context) (token string, expires time.Time, err error)
	// RevokeSessions invalidates every session issued before now
	RevokeSessions(ctx context.Context) (time.Time, error)
	// ListAudit returns the most recent audit records, newest first
	ListAudit(ctx context.Context, limit int) ([]db.AuditEntry, error)
	// RecordAudit stores one audit record. Callers treat failure as non-fatal.
	RecordAudit(ctx context.Context, entry db.AuditEntry) error
	// AuditTargetName returns the label of an app or node for the audit log, or "" when it is not known
	AuditTargetName(ctx context.Context, targetType, id string) string
}

// Kinds of thing the audit log can name by looking them up.
const (
	AuditTargetApp  = "app"
	AuditTargetNode = "node"
)

// DeployHook is a named, per-app secret an external pipeline presents to trigger a pull and
// restart. The token itself is never read back after creation; only this metadata is.
type DeployHook struct {
	ID         string     `json:"id"`
	AppID      string     `json:"app_id"`
	Name       string     `json:"name"`
	SourceKind string     `json:"source_kind"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	LastUsedIP string     `json:"last_used_ip"`
}

// DeployHookService defines the primary port for creating, listing and revoking per-app deploy
// hooks, and for verifying one and triggering the update it authorizes. A hook's SourceKind names
// which class of caller it is for (today only DeploySourceGeneric exists); adding a kind that needs
// more than a bearer-token check means adding a verifier in internal/service, not changing this port.
type DeployHookService interface {
	// CreateDeployHook issues a new hook for an app. Only its hash is stored, so the plaintext token
	// is returned once and cannot be recovered later.
	CreateDeployHook(ctx context.Context, appID, name string) (token string, hook *DeployHook, err error)
	// ListDeployHooks returns an app's hooks, newest first, never nil. Tokens are never included.
	ListDeployHooks(ctx context.Context, appID string) ([]*DeployHook, error)
	// RevokeDeployHook deletes one hook. It stops working immediately; other hooks on the app are unaffected.
	RevokeDeployHook(ctx context.Context, appID, hookID string) error
	// TriggerDeploy verifies a presented token against appID's hooks and, on success, starts the same
	// update job the manual Update button starts. callerIP is recorded against the hook, never used
	// to authenticate. Returns ErrDeployTriggerUnauthorized for a missing, wrong or revoked token.
	TriggerDeploy(ctx context.Context, appID, presentedToken, callerIP string) (*db.Job, *DeployHook, error)
}

// AutoRegisterRequest is what a secondary sends to register itself with the primary
type AutoRegisterRequest struct {
	ID          string
	Name        string
	APIEndpoint string
	APIKey      string
	Token       string // shared registration token or single-use join token
}

// AutoRegistration says what registering did
type AutoRegistration struct {
	Node    *db.Node
	Created bool // false: an already-registered node refreshed its record
}
