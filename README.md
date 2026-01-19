# Selfhostly

A web-based tool for managing self-hosted applications with Cloudflare tunnel management and zero-downtime updates.

## Tech Stack

- **Backend**: Go (Gin framework)
- **Frontend**: React + TypeScript + Vite
- **Design System**: shadcn/ui (Radix UI + Tailwind CSS)
- **Database**: SQLite
- **Container Runtime**: Docker
- **Tunnel**: Cloudflare Tunnels

## Features

- Create and manage self-hosted applications
- Automatic Cloudflare tunnel setup
- Zero-downtime container updates
- Real-time app status monitoring
- Docker-compose YAML editor with validation
- Preview merged compose files
- Optional authentication

## Project Structure

```
selfhostly/
├── cmd/server/           # Backend entry point
├── internal/             # Private application code
│   ├── config/         # Configuration management
│   ├── db/             # Database models and operations
│   ├── docker/          # Docker compose operations
│   ├── cloudflare/      # Cloudflare API integration
│   ├── http/            # HTTP handlers
│   ├── app/             # App business logic
│   └── domain/          # Domain entities
├── web/                  # Frontend (React)
│   └── src/
│       ├── app/          # App-level setup
│       ├── features/     # Feature-based components
│       └── shared/        # Shared utilities
└── pkg/                  # Public libraries
```

## 🚀 Quick Start (Docker)

The easiest way to run Selfhostly is with Docker:

```bash
# 1. Pull the image
docker pull ghcr.io/YOUR_USERNAME/selfhostly:latest

# 2. Create a .env file (copy from env.example)
cp env.example .env
# Edit .env with your configuration

# 3. Run with docker-compose
docker compose -f docker-compose.prod.yml up -d
```

### With Cloudflare Tunnel

```bash
# Include the tunnel profile
docker compose -f docker-compose.prod.yml --profile tunnel up -d
```

## ⚙️ Configuration

Copy `env.example` to `.env` and configure:

| Variable | Required | Description |
|----------|----------|-------------|
| `GITHUB_CLIENT_ID` | Yes | GitHub OAuth App Client ID |
| `GITHUB_CLIENT_SECRET` | Yes | GitHub OAuth App Client Secret |
| `JWT_SECRET` | Yes | Random secret for JWT tokens |
| `BASE_URL` | Yes | Public URL (e.g., `https://selfhostly.example.com`) |
| `CLOUDFLARE_API_TOKEN` | No | For tunnel management |
| `CLOUDFLARE_ACCOUNT_ID` | No | For tunnel management |

### Creating a GitHub OAuth App

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click "New OAuth App"
3. Set **Homepage URL** to your `BASE_URL`
4. Set **Authorization callback URL** to `{BASE_URL}/auth/github/callback`
5. Copy the Client ID and Client Secret to your `.env`

## 🔧 Development Setup

### Backend Setup

1. Install Go dependencies:
```bash
go get github.com/gin-gonic/gin
go get github.com/mattn/go-sqlite3
go get golang.org/x/crypto/bcrypt
go get gopkg.in/yaml.v3
```

2. Configure environment variables:
```bash
export SERVER_ADDRESS=:8080
export DATABASE_PATH=./data/automaton.db
export CLOUDFLARE_API_TOKEN=your_token_here
export CLOUDFLARE_ACCOUNT_ID=your_account_id
export JWT_SECRET=your_jwt_secret
```

3. Run the server:
```bash
go run cmd/server/main.go
```

### Frontend Setup

1. Install dependencies:
```bash
cd web
npm install
```

2. Start development server:
```bash
npm run dev
```

3. Build for production:
```bash
npm run build
```

## API Endpoints

### Apps

- `GET /api/apps` - List all apps
- `POST /api/apps` - Create new app
- `GET /api/apps/:id` - Get app details
- `PUT /api/apps/:id` - Update app
- `DELETE /api/apps/:id` - Delete app
- `POST /api/apps/:id/start` - Start app
- `POST /api/apps/:id/stop` - Stop app
- `POST /api/apps/:id/update` - Update containers (zero-downtime)
- `GET /api/apps/:id/logs` - Get app logs

### Settings

- `GET /api/settings` - Get settings
- `PUT /api/settings` - Update settings

### Authentication (when enabled)

- `POST /api/auth/login` - Login
- `POST /api/auth/create-user` - Create user

## Usage

1. **Create an App**:
   - Click "New App" button
   - Provide app name and description
   - Upload or paste docker-compose.yml
   - Review and deploy

2. **Manage Apps**:
   - View all apps on dashboard
   - Start/stop apps with one click
   - View logs and update containers
   - Access apps via public Cloudflare URLs

3. **Update Apps**:
   - Click update button for zero-downtime update
   - Monitor progress in real-time
   - Tunnel stays active during update

4. **Configure Settings**:
   - Set up Cloudflare API credentials
   - Enable/disable authentication
   - Manage app preferences

## Docker Compose Injection

The tool automatically injects a cloudflared service into your docker-compose.yml:

```yaml
cloudflared:
  image: cloudflare/cloudflared:latest
  container_name: ${APP_NAME}-cloudflared
  command: tunnel --no-autoupdate run --token ${TUNNEL_TOKEN}
  restart: unless-stopped
  networks:
    - ${APP_NETWORK}  # Only if app defines networks
```

If your app defines custom networks, cloudflared will automatically connect to them.

## Security

- Store Cloudflare API tokens securely
- Use bcrypt for password hashing
- Validate all user inputs
- Sanitize YAML content before parsing
- Restrict Docker API access to local only

## Development

### Backend Development

```bash
# Run with hot reload
go run cmd/server/main.go

# Or build and run
go build -o bin/server cmd/server/main.go
./bin/server
```

### Frontend Development

```bash
# Start dev server with proxy to backend
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

## License

MIT
