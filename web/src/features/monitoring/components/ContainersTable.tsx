import { useState, useMemo } from 'react';
import { Card, CardContent } from '@/shared/components/ui/Card';
import { Badge } from '@/shared/components/ui/Badge';
import { Button } from '@/shared/components/ui/Button';
import { Network, HardDrive, ArrowUpDown, ArrowUp, ArrowDown } from 'lucide-react';
import type { ContainerInfo } from '@/shared/types/api';
import ContainerActions from './ContainerActions';

type SortField = 'name' | 'cpu' | 'memory' | 'app';
type SortOrder = 'asc' | 'desc';

interface ContainersTableProps {
  containers: ContainerInfo[];
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
}

function getStateBadgeColor(state: string) {
  switch (state) {
    case 'running':
      return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400';
    case 'paused':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400';
    case 'stopped':
      return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400';
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400';
  }
}

function ContainersTable({ containers }: ContainersTableProps) {
  const [sortField, setSortField] = useState<SortField>('cpu');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');

  // Sort containers
  const sortedContainers = useMemo(() => {
    return [...containers].sort((a, b) => {
      let aValue: number | string = 0;
      let bValue: number | string = 0;

      switch (sortField) {
        case 'name':
          aValue = a.name.toLowerCase();
          bValue = b.name.toLowerCase();
          break;
        case 'cpu':
          aValue = a.cpu_percent;
          bValue = b.cpu_percent;
          break;
        case 'memory':
          aValue = a.memory_usage_bytes;
          bValue = b.memory_usage_bytes;
          break;
        case 'app':
          aValue = a.app_name.toLowerCase();
          bValue = b.app_name.toLowerCase();
          break;
      }

      if (aValue < bValue) return sortOrder === 'asc' ? -1 : 1;
      if (aValue > bValue) return sortOrder === 'asc' ? 1 : -1;
      return 0;
    });
  }, [containers, sortField, sortOrder]);

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortOrder(field === 'cpu' || field === 'memory' ? 'desc' : 'asc');
    }
  };

  const getSortIcon = (field: SortField) => {
    if (sortField !== field) {
      return <ArrowUpDown className="h-3.5 w-3.5" />;
    }
    return sortOrder === 'asc' ? (
      <ArrowUp className="h-3.5 w-3.5" />
    ) : (
      <ArrowDown className="h-3.5 w-3.5" />
    );
  };

  if (containers.length === 0) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div className="text-center py-12">
            <p className="text-muted-foreground">No containers found</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-3 sm:space-y-4">
      {/* Sort Controls - Horizontal scroll on mobile */}
      <div className="overflow-x-auto scrollbar-hide -mx-3 sm:mx-0 px-3 sm:px-0 pb-1">
        <div className="flex items-center gap-2 w-fit">
          <span className="text-xs text-muted-foreground whitespace-nowrap">Sort by:</span>
          <div className="flex items-center gap-1.5 bg-muted/50 rounded-lg p-1">
            <Button
              variant={sortField === 'cpu' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => toggleSort('cpu')}
              className="gap-1 text-xs h-8 px-3 whitespace-nowrap"
            >
              CPU {getSortIcon('cpu')}
            </Button>
            <Button
              variant={sortField === 'memory' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => toggleSort('memory')}
              className="gap-1 text-xs h-8 px-3 whitespace-nowrap"
            >
              Memory {getSortIcon('memory')}
            </Button>
            <Button
              variant={sortField === 'name' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => toggleSort('name')}
              className="gap-1 text-xs h-8 px-3 whitespace-nowrap"
            >
              Name {getSortIcon('name')}
            </Button>
            <Button
              variant={sortField === 'app' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => toggleSort('app')}
              className="gap-1 text-xs h-8 px-3 whitespace-nowrap"
            >
              App {getSortIcon('app')}
            </Button>
          </div>
        </div>
      </div>

      {/* Containers */}
      <div className="space-y-3">
        {sortedContainers.map((container) => {
          const memPercent =
            container.memory_limit_bytes > 0
              ? (container.memory_usage_bytes / container.memory_limit_bytes) * 100
              : 0;

          // Determine resource usage level for visual indicator
          const isHighUsage = container.state === 'running' && (container.cpu_percent > 80 || memPercent > 80);
          const isMediumUsage = container.state === 'running' && (container.cpu_percent > 50 || memPercent > 50);

          const borderClass = isHighUsage
            ? 'border-l-4 border-l-red-500'
            : isMediumUsage
              ? 'border-l-4 border-l-yellow-500'
              : '';

          return (
            <Card key={container.id} className={`overflow-hidden ${borderClass}`}>
              <CardContent className="pt-4 sm:pt-6 p-4 sm:p-6">
                {/* Header Row */}
                <div className="flex items-start justify-between mb-3 sm:mb-4">
                  <div className="flex-1 min-w-0">
                    {/* Container Name and State */}
                    <div className="flex items-center gap-2 mb-2 flex-wrap">
                      <h3 className="font-mono text-sm font-medium truncate">{container.name}</h3>
                      <Badge className={getStateBadgeColor(container.state)}>{container.state}</Badge>
                      {container.restart_count > 0 && (
                        <Badge variant="outline" className="text-xs px-1.5 py-0 h-5 bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-400 border-yellow-200 dark:border-yellow-800">
                          Restarts: {container.restart_count}
                        </Badge>
                      )}
                    </div>

                    {/* App and Container Info */}
                    <div className="flex items-center gap-3 flex-wrap text-xs text-muted-foreground">
                      {/* App Name with Management Badge */}
                      <div className="flex items-center gap-1.5">
                        <span className="text-muted-foreground/70">App:</span>
                        <span className="font-medium text-foreground">{container.app_name}</span>
                        {container.is_managed && (
                          <Badge variant="outline" className="text-xs px-1.5 py-0 h-5 bg-blue-50 dark:bg-blue-950 text-blue-700 dark:text-blue-400 border-blue-200 dark:border-blue-800">
                            Managed
                          </Badge>
                        )}
                        {!container.is_managed && container.app_name !== 'unmanaged' && (
                          <Badge variant="outline" className="text-xs px-1.5 py-0 h-5 bg-amber-50 dark:bg-amber-950 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800">
                            External
                          </Badge>
                        )}
                      </div>

                      {/* Container ID */}
                      <div className="flex items-center gap-1.5">
                        <span className="text-muted-foreground/70">ID:</span>
                        <span className="font-mono text-foreground">{container.id.substring(0, 12)}</span>
                      </div>
                    </div>
                  </div>
                  <ContainerActions
                    containerId={container.id}
                    containerName={container.name}
                    containerState={container.state}
                    appName={container.app_name}
                    nodeId={container.node_id}
                    isManaged={container.is_managed}
                  />
                </div>

                {/* Stats Grid - Only show for running containers */}
                {container.state === 'running' && (
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4">
                    {/* CPU */}
                    <div>
                      <p className="text-xs text-muted-foreground mb-1">CPU</p>
                      <div className="flex items-baseline gap-1">
                        <p className="text-sm font-medium">{container.cpu_percent.toFixed(1)}%</p>
                      </div>
                      <div className="mt-2 w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                        <div
                          className={`h-1.5 rounded-full transition-all ${container.cpu_percent > 80
                            ? 'bg-red-500'
                            : container.cpu_percent > 50
                              ? 'bg-yellow-500'
                              : 'bg-blue-500'
                            }`}
                          style={{ width: `${Math.min(container.cpu_percent, 100)}%` }}
                        />
                      </div>
                    </div>

                    {/* Memory */}
                    <div>
                      <p className="text-xs text-muted-foreground mb-1">Memory</p>
                      <div className="flex items-baseline gap-1">
                        <p className="text-sm font-medium">
                          {formatBytes(container.memory_usage_bytes)}
                        </p>
                        {container.memory_limit_bytes > 0 && (
                          <p className="text-xs text-muted-foreground">
                            / {formatBytes(container.memory_limit_bytes)}
                          </p>
                        )}
                      </div>
                      <div className="mt-2 w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                        <div
                          className={`h-1.5 rounded-full transition-all ${memPercent > 80
                            ? 'bg-red-500'
                            : memPercent > 50
                              ? 'bg-yellow-500'
                              : 'bg-green-500'
                            }`}
                          style={{ width: `${Math.min(memPercent, 100)}%` }}
                        />
                      </div>
                    </div>

                    {/* Network I/O */}
                    <div>
                      <p className="text-xs text-muted-foreground mb-1 flex items-center gap-1">
                        <Network className="h-3 w-3" />
                        Network I/O
                      </p>
                      <div className="space-y-1">
                        <div className="flex items-center gap-1 text-xs">
                          <span className="text-blue-600 dark:text-blue-400">↓</span>
                          <span className="font-medium">{formatBytes(container.network_rx_bytes)}</span>
                        </div>
                        <div className="flex items-center gap-1 text-xs">
                          <span className="text-green-600 dark:text-green-400">↑</span>
                          <span className="font-medium">{formatBytes(container.network_tx_bytes)}</span>
                        </div>
                      </div>
                    </div>

                    {/* Disk I/O */}
                    <div>
                      <p className="text-xs text-muted-foreground mb-1 flex items-center gap-1">
                        <HardDrive className="h-3 w-3" />
                        Disk I/O
                      </p>
                      <div className="space-y-1">
                        <div className="flex items-center gap-1 text-xs">
                          <span className="text-blue-600 dark:text-blue-400">R</span>
                          <span className="font-medium">{formatBytes(container.block_read_bytes)}</span>
                        </div>
                        <div className="flex items-center gap-1 text-xs">
                          <span className="text-orange-600 dark:text-orange-400">W</span>
                          <span className="font-medium">{formatBytes(container.block_write_bytes)}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {/* Stopped container message */}
                {container.state !== 'running' && (
                  <div className="text-sm text-muted-foreground">
                    Container is {container.state}. Resource metrics are only available for running containers.
                  </div>
                )}
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}

export default ContainersTable;
