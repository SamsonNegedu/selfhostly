import { useState, useEffect } from 'react';
import { Card, CardContent } from '@/shared/components/ui/card';
import { Cpu, HardDrive, Server, Activity, ChevronDown, ChevronUp } from 'lucide-react';
import type { SystemStats } from '@/shared/types/api';

interface SystemOverviewProps {
  stats: SystemStats;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
}

function getColorForUsage(percent: number) {
  if (percent > 85) return 'red';
  if (percent > 65) return 'yellow';
  return 'green';
}

function SystemOverview({ stats }: SystemOverviewProps) {
  const cpuColor = getColorForUsage(stats.cpu.usage_percent);
  const memColor = getColorForUsage(stats.memory.usage_percent);
  const diskColor = getColorForUsage(stats.disk.usage_percent);

  // Collapse state for mobile - persist to localStorage
  const [isCollapsed, setIsCollapsed] = useState(() => {
    const saved = localStorage.getItem('system-resources-collapsed');
    return saved ? JSON.parse(saved) : false;
  });

  useEffect(() => {
    localStorage.setItem('system-resources-collapsed', JSON.stringify(isCollapsed));
  }, [isCollapsed]);

  return (
    <>
      {/* Mobile Grouped View */}
      <Card className="sm:hidden">
        <CardContent className="p-4">
          {/* Header with Toggle */}
          <button
            onClick={() => setIsCollapsed(!isCollapsed)}
            className="w-full flex items-center justify-between mb-3 group"
          >
            <h3 className="text-sm font-semibold text-muted-foreground">System Resources</h3>
            <div className="flex items-center gap-1 text-muted-foreground">
              {!isCollapsed && (
                <span className="text-xs opacity-0 group-hover:opacity-100 transition-opacity">Hide</span>
              )}
              {isCollapsed ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronUp className="h-4 w-4" />
              )}
            </div>
          </button>

          {/* Collapsed Summary */}
          {isCollapsed && (
            <div className="flex items-center justify-around py-2 text-xs animate-in fade-in duration-200">
              <div className="text-center">
                <p className="font-bold text-base">{stats.cpu.usage_percent.toFixed(0)}%</p>
                <p className="text-muted-foreground">CPU</p>
              </div>
              <div className="text-center">
                <p className="font-bold text-base">{formatBytes(stats.memory.used_bytes)}</p>
                <p className="text-muted-foreground">Memory</p>
              </div>
              <div className="text-center">
                <p className="font-bold text-base">{stats.docker.running}/{stats.docker.total_containers}</p>
                <p className="text-muted-foreground">Containers</p>
              </div>
            </div>
          )}
          
          {/* Collapsible Content */}
          {!isCollapsed && (
          <div className="space-y-4 animate-in fade-in slide-in-from-top-2 duration-200">
            {/* CPU */}
            <div className="flex items-center gap-3">
              <div
                className={`p-2 rounded-lg flex-shrink-0 ${
                  cpuColor === 'red'
                    ? 'bg-red-100 dark:bg-red-900/30'
                    : cpuColor === 'yellow'
                    ? 'bg-yellow-100 dark:bg-yellow-900/30'
                    : 'bg-blue-100 dark:bg-blue-900/30'
                }`}
              >
                <Cpu
                  className={`h-5 w-5 ${
                    cpuColor === 'red'
                      ? 'text-red-600 dark:text-red-400'
                      : cpuColor === 'yellow'
                      ? 'text-yellow-600 dark:text-yellow-400'
                      : 'text-blue-600 dark:text-blue-400'
                  }`}
                />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-baseline justify-between mb-1">
                  <p className="text-xs text-muted-foreground">CPU Usage</p>
                  <p className="text-lg font-bold">{stats.cpu.usage_percent.toFixed(1)}%</p>
                </div>
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                  <div
                    className={`h-1.5 rounded-full transition-all ${
                      cpuColor === 'red'
                        ? 'bg-red-500'
                        : cpuColor === 'yellow'
                        ? 'bg-yellow-500'
                        : 'bg-blue-500'
                    }`}
                    style={{ width: `${Math.min(stats.cpu.usage_percent, 100)}%` }}
                  />
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">{stats.cpu.cores} cores</p>
              </div>
            </div>

            {/* Memory */}
            <div className="flex items-center gap-3">
              <div
                className={`p-2 rounded-lg flex-shrink-0 ${
                  memColor === 'red'
                    ? 'bg-red-100 dark:bg-red-900/30'
                    : memColor === 'yellow'
                    ? 'bg-yellow-100 dark:bg-yellow-900/30'
                    : 'bg-green-100 dark:bg-green-900/30'
                }`}
              >
                <Activity
                  className={`h-5 w-5 ${
                    memColor === 'red'
                      ? 'text-red-600 dark:text-red-400'
                      : memColor === 'yellow'
                      ? 'text-yellow-600 dark:text-yellow-400'
                      : 'text-green-600 dark:text-green-400'
                  }`}
                />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-baseline justify-between mb-1">
                  <p className="text-xs text-muted-foreground">Memory</p>
                  <p className="text-lg font-bold">{formatBytes(stats.memory.used_bytes)}</p>
                </div>
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                  <div
                    className={`h-1.5 rounded-full transition-all ${
                      memColor === 'red'
                        ? 'bg-red-500'
                        : memColor === 'yellow'
                        ? 'bg-yellow-500'
                        : 'bg-green-500'
                    }`}
                    style={{ width: `${Math.min(stats.memory.usage_percent, 100)}%` }}
                  />
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">of {formatBytes(stats.memory.total_bytes)}</p>
              </div>
            </div>

            {/* Disk */}
            <div className="flex items-center gap-3">
              <div
                className={`p-2 rounded-lg flex-shrink-0 ${
                  diskColor === 'red'
                    ? 'bg-red-100 dark:bg-red-900/30'
                    : diskColor === 'yellow'
                    ? 'bg-yellow-100 dark:bg-yellow-900/30'
                    : 'bg-purple-100 dark:bg-purple-900/30'
                }`}
              >
                <HardDrive
                  className={`h-5 w-5 ${
                    diskColor === 'red'
                      ? 'text-red-600 dark:text-red-400'
                      : diskColor === 'yellow'
                      ? 'text-yellow-600 dark:text-yellow-400'
                      : 'text-purple-600 dark:text-purple-400'
                  }`}
                />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-baseline justify-between mb-1">
                  <p className="text-xs text-muted-foreground">Disk Space</p>
                  <p className="text-lg font-bold">{formatBytes(stats.disk.used_bytes)}</p>
                </div>
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                  <div
                    className={`h-1.5 rounded-full transition-all ${
                      diskColor === 'red'
                        ? 'bg-red-500'
                        : diskColor === 'yellow'
                        ? 'bg-yellow-500'
                        : 'bg-purple-500'
                    }`}
                    style={{ width: `${Math.min(stats.disk.usage_percent, 100)}%` }}
                  />
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">of {formatBytes(stats.disk.total_bytes)}</p>
              </div>
            </div>

            {/* Docker - Different layout */}
            <div className="pt-3 border-t">
              <div className="flex items-center gap-3 mb-3">
                <div className="p-2 rounded-lg flex-shrink-0 bg-blue-100 dark:bg-blue-900/30">
                  <Server className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                </div>
                <div className="flex-1">
                  <p className="text-xs text-muted-foreground">Docker Containers</p>
                  <p className="text-lg font-bold">{stats.docker.running} / {stats.docker.total_containers}</p>
                </div>
              </div>
              <div className="grid grid-cols-3 gap-2 text-center">
                <div className="bg-muted/30 rounded-lg py-2">
                  <p className="text-base font-bold text-green-600 dark:text-green-400">
                    {stats.docker.running}
                  </p>
                  <p className="text-xs text-muted-foreground">Running</p>
                </div>
                <div className="bg-muted/30 rounded-lg py-2">
                  <p className="text-base font-bold">{stats.docker.stopped}</p>
                  <p className="text-xs text-muted-foreground">Stopped</p>
                </div>
                <div className="bg-muted/30 rounded-lg py-2">
                  <p className="text-base font-bold">{stats.docker.images}</p>
                  <p className="text-xs text-muted-foreground">Images</p>
                </div>
              </div>
            </div>
          </div>
          )}
        </CardContent>
      </Card>

      {/* Desktop Individual Cards */}
      <div className="hidden sm:grid sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
        {/* CPU Card */}
        <Card>
          <CardContent className="pt-4 sm:pt-6 p-4 sm:p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs sm:text-sm text-muted-foreground mb-1">CPU Usage</p>
                <p className="text-xl sm:text-2xl font-bold">{stats.cpu.usage_percent.toFixed(1)}%</p>
                <p className="text-xs text-muted-foreground mt-0.5 sm:mt-1">{stats.cpu.cores} cores</p>
              </div>
              <div
                className={`p-2 sm:p-3 rounded-full ${
                  cpuColor === 'red'
                    ? 'bg-red-100 dark:bg-red-900/30'
                    : cpuColor === 'yellow'
                    ? 'bg-yellow-100 dark:bg-yellow-900/30'
                    : 'bg-blue-100 dark:bg-blue-900/30'
                }`}
              >
                <Cpu
                  className={`h-6 w-6 ${
                    cpuColor === 'red'
                      ? 'text-red-600 dark:text-red-400'
                      : cpuColor === 'yellow'
                      ? 'text-yellow-600 dark:text-yellow-400'
                      : 'text-blue-600 dark:text-blue-400'
                  }`}
                />
              </div>
            </div>
            {/* Progress bar */}
            <div className="mt-4 w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
              <div
                className={`h-1.5 sm:h-2 rounded-full transition-all ${
                  cpuColor === 'red'
                    ? 'bg-red-500'
                    : cpuColor === 'yellow'
                    ? 'bg-yellow-500'
                    : 'bg-blue-500'
                }`}
                style={{ width: `${Math.min(stats.cpu.usage_percent, 100)}%` }}
              />
            </div>
          </CardContent>
        </Card>

        {/* Memory Card */}
        <Card>
          <CardContent className="pt-4 sm:pt-6 p-4 sm:p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs sm:text-sm text-muted-foreground mb-1">Memory</p>
                <p className="text-xl sm:text-2xl font-bold">{formatBytes(stats.memory.used_bytes)}</p>
                <p className="text-xs text-muted-foreground mt-0.5 sm:mt-1">
                  of {formatBytes(stats.memory.total_bytes)}
                </p>
              </div>
              <div
                className={`p-2 sm:p-3 rounded-full ${
                  memColor === 'red'
                    ? 'bg-red-100 dark:bg-red-900/30'
                    : memColor === 'yellow'
                    ? 'bg-yellow-100 dark:bg-yellow-900/30'
                    : 'bg-green-100 dark:bg-green-900/30'
                }`}
              >
                <Activity
                  className={`h-6 w-6 ${
                    memColor === 'red'
                      ? 'text-red-600 dark:text-red-400'
                      : memColor === 'yellow'
                      ? 'text-yellow-600 dark:text-yellow-400'
                      : 'text-green-600 dark:text-green-400'
                  }`}
                />
              </div>
            </div>
            {/* Progress bar */}
            <div className="mt-3 sm:mt-4 w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5 sm:h-2">
              <div
                className={`h-1.5 sm:h-2 rounded-full transition-all ${
                  memColor === 'red'
                    ? 'bg-red-500'
                    : memColor === 'yellow'
                    ? 'bg-yellow-500'
                    : 'bg-green-500'
                }`}
                style={{ width: `${Math.min(stats.memory.usage_percent, 100)}%` }}
              />
            </div>
          </CardContent>
        </Card>

        {/* Disk Card */}
        <Card>
          <CardContent className="pt-4 sm:pt-6 p-4 sm:p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs sm:text-sm text-muted-foreground mb-1">Disk Space</p>
                <p className="text-xl sm:text-2xl font-bold">{formatBytes(stats.disk.used_bytes)}</p>
                <p className="text-xs text-muted-foreground mt-0.5 sm:mt-1">
                  of {formatBytes(stats.disk.total_bytes)}
                </p>
              </div>
              <div
                className={`p-2 sm:p-3 rounded-full ${
                  diskColor === 'red'
                    ? 'bg-red-100 dark:bg-red-900/30'
                    : diskColor === 'yellow'
                    ? 'bg-yellow-100 dark:bg-yellow-900/30'
                    : 'bg-purple-100 dark:bg-purple-900/30'
                }`}
              >
                <HardDrive
                  className={`h-6 w-6 ${
                    diskColor === 'red'
                      ? 'text-red-600 dark:text-red-400'
                      : diskColor === 'yellow'
                      ? 'text-yellow-600 dark:text-yellow-400'
                      : 'text-purple-600 dark:text-purple-400'
                  }`}
                />
              </div>
            </div>
            {/* Progress bar */}
            <div className="mt-3 sm:mt-4 w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5 sm:h-2">
              <div
                className={`h-1.5 sm:h-2 rounded-full transition-all ${
                  diskColor === 'red'
                    ? 'bg-red-500'
                    : diskColor === 'yellow'
                    ? 'bg-yellow-500'
                    : 'bg-purple-500'
                }`}
                style={{ width: `${Math.min(stats.disk.usage_percent, 100)}%` }}
              />
            </div>
          </CardContent>
        </Card>

        {/* Docker Card */}
        <Card>
          <CardContent className="pt-4 sm:pt-6 p-4 sm:p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs sm:text-sm text-muted-foreground mb-1">Docker</p>
                <p className="text-xl sm:text-2xl font-bold">{stats.docker.running}</p>
                <p className="text-xs text-muted-foreground mt-0.5 sm:mt-1">
                  of {stats.docker.total_containers} containers
                </p>
              </div>
              <div className="p-2 sm:p-3 rounded-full bg-blue-100 dark:bg-blue-900/30">
                <Server className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              </div>
            </div>
            {/* Stats breakdown */}
            <div className="mt-3 sm:mt-4 grid grid-cols-3 gap-2 text-xs">
              <div className="text-center">
                <p className="font-medium text-green-600 dark:text-green-400">
                  {stats.docker.running}
                </p>
                <p className="text-muted-foreground">Running</p>
              </div>
              <div className="text-center">
                <p className="font-medium text-gray-600 dark:text-gray-400">{stats.docker.stopped}</p>
                <p className="text-muted-foreground">Stopped</p>
              </div>
              <div className="text-center">
                <p className="font-medium text-gray-600 dark:text-gray-400">{stats.docker.images}</p>
                <p className="text-muted-foreground">Images</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </>
  );
}

export default SystemOverview;
