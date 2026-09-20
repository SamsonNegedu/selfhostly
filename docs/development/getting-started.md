# Development Guide

`make help` lists the commands. The short version:

```bash
make dev        # backend and frontend in Docker with live reload
make backend    # or on this machine, no Docker: backend with hot reload on :8080
make frontend   # frontend dev server on :5173 (run `cd web && npm install` once first)
make test       # Go tests
make down       # stop the containers
```

Open http://localhost:5173. Vite proxies `/api` and `/auth` to the backend.

## Live Reload Setup

This project uses [Air](https://github.com/cosmtrek/air) for Go live reload during development.

### Quick Start

To run the application with live reload:

```bash
# Start all services with live reload
docker-compose -f docker-compose.dev.yml up

# Or start just the backend with live reload
docker-compose -f docker-compose.dev.yml up backend
```

### How It Works

- **Air** watches your Go source files for changes
- When a file changes, Air automatically rebuilds and restarts the server
- Configuration is stored in `.air.toml`
- Temporary build artifacts are stored in `tmp/` (gitignored)

### Local Development (without Docker)

If you prefer to run the Go server locally without Docker:

1. Install Air:
   ```bash
   go install github.com/air-verse/air@latest
   ```

2. Run with Air:
   ```bash
   air
   ```

3. Or run directly:
   ```bash
   go run ./cmd/server
   ```

### Configuration

The Air configuration (`.air.toml`) includes:

- **Watched directories**: All Go files except `web/`, `tmp/`, `vendor/`, etc.
- **Excluded files**: Test files (`*_test.go`)
- **Build command**: `go build -o ./tmp/main ./cmd/server`
- **Restart delay**: 1 second after file changes

### Environment Variables

Create a `.env` file in the project root:

```bash
cp env.example .env
```

Then edit `.env` with your configuration.

### Production vs Development

- **Development**: `docker-compose.dev.yml` - Uses `Dockerfile.dev` with Air, mounts source code
- **Production**: `docker-compose.prod.yml` - Uses optimized multi-stage build, smaller image

### Tips

- Air will only watch Go files - frontend changes use the Vite dev server
- If Air gets stuck, restart the container: `docker-compose -f docker-compose.dev.yml restart backend`
- Check build errors in `build-errors.log` if the server doesn't start

## Run with the API gateway locally

The primary and the gateway need different ports. Put these in `.env` (or a file passed as `ENV_FILE`):

```env
# primary
SERVER_ADDRESS=:8082
NODE_API_ENDPOINT=http://localhost:8082
GATEWAY_API_KEY=dev-gateway-secret

# gateway
GATEWAY_LISTEN_ADDRESS=:8080
PRIMARY_BACKEND_URL=http://localhost:8082
```

```bash
make backend    # primary on :8082
make gateway    # gateway on :8080, hot reloads
make frontend   # Vite proxies to the gateway on :8080; open http://localhost:5173
```

The path is browser 5173 to the gateway on 8080 to the primary on 8082. `ENV_FILE=.env.primary` and
`ENV_FILE=.env.gateway` keep the two configurations apart.

## Building

```bash
cd web && npm run build          # frontend
go build -o bin/server ./cmd/server
go build -o bin/gateway ./cmd/gateway
```

Images: `Dockerfile.backend` (primary and secondary), `Dockerfile.gateway`, `web/Dockerfile` (frontend),
`Dockerfile.dev` (live reload). `go test ./...` and `cd web && npm run lint` are the checks.

The command line tool has its own build (`make ctl` gives `bin/selfhostlyctl`). After changing one of its commands or
flags, run `make docs` to regenerate [the reference](../reference/selfhostlyctl.md). Deployment-path tests:
[testing-deployments.md](testing-deployments.md).

## UI gallery

With the dev server running, open `/dev/ui` to see every UI primitive (buttons, fields, tabs, tables, status pills,
charts, terminal, dialogs, toasts, job progress) in both themes, and `/dev/login` for the sign-in page while you are
signed in. Both routes exist only in development and are left out of production builds. The design tokens they use
are defined in `web/src/styles/globals.css`.
