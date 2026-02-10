# Selfhostly

<p>
	<a href="https://github.com/user-attachments/assets/4d8df016-2bbe-4975-9dfd-474a79b12c85" target="_blank">
		<img alt="Screenshot 2026-02-10 at 20 57 15" src="https://github.com/user-attachments/assets/4d8df016-2bbe-4975-9dfd-474a79b12c85" width="45%" height="auto">
	</a>
	<a href="https://github.com/user-attachments/assets/822001e0-ab5c-42e7-9b08-e993b7bc4283" target="_blank">
		<img alt="Screenshot 2026-02-10 at 20 59 52" src="https://github.com/user-attachments/assets/822001e0-ab5c-42e7-9b08-e993b7bc4283" width="45%" height="auto">
	</a>
</p>

A web-based platform for managing self-hosted applications on your Raspberry Pi or Linux server. Deploy Docker Compose apps, configure Cloudflare tunnels, monitor system resources, and manage containers through a centralized interface.

## Features

### Multi-Node Architecture
- **Distributed Deployment** - Manage applications across multiple servers from a single UI
- **Automatic Health Checks** - Smart monitoring with exponential backoff for offline nodes
- **Node Heartbeats** - Secondary nodes proactively report online status
- **Secure Node Communication** - API key-based authentication between nodes
- **Unified Monitoring** - View stats from all nodes in one dashboard
- **Horizontal Scaling** - Add more nodes to distribute workload

### Application Management
- **Deploy Docker Compose Apps** - Deploy applications through an intuitive web interface
- **Built-in Editor** - Monaco editor with YAML syntax highlighting and validation
- **Version History** - Automatic versioning with complete rollback capability
- **Zero-Downtime Updates** - Pull new images and update containers without interruption
- **Activity Timeline** - Track all changes, deployments, and updates

### Cloudflare Integration
- **Automatic Tunnel Setup** - Create and configure Cloudflare tunnels directly from the UI
- **Ingress Rules Management** - Configure routing rules without accessing the Cloudflare dashboard
- **Tunnel Management** - View, restart, and manage cloudflared containers
- **DNS Management** - Automatic DNS record creation for tunnel routes

### Monitoring & Observability
- **System Monitoring** - Real-time CPU, memory, disk, and Docker daemon statistics
- **Container Metrics** - View resource usage for every container
- **Search & Filter** - Find containers across all apps by name, ID, or status
- **Resource Alerts** - Automatic warnings for high CPU, memory, or disk usage
- **Quick Actions** - Restart or stop containers directly from the dashboard
- **Auto-Refresh** - Metrics update every 10 seconds (pauses when tab is inactive)

### User Interface
- **Theme Support** - Light and dark modes with system preference detection
- **Responsive Design** - Works on mobile, tablet, and desktop devices
- **Modern Stack** - Built with React, TypeScript, TailwindCSS, and Radix UI
- **Flexible Authentication** - Cloudflare Zero Trust or GitHub OAuth support

## Tech Stack

**Backend:** Go • Gin • SQLite (modernc.org/sqlite - pure Go, no CGO) • Docker API • gopsutil

**Frontend:** React • TypeScript • TanStack Query • Zustand • TailwindCSS • Radix UI • Monaco Editor • Lucide Icons

**Infrastructure:** Docker • Docker Compose • Cloudflare Tunnels • Air (live reload)

## Security Notice

**Single-user design only.** This application is intended for personal use (e.g., managing a home server or Raspberry Pi).

- **Recommended:** Deploy behind [Cloudflare Zero Trust](./docs/CLOUDFLARE_ZERO_TRUST.md) for secure authentication
- **Alternative:** [GitHub OAuth authentication](./docs/GITHUB_WHITELIST.md) with username whitelist
- **Not suitable** for multi-user or multi-tenant environments

See [Security Documentation](./docs/SECURITY.md) for full details.

## Recent Updates

- **Multi-Node Architecture** - Manage applications across multiple servers from a single UI with automatic health checks
- **Smart Health Monitoring** - Exponential backoff for health checks and heartbeat mechanism for faster node recovery
- **System Monitoring** - Comprehensive dashboard showing CPU, memory, disk, and per-container metrics with real-time updates
- **Theme Support** - Dark and light modes with automatic system preference detection
- **Version Control** - Automatic versioning and rollback capability for all compose file changes
- **Development Environment** - Air integration for Go live reload and hot module reloading for frontend
- **Container Actions** - Quick restart/stop actions for any container from the monitoring dashboard
- **Cloudflare Improvements** - Enhanced tunnel management with better duplicate handling and restart functionality
- **Pure Go SQLite** - Switched to modernc.org/sqlite (no CGO dependencies) for easier cross-compilation

## Requirements

- **Docker** - Version 20.10+ with Docker Compose
- **Linux/macOS** - Tested on Raspberry Pi OS, Ubuntu, and macOS (Windows via WSL2)
- **Architecture** - amd64 or arm64 (Raspberry Pi 4/5 supported)
- **Resources** - Minimum 1GB RAM, 2GB+ recommended
- **Optional** - Cloudflare account for tunnel integration

## Quick Start

### Production Deployment

```bash
# 1. Clone the repository
git clone https://github.com/yourusername/selfhostly.git
cd selfhostly

# 2. Configure environment
cp env.example .env
# Edit .env with your settings (see Configuration section below)

# 3. Run with Docker Compose
docker compose -f docker-compose.prod.yml up -d

# Or use Make command
make prod

# Optional: Run with Cloudflare Tunnel
docker compose -f docker-compose.prod.yml --profile tunnel up -d
```

Access the web interface at `http://localhost:8080` (or your configured address).

### Development Setup

```bash
# Option 1: Start both backend and frontend with one command
make dev

# Option 2: Start services separately
make dev-backend      # Terminal 1: Backend with live reload
cd web && npm run dev # Terminal 2: Frontend dev server
```

Frontend dev server runs at `http://localhost:5173` with hot module reloading.  
Backend API runs at `http://localhost:8080` with automatic rebuild on code changes.

#### Run everything locally (no Docker)

| What | Command | Port |
|------|---------|------|
| Backend (primary) | `make run-local` (with Air hot reload) | 8080 |
| Frontend | `cd web && npm run dev` | 5173 |

Use the same `.env` (e.g. `cp env.example .env`). Open **http://localhost:5173**; Vite proxies `/api` and `/auth` to the backend.

#### Run with API gateway (primary + gateway + frontend)

When testing the gateway locally, primary and gateway need different ports. Example:

1. **Primary backend** (e.g. `.env` or `ENV_FILE`):
   ```env
   SERVER_ADDRESS=:8082
   NODE_API_ENDPOINT=http://localhost:8082
   GATEWAY_API_KEY=dev-gateway-secret
   ```
   ```bash
   make run-local   # or: make run-local ENV_FILE=.env.primary
   ```
   → Primary listens on **8082**.

2. **Gateway** (same `.env` or a gateway-specific one):
   ```env
   GATEWAY_LISTEN_ADDRESS=:8080
   PRIMARY_BACKEND_URL=http://localhost:8082
   GATEWAY_API_KEY=dev-gateway-secret
   ```
   ```bash
   make run-gateway   # with Air hot reload
   # or: make run-gateway ENV_FILE=.env.gateway
   ```
   → Gateway listens on **8080** (hot reloads on code changes).

3. **Frontend** (proxy points at gateway):
   ```bash
   cd web && npm run dev
   ```
   Vite proxies to `http://localhost:8080` (gateway). Open **http://localhost:5173**.

Summary: **Primary 8082 → Gateway 8080 → Frontend proxy → Browser 5173.**

See [Development Guide](./docs/DEVELOPMENT.md) for more details.

## Make Commands

The project includes a Makefile with convenient commands for common tasks. Run `make help` to see all available commands.

### Development Commands

```bash
make dev              # Start all services with live reload (backend + frontend)
make dev-backend      # Start only backend with live reload
make dev-frontend     # Start only frontend dev server
make dev-build        # Rebuild dev containers
```

### Production Commands

```bash
make prod             # Start production services
make prod-build       # Build and start production services
```

### Control Commands

```bash
make down             # Stop all running containers
make clean            # Clean build artifacts, containers, and volumes
make logs             # Show logs from all services
make logs-backend     # Show backend logs only
make restart-backend  # Restart backend service
```

### Local Development (No Docker)

```bash
make install-air      # Install Air for local development
make run-local        # Run backend locally with Air
make run-local-no-air # Run backend locally without Air
make run-gateway      # Run API gateway (set GATEWAY_* and GATEWAY_API_KEY in .env)
```

### Getting Help

```bash
make help             # Show all available commands with descriptions
```

### Quick Reference

| Command | Description |
|---------|-------------|
| `make dev` | Start full dev environment |
| `make run-local` | Run backend locally (Air) |
| `make run-gateway` | Run API gateway locally |
| `make prod` | Start production |
| `make logs` | View all logs |
| `make down` | Stop everything |
| `make clean` | Clean up completely |
| `make help` | Show all commands |

## Configuration

### Core Settings

```env
# Server Configuration
SERVER_ADDRESS=:8080           # Address to bind the web server
DATABASE_PATH=./data/selfhostly.db  # SQLite database location
```

### Multi-Node Configuration (Optional)

Deploy across multiple servers for distributed app management:

```env
# Primary Node (main server with UI and database)
NODE_IS_PRIMARY=true
NODE_NAME=primary
NODE_API_ENDPOINT=http://192.168.1.10:8080        # This node's reachable URL
NODE_API_KEY=your-secure-api-key-here             # Generate with: openssl rand -base64 32
REGISTRATION_TOKEN=your-secure-registration-token # Generate with: openssl rand -base64 32

# Secondary Node (worker server) - Auto-registers on startup!
NODE_IS_PRIMARY=false
NODE_NAME=worker-1
NODE_API_ENDPOINT=http://192.168.1.50:9090        # This node's reachable URL
NODE_API_KEY=your-secure-api-key-here             # Auto-generated or set explicitly
PRIMARY_NODE_URL=http://192.168.1.10:8080         # URL of primary node
REGISTRATION_TOKEN=your-secure-registration-token # Same as primary - enables auto-registration
```

**Benefits:**
- **Auto-registration** - Secondary nodes register themselves on startup
- **Simple Configuration** - Set environment variables and deploy
- **Centralized Management** - Control multiple servers from one UI
- **Unified Monitoring** - View metrics across all nodes
- **Automatic Health Checks** - Continuous monitoring and heartbeats

See [Multi-Node Setup Guide](./docs/MULTI_NODE.md) for complete configuration, authentication, and troubleshooting.

### Cloudflare Integration (Optional)

Required for automatic tunnel creation and management:

```env
CLOUDFLARE_API_TOKEN=your_cloudflare_api_token
CLOUDFLARE_ACCOUNT_ID=your_cloudflare_account_id
```

**How to get these:**
1. Go to [Cloudflare API Tokens](https://dash.cloudflare.com/profile/api-tokens)
2. Create a token with `Cloudflare Tunnel:Edit` and `Zone:DNS:Edit` permissions
3. Find your Account ID in any zone's overview page

### Authentication (Optional)

**Option 1: Cloudflare Zero Trust (Recommended)**

Deploy behind Cloudflare Zero Trust Access for enterprise-grade authentication:

```env
AUTH_ENABLED=false  # Cloudflare handles authentication
CLOUDFLARE_API_TOKEN=your_token
CLOUDFLARE_ACCOUNT_ID=your_account_id
```

**Benefits:**
- No OAuth configuration required
- Support for multiple identity providers (Google, GitHub, Okta, etc.)
- Email-based access control
- Built-in 2FA support

See [Cloudflare Zero Trust Setup Guide](./docs/CLOUDFLARE_ZERO_TRUST.md)

**Option 2: GitHub OAuth**

Use GitHub OAuth for simple username-based authentication:

```env
AUTH_ENABLED=true
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_ALLOWED_USERS=username1,username2
NODE_API_ENDPOINT=https://your-domain.com
```

See [GitHub OAuth Setup Guide](./docs/GITHUB_WHITELIST.md)

**Security Note:** This application is designed for single-user deployments. See [Security Documentation](./docs/SECURITY.md) for details.

## Development

### Local Development

**Option 1: Make Commands (Easiest)**

```bash
# Start everything with one command
make dev

# Or start services separately
make dev-backend      # Terminal 1: Backend with live reload
make dev-frontend     # Terminal 2: Frontend dev server
```

**Option 2: Docker Compose**

```bash
# Start backend with Air (live reload)
docker compose -f docker-compose.dev.yml up backend

# Start frontend in another terminal
cd web && npm install && npm run dev
```

**Option 3: Native Go + npm**

```bash
# Install Air for live reload
make install-air
# Or: go install github.com/air-verse/air@latest

# Terminal 1: Backend with live reload
make run-local
# Or: air

# Terminal 2: Frontend dev server
cd web && npm install && npm run dev
```

**Access:**
- Frontend: `http://localhost:5173` (proxies API requests to backend)
- Backend API: `http://localhost:8080`

### Building for Production

```bash
# Build frontend assets
cd web && npm run build

# Build Go binaries
go build -o bin/server cmd/server/main.go
go build -o bin/gateway cmd/gateway/main.go

# Or build Docker images
docker build -t selfhostly-backend -f Dockerfile.backend .
docker build -t selfhostly-gateway -f Dockerfile.gateway .
docker build -t selfhostly-frontend -f web/Dockerfile ./web
```

**Available Dockerfiles:**
- `Dockerfile.backend` - API server (universal for primary and secondary nodes)
- `Dockerfile.gateway` - API gateway for routing requests
- `web/Dockerfile` - Frontend static file server
- `Dockerfile.dev` - Development environment with live reload

### Project Structure

```
selfhostly/
├── cmd/
│   ├── server/           # Primary backend entry point
│   └── gateway/          # Gateway entry point
├── internal/
│   ├── cloudflare/       # Cloudflare API client and tunnel management
│   ├── config/           # Configuration loading
│   ├── db/               # Database models and queries
│   ├── docker/           # Docker and Compose management
│   ├── gateway/          # Gateway proxy logic
│   ├── http/             # HTTP handlers and routing
│   ├── node/             # Multi-node communication
│   ├── routing/          # Request routing and aggregation
│   ├── service/          # Business logic layer
│   ├── system/           # System metrics collection
│   └── tunnel/           # Tunnel provider abstraction
├── web/
│   └── src/
│       ├── features/     # Feature-based modules (dashboard, monitoring, etc.)
│       └── shared/       # Shared components, utilities, types
└── docs/                 # Documentation
```

### Testing

```bash
# Run Go tests
go test ./...

# Run specific package tests
go test ./internal/docker/

# Frontend linting
cd web && npm run lint
```

### Viewing Logs

```bash
# View all service logs
make logs

# View backend logs only
make logs-backend

# Or with Docker Compose directly
docker compose -f docker-compose.dev.yml logs -f
```

### Cleaning Up

```bash
# Stop all containers
make down

# Clean everything (containers, volumes, build artifacts)
make clean
```

See [Development Guide](./docs/DEVELOPMENT.md) for live reload setup and debugging tips.

## Documentation

### Getting Started
- [Development Guide](./docs/DEVELOPMENT.md) - Local setup, live reload, and debugging
- [Multi-Node Setup](./docs/MULTI_NODE.md) - Distributed deployment, authentication, and health checks
- [Security Model](./docs/SECURITY.md) - Single-user design and limitations

### Features
- [Monitoring Dashboard](./docs/MONITORING.md) - System metrics, container monitoring, and resource alerts
- [Compose Versioning](./docs/COMPOSE_VERSIONING.md) - Version control and rollback system
- [Cloudflare Integration](./docs/CLOUDFLARE_ZERO_TRUST.md) - Tunnel setup and Zero Trust configuration

### Authentication & Security
- [Multi-Node Authentication](./docs/MULTI_NODE.md#authentication-strategies) - Node-to-node API keys and user auth
- [Cloudflare Zero Trust Setup](./docs/CLOUDFLARE_ZERO_TRUST.md) - Recommended authentication method
- [GitHub OAuth Setup](./docs/GITHUB_WHITELIST.md) - Alternative authentication option

## Key Workflows

### Deploying a New App

1. **Create App** - Navigate to "Create App" and provide a name and description
2. **Write Compose File** - Use the Monaco editor to write or paste your docker-compose.yml
3. **Configure Ingress** - Optionally add Cloudflare ingress rules for public access
4. **Deploy** - Click "Create & Deploy" to start your containers

### Managing Cloudflare Tunnels

1. **Create Tunnel** - Go to Cloudflare Management and create a new tunnel with one click
2. **Configure Ingress** - Add ingress rules to route subdomains to your services
3. **Update Rules** - Edit ingress rules anytime; cloudflared will automatically restart
4. **View Status** - See tunnel status and metrics from the dashboard

### Monitoring Your System

1. **System Overview** - View real-time CPU, memory, disk, and Docker stats at `/monitoring`
2. **Container Metrics** - See resource usage for every container across all apps
3. **Search & Filter** - Find specific containers by name, app, or status
4. **Take Action** - Restart or stop containers directly from the monitoring page
5. **Resource Alerts** - Get automatic warnings for high resource usage

### Updating an App

1. **Navigate to App** - Go to the app details page
2. **Pull New Images** - Click "Update Containers" to pull latest images
3. **View Progress** - Monitor the update process in real-time
4. **Zero Downtime** - Containers are updated with zero downtime strategy

### Version Control & Rollback

1. **Auto Versioning** - Every compose file change is automatically versioned
2. **View History** - See all previous versions with timestamps and change reasons
3. **Rollback** - Click "Rollback" on any version to restore previous configuration
4. **Activity Timeline** - Track all changes and deployments in the activity log

## Use Cases

**Ideal for:**
- Self-hosting enthusiasts managing apps on a Raspberry Pi or home server
- Multi-server home labs with distributed workloads
- Developers running multiple services across different machines
- Managing Docker containers without SSH access
- Quick deployment of Docker Compose applications with public URLs
- Monitoring resource usage across multiple nodes
- Geographic distribution of services (edge nodes + main server)

**Not suitable for:**
- Multi-user or multi-tenant environments (single-user design)
- Production SaaS platforms (no resource isolation between users)
- Enterprise deployments requiring RBAC and audit logs

## Troubleshooting

### Application won't start

- Check Docker is running: `docker ps`
- Verify `.env` file exists and has correct values
- Check logs: `make logs` or `docker compose -f docker-compose.prod.yml logs -f`
- Ensure port 8080 is not in use
- Try rebuilding: `make prod-build`

### Cloudflare tunnel not working

- Verify `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` in `.env`
- Check token permissions: needs `Cloudflare Tunnel:Edit` and `Zone:DNS:Edit`
- Review tunnel logs in the app details page
- Ensure ingress rules point to correct service:port

### No containers showing in monitoring

- Ensure apps have been deployed
- Verify Docker socket access: `/var/run/docker.sock`
- Check that apps have running containers: `docker ps`

### Stats not updating

- Check browser console for API errors
- Verify authentication is working
- Ensure tab is visible (auto-refresh pauses when tab is hidden)

### Database errors

- Check `DATABASE_PATH` directory exists and is writable
- Ensure SQLite file has correct permissions
- Try removing database and restart (will lose data)

### Can't access on network

- Check `SERVER_ADDRESS` in `.env` - use `0.0.0.0:8080` for all interfaces
- Verify firewall allows port 8080
- For remote access, consider using Cloudflare tunnel instead of exposing port

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](./LICENSE) file for details

## Acknowledgments

- Built with [Gin](https://github.com/gin-gonic/gin) - HTTP web framework for Go
- Authentication via [go-pkgz/auth](https://github.com/go-pkgz/auth) - OAuth and JWT handling
- UI powered by [Radix UI](https://www.radix-ui.com/) and [TailwindCSS](https://tailwindcss.com/)
- System metrics via [gopsutil](https://github.com/shirou/gopsutil)
- Code editor via [Monaco Editor](https://microsoft.github.io/monaco-editor/)
- Database via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) - Pure Go SQLite implementation
