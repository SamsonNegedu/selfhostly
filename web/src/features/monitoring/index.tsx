import { useState, useMemo } from 'react';
import { useSystemStats } from '@/shared/services/api';
import { AlertCircle, Search, X } from 'lucide-react';
import { DashboardSkeleton } from '@/shared/components/ui/Skeleton';
import { Button } from '@/shared/components/ui/button';
import SystemOverview from './components/SystemOverview';
import ContainersTable from './components/ContainersTable';
import ResourceAlerts from './components/ResourceAlerts';

function Monitoring() {
  const { data: stats, isLoading, error, dataUpdatedAt } = useSystemStats(10000);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'running' | 'stopped'>('all');

  // Filter containers based on search and status
  const filteredContainers = useMemo(() => {
    if (!stats?.containers) return [];

    let filtered = stats.containers;

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
  }, [stats?.containers, searchQuery, statusFilter]);

  if (isLoading && !stats) {
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

  if (!stats) {
    return null;
  }

  const lastUpdated = new Date(dataUpdatedAt);
  const secondsAgo = Math.floor((Date.now() - lastUpdated.getTime()) / 1000);

  return (
    <div className="fade-in space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">System Monitoring</h1>
        <p className="text-muted-foreground mt-2">
          Real-time monitoring of {stats.node_name} • Updated {secondsAgo}s ago
        </p>
      </div>

      {/* Resource Alerts */}
      <ResourceAlerts stats={stats} />

      {/* System Overview Cards */}
      <SystemOverview stats={stats} />

      {/* Containers Section */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold">
            All Containers ({filteredContainers.length})
          </h2>
        </div>

        {/* Search and Filters */}
        {stats.containers.length > 0 && (
          <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center">
            {/* Search Bar */}
            <div className="relative flex-1 max-w-md w-full">
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

            {/* Status Filter */}
            <div className="flex items-center gap-2">
              <Button
                variant={statusFilter === 'all' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setStatusFilter('all')}
              >
                All
              </Button>
              <Button
                variant={statusFilter === 'running' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setStatusFilter('running')}
              >
                Running
              </Button>
              <Button
                variant={statusFilter === 'stopped' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setStatusFilter('stopped')}
              >
                Stopped
              </Button>
            </div>
          </div>
        )}

        {/* Containers Table */}
        <ContainersTable containers={filteredContainers} />
      </div>
    </div>
  );
}

export default Monitoring;
