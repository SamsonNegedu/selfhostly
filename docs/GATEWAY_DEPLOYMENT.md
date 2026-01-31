# Gateway Architecture Deployment Guide

## Overview

The production deployment uses a **gateway architecture** to enable multi-node routing and scalability:

```
Internet → Cloudflare Tunnel → Gateway (port 8080) → Backend Nodes (port 8082+)
```

### Components

1. **Gateway** - Single entry point that:
   - Receives all incoming requests
   - Routes requests to appropriate backend nodes
   - Handles authentication and authorization
   - Maintains a registry of available nodes

2. **Primary Backend** - Main application server that:
   - Manages the database
   - Executes Docker operations
   - Handles user authentication
   - Coordinates with secondary nodes

3. **Secondary Nodes** (optional) - Additional worker nodes that:
   - Register with the primary on startup
   - Execute their own Docker operations
   - Report health status to primary

## Deployment

### 1. Build the Images

Three optimized images are available:

```bash
# Gateway (lean routing - ~20MB)
docker build -f Dockerfile.gateway -t ghcr.io/selfhostly/gateway:latest .

# Primary backend with UI (~150MB)
docker build -f Dockerfile.primary -t ghcr.io/selfhostly/primary:latest .

# Secondary backend without UI (~80MB)
docker build -f Dockerfile.backend -t ghcr.io/selfhostly/backend:latest .
```

See [BUILD_IMAGES.md](../BUILD_IMAGES.md) for details on image differences.

### 2. Configure Environment Variables

Create a `.env` file with the required configuration:

```bash
# Gateway API Key (shared secret between gateway and backends)
GATEWAY_API_KEY=your-secure-random-key-here

# Authentication
JWT_SECRET=your-jwt-secret
AUTH_ENABLED=true
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
AUTH_BASE_URL=https://your-domain.com
GITHUB_ALLOWED_USERS=user1,user2
AUTH_SECURE_COOKIE=true

# Node Configuration
NODE_ID=450359e5-52c3-47e8-a256-6ea537528a06
NODE_NAME=primary
REGISTRATION_TOKEN=your-registration-token

# Cloudflare (optional)
CLOUDFLARE_API_TOKEN=your-cloudflare-token
CLOUDFLARE_ACCOUNT_ID=your-cloudflare-account-id
TUNNEL_TOKEN=your-tunnel-token
```

**Important:** Generate a strong random `GATEWAY_API_KEY`:

```bash
openssl rand -base64 64
```

### 3. Deploy with Docker Compose

```bash
docker compose -f docker-compose.prod.yml up -d
```

This will start:
- Gateway on port 8080 (public)
- Primary backend on port 8082 (internal only)
- Cloudflared tunnel (if configured)

### 4. Verify Deployment

Check that services are healthy:

```bash
docker compose -f docker-compose.prod.yml ps
```

Test the gateway:

```bash
curl http://localhost:8080/api/health
```

## Architecture Details

### Port Configuration

- **Gateway**: Port 8080 (public-facing)
  - Receives all external requests
  - Routes to backend nodes
  - Exposed to host and Cloudflare tunnel

- **Primary Backend**: Port 8082 (internal)
  - Not exposed to host network
  - Only accessible within Docker network
  - Gateway forwards requests here

- **Secondary Nodes**: Ports 8083+ (internal)
  - Each node uses a unique port
  - Register with primary on startup

### Request Flow

1. **User Request** → Cloudflare Tunnel → Gateway (port 8080)

2. **Gateway** analyzes the request:
   - Global operations (list apps, stats) → Primary
   - Node-specific operations → Target node (using `node_id` param)

3. **Gateway** forwards request with:
   - Original auth headers (JWT, cookies)
   - `X-Gateway-API-Key` header (for node management)
   - `X-Forwarded-Host` header (for OAuth redirects)

4. **Backend** processes and returns response

5. **Gateway** returns response to client

### Authentication Flow

The gateway validates user authentication:

- **Enabled** (`AUTH_ENABLED=true`): Gateway checks JWT tokens
- **Disabled** (`AUTH_ENABLED=false`): Gateway passes all requests through

Node management endpoints always require `GATEWAY_API_KEY` regardless of user auth.

## Adding Secondary Nodes

To add additional worker nodes:

1. Create a new service in docker-compose:

```yaml
  secondary-node-1:
    image: ghcr.io/selfhostly/backend:latest  # No UI needed
    container_name: selfhostly-node-1
    environment:
      SERVER_ADDRESS: ":8083"
      NODE_IS_PRIMARY: "false"
      NODE_NAME: "worker-1"
      GATEWAY_API_KEY: ${GATEWAY_API_KEY}
      PRIMARY_NODE_URL: http://primary:8082
      REGISTRATION_TOKEN: ${REGISTRATION_TOKEN}
      # ... other env vars
    volumes:
      - /path/to/node1/apps:/app/apps
      - /var/run/docker.sock:/var/run/docker.sock
    networks:
      - selfhostly-network
    expose:
      - "8083"
```

2. The node will automatically register with the primary on startup

3. Gateway will discover the node and route requests to it

## Load Balancing

For high availability, you can run multiple gateway instances behind a load balancer:

1. Remove port mapping from gateway service
2. Run multiple gateway replicas
3. Use nginx/haproxy to load balance across gateway instances

Example with docker compose scale:

```bash
docker compose -f docker-compose.prod.yml up -d --scale gateway=3
```

Then configure nginx to proxy to all gateway instances.

## Security Considerations

1. **GATEWAY_API_KEY**: Keep this secret secure. It grants full access to node management APIs.

2. **JWT_SECRET**: Must be the same on gateway and all backends for authentication to work.

3. **Network Isolation**: Backend nodes should not be exposed to the public internet directly.

4. **TLS**: Use Cloudflare Tunnel or reverse proxy with TLS for production.

## Monitoring

Gateway logs routing decisions:

```bash
docker compose -f docker-compose.prod.yml logs -f gateway
```

Look for:
- `gateway: incoming request` - All incoming requests
- `gateway: routing request` - Target resolution
- `router: resolved by node_id` - Node-specific routing
- `router: primary-only route` - Primary-only operations

## Troubleshooting

### Gateway can't reach primary

**Symptom**: `gateway: upstream request failed`

**Solution**: 
- Verify primary is healthy: `docker compose ps`
- Check network connectivity: `docker compose exec gateway ping primary`
- Verify `PRIMARY_BACKEND_URL` is correct

### Node not found errors

**Symptom**: `router: node not found node_id=...`

**Solution**:
- Node may not be registered yet (check primary logs)
- Node may have restarted with a new ID (clear stale data)
- Verify node's `NODE_API_ENDPOINT` is reachable from gateway

### Authentication issues

**Symptom**: `gateway: auth required`

**Solution**:
- Verify `JWT_SECRET` matches between gateway and backends
- Check if `AUTH_ENABLED` is correctly set
- Verify JWT token is valid and not expired

## Migration from Single-Node Setup

If you're migrating from a single-node deployment:

1. Backup your database: `cp /path/to/selfhostly.db /path/to/backup/`

2. Update docker-compose.yml to use the gateway architecture

3. Set `GATEWAY_API_KEY` in environment

4. Change backend port from 8080 to 8082

5. Deploy and verify gateway routes correctly

6. Update any external services to point to gateway (port 8080)

## Rolling Updates

To update without downtime:

1. Update the images:
   ```bash
   docker pull ghcr.io/selfhostly/gateway:latest
   docker pull ghcr.io/selfhostly/primary:latest
   docker pull ghcr.io/selfhostly/backend:latest  # If using secondaries
   ```

2. Update backends first:
   ```bash
   docker compose -f docker-compose.prod.yml up -d primary
   ```

3. Wait for primary to be healthy

4. Update gateway:
   ```bash
   docker compose -f docker-compose.prod.yml up -d gateway
   ```

Gateway will continue routing to old backend during update, minimizing downtime.
