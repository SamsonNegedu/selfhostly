package domain

import (
	"context"
	"errors"
	"time"

	"github.com/selfhostly/internal/db"
)

// ErrRegistrationUnauthorized: the registration token is wrong, expired or was already used
var ErrRegistrationUnauthorized = errors.New("invalid registration token")

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
