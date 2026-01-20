# Selfhostly

A web-based tool for managing self-hosted applications with automatic Cloudflare tunnel setup and zero-downtime updates.

## Features

- 🚀 Create and manage Docker Compose applications
- 🌐 Automatic Cloudflare tunnel configuration
- 🔄 Zero-downtime container updates
- 📊 Real-time app status monitoring
- 📝 Docker Compose editor with YAML validation
- 📜 Version history and rollback support
- 🔒 Optional authentication (Cloudflare Zero Trust recommended)

## Tech Stack

Go • React + TypeScript • SQLite • Docker • Cloudflare Tunnels

## ⚠️ Security Notice

**Single-user design only.** Intended for personal use (e.g., Raspberry Pi hosting).

- ✅ **Recommended:** Deploy behind [Cloudflare Zero Trust](./docs/CLOUDFLARE_ZERO_TRUST.md) (no OAuth needed)
- ✅ Alternative: [GitHub OAuth authentication](./docs/GITHUB_WHITELIST.md)
- ❌ Not suitable for multi-user environments

📖 Full details: [Security Documentation](./docs/SECURITY.md)

## 🚀 Quick Start

```bash
# 1. Clone and configure
git clone <repo-url>
cp env.example .env
# Edit .env with your settings

# 2. Run with Docker Compose
docker compose -f docker-compose.prod.yml up -d

# With Cloudflare Tunnel (optional)
docker compose -f docker-compose.prod.yml --profile tunnel up -d
```

Access at `http://localhost:8080`

## ⚙️ Configuration

### Required Environment Variables

```env
SERVER_ADDRESS=:8080
DATABASE_PATH=./data/automaton.db
```

### Authentication (Optional)

**Option 1 - Cloudflare Zero Trust** (Recommended)
```env
AUTH_ENABLED=false  # Cloudflare handles auth
CLOUDFLARE_API_TOKEN=your_token
CLOUDFLARE_ACCOUNT_ID=your_account_id
```
See: [Cloudflare Zero Trust Setup](./docs/CLOUDFLARE_ZERO_TRUST.md)

**Option 2 - GitHub OAuth**
```env
AUTH_ENABLED=true
GITHUB_CLIENT_ID=your_client_id
GITHUB_CLIENT_SECRET=your_client_secret
GITHUB_ALLOWED_USERS=username1,username2
AUTH_BASE_URL=https://your-domain.com
```
See: [GitHub OAuth Setup](./docs/GITHUB_WHITELIST.md)

## 🔧 Development

**Backend:**
```bash
go run cmd/server/main.go  # Backend runs on :8080
```

**Frontend:**
```bash
cd web
npm install
npm run dev  # Frontend runs on :5173 (proxies to backend)
```

**Build:**
```bash
cd web && npm run build  # Builds to web/dist
go build -o bin/server cmd/server/main.go
```

## 📚 Documentation

- [Security Model](./docs/SECURITY.md) - Single-user design, limitations
- [Cloudflare Zero Trust Setup](./docs/CLOUDFLARE_ZERO_TRUST.md) - Recommended auth
- [GitHub OAuth Setup](./docs/GITHUB_WHITELIST.md) - Alternative auth
- [Compose Versioning](./docs/COMPOSE_VERSIONING.md) - Version control system
- [API Documentation](./docs/API_CLIENT_REFACTORING.md) - API reference

## License

MIT
