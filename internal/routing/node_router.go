package routing

import (
	"context"
	"log/slog"
	"sync"

	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/node"
)

// NodeRouter handles routing operations to local or remote nodes
type NodeRouter struct {
	database    *db.DB
	nodeClient  *node.Client
	localNodeID string
	logger      *slog.Logger
}

// NewNodeRouter creates a new node router
func NewNodeRouter(database *db.DB, nodeClient *node.Client, localNodeID string, logger *slog.Logger) *NodeRouter {
	return &NodeRouter{
		database:    database,
		nodeClient:  nodeClient,
		localNodeID: localNodeID,
		logger:      logger,
	}
}

// DetermineTargetNodes resolves which nodes to query based on nodeIDs parameter
// If nodeIDs is empty or contains "all", returns all nodes
// Otherwise returns only the specified nodes
func (r *NodeRouter) DetermineTargetNodes(ctx context.Context, nodeIDs []string) ([]*db.Node, error) {
	if len(nodeIDs) == 0 || (len(nodeIDs) == 1 && nodeIDs[0] == "all") {
		// Fetch from all nodes
		allNodes, err := r.database.GetAllNodes()
		if err != nil {
			r.logger.ErrorContext(ctx, "failed to get nodes", "error", err)
			return nil, domain.WrapDatabaseOperation("get nodes", err)
		}
		return allNodes, nil
	}

	// Fetch from specific nodes
	var targetNodes []*db.Node
	for _, nodeID := range nodeIDs {
		node, err := r.database.GetNode(nodeID)
		if err != nil {
			r.logger.WarnContext(ctx, "node not found", "nodeID", nodeID, "error", err)
			continue
		}
		targetNodes = append(targetNodes, node)
	}

	return targetNodes, nil
}

// AggregateFromNodes fetches data from multiple nodes in parallel
// localFetcher is called if the node is the local node
// remoteFetcher is called for remote nodes
func (r *NodeRouter) AggregateFromNodes(
	ctx context.Context,
	nodes []*db.Node,
	localFetcher func() (interface{}, error),
	remoteFetcher func(*db.Node) (interface{}, error),
) ([]interface{}, error) {
	var (
		results []interface{}
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	for _, node := range nodes {
		wg.Add(1)
		go func(n *db.Node) {
			defer wg.Done()

			var result interface{}
			var err error

			if n.ID == r.localNodeID {
				// Fetch local data
				result, err = localFetcher()
				if err != nil {
					r.logger.ErrorContext(ctx, "failed to retrieve local data", "error", err)
					return
				}
			} else {
				// Fetch from remote node
				result, err = remoteFetcher(n)
				if err != nil {
					r.logger.WarnContext(ctx, "failed to fetch from remote node", "nodeID", n.ID, "nodeName", n.Name, "error", err)
					return
				}
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(node)
	}

	// Wait for all fetches to complete
	wg.Wait()

	return results, nil
}

// IsLocalNode checks if a node ID is the local node
func (r *NodeRouter) IsLocalNode(nodeID string) bool {
	return nodeID == r.localNodeID || nodeID == ""
}
