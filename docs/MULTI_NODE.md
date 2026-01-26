# Multi-Node Architecture

This document covers Selfhostly's multi-node architecture, authentication strategies, setup procedures, and operational details.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Authentication Strategies](#authentication-strategies)
- [Setup Guide](#setup-guide)
- [Health Check System](#health-check-system)
- [Troubleshooting](#troubleshooting)

## Overview

Selfhostly supports distributed deployments with one **primary node** and multiple **secondary nodes**. This allows you to:

- Manage applications across multiple servers from a single UI
- Distribute workloads geographically or by purpose
- Maintain centralized configuration and monitoring
- Scale horizontally by adding more nodes

### Key Concepts

- **Primary Node**: Central management server with the UI and database
- **Secondary Nodes**: Worker nodes that run applications and report to primary
- **Node Authentication**: API key-based authentication for inter-node communication
- **User Authentication**: GitHub OAuth or Cloudflare Zero Trust for UI access
- **Health Checks**: Automatic monitoring with exponential backoff
- **Heartbeats**: Secondary nodes proactively announce they're online

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Primary Node                            │
│  ┌────────────┐  ┌──────────┐  ┌────────────┐  ┌────────────┐ │
│  │   Web UI   │  │ Database │  │ Health     │  │  Cloudflare│ │
│  │  (React)   │  │ (SQLite) │  │ Checker    │  │  Tunnels   │ │
│  └────────────┘  └──────────┘  └────────────┘  └────────────┘ │
│         │              │               │               │        │
│         └──────────────┴───────────────┴───────────────┘        │
│                         HTTP API                                │
└───────────────────────────┬─────────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            │               │               │
    ┌───────▼──────┐ ┌─────▼──────┐ ┌─────▼──────┐
    │  Secondary   │ │ Secondary  │ │ Secondary  │
    │   Node #1    │ │  Node #2   │ │  Node #3   │
    │              │ │            │ │            │
    │  - Docker    │ │ - Docker   │ │ - Docker   │
    │  - Apps      │ │ - Apps     │ │ - Apps     │
    │  - Stats     │ │ - Stats    │ │ - Stats    │
    │  - Heartbeat │ │ - Heartbeat│ │ - Heartbeat│
    └──────────────┘ └────────────┘ └────────────┘
```

### Communication Patterns

1. **Primary → Secondary**: Health checks, app deployment, stats fetching
2. **Secondary → Primary**: Heartbeats, self-reporting status
3. **User → Primary**: UI access, management operations
4. **Primary ↔ Cloudflare**: Tunnel management, DNS updates

## Authentication Strategies

Selfhostly uses different authentication mechanisms for different purposes:

### 1. Node-to-Node Authentication (API Keys)

**Purpose**: Secure communication between nodes in the cluster.

**Mechanism**: Each node has a unique API key. When nodes communicate, they include:
- `X-Node-ID`: The calling node's unique identifier
- `X-Node-API-Key`: The calling node's API key

**Validation Flow**:

```go
// Secondary sends heartbeat to primary
POST /api/internal/nodes/{id}/heartbeat
Headers:
  X-Node-ID: abc-123-def-456
  X-Node-API-Key: xyz789secretkey

// Primary validates:
1. Look up node by ID in database
2. Compare provided API key with stored key
3. Accept if match, reject if mismatch
```

**Endpoints Protected by Node Auth**:
- `/api/internal/nodes/:id/heartbeat` - Node heartbeats
- `/api/internal/apps` - App operations
- `/api/internal/system/stats` - System statistics
- `/api/internal/settings` - Settings sync
- `/api/internal/cloudflare/tunnels` - Tunnel info

### 2. User Authentication (GitHub OAuth or Cloudflare Zero Trust)

**Purpose**: Secure access to the web UI and management operations.

**Option A: Cloudflare Zero Trust (Recommended)**

Deploy behind Cloudflare Access and disable built-in authentication:

```bash
AUTH_ENABLED=false
```

Cloudflare handles authentication at the edge - simpler and more secure!

**Option B: GitHub OAuth with Username Whitelist**

Enable GitHub authentication with user whitelist:

```bash
AUTH_ENABLED=true
JWT_SECRET=your-strong-random-secret-at-least-32-characters
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_ALLOWED_USERS=alice,bob,charlie
NODE_API_ENDPOINT=https://your-primary-node.com
```

**Endpoints Protected by User Auth**:
- `/api/apps` - App management
- `/api/nodes` - Node management
- `/api/settings` - Settings management
- `/api/system/stats` - System monitoring
- All UI routes

### 3. No Authentication (Internal Endpoints)

Some endpoints are public for operational purposes:
- `/api/health` - Health check endpoint
- `/auth/*` - OAuth callback endpoints
- `/avatar/*` - User avatars

## Setup Guide

### Prerequisites

- Go 1.21+ and Node.js 18+ (for building from source)
- Docker and Docker Compose installed on all nodes
- Network connectivity between nodes
- (Optional) GitHub OAuth app credentials
- (Optional) Cloudflare account with API token

### Step 1: Generate API Keys

Generate a unique API key for each node:

```bash
# For each node, generate a secure random key
openssl rand -base64 32
```

Save these keys securely - you'll need them during setup.

**Example Output**:
```
Primary:   Kx9mP2vQ8wR5tY7uI0oP3aS4dF6gH8jK9lZ1xC2vB4n=
Secondary1: Mq3nR5vT9xS7yU1jL4oQ7bS9dG2hJ5kM8pZ3xD6wC9n=
Secondary2: Np7qS9wU3zT5yV8kM2oR5cT7eH4jL7nP1qZ6xE9xD2o=
```

### Step 2: Setup Primary Node

**1. Configure Environment**

Create `.env` file:

```bash
# Application
APP_ENV=production
SERVER_ADDRESS=:8080
DATABASE_PATH=./data/selfhostly.db

# Authentication (choose one)
# Option A: Cloudflare Zero Trust (recommended)
AUTH_ENABLED=false

# Option B: GitHub OAuth
# AUTH_ENABLED=true
# JWT_SECRET=your-strong-random-secret-32-chars-min
# GITHUB_CLIENT_ID=your_github_client_id
# GITHUB_CLIENT_SECRET=your_github_client_secret
# GITHUB_ALLOWED_USERS=alice,bob
# NODE_API_ENDPOINT=https://primary.example.com

# Node Configuration
NODE_IS_PRIMARY=true
NODE_NAME=primary
NODE_API_ENDPOINT=http://192.168.1.10:8080  # This node's reachable URL
NODE_API_KEY=Kx9mP2vQ8wR5tY7uI0oP3aS4dF6gH8jK9lZ1xC2vB4n=

# Cloudflare (optional)
CLOUDFLARE_API_TOKEN=your_token
CLOUDFLARE_ACCOUNT_ID=your_account_id
```

**2. Start Primary Node**

```bash
# Using Docker Compose
docker-compose up -d

# Or build and run directly
go build -o selfhostly ./cmd/server
./selfhostly
```

**3. Access UI**

Navigate to `http://your-primary-ip:8080` and log in.

### Step 3: Setup Secondary Nodes

**1. Configure Environment**

Create `.env` file on secondary node:

```bash
# Application
APP_ENV=production
SERVER_ADDRESS=:8080
DATABASE_PATH=./data/selfhostly.db

# Node Configuration - SECONDARY
NODE_IS_PRIMARY=false
NODE_NAME=worker-1
NODE_API_ENDPOINT=http://192.168.1.50:8080  # This secondary's reachable URL
NODE_API_KEY=Mq3nR5vT9xS7yU1jL4oQ7bS9dG2hJ5kM8pZ3xD6wC9n=

# Primary Node Connection
PRIMARY_NODE_URL=http://192.168.1.10:8080

# DO NOT set Cloudflare vars - synced from primary
# DO NOT set authentication - not needed on secondary
```

**Important Notes**:
- `NODE_API_ENDPOINT`: This node's reachable URL for inter-node communication
- `NODE_API_KEY`: This secondary's own API key (for authentication)
- `PRIMARY_NODE_URL`: Must be reachable from this secondary node
- `PRIMARY_NODE_API_KEY`: Currently unused, reserved for future features
- Secondary nodes sync Cloudflare credentials from primary automatically

**2. Start Secondary Node**

```bash
# Using Docker Compose
docker-compose up -d

# Or build and run directly
go build -o selfhostly ./cmd/server
./selfhostly
```

**3. Verify Startup Heartbeat**

Check logs for successful heartbeat:

```bash
# Should see:
INFO sending startup heartbeat to primary url=http://192.168.1.10:8080/api/internal/nodes/{id}/heartbeat
INFO startup heartbeat sent successfully to primary
```

### Step 4: Register Secondary on Primary

**1. Navigate to Nodes Page**

In the primary UI, go to: **Settings → Nodes**

**2. Click "Register Node"**

**3. Fill in Node Details**:
- **Node ID**: `032d3f54-41d1-4733-a9ff-0eb19f28970e` (from secondary's startup logs)
- **Name**: `worker-1` (must match `NODE_NAME` in secondary's .env)
- **API Endpoint**: `http://192.168.1.50:8080` (secondary's reachable URL)
- **API Key**: `Mq3nR5vT9xS7yU1jL4oQ7bS9dG2hJ5kM8pZ3xD6wC9n=` (from secondary's .env)

**Critical**: The Node ID and API key entered here MUST match what's in the secondary node for heartbeat authentication to work!

**4. Submit**

The primary will:
1. Store the node in the database
2. Perform an initial health check
3. Mark the node as "online" if reachable

**5. Verify Registration**

The node should appear in the nodes list with a green "Online" badge.

### Step 5: Verify Multi-Node Setup

**1. Check Node Status**

Navigate to **Settings → Nodes** - all nodes should show "Online"

**2. Deploy Test App**

Create an app and select a secondary node as the target:

```yaml
version: '3.8'
services:
  nginx:
    image: nginx:alpine
    ports:
      - "8081:80"
```

**3. Verify Monitoring**

Go to **Monitoring** - you should see stats from all nodes.

## Health Check System

The primary node continuously monitors secondary nodes using a smart health check system with exponential backoff.

### How It Works

**1. Periodic Background Checks**

The primary runs health checks every **30 seconds** in the background:

```go
// On server startup:
go runPeriodicHealthChecks()

// Every 30 seconds:
healthCheckAllNodes()
```

**2. Exponential Backoff**

To reduce unnecessary checks on persistently down nodes, the system uses exponential backoff:

| Consecutive Failures | Check Interval | Status        |
|---------------------|----------------|---------------|
| 0-2                 | Every 30s      | "online"      |
| 3-5                 | Every 2 min    | "offline"     |
| 6-10                | Every 5 min    | "offline"     |
| 11-20               | Every 15 min   | "unreachable" |
| 21+                 | Every 30 min   | "unreachable" |

**3. Health Check Process**

```go
1. Check if node is local (primary)
   → Mark as "online" immediately
   
2. Check if enough time has passed (exponential backoff)
   → Skip if checked too recently
   
3. Send GET request to: /api/internal/system/stats
   → Include node auth headers
   
4. On Success:
   → Reset consecutive_failures to 0
   → Set status to "online"
   → Update last_health_check
   
5. On Failure:
   → Increment consecutive_failures
   → Update status based on failure count
   → Update last_health_check
```

**4. Manual Health Checks**

You can trigger a manual check from the UI:

```
POST /api/nodes/:id/check
```

This bypasses exponential backoff for immediate verification.

### Heartbeat System

Secondary nodes proactively announce they're online using heartbeats.

**1. Startup Heartbeat**

When a secondary node starts, it sends a heartbeat to the primary after 2 seconds:

```go
// Secondary sends:
POST /api/internal/nodes/{id}/heartbeat
Headers:
  X-Node-ID: {secondary-id}
  X-Node-API-Key: {secondary-api-key}

// Primary responds:
1. Validates authentication
2. Resets consecutive_failures to 0
3. Sets status to "online"
4. Updates last_seen timestamp
5. Updates last_health_check timestamp
```

**2. Benefits**

- **Immediate Status Update**: Node appears online instantly, no need to wait for next health check
- **Reduced Load**: Primary doesn't need to check as frequently
- **Better UX**: Faster feedback when nodes come back online

**3. Future Enhancement**

Secondary nodes could send periodic heartbeats (every 60s) to maintain online status even longer between health checks.

## Troubleshooting

### Node Shows as Offline

**Symptoms**: Secondary node is running but shows "offline" on primary.

**Check 1: Network Connectivity**

```bash
# From primary, test connection to secondary
curl http://secondary-ip:8080/api/health

# Should return:
{"status":"healthy","service":"selfhostly"}
```

**Check 2: API Key Mismatch**

```bash
# On secondary, check NODE_API_KEY
grep NODE_API_KEY .env

# On primary UI, verify the registered API key matches exactly
# Settings → Nodes → [node] → Edit
```

**Check 3: Firewall**

```bash
# Ensure port 8080 is open on secondary
sudo ufw status
sudo ufw allow 8080/tcp
```

**Check 4: Heartbeat Logs**

```bash
# On secondary, check logs for heartbeat errors
docker-compose logs | grep heartbeat

# Look for:
# ✅ "startup heartbeat sent successfully"
# ❌ "heartbeat failed" - check authentication
# ❌ "failed to send heartbeat" - check network
```

### Authentication Errors

**Symptoms**: "Invalid API key" or "Unauthorized" errors in logs.

**Solution**:

1. Regenerate API key on secondary:
```bash
openssl rand -base64 32
```

2. Update secondary's `.env`:
```bash
NODE_API_KEY=new-key-here
```

3. Update registered node on primary UI with same key

4. Restart secondary:
```bash
docker-compose restart
```

### Health Checks Not Running

**Symptoms**: Nodes never update status, stuck on initial state.

**Check 1: Primary Node Logs**

```bash
# Should see periodic health check logs
docker-compose logs | grep "health check"

# Look for:
# ✅ "background tasks started" health_check_interval=30s
# ✅ "health check completed"
# ❌ "health check failed" - check error details
```

**Check 2: Database Issues**

```bash
# Check if database is accessible
ls -lh data/selfhostly.db

# Check permissions
chmod 644 data/selfhostly.db
```

### Node Can't Reach Primary

**Symptoms**: Secondary logs show "failed to send heartbeat to primary".

**Check 1: PRIMARY_NODE_URL**

```bash
# On secondary, verify URL is correct
grep PRIMARY_NODE_URL .env

# Should be reachable from secondary:
curl http://primary-url:8080/api/health
```

**Check 2: DNS Resolution**

```bash
# If using hostnames, verify DNS works
ping primary.example.com
```

**Check 3: Reverse Proxy**

If primary is behind a reverse proxy (nginx, Caddy):

```nginx
# Ensure /api/internal/* is proxied
location /api/internal/ {
    proxy_pass http://localhost:8080;
    proxy_set_header X-Node-ID $http_x_node_id;
    proxy_set_header X-Node-API-Key $http_x_node_api_key;
}
```

### Exponential Backoff Too Aggressive

**Symptoms**: Node takes too long to be marked offline or back online.

**Tune Health Check Intervals**:

Currently hardcoded in `internal/service/node_service.go`:

```go
func shouldCheckNode(node *db.Node, now time.Time) bool {
    failures := node.ConsecutiveFailures
    
    // Adjust these intervals to your needs
    var interval time.Duration
    switch {
    case failures <= 2:
        interval = 30 * time.Second  // ← More aggressive
    case failures <= 5:
        interval = 2 * time.Minute   // ← Adjust as needed
    // ... etc
    }
}
```

### Apps Not Deploying to Secondary Nodes

**Symptoms**: Apps only deploy to primary, secondary option not working.

**Check 1: Node Registration**

Ensure node is registered and online:
- Settings → Nodes
- Node should show green "Online" badge

**Check 2: Node Selection**

When creating app, verify node_id parameter:

```bash
# Should include node_id in query string
POST /api/apps?node_id=abc-123-def-456
```

**Check 3: Secondary Logs**

```bash
# On secondary, watch for app operations
docker-compose logs -f | grep -i "app\|deploy\|compose"
```

## Advanced Configuration

### Custom Health Check Endpoint

By default, health checks use `/api/internal/system/stats`. This could be customized to use a lighter endpoint if needed.

### Multiple Primary Nodes (Future)

Currently, Selfhostly supports only one primary node. High availability with multiple primaries is a planned feature.

### TLS/HTTPS Between Nodes

For production deployments, use TLS:

```bash
# Primary
NODE_API_ENDPOINT=https://primary.example.com

# Secondary
PRIMARY_NODE_URL=https://primary.example.com
```

Ensure valid certificates are configured on both nodes.

### Monitoring Health Check Performance

Track health check duration and failures:

```bash
# Check logs for timing information
docker-compose logs | grep "health check" | grep "duration"
```

## API Reference

### Node Heartbeat Endpoint

**Request**:
```http
POST /api/internal/nodes/{id}/heartbeat
X-Node-ID: abc-123-def-456
X-Node-API-Key: your-api-key
```

**Response**:
```json
{
  "message": "Heartbeat received",
  "nodeID": "abc-123-def-456"
}
```

### Manual Health Check Endpoint

**Request**:
```http
POST /api/nodes/{id}/check
Authorization: Bearer {user-jwt-token}
```

**Response**:
```json
{
  "message": "Health check completed successfully",
  "node": {
    "id": "abc-123",
    "name": "worker-1",
    "status": "online",
    "last_seen": "2026-01-26T20:00:00Z"
  }
}
```

## Security Best Practices

1. **Use Strong API Keys**: Generate 32+ character random keys
2. **Rotate Keys Periodically**: Update keys every 90 days
3. **Network Isolation**: Use private networks or VPNs between nodes
4. **Enable TLS**: Use HTTPS for all node communication
5. **Firewall Rules**: Restrict access to port 8080 to known IPs
6. **Monitor Logs**: Watch for authentication failures
7. **Cloudflare Zero Trust**: Recommended for UI access

## Performance Considerations

- **Health Check Interval**: 30 seconds provides good balance between responsiveness and overhead
- **Exponential Backoff**: Reduces load on persistently down nodes
- **Database Size**: SQLite handles hundreds of nodes efficiently
- **Network Latency**: Keep nodes in same region for best performance
- **Concurrent Operations**: Health checks run in parallel with goroutines

## Future Enhancements

- [ ] Periodic heartbeats from secondary nodes (every 60s)
- [ ] Primary node high availability (multiple primaries)
- [ ] Automatic node discovery (mDNS/broadcast)
- [ ] Node groups/labels for better organization
- [ ] Metrics collection (Prometheus/Grafana integration)
- [ ] Node-to-node TLS certificate management
- [ ] Quorum-based decisions for cluster operations

---

## Related Documentation

- [Development Guide](./DEVELOPMENT.md)
- [Cloudflare Zero Trust Setup](./CLOUDFLARE_ZERO_TRUST.md)
- [GitHub Whitelist Authentication](./GITHUB_WHITELIST.md)
- [Monitoring Guide](./MONITORING.md)
