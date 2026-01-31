package routing

import (
	"context"
	"log/slog"
	"sync"

	"github.com/selfhostly/internal/db"
)

// AppsAggregator aggregates apps from multiple nodes
type AppsAggregator struct {
	router *NodeRouter
	logger *slog.Logger
}

// NewAppsAggregator creates a new apps aggregator
func NewAppsAggregator(router *NodeRouter, logger *slog.Logger) *AppsAggregator {
	return &AppsAggregator{
		router: router,
		logger: logger,
	}
}

// AggregateApps fetches apps from multiple nodes in parallel
func (a *AppsAggregator) AggregateApps(
	ctx context.Context,
	nodes []*db.Node,
	localFetcher func() ([]*db.App, error),
	remoteFetcher func(*db.Node) ([]*db.App, error),
) ([]*db.App, error) {
	var (
		allApps []*db.App
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	for _, node := range nodes {
		wg.Add(1)
		go func(n *db.Node) {
			defer wg.Done()

			if n.ID == a.router.localNodeID {
				// Fetch local apps
				localApps, err := localFetcher()
				if err != nil {
					a.logger.ErrorContext(ctx, "failed to retrieve local apps", "error", err)
					return
				}

				// Add node ID to each app for display
				for _, app := range localApps {
					app.NodeID = n.ID
				}

				mu.Lock()
				allApps = append(allApps, localApps...)
				mu.Unlock()
			} else {
				// Fetch from remote node
				a.logger.InfoContext(ctx, "fetching apps from remote node", "nodeID", n.ID, "nodeName", n.Name)
				remoteApps, err := remoteFetcher(n)
				if err != nil {
					a.logger.WarnContext(ctx, "failed to fetch apps from remote node", "nodeID", n.ID, "nodeName", n.Name, "error", err)
					return
				}

				mu.Lock()
				allApps = append(allApps, remoteApps...)
				mu.Unlock()
			}
		}(node)
	}

	// Wait for all fetches to complete
	wg.Wait()

	return allApps, nil
}

// TunnelsAggregator aggregates Cloudflare tunnels from multiple nodes
type TunnelsAggregator struct {
	router *NodeRouter
	logger *slog.Logger
}

// NewTunnelsAggregator creates a new tunnels aggregator
func NewTunnelsAggregator(router *NodeRouter, logger *slog.Logger) *TunnelsAggregator {
	return &TunnelsAggregator{
		router: router,
		logger: logger,
	}
}

// AggregateTunnels fetches tunnels from multiple nodes in parallel
func (a *TunnelsAggregator) AggregateTunnels(
	ctx context.Context,
	nodes []*db.Node,
	localFetcher func() ([]*db.CloudflareTunnel, error),
	remoteFetcher func(*db.Node) ([]*db.CloudflareTunnel, error),
) ([]*db.CloudflareTunnel, error) {
	var (
		allTunnels []*db.CloudflareTunnel
		mu         sync.Mutex
		wg         sync.WaitGroup
	)

	for _, node := range nodes {
		wg.Add(1)
		go func(n *db.Node) {
			defer wg.Done()

			if n.ID == a.router.localNodeID {
				// Fetch local tunnels
				localTunnels, err := localFetcher()
				if err != nil {
					a.logger.ErrorContext(ctx, "failed to retrieve local tunnels", "error", err)
					return
				}

				mu.Lock()
				allTunnels = append(allTunnels, localTunnels...)
				mu.Unlock()
			} else {
				// Fetch from remote node
				a.logger.InfoContext(ctx, "fetching tunnels from remote node", "nodeID", n.ID, "nodeName", n.Name)
				remoteTunnels, err := remoteFetcher(n)
				if err != nil {
					a.logger.WarnContext(ctx, "failed to fetch tunnels from remote node", "nodeID", n.ID, "nodeName", n.Name, "error", err)
					return
				}

				mu.Lock()
				allTunnels = append(allTunnels, remoteTunnels...)
				mu.Unlock()
			}
		}(node)
	}

	// Wait for all fetches to complete
	wg.Wait()

	return allTunnels, nil
}
