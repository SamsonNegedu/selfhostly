import { useState, useMemo } from 'react';
import { useSystemStats } from '@/shared/services/api';
import { useNodeContext } from '@/shared/contexts/NodeContext';
import { AlertCircle, Search, X, WifiOff, ServerCrash } from 'lucide-react';
import { DashboardSkeleton } from '@/shared/components/ui/Skeleton';
import { Button } from '@/shared/components/ui/button';
import { Card, CardContent } from '@/shared/components/ui/card';
import SystemOverview from './components/SystemOverview';
import ContainersTable from './components/ContainersTable';
import ResourceAlerts from './components/ResourceAlerts';
import type { ContainerInfo, SystemStats } from '@/shared/types/api';

function Monitoring() {
  // Get global node context for filtering stats by selected nodes
  const { selectedNodeIds } = useNodeContext();

  const { data: statsArray, isLoading, error, dataUpdatedAt } = useSystemStats(10000, selectedNodeIds);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'running' | 'stopped'>('all');

  // Separate online and offline/error nodes
  const { onlineNodes, offlineNodes } = useMemo(() => {
    if (!statsArray || !Array.isArray(statsArray)) {
      return { onlineNodes: [], offlineNodes: [] };
    }
    const online: SystemStats[] = [];
    const offline: SystemStats[] = [];

    statsArray.forEach(stat => {
      if (stat.status === 'online') {
        online.push(stat);
      } else {
        offline.push(stat);
      }
    });

    return { onlineNodes: online, offlineNodes: offline };
  }, [statsArray]);

  // For now, show first online node's stats (or null if none)
  const stats = useMemo(() => {
    return onlineNodes.length > 0 ? onlineNodes[0] : null;
  }, [onlineNodes]);

  // Aggregate containers from all ONLINE nodes only
  const allContainers: ContainerInfo[] = useMemo(() => {
    if (!statsArray || !Array.isArray(statsArray)) {
      return (stats?.containers || []) as ContainerInfo[];
    }
    // Combine containers from all online nodes only
    return onlineNodes.flatMap(nodeStats => (nodeStats.containers || []) as ContainerInfo[]);
  }, [onlineNodes, stats]);

  // Filter containers based on search and status
  const filteredContainers: ContainerInfo[] = useMemo(() => {
    if (!allContainers || allContainers.length === 0) return [];

    let filtered = allContainers;

    // Filter by search query
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (container) =>
          container.name.toLowerCase().includes(query) ||
          container.app_name.toLowerCase().includes(query) ||
          container.id.toLowerCase().includes(query)
      );
    }

    // Filter by status
    if (statusFilter !== 'all') {
      filtered = filtered.filter((container) => container.state === statusFilter);
    }

    return filtered;
  }, [allContainers, searchQuery, statusFilter]);

  // Get node names for display (must be before early returns) - include both online and offline
  const nodeNames = useMemo(() => {
    if (!statsArray || !Array.isArray(statsArray)) {
      return stats?.node_name ? [stats.node_name] : [];
    }
    return statsArray.map(s => s.node_name).filter(Boolean);
  }, [statsArray]);

  // Calculate seconds ago (must be before early returns)
  const secondsAgo = useMemo(() => {
    if (!dataUpdatedAt) return 0;
    const lastUpdated = new Date(dataUpdatedAt);
    return Math.floor((Date.now() - lastUpdated.getTime()) / 1000);
  }, [dataUpdatedAt]);

  // Early returns after all hooks
  if (isLoading && !statsArray) {
    return <DashboardSkeleton />;
  }

  if (error) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center max-w-md fade-in">
          <AlertCircle className="h-12 w-12 text-destructive mx-auto mb-4" />
          <h2 className="text-xl font-semibold mb-2">Failed to load system statistics</h2>
          <p className="text-muted-foreground mb-4">
            There was an error loading the monitoring data. Please try again.
          </p>
          <button
            onClick={() => window.location.reload()}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:opacity-90 transition-opacity button-press"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  // If there are no stats at all (not even offline nodes), return null
  if (!stats && offlineNodes.length === 0) {
    return null;
  }

  // Ensure stats has required properties with defaults (if we have online nodes)
  const safeStats = stats ? {
    ...stats,
    containers: stats.containers || [],
    cpu: stats.cpu || { usage_percent: 0, cores: 0 },
    memory: stats.memory || { usage_percent: 0, total_bytes: 0, used_bytes: 0, free_bytes: 0, available_bytes: 0 },
    disk: stats.disk || { usage_percent: 0, total_bytes: 0, used_bytes: 0, free_bytes: 0, path: '/' },
    docker: stats.docker || { total_containers: 0, running: 0, stopped: 0, paused: 0, images: 0, version: '' },
  } : null;

  return (
    <div className="fade-in space-y-4 sm:space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl sm:text-3xl font-bold">System Monitoring</h1>
        <p className="text-muted-foreground mt-1 sm:mt-2 text-sm sm:text-base">
          {nodeNames.length > 0
            ? `Real-time monitoring of ${nodeNames.length > 1 ? `${nodeNames.length} nodes` : nodeNames[0]}`
            : 'Real-time monitoring'
          } • Updated {secondsAgo > 0 ? `${secondsAgo}s ago` : 'just now'}
        </p>
      </div>

      {/* Offline/Error Nodes Alert */}
      {offlineNodes.length > 0 && (
        <div className="space-y-2">
          {offlineNodes.map(node => (
            <Card key={node.node_id} className="border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-950/30">
              <CardContent className="pt-6">
                <div className="flex items-start gap-3">
                  {node.status === 'offline' ? (
                    <WifiOff className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
                  ) : (
                    <ServerCrash className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
                  )}
                  <div className="flex-1 min-w-0">
                    <h3 className="font-semibold text-red-900 dark:text-red-100">
                      {node.status === 'offline' ? 'Node Unreachable' : 'Error Fetching Stats'}: {node.node_name}
                    </h3>
                    <p className="text-sm text-red-800 dark:text-red-200 mt-1">
                      {node.error || 'Unable to connect to this node. Please check if the node is running and accessible.'}
                    </p>
                    {node.status === 'offline' && (
                      <p className="text-xs text-red-700 dark:text-red-300 mt-2">
                        💡 Tip: Verify the node's API endpoint is correct and the node service is running.
                      </p>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Resource Alerts - Only show if we have online nodes */}
      {stats && safeStats && <ResourceAlerts stats={safeStats} />}

      {/* System Overview Cards - Only show if we have online nodes */}
      {stats && safeStats ? (
        <SystemOverview stats={safeStats} />
      ) : onlineNodes.length === 0 && offlineNodes.length > 0 ? (
        <Card className="border-gray-200 dark:border-gray-700">
          <CardContent className="pt-6">
            <div className="text-center py-8">
              <ServerCrash className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2">No Online Nodes</h3>
              <p className="text-muted-foreground">
                All selected nodes are currently offline or unreachable. Please check the alerts above for details.
              </p>
            </div>
          </CardContent>
        </Card>
      ) : null}

      {/* Containers Section - Only show if we have online nodes */}
      {stats && (
        <div className="space-y-3 sm:space-y-4">
          {/* Header with count */}
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <h2 className="text-xl sm:text-2xl font-bold">
              All Containers ({filteredContainers.length})
            </h2>

            {/* Status Filter Buttons - Compact on mobile */}
            {allContainers.length > 0 && (
              <div className="flex items-center gap-1.5 sm:gap-2 bg-muted/50 rounded-lg p-1 w-fit">
                <Button
                  variant={statusFilter === 'all' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setStatusFilter('all')}
                  className="h-8 px-3 text-xs"
                >
                  All
                </Button>
                <Button
                  variant={statusFilter === 'running' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setStatusFilter('running')}
                  className="h-8 px-3 text-xs"
                >
                  Running
                </Button>
                <Button
                  variant={statusFilter === 'stopped' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setStatusFilter('stopped')}
                  className="h-8 px-3 text-xs"
                >
                  Stopped
                </Button>
              </div>
            )}
          </div>

          {/* Search Bar - Full width */}
          {allContainers.length > 0 && (
            <div className="relative w-full">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Search containers, apps, or IDs..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 pl-10 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery('')}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
          )}

          {/* Containers Table */}
          <ContainersTable containers={filteredContainers} />
        </div>
      )}
    </div>
  );
}

export default Monitoring;
