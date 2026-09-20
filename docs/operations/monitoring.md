# System Monitoring Dashboard

## Overview

The monitoring page shows CPU, memory, disk and every container across all your nodes in one place, so you do not need to SSH in and run `htop`. Open it at `/monitoring`.

## Features

### System Overview
- **CPU Usage**: Real-time CPU utilization percentage across all cores
- **Memory Usage**: Current memory consumption with available/total breakdown
- **Disk Space**: Disk usage with free space remaining
- **Docker Stats**: Container counts (running/stopped/paused) and image count

### Container Monitoring
- **All Containers View**: See every container across all apps in one place
- **Resource Metrics**: CPU, memory, network I/O, and disk I/O per container
- **Container State**: Visual badges showing running/stopped/paused status
- **Restart Tracking**: See how many times each container has restarted

### Real-time Updates
- Auto-refreshes every 10 seconds
- Pauses when browser tab is not visible (saves resources)
- Shows "Updated X seconds ago" timestamp

### Search & Filtering
- Search by container name, app name, or container ID
- Filter by status (All, Running, Stopped)
- Results update instantly as you type

### Resource Alerts
Automatic alerts for:
- System CPU > 90% (critical) or > 80% (warning)
- System Memory > 95% (critical) or > 85% (warning)
- Disk Space > 95% (critical) or > 85% (warning)
- Containers using > 90% CPU
- Containers using > 85% memory
- Stopped containers
- Containers with high restart counts (> 5 restarts)

### Quick Actions
- **Restart Container**: Restart any running container with confirmation
- **Stop Container**: Stop any running container with confirmation
- Immediate feedback with success/error notifications

## API Endpoints

### GET /api/system/stats
Returns system statistics: CPU, memory, disk, Docker daemon info and every container's metrics.

- `?node_ids=all` (the default) or `?node_ids=id1,id2` chooses the nodes. The nodes are queried in parallel.
- The response is an **array with one object per node**; a node that is offline or unreachable is left out. One element
  is shown below. Linked nodes are queried through their connection, so they need no reachable address.

**Response element:**
```json
{
  "node_id": "raspberrypi",
  "node_name": "raspberrypi",
  "cpu": {
    "usage_percent": 45.2,
    "cores": 4
  },
  "memory": {
    "total_bytes": 8589934592,
    "used_bytes": 4294967296,
    "free_bytes": 4294967296,
    "available_bytes": 5368709120,
    "usage_percent": 50.0
  },
  "disk": {
    "total_bytes": 128849018880,
    "used_bytes": 51539607552,
    "free_bytes": 77309411328,
    "usage_percent": 40.0,
    "path": "/"
  },
  "docker": {
    "total_containers": 12,
    "running": 8,
    "stopped": 4,
    "paused": 0,
    "images": 15,
    "version": "24.0.7"
  },
  "containers": [
    {
      "id": "abc123...",
      "name": "myapp-web-1",
      "app_name": "myapp",
      "status": "running",
      "state": "running",
      "cpu_percent": 12.5,
      "memory_usage_bytes": 536870912,
      "memory_limit_bytes": 2147483648,
      "network_rx_bytes": 1048576,
      "network_tx_bytes": 524288,
      "block_read_bytes": 10485760,
      "block_write_bytes": 5242880,
      "created_at": "2024-01-15T10:30:00Z",
      "restart_count": 0
    }
  ],
  "timestamp": "2024-01-15T14:30:00Z"
}
```

### POST /api/system/containers/:id/restart
Restarts a specific container by ID.

**Response:**
```json
{
  "message": "Container restarted successfully",
  "container_id": "abc123..."
}
```

### POST /api/system/containers/:id/stop
Stops a specific container by ID.

**Response:**
```json
{
  "message": "Container stopped successfully",
  "container_id": "abc123..."
}
```

## Architecture

### Backend
- **`internal/system/stats.go`**: System metrics collector using gopsutil
- **`internal/http/system.go`**: HTTP handlers for monitoring endpoints
- **`internal/docker/manager.go`**: Container control methods (restart/stop)

### Frontend
- **`web/src/features/monitoring/`**: Main monitoring page
- **`web/src/features/monitoring/components/`**: Reusable monitoring components
  - `SystemOverview.tsx`: System-level metrics cards
  - `ContainersTable.tsx`: Container list with metrics
  - `ContainerActions.tsx`: Quick action buttons
  - `ResourceAlerts.tsx`: Alert banners

## Usage

1. **Access**: Navigate to `/monitoring` in the web interface
2. **View System Health**: Check the overview cards for CPU, memory, disk, and Docker stats
3. **Monitor Containers**: Scroll to see all containers across all apps
4. **Search**: Use the search bar to find specific containers
5. **Take Action**: Click restart/stop buttons on any container
6. **Check Alerts**: Review any alerts at the top of the page

## Troubleshooting

### No containers showing up
- Docker must be running (`docker ps`) and apps must be deployed.
- The backend needs the Docker socket: `selfhostlyctl doctor` checks it and names the fix.

### Every container shows 0 MB of memory (Raspberry Pi)
The kernel does not account container memory on a Pi unless asked to. `selfhostlyctl check --fix` offers the change to the boot
command line (it needs a reboot).

### A node is missing
Offline and unreachable nodes are left out of the stats. Check **Settings → Nodes**; a linked node returns by itself when its
connection is back ([multi-node.md](multi-node.md)).

### Stats not updating
- Check browser console for API errors
- Verify `/api/system/stats` endpoint is accessible
- Ensure authentication is working

### High CPU on stats collection
- Stats collection runs on-demand only when page is open
- Collection pauses when tab is not visible
- Consider increasing refresh interval if needed

### Permission errors (gopsutil)
- gopsutil may need elevated permissions on some systems
- CPU/memory stats generally work without sudo
- Disk stats may require read permissions on mount points
