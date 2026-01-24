import { Card, CardContent } from '@/shared/components/ui/card';
import { Cpu, HardDrive, Server, Activity } from 'lucide-react';
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

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {/* CPU Card */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground mb-1">CPU Usage</p>
              <p className="text-2xl font-bold">{stats.cpu.usage_percent.toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground mt-1">{stats.cpu.cores} cores</p>
            </div>
            <div
              className={`p-3 rounded-full ${
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
              className={`h-2 rounded-full transition-all ${
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
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground mb-1">Memory</p>
              <p className="text-2xl font-bold">{formatBytes(stats.memory.used_bytes)}</p>
              <p className="text-xs text-muted-foreground mt-1">
                of {formatBytes(stats.memory.total_bytes)}
              </p>
            </div>
            <div
              className={`p-3 rounded-full ${
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
          <div className="mt-4 w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
            <div
              className={`h-2 rounded-full transition-all ${
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
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground mb-1">Disk Space</p>
              <p className="text-2xl font-bold">{formatBytes(stats.disk.used_bytes)}</p>
              <p className="text-xs text-muted-foreground mt-1">
                of {formatBytes(stats.disk.total_bytes)}
              </p>
            </div>
            <div
              className={`p-3 rounded-full ${
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
          <div className="mt-4 w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
            <div
              className={`h-2 rounded-full transition-all ${
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
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground mb-1">Docker</p>
              <p className="text-2xl font-bold">{stats.docker.running}</p>
              <p className="text-xs text-muted-foreground mt-1">
                of {stats.docker.total_containers} containers
              </p>
            </div>
            <div className="p-3 rounded-full bg-blue-100 dark:bg-blue-900/30">
              <Server className="h-6 w-6 text-blue-600 dark:text-blue-400" />
            </div>
          </div>
          {/* Stats breakdown */}
          <div className="mt-4 grid grid-cols-3 gap-2 text-xs">
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
  );
}

export default SystemOverview;
