package routing

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/system"
)

// StatsAggregator aggregates system stats from multiple nodes
type StatsAggregator struct {
	router *NodeRouter
	logger *slog.Logger
}

// NewStatsAggregator creates a new stats aggregator
func NewStatsAggregator(router *NodeRouter, logger *slog.Logger) *StatsAggregator {
	return &StatsAggregator{
		router: router,
		logger: logger,
	}
}

// AggregateStats fetches system stats from multiple nodes in parallel
func (a *StatsAggregator) AggregateStats(
	ctx context.Context,
	nodes []*db.Node,
	localFetcher func() (*system.SystemStats, error),
	remoteFetcher func(*db.Node) (map[string]interface{}, error),
	mapConverter func(map[string]interface{}, string, string) (*system.SystemStats, error),
) ([]*system.SystemStats, error) {
	var (
		allStats []*system.SystemStats
		mu       sync.Mutex
		wg       sync.WaitGroup
	)

	for _, node := range nodes {
		wg.Add(1)
		go func(n *db.Node) {
			defer wg.Done()

			if n.ID == a.router.localNodeID {
				// Fetch local stats
				localStats, err := localFetcher()
				if err != nil {
					a.logger.ErrorContext(ctx, "failed to retrieve local system stats", "error", err)
					return
				}

				mu.Lock()
				allStats = append(allStats, localStats)
				mu.Unlock()
			} else {
				// Fetch from remote node
				a.logger.InfoContext(ctx, "fetching system stats from remote node", "nodeID", n.ID, "nodeName", n.Name)
				remoteStatsMap, err := remoteFetcher(n)
				if err != nil {
					a.logger.WarnContext(ctx, "failed to fetch system stats from remote node", "nodeID", n.ID, "nodeName", n.Name, "error", err)

					// Create an error stat object so the UI knows the node is offline
					errorStats := &system.SystemStats{
						NodeID:     n.ID,
						NodeName:   n.Name,
						Status:     "offline",
						Error:      err.Error(),
						Timestamp:  time.Now(),
						Containers: []system.ContainerInfo{}, // Empty slice, not nil
					}

					mu.Lock()
					allStats = append(allStats, errorStats)
					mu.Unlock()
					return
				}

				// Convert map to SystemStats struct
				remoteStats, err := mapConverter(remoteStatsMap, n.ID, n.Name)
				if err != nil {
					a.logger.WarnContext(ctx, "failed to convert remote stats", "nodeID", n.ID, "error", err)

					// Create an error stat object
					errorStats := &system.SystemStats{
						NodeID:     n.ID,
						NodeName:   n.Name,
						Status:     "error",
						Error:      err.Error(),
						Timestamp:  time.Now(),
						Containers: []system.ContainerInfo{},
					}

					mu.Lock()
					allStats = append(allStats, errorStats)
					mu.Unlock()
					return
				}

				mu.Lock()
				allStats = append(allStats, remoteStats)
				mu.Unlock()
			}
		}(node)
	}

	// Wait for all fetches to complete
	wg.Wait()

	return allStats, nil
}
