package domain

import (
	"context"
	"errors"

	"github.com/selfhostly/internal/db"
)

// Errors a NodeLinkService returns for reasons the caller answers with a specific status.
var (
	// ErrLinkUnauthorized: the node's key or the join token is wrong, expired or already used
	ErrLinkUnauthorized = errors.New("invalid link credentials")
	// ErrLinkNameTaken: another node already has that name
	ErrLinkNameTaken = errors.New("node name already exists")
)

// LinkRequest is what a secondary presents when it dials out to the primary.
type LinkRequest struct {
	ID    string
	Name  string
	Key   string
	Token string // single-use join token or the shared registration token; only needed by a new node
}

// LinkAdmission says what admitting a node did, so it can be audited.
type LinkAdmission struct {
	Node     *db.Node
	Created  bool // a new node was registered
	Switched bool // an existing directly-reached node now uses its link
}

// NodeLinkService holds the rules for nodes that connect out to the primary (see docs/design/rfcs/node-link.md).
type NodeLinkService interface {
	// Admit authenticates the node and records it: a known node by its key, a new node by a token.
	Admit(ctx context.Context, req LinkRequest) (*LinkAdmission, error)
	// LinkedNode returns the node when it is reached over its link, and nil when it is unknown or reached directly.
	LinkedNode(ctx context.Context, nodeID string) (*db.Node, error)
	// MarkOffline records that a linked node's connection dropped.
	MarkOffline(ctx context.Context, nodeID string) error
}
