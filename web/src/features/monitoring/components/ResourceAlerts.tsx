import { AlertTriangle, AlertCircle } from 'lucide-react';
import type { SystemStats } from '@/shared/types/api';

interface ResourceAlertsProps {
  stats: SystemStats;
}

function ResourceAlerts({ stats }: ResourceAlertsProps) {
  const alerts: { type: 'critical' | 'warning'; message: string }[] = [];

  // System-level alerts
  if (stats.cpu.usage_percent > 90) {
    alerts.push({ type: 'critical', message: `System CPU usage is critical (${stats.cpu.usage_percent.toFixed(1)}%)` });
  } else if (stats.cpu.usage_percent > 80) {
    alerts.push({ type: 'warning', message: `System CPU usage is high (${stats.cpu.usage_percent.toFixed(1)}%)` });
  }

  if (stats.memory.usage_percent > 95) {
    alerts.push({ type: 'critical', message: `System memory usage is critical (${stats.memory.usage_percent.toFixed(1)}%)` });
  } else if (stats.memory.usage_percent > 85) {
    alerts.push({ type: 'warning', message: `System memory usage is high (${stats.memory.usage_percent.toFixed(1)}%)` });
  }

  if (stats.disk.usage_percent > 95) {
    alerts.push({ type: 'critical', message: `Disk space is critically low (${stats.disk.usage_percent.toFixed(1)}%)` });
  } else if (stats.disk.usage_percent > 85) {
    alerts.push({ type: 'warning', message: `Disk space is running low (${stats.disk.usage_percent.toFixed(1)}%)` });
  }

  // Container-level alerts (only if containers array exists)
  const containers = stats.containers || [];

  const highCPUContainers = containers.filter(
    (c) => c.state === 'running' && c.cpu_percent > 90
  );
  if (highCPUContainers.length > 0) {
    alerts.push({
      type: 'warning',
      message: `${highCPUContainers.length} container${highCPUContainers.length > 1 ? 's' : ''} using high CPU (>90%): ${highCPUContainers.map(c => c.name).join(', ')}`,
    });
  }

  const highMemContainers = containers.filter(
    (c) => c.state === 'running' && c.memory_limit_bytes > 0 && (c.memory_usage_bytes / c.memory_limit_bytes) * 100 > 85
  );
  if (highMemContainers.length > 0) {
    alerts.push({
      type: 'warning',
      message: `${highMemContainers.length} container${highMemContainers.length > 1 ? 's' : ''} using high memory (>85%): ${highMemContainers.map(c => c.name).join(', ')}`,
    });
  }

  const stoppedContainers = containers.filter((c) => c.state === 'stopped');
  if (stoppedContainers.length > 0) {
    alerts.push({
      type: 'warning',
      message: `${stoppedContainers.length} container${stoppedContainers.length > 1 ? 's are' : ' is'} stopped: ${stoppedContainers.map(c => c.name).join(', ')}`,
    });
  }

  const restartedContainers = containers.filter((c) => c.restart_count > 5);
  if (restartedContainers.length > 0) {
    alerts.push({
      type: 'warning',
      message: `${restartedContainers.length} container${restartedContainers.length > 1 ? 's have' : ' has'} restarted multiple times: ${restartedContainers.map(c => `${c.name} (${c.restart_count})`).join(', ')}`,
    });
  }

  if (alerts.length === 0) {
    return null;
  }

  return (
    <div className="space-y-2">
      {alerts.map((alert, index) => (
        <div
          key={index}
          className={`flex items-start gap-2 sm:gap-3 p-3 sm:p-4 rounded-lg border ${alert.type === 'critical'
            ? 'bg-red-50 dark:bg-red-900/10 border-red-200 dark:border-red-900/50'
            : 'bg-yellow-50 dark:bg-yellow-900/10 border-yellow-200 dark:border-yellow-900/50'
            }`}
        >
          {alert.type === 'critical' ? (
            <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
          ) : (
            <AlertTriangle className="h-5 w-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5" />
          )}
          <div className="flex-1 min-w-0">
            <p
              className={`text-xs sm:text-sm font-medium break-words ${alert.type === 'critical'
                ? 'text-red-900 dark:text-red-200'
                : 'text-yellow-900 dark:text-yellow-200'
                }`}
            >
              {alert.message}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}

export default ResourceAlerts;
