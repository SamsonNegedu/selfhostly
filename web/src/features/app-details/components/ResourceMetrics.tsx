import { useEffect, useState, useRef } from 'react';
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card';
import { apiClient } from '@/shared/lib/api-client';
import type { AppStats } from '@/shared/types/api';
import { Activity, Cpu, HardDrive, Network, Server } from 'lucide-react';

interface ResourceMetricsProps {
  appId: string;
  appStatus: string;
  isFullPage?: boolean;
}

export function ResourceMetrics({ appId, appStatus, isFullPage = false }: ResourceMetricsProps) {
  const [stats, setStats] = useState<AppStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<number | null>(null);
  const isVisibleRef = useRef(true);

  const fetchStats = async () => {
    // Only fetch if tab is visible and app is running
    if (!isVisibleRef.current || appStatus !== 'running') {
      return;
    }

    try {
      const data = await apiClient.get<AppStats>(`/api/apps/${appId}/stats`);
      setStats(data);
      setError(null);
    } catch (err) {
      console.error('Failed to fetch stats:', err);
      setError('Failed to load resource metrics');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // Handle page visibility changes to pause/resume polling
    const handleVisibilityChange = () => {
      isVisibleRef.current = !document.hidden;
      if (isVisibleRef.current) {
        // Resume polling immediately when tab becomes visible
        fetchStats();
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);

    // Initial fetch
    fetchStats();

    // Poll more frequently in full page mode (5s), less frequently when embedded (10s)
    const pollInterval = isFullPage ? 5000 : 10000;
    intervalRef.current = window.setInterval(fetchStats, pollInterval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [appId, appStatus, isFullPage]);

  if (loading && !stats) {
    return (
      <Card className="p-6">
        <h3 className="text-lg font-semibold mb-4">Resource Usage</h3>
        <div className="text-sm text-gray-500">Loading metrics...</div>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="p-6">
        <h3 className="text-lg font-semibold mb-4">Resource Usage</h3>
        <div className="text-sm text-red-500">{error}</div>
      </Card>
    );
  }

  if (!stats || appStatus !== 'running') {
    return (
      <Card className="p-6">
        <h3 className="text-lg font-semibold mb-4">Resource Usage</h3>
        <div className="text-sm text-gray-500">
          {stats?.message || 'App is not running. Start the app to view resource metrics.'}
        </div>
      </Card>
    );
  }

  if (stats.containers.length === 0) {
    return (
      <Card className="p-6">
        <h3 className="text-lg font-semibold mb-4">Resource Usage</h3>
        <div className="text-sm text-gray-500">No running containers found.</div>
      </Card>
    );
  }

  const memoryPercent = stats.memory_limit_bytes > 0
    ? (stats.total_memory_bytes / stats.memory_limit_bytes) * 100
    : 0;

  // Calculate totals for network and disk I/O
  const totalNetworkRx = stats.containers.reduce((sum, c) => sum + c.network_rx_bytes, 0);
  const totalNetworkTx = stats.containers.reduce((sum, c) => sum + c.network_tx_bytes, 0);
  const totalBlockRead = stats.containers.reduce((sum, c) => sum + c.block_read_bytes, 0);
  const totalBlockWrite = stats.containers.reduce((sum, c) => sum + c.block_write_bytes, 0);

  if (isFullPage) {
    return (
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground mb-1">CPU Usage</p>
                  <p className="text-2xl font-bold">{stats.total_cpu_percent.toFixed(1)}%</p>
                </div>
                <div className={`p-3 rounded-full ${stats.total_cpu_percent > 80
                    ? 'bg-red-100 dark:bg-red-900/30'
                    : stats.total_cpu_percent > 50
                      ? 'bg-yellow-100 dark:bg-yellow-900/30'
                      : 'bg-blue-100 dark:bg-blue-900/30'
                  }`}>
                  <Cpu className={`h-6 w-6 ${stats.total_cpu_percent > 80
                      ? 'text-red-600 dark:text-red-400'
                      : stats.total_cpu_percent > 50
                        ? 'text-yellow-600 dark:text-yellow-400'
                        : 'text-blue-600 dark:text-blue-400'
                    }`} />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground mb-1">Memory</p>
                  <p className="text-2xl font-bold">{formatBytes(stats.total_memory_bytes)}</p>
                  {stats.memory_limit_bytes > 0 && (
                    <p className="text-xs text-muted-foreground">{memoryPercent.toFixed(1)}% of {formatBytes(stats.memory_limit_bytes)}</p>
                  )}
                </div>
                <div className={`p-3 rounded-full ${memoryPercent > 80
                    ? 'bg-red-100 dark:bg-red-900/30'
                    : memoryPercent > 50
                      ? 'bg-yellow-100 dark:bg-yellow-900/30'
                      : 'bg-green-100 dark:bg-green-900/30'
                  }`}>
                  <HardDrive className={`h-6 w-6 ${memoryPercent > 80
                      ? 'text-red-600 dark:text-red-400'
                      : memoryPercent > 50
                        ? 'text-yellow-600 dark:text-yellow-400'
                        : 'text-green-600 dark:text-green-400'
                    }`} />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground mb-1">Network I/O</p>
                  <p className="text-sm font-medium text-blue-600">↓ {formatBytes(totalNetworkRx)}</p>
                  <p className="text-sm font-medium text-green-600">↑ {formatBytes(totalNetworkTx)}</p>
                </div>
                <div className="p-3 rounded-full bg-purple-100 dark:bg-purple-900/30">
                  <Network className="h-6 w-6 text-purple-600 dark:text-purple-400" />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground mb-1">Disk I/O</p>
                  <p className="text-sm font-medium text-blue-600">R {formatBytes(totalBlockRead)}</p>
                  <p className="text-sm font-medium text-orange-600">W {formatBytes(totalBlockWrite)}</p>
                </div>
                <div className="p-3 rounded-full bg-orange-100 dark:bg-orange-900/30">
                  <Activity className="h-6 w-6 text-orange-600 dark:text-orange-400" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Overall Usage Progress Bars */}
        <Card>
          <CardHeader>
            <div className="flex justify-between items-center">
              <CardTitle className="text-lg">Overall Usage</CardTitle>
              <span className="text-xs text-muted-foreground">
                Updated {new Date(stats.timestamp).toLocaleTimeString()}
              </span>
            </div>
          </CardHeader>
          <CardContent className="space-y-6">
            <div>
              <div className="flex justify-between mb-2 text-sm">
                <span className="font-medium">CPU</span>
                <span className="text-muted-foreground">{stats.total_cpu_percent.toFixed(2)}%</span>
              </div>
              <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
                <div
                  className={`h-3 rounded-full transition-all ${stats.total_cpu_percent > 80
                      ? 'bg-red-500'
                      : stats.total_cpu_percent > 50
                        ? 'bg-yellow-500'
                        : 'bg-blue-500'
                    }`}
                  style={{ width: `${Math.min(stats.total_cpu_percent, 100)}%` }}
                />
              </div>
            </div>

            <div>
              <div className="flex justify-between mb-2 text-sm">
                <span className="font-medium">Memory</span>
                <span className="text-muted-foreground">
                  {formatBytes(stats.total_memory_bytes)}
                  {stats.memory_limit_bytes > 0 && (
                    <> / {formatBytes(stats.memory_limit_bytes)} ({memoryPercent.toFixed(1)}%)</>
                  )}
                </span>
              </div>
              <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
                <div
                  className={`h-3 rounded-full transition-all ${memoryPercent > 80
                      ? 'bg-red-500'
                      : memoryPercent > 50
                        ? 'bg-yellow-500'
                        : 'bg-green-500'
                    }`}
                  style={{ width: `${Math.min(memoryPercent || 0, 100)}%` }}
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Container Details */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Server className="h-5 w-5 text-primary" />
              Containers ({stats.containers.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {stats.containers.map((container) => {
                const containerMemPercent = container.memory_limit_bytes > 0
                  ? (container.memory_usage_bytes / container.memory_limit_bytes) * 100
                  : 0;

                return (
                  <Card key={container.container_id} className="overflow-hidden">
                    <CardContent className="pt-6">
                      <div className="flex items-start justify-between mb-4">
                        <div className="flex-1 min-w-0">
                          <div className="font-mono text-sm font-medium truncate">
                            {container.container_name}
                          </div>
                          <div className="font-mono text-xs text-muted-foreground truncate">
                            {container.container_id.substring(0, 12)}
                          </div>
                        </div>
                      </div>

                      <div className="space-y-4">
                        {/* CPU */}
                        <div>
                          <div className="flex justify-between mb-1 text-xs">
                            <span className="text-muted-foreground">CPU</span>
                            <span className="font-medium">{container.cpu_percent.toFixed(2)}%</span>
                          </div>
                          <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                            <div
                              className="h-1.5 rounded-full bg-blue-500 transition-all"
                              style={{ width: `${Math.min(container.cpu_percent, 100)}%` }}
                            />
                          </div>
                        </div>

                        {/* Memory */}
                        <div>
                          <div className="flex justify-between mb-1 text-xs">
                            <span className="text-muted-foreground">Memory</span>
                            <span className="font-medium">
                              {formatBytes(container.memory_usage_bytes)}
                              {container.memory_limit_bytes > 0 && (
                                <> ({containerMemPercent.toFixed(1)}%)</>
                              )}
                            </span>
                          </div>
                          <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                            <div
                              className="h-1.5 rounded-full bg-green-500 transition-all"
                              style={{ width: `${Math.min(containerMemPercent || 0, 100)}%` }}
                            />
                          </div>
                        </div>

                        {/* Network & Disk I/O */}
                        <div className="grid grid-cols-2 gap-3 pt-2 border-t border-border">
                          <div>
                            <div className="text-xs text-muted-foreground mb-1">Network I/O</div>
                            <div className="text-xs font-medium space-y-0.5">
                              <div className="text-blue-600">↓ {formatBytes(container.network_rx_bytes)}</div>
                              <div className="text-green-600">↑ {formatBytes(container.network_tx_bytes)}</div>
                            </div>
                          </div>
                          <div>
                            <div className="text-xs text-muted-foreground mb-1">Disk I/O</div>
                            <div className="text-xs font-medium space-y-0.5">
                              <div className="text-blue-600">R {formatBytes(container.block_read_bytes)}</div>
                              <div className="text-orange-600">W {formatBytes(container.block_write_bytes)}</div>
                            </div>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  // Compact embedded view
  return (
    <Card className="p-6">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold">Resource Usage</h3>
        <span className="text-xs text-gray-500">
          Updated {new Date(stats.timestamp).toLocaleTimeString()}
        </span>
      </div>

      {/* Overall metrics */}
      <div className="space-y-4 mb-6">
        <div>
          <div className="flex justify-between mb-1 text-sm">
            <span className="font-medium">CPU</span>
            <span className="text-gray-600">{stats.total_cpu_percent.toFixed(2)}%</span>
          </div>
          <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
            <div
              className={`h-2 rounded-full transition-all ${stats.total_cpu_percent > 80
                  ? 'bg-red-500'
                  : stats.total_cpu_percent > 50
                    ? 'bg-yellow-500'
                    : 'bg-blue-500'
                }`}
              style={{ width: `${Math.min(stats.total_cpu_percent, 100)}%` }}
            />
          </div>
        </div>

        <div>
          <div className="flex justify-between mb-1 text-sm">
            <span className="font-medium">Memory</span>
            <span className="text-gray-600">
              {formatBytes(stats.total_memory_bytes)}
              {stats.memory_limit_bytes > 0 && (
                <>
                  {' / '}
                  {formatBytes(stats.memory_limit_bytes)}
                  {' '}
                  ({memoryPercent.toFixed(1)}%)
                </>
              )}
            </span>
          </div>
          <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
            <div
              className={`h-2 rounded-full transition-all ${memoryPercent > 80
                  ? 'bg-red-500'
                  : memoryPercent > 50
                    ? 'bg-yellow-500'
                    : 'bg-green-500'
                }`}
              style={{ width: `${Math.min(memoryPercent, 100)}%` }}
            />
          </div>
        </div>
      </div>

      {/* Container breakdown */}
      <div>
        <h4 className="font-medium text-sm mb-3">Containers ({stats.containers.length})</h4>
        <div className="space-y-3">
          {stats.containers.map((container) => {
            const containerMemPercent = container.memory_limit_bytes > 0
              ? (container.memory_usage_bytes / container.memory_limit_bytes) * 100
              : 0;

            return (
              <div
                key={container.container_id}
                className="border border-gray-200 dark:border-gray-700 rounded-lg p-3"
              >
                <div className="flex items-start justify-between mb-2">
                  <div className="flex-1 min-w-0">
                    <div className="font-mono text-xs truncate text-gray-900 dark:text-gray-100">
                      {container.container_name}
                    </div>
                    <div className="font-mono text-xs text-gray-400 truncate">
                      {container.container_id.substring(0, 12)}
                    </div>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3 text-xs">
                  <div>
                    <div className="text-gray-500 mb-1">CPU</div>
                    <div className="font-medium">{container.cpu_percent.toFixed(2)}%</div>
                  </div>
                  <div>
                    <div className="text-gray-500 mb-1">Memory</div>
                    <div className="font-medium">
                      {formatBytes(container.memory_usage_bytes)}
                      {container.memory_limit_bytes > 0 && (
                        <span className="text-gray-500">
                          {' '}({containerMemPercent.toFixed(0)}%)
                        </span>
                      )}
                    </div>
                  </div>
                  <div>
                    <div className="text-gray-500 mb-1">Network I/O</div>
                    <div className="font-medium">
                      <span className="text-blue-600">↓ {formatBytes(container.network_rx_bytes)}</span>
                      {' / '}
                      <span className="text-green-600">↑ {formatBytes(container.network_tx_bytes)}</span>
                    </div>
                  </div>
                  <div>
                    <div className="text-gray-500 mb-1">Disk I/O</div>
                    <div className="font-medium">
                      <span className="text-blue-600">R {formatBytes(container.block_read_bytes)}</span>
                      {' / '}
                      <span className="text-orange-600">W {formatBytes(container.block_write_bytes)}</span>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </Card>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
}
