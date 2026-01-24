# System Monitoring Dashboard

## Overview

The monitoring dashboard provides comprehensive real-time visibility into your self-hosted infrastructure, eliminating the need to SSH into your Pi and run htop.

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
Returns comprehensive system statistics including CPU, memory, disk, Docker daemon info, and all container metrics.

**Response:**
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

## Multi-Node Readiness

The architecture is designed to support multiple Raspberry Pis in the future:

### Current (Phase 1)
- All stats report from local node (hostname as node_id)
- Single API endpoint for system stats
- Container actions execute locally

### Future (Phase 2)
When adding more Pis:
1. Add `nodes` table to database
2. Implement agent/heartbeat system for worker nodes
3. Add node switcher in UI
4. Aggregate stats across all nodes

**Migration Path:**
- `node_id` and `node_name` fields already included in SystemStats
- API can be extended to accept `?node=pi-worker-2` parameter
- Frontend components designed to work with single or multiple nodes

## Usage

1. **Access**: Navigate to `/monitoring` in the web interface
2. **View System Health**: Check the overview cards for CPU, memory, disk, and Docker stats
3. **Monitor Containers**: Scroll to see all containers across all apps
4. **Search**: Use the search bar to find specific containers
5. **Take Action**: Click restart/stop buttons on any container
6. **Check Alerts**: Review any alerts at the top of the page

## Performance

- Stats collection completes in < 500ms on Raspberry Pi 4
- Frontend only polls when tab is visible
- Minimal overhead on system resources
- Container list handles 50+ containers efficiently

## Dependencies

### Backend
- `github.com/shirou/gopsutil/v3`: Cross-platform system stats library
  - Provides CPU, memory, and disk usage
  - Works on Linux, macOS, Windows

### Frontend
- Uses existing React Query for data fetching
- Uses existing UI components (Cards, Badges, Buttons)
- No additional dependencies required

## Testing

To test the monitoring dashboard:

1. Start the backend server:
   ```bash
   make dev-server
   ```

2. Start the frontend dev server:
   ```bash
   cd web && npm run dev
   ```

3. Navigate to `http://localhost:5173/monitoring`

4. Verify:
   - System stats display correctly
   - Containers show up with metrics
   - Search and filtering work
   - Container actions (restart/stop) work with confirmation
   - Alerts appear when resource usage is high
   - Auto-refresh updates data every 10 seconds

## Troubleshooting

### No containers showing up
- Ensure Docker is running
- Check that apps have been deployed
- Verify docker-compose.yml files exist in app directories

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

## Future Enhancements

Potential additions for Phase 2:
- Historical metrics (store last hour/day of data)
- Graphs and charts for CPU/memory over time
- Email/webhook alerts for critical issues
- Container log streaming in monitoring view
- Batch container operations (restart all, stop all)
- Custom alert thresholds in settings
- Export metrics to Prometheus/Grafana
