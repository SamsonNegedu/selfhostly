package routing

import (
	"context"
	"log/slog"
	"sync"

	"github.com/selfhostly/internal/constants"
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
// Filters out offline and unreachable nodes to avoid unnecessary request attempts
func (r *NodeRouter) DetermineTargetNodes(ctx context.Context, nodeIDs []string) ([]*db.Node, error) {
	var allNodes []*db.Node
	var err error

	if len(nodeIDs) == 0 || (len(nodeIDs) == 1 && nodeIDs[0] == "all") {
		// Fetch from all nodes
		allNodes, err = r.database.GetAllNodes()
		if err != nil {
			r.logger.ErrorContext(ctx, "failed to get nodes", "error", err)
			return nil, domain.WrapDatabaseOperation("get nodes", err)
		}
	} else {
		// Fetch from specific nodes
		for _, nodeID := range nodeIDs {
			node, getErr := r.database.GetNode(nodeID)
			if getErr != nil {
				r.logger.WarnContext(ctx, "node not found", "nodeID", nodeID, "error", getErr)
				continue
			}
			allNodes = append(allNodes, node)
		}
	}

	// Filter out offline and unreachable nodes (but always include local node)
	var targetNodes []*db.Node
	for _, node := range allNodes {
		// Always include local node regardless of status
		if node.ID == r.localNodeID {
			targetNodes = append(targetNodes, node)
			continue
		}

		// Skip offline and unreachable nodes for remote requests
		if node.Status == constants.NodeStatusOffline || node.Status == constants.NodeStatusUnreachable {
			r.logger.DebugContext(ctx, "skipping offline/unreachable node for request forwarding",
				"nodeID", node.ID,
				"nodeName", node.Name,
				"status", node.Status)
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
