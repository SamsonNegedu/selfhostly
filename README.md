# Selfhostly

A comprehensive web-based platform for managing self-hosted applications on your Raspberry Pi or any Linux server. Deploy apps, configure Cloudflare tunnels, monitor resources, and manage containers—all without SSH access.

## Features

### App Management
- 🚀 **Create & Deploy Apps** - Deploy Docker Compose applications through an intuitive web UI
- 📝 **Compose Editor** - Built-in Monaco editor with YAML syntax highlighting and validation
- 📜 **Version History** - Automatic versioning with complete rollback capability for all compose file changes
- 🔄 **Zero-Downtime Updates** - Pull new images and update containers without service interruption
- 📋 **Activity Timeline** - Track all changes, deployments, and updates for each app

### Cloudflare Integration
- 🌐 **Automatic Tunnel Setup** - Create and configure Cloudflare tunnels directly from the UI
- 🔗 **Ingress Rules Management** - Configure and edit ingress rules without touching the Cloudflare dashboard
- 🚇 **Tunnel Management** - View, restart, and manage cloudflared containers per app
- 📍 **DNS Record Management** - Automatic DNS record creation for tunnel routes

### Monitoring & Observability
- 📊 **System Monitoring** - Real-time CPU, memory, disk, and Docker daemon statistics
- 🐳 **Container Metrics** - View CPU, memory, network I/O, and disk I/O for every container
- 🔍 **Container Search & Filter** - Find containers across all apps by name, ID, or status
- 🚨 **Resource Alerts** - Automatic alerts for high CPU, memory, disk usage, and container issues
- ⚡ **Quick Actions** - Restart or stop any container directly from the monitoring dashboard
- 🔄 **Auto-Refresh** - Metrics update every 10 seconds (pauses when tab is inactive)

### User Experience
- 🎨 **Theme Support** - Beautiful light and dark modes with system preference detection
- 📱 **Responsive Design** - Fully functional on mobile, tablet, and desktop
- ⚡ **Fast & Modern UI** - Built with React, TypeScript, TailwindCSS, and Radix UI
- 🔐 **Optional Authentication** - Cloudflare Zero Trust or GitHub OAuth support

## Tech Stack

**Backend:** Go • Fiber • SQLite (modernc.org/sqlite - pure Go, no CGO) • Docker API • gopsutil

**Frontend:** React • TypeScript • TanStack Query • Zustand • TailwindCSS • Radix UI • Monaco Editor • Lucide Icons

**Infrastructure:** Docker • Docker Compose • Cloudflare Tunnels • Air (live reload)

## ⚠️ Security Notice

**Single-user design only.** Intended for personal use (e.g., Raspberry Pi hosting).

- ✅ **Recommended:** Deploy behind [Cloudflare Zero Trust](./docs/CLOUDFLARE_ZERO_TRUST.md) (no OAuth needed)
- ✅ Alternative: [GitHub OAuth authentication](./docs/GITHUB_WHITELIST.md)
- ❌ Not suitable for multi-user environments

📖 Full details: [Security Documentation](./docs/SECURITY.md)

## ✨ Recent Updates

- **🔧 System Monitoring** - Comprehensive dashboard showing CPU, memory, disk, and per-container metrics with real-time updates
- **🎨 Theme Support** - Beautiful dark and light modes with automatic system preference detection
- **📜 Version Control** - Automatic versioning and rollback capability for all compose file changes
- **⚡ Development Environment** - Air integration for Go live reload and hot module reloading for frontend
- **🐳 Container Actions** - Quick restart/stop actions for any container from the monitoring dashboard
- **🚇 Cloudflare Improvements** - Enhanced tunnel management with better duplicate handling and restart functionality
- **🗄️ Pure Go SQLite** - Switched to modernc.org/sqlite (no CGO dependencies) for easier cross-compilation

## 📋 Requirements

- **Docker** - Version 20.10+ with Docker Compose
- **Linux/macOS** - Tested on Raspberry Pi OS, Ubuntu, and macOS (Windows via WSL2)
- **Architecture** - amd64 or arm64 (Raspberry Pi 4/5 supported)
- **Resources** - Minimum 1GB RAM, 2GB+ recommended
- **Optional** - Cloudflare account for tunnel integration

## 🚀 Quick Start

### Production Deployment

```bash
# 1. Clone the repository
git clone https://github.com/yourusername/selfhost-automaton.git
cd selfhost-automaton

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

See [Development Guide](./docs/DEVELOPMENT.md) for detailed instructions.

## 🛠️ Make Commands

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
```

### Getting Help

```bash
make help             # Show all available commands with descriptions
```

### Quick Reference

| Command | Description |
|---------|-------------|
| `make dev` | 🚀 Start full dev environment |
| `make prod` | 🏭 Start production |
| `make logs` | 📜 View all logs |
| `make down` | 🛑 Stop everything |
| `make clean` | 🧹 Clean up completely |
| `make help` | ❓ Show all commands |

## ⚙️ Configuration

### Core Settings

```env
# Server Configuration
SERVER_ADDRESS=:8080           # Address to bind the web server
DATABASE_PATH=./data/automaton.db  # SQLite database location
```

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

**Option 1 - Cloudflare Zero Trust** ✅ Recommended

Deploy behind Cloudflare Zero Trust Access for enterprise-grade authentication:

```env
AUTH_ENABLED=false  # Cloudflare handles authentication
CLOUDFLARE_API_TOKEN=your_token
CLOUDFLARE_ACCOUNT_ID=your_account_id
```

**Benefits:**
- No OAuth configuration needed
- Support for multiple identity providers (Google, GitHub, Okta, etc.)
- Email-based access control
- Built-in 2FA support

📖 [Cloudflare Zero Trust Setup Guide](./docs/CLOUDFLARE_ZERO_TRUST.md)

**Option 2 - GitHub OAuth**

Use GitHub OAuth for simple username-based authentication:

```env
AUTH_ENABLED=true
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_ALLOWED_USERS=username1,username2
AUTH_BASE_URL=https://your-domain.com
```

📖 [GitHub OAuth Setup Guide](./docs/GITHUB_WHITELIST.md)

**⚠️ Security Note:** This is a single-user tool. See [Security Documentation](./docs/SECURITY.md) for details.

## 🔧 Development

### Local Development

**Option 1 - Make Commands (Easiest)**

```bash
# Start everything with one command
make dev

# Or start services separately
make dev-backend      # Terminal 1: Backend with live reload
make dev-frontend     # Terminal 2: Frontend dev server
```

**Option 2 - Docker Compose**

```bash
# Start backend with Air (live reload)
docker compose -f docker-compose.dev.yml up backend

# Start frontend in another terminal
cd web && npm install && npm run dev
```

**Option 3 - Native Go + npm**

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

# Build Go binary
go build -o bin/server cmd/server/main.go

# Or build Docker image
docker build -t selfhostly -f Dockerfile.unified .
```

### Project Structure

```
selfhost-automaton/
├── cmd/server/           # Main application entry point
├── internal/
│   ├── cloudflare/       # Cloudflare API client and tunnel management
│   ├── config/           # Configuration loading
│   ├── db/               # Database models and queries
│   ├── docker/           # Docker and Compose management
│   ├── http/             # HTTP handlers and routing
│   └── system/           # System metrics collection
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

📖 See [Development Guide](./docs/DEVELOPMENT.md) for live reload setup and debugging tips.

## 📚 Documentation

### Getting Started
- [Development Guide](./docs/DEVELOPMENT.md) - Local setup, live reload, and debugging
- [Security Model](./docs/SECURITY.md) - Single-user design and limitations

### Features
- [Monitoring Dashboard](./docs/MONITORING.md) - System metrics, container monitoring, and resource alerts
- [Compose Versioning](./docs/COMPOSE_VERSIONING.md) - Version control and rollback system
- [Cloudflare Integration](./docs/CLOUDFLARE_ZERO_TRUST.md) - Tunnel setup and Zero Trust configuration

### Authentication
- [Cloudflare Zero Trust Setup](./docs/CLOUDFLARE_ZERO_TRUST.md) - Recommended authentication method
- [GitHub OAuth Setup](./docs/GITHUB_WHITELIST.md) - Alternative authentication option

## 🚦 Key Workflows

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

## 🎯 Use Cases

Perfect for:
- 🏠 Self-hosting enthusiasts managing apps on a Raspberry Pi
- 🔧 Developers running multiple services on a home lab server
- 📦 Anyone tired of SSH-ing to manage Docker containers
- 🚀 Quick deployment of Docker Compose applications with public URLs
- 📊 Monitoring resource usage without htop or SSH access

Not suitable for:
- ❌ Multi-user/multi-tenant environments (single-user design)
- ❌ Production SaaS platforms (no resource isolation between users)
- ❌ Enterprise deployments requiring RBAC and audit logs

## 🔧 Troubleshooting

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

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

MIT License - see [LICENSE](./LICENSE) file for details

## 🙏 Acknowledgments

- Built with [Fiber](https://gofiber.io/) - Fast HTTP framework for Go
- UI powered by [Radix UI](https://www.radix-ui.com/) and [TailwindCSS](https://tailwindcss.com/)
- System metrics via [gopsutil](https://github.com/shirou/gopsutil)
- Code editor via [Monaco Editor](https://microsoft.github.io/monaco-editor/)

---

**Made with ❤️ for the self-hosting community**
