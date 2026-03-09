# AGENTS.md

This document provides guidance for AI agents working on the Selfhostly project. It outlines the architecture, key components, and development patterns to ensure consistent and effective assistance.

## Project Overview

Selfhostly is a web-based platform for managing self-hosted applications on Raspberry Pi or Linux servers. It features:

- **Multi-Node Architecture** - Manage applications across multiple servers from a single UI
- **Docker Compose Management** - Deploy and manage applications through an intuitive web interface
- **Cloudflare Integration** - Automatic tunnel setup and ingress rule management
- **System Monitoring** - Real-time CPU, memory, disk, and container statistics
- **Version Control** - Automatic versioning with rollback capability

## Architecture Overview

### Backend (Go)
- **Main Server**: Primary node with UI and database (`cmd/server/main.go`)
- **API Gateway**: Optional gateway for routing requests (`cmd/gateway/main.go`)
- **Secondary Nodes**: Worker nodes for distributed app management

### Frontend (React/TypeScript)
- Built with React, TypeScript, TailwindCSS, and Radix UI
- Monaco editor for YAML compose file editing
- Real-time updates using TanStack Query and Zustand

### Key Components

#### Backend Components
```
internal/
├── config/          # Configuration loading and management
├── db/              # Database models and operations
├── docker/          # Docker and Compose management
├── gateway/         # API gateway proxy logic
├── http/            # HTTP handlers and routing
├── node/            # Multi-node communication
├── routing/         # Request routing and aggregation
├── service/         # Business logic layer
├── system/          # System metrics collection
├── tunnel/          # Tunnel provider abstraction
└── cloudflare/      # Cloudflare API client and tunnel management
```

#### Frontend Components
```
web/src/
├── features/        # Feature-based modules
│   ├── app-details/ # App detail view and actions
│   ├── monitoring/  # System and container monitoring
│   └── dashboard/   # Main dashboard view
└── shared/          # Shared utilities, types, and components
```

## Key Development Patterns

### Backend Patterns

#### 1. Service Layer Architecture
- Business logic is encapsulated in service packages (`internal/service/`)
- Services implement interfaces defined in `internal/domain/ports.go`
- Services depend on repositories/data access interfaces

#### 2. Handler Pattern
- HTTP handlers in `internal/http/` package are thin controllers
- Handler calls service methods for business logic
- Input validation happens via `internal/validation/` package

#### 3. Error Handling
- Domain errors are defined in `internal/domain/errors.go`
- Errors are wrapped with context and propagated up the stack
- HTTP handlers convert domain errors to appropriate HTTP responses

#### 4. Docker Management
- Docker operations are abstracted through `internal/docker/manager.go`
- Commands are executed asynchronously through job processors
- Container stats are collected via gopsutil

#### 5. Multi-Node Communication
- Nodes communicate via HTTP with API key authentication
- Heartbeats report node status automatically
- Circuit breakers handle node failures gracefully

### Frontend Patterns

#### 1. Feature-Based Architecture
- Each major feature has its own directory in `web/src/features/`
- Components are organized by feature rather than type
- Shared components go in `web/src/shared/`

#### 2. State Management
- Zustand for simple global state
- TanStack Query for server state
- Local component state for UI interactions

#### 3. TypeScript Types
- Strong typing throughout the frontend
- API responses typed based on backend models
- Component props are explicitly typed

## Important Development Notes

### Multi-Node Considerations
- Secondary nodes auto-register with primary node on startup
- All operations are designed to work across multiple nodes
- Node failures should be handled gracefully

### Security Design
- Application is designed for single-user deployments only
- No multi-tenant or multi-user support
- Either Cloudflare Zero Trust or GitHub OAuth for authentication

### Database
- Uses SQLite with modernc.org/sqlite (pure Go implementation)
- No CGO dependencies for easier cross-compilation
- Automatic migrations handle schema changes

### Job Processing
- Background jobs handle long-running operations
- Progress tracking for job status
- Job registry manages job lifecycle

### Docker Integration
- Direct Docker API access via socket
- Compose file parsing and validation
- Container lifecycle management

## When Working on This Project

### For AI Agents

1. **Always Read the README First** - Understand the current state and architecture
2. **Follow Existing Patterns** - Maintain consistency with existing code structure
3. **Maintain Separation of Concerns** - Keep handlers thin, services focused
4. **Test Your Changes** - Include both Go and frontend tests where appropriate
5. **Update Documentation** - Document new features or changes to existing ones
6. **Consider Multi-Node Impact** - Will changes work across multiple nodes?

### Common Tasks

#### Adding New Features
1. Define models in `internal/db/models.go`
2. Add service methods in appropriate service package
3. Create HTTP handlers for API endpoints
4. Implement frontend components in appropriate feature directory
5. Update TypeScript types for frontend-backend communication

#### Bug Fixes
1. Reproduce the issue first
2. Add tests to verify the fix
3. Ensure fix doesn't break existing functionality
4. Update any related documentation

#### Performance Optimizations
1. Profile before and after changes
2. Consider cache invalidation strategies
3. Monitor impact on multi-node performance
4. Test with large numbers of containers

### Testing Guidelines

#### Backend Tests
- Unit tests for service methods
- Integration tests for HTTP handlers
- Mock external dependencies (Docker, Cloudflare)
- Test error cases and edge conditions

#### Frontend Tests
- Component tests for UI interactions
- Integration tests for API calls
- Mock API responses for predictable testing
- Test error states and loading indicators

## Development Commands Reference

### Backend Development
```bash
make dev                 # Start full dev environment
make dev-backend        # Start only backend with live reload
make run-local          # Run backend locally with Air
make test              # Run Go tests
make logs-backend       # View backend logs
```

### Frontend Development
```bash
cd web && npm run dev    # Start frontend dev server
cd web && npm run build # Build for production
cd web && npm run lint  # Run linting
```

### Multi-Node Setup
- Configure primary node with `NODE_IS_PRIMARY=true`
- Configure secondary nodes with `NODE_IS_PRIMARY=false`
- Set `PRIMARY_NODE_URL` to point to primary node
- Secondary nodes auto-register on startup

## API Considerations

### Authentication
- Cloudflare Zero Trust preferred (handled by proxy)
- GitHub OAuth as alternative with username whitelist
- API key for node-to-node communication

### Response Format
- JSON API responses with consistent error format
- Timestamps in ISO 8601 format
- Pagination for large result sets
- Real-time updates via WebSocket where appropriate

### Rate Limiting
- Node API endpoints have rate limiting
- User operations have reasonable limits
- Long-running operations use job queues

## Security Validation

### Docker Compose Security

As of the latest version, Selfhostly implements comprehensive security validation for Docker Compose files to prevent host machine compromise.

#### Blocked Configurations

The following configurations are **automatically blocked** during compose file validation:

1. **Privileged Mode** - `privileged: true`
2. **Dangerous Volume Mounts**:
   - `/var/run/docker.sock` (Docker socket)
   - `/` (root filesystem)
   - `/etc`, `/proc`, `/sys`, `/dev`, `/boot` (system directories)
   - `/root`, `/home/*` (user directories with credentials)
   - `/var/lib/docker` (Docker internals)
3. **Device Access** - `devices:` section
4. **Host Network Mode** - `network_mode: host`
5. **Host PID Namespace** - `pid: host`
6. **Host IPC Namespace** - `ipc: host`
7. **Dangerous Capabilities** - `cap_add: [SYS_ADMIN, SYS_MODULE, SYS_PTRACE, NET_ADMIN, DAC_OVERRIDE, ALL]`
8. **Disabled Security Features** - `security_opt: [apparmor=unconfined, seccomp=unconfined, label=disable]`
9. **Custom Cgroup Parent** - `cgroup_parent:`
10. **Dangerous Tmpfs Mounts** - tmpfs on system paths

#### Allowed Configurations

Safe configurations that work normally:

- **Safe Volume Mounts**: `/data/*`, `/mnt/*`, `/opt/*`, named volumes, relative paths
- **Safe Capabilities**: `NET_BIND_SERVICE`, `CHOWN`, `SETUID`, `SETGID`
- **Safe Network Modes**: `bridge`, `container:name`, custom networks
- **Safe Security Options**: `no-new-privileges=true`, custom AppArmor/seccomp profiles

#### Implementation Location

- **Validation Logic**: `internal/validation/validation.go`
- **Test Suite**: `internal/validation/security_test.go`
- **Documentation**: `docs/SECURITY.md`
- **Quick Reference**: `docs/SECURITY_QUICK_REFERENCE.md`
- **Examples**: `examples/security-examples.yml`

#### When Working on Security

1. **Never bypass security validation** - These protections are critical
2. **Add tests for new validations** - Maintain test coverage
3. **Document security changes** - Update `docs/SECURITY.md`
4. **Consider false positives** - Ensure legitimate use cases still work
5. **Test attack scenarios** - Verify malicious configs are blocked

## Troubleshooting Common Issues

### Docker Issues
- Check Docker socket permissions
- Verify container connectivity
- Monitor resource usage
- Check Docker daemon health

### Multi-Node Issues
- Verify network connectivity
- Check API key authentication
- Monitor heartbeat status
- Ensure proper node registration

### Frontend Issues
- Check browser console for errors
- Verify API response format
- Test authentication flow
- Ensure CORS policies are correct

### Security Validation Issues
- Review compose file for blocked configurations
- Check error message for specific violation
- Consult `docs/SECURITY_QUICK_REFERENCE.md`
- Use safe alternatives (e.g., named volumes instead of host paths)

---

This document should be updated as the project evolves to reflect new patterns, technologies, or architectural decisions.
