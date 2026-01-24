# Development Guide

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
   go run cmd/server/main.go
   ```

### Configuration

The Air configuration (`.air.toml`) includes:

- **Watched directories**: All Go files except `web/`, `tmp/`, `vendor/`, etc.
- **Excluded files**: Test files (`*_test.go`)
- **Build command**: `go build -o ./tmp/main ./cmd/server/main.go`
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
