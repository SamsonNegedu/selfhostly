# Project Architecture Documentation

## Table of Contents

1. [System Overview](#system-overview)
2. [Architecture Diagram](#architecture-diagram)
3. [Core Components](#core-components)
4. [Data Flow](#data-flow)
5. [Technology Stack](#technology-stack)
6. [Design Principles](#design-principles)
7. [API Architecture](#api-architecture)
8. [Database Schema](#database-schema)
9. [Security Architecture](#security-architecture)
10. [Deployment Architecture](#deployment-architecture)

---

## System Overview

**Selfhostly** is a web-based platform designed for managing self-hosted Docker Compose applications with integrated Cloudflare tunnel support. The system provides a comprehensive solution for deploying, monitoring, and managing containerized applications without requiring SSH access to the host server.

### Key Capabilities

- **Application Management**: Deploy and manage Docker Compose applications through a web UI
- **Cloudflare Integration**: Automatic tunnel creation and DNS configuration for public access
- **System Monitoring**: Real-time monitoring of system resources and container metrics
- **Version Control**: Complete versioning and rollback capability for compose files
- **Zero-Downtime Updates**: Pull and update containers without service interruption

### Target Use Case

Single-user self-hosting environment (e.g., Raspberry Pi, home lab server) where the user needs to manage multiple Docker-based applications remotely without SSH access.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Frontend Layer                            │
│                                                                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  Dashboard   │  │   App Mgmt   │  │  Monitoring  │              │
│  │  (React)     │  │  (React)     │  │  (React)     │              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │
│         │                  │                  │                       │
│         └──────────────────┼──────────────────┘                       │
│                            │                                          │
│                   ┌────────▼─────────┐                               │
│                   │   API Client     │                               │
│                   │   (TanStack      │                               │
│                   │    Query)        │                               │
│                   └────────┬─────────┘                               │
└────────────────────────────┼──────────────────────────────────────────┘
                             │ HTTP/REST
                             │
┌────────────────────────────▼──────────────────────────────────────────┐
│                          Backend Layer (Go)                           │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    HTTP Server (Gin)                          │   │
│  │  ┌─────────────┐  ┌────────────┐  ┌──────────────┐          │   │
│  │  │   Routes    │  │Middleware  │  │   Auth       │          │   │
│  │  │   Handler   │  │(CORS, etc) │  │   (go-pkgz)  │          │   │
│  │  └──────┬──────┘  └────────────┘  └──────────────┘          │   │
│  └─────────┼─────────────────────────────────────────────────────┘   │
│            │                                                           │
│  ┌─────────▼──────────────────────────────────────────────────────┐  │
│  │                      Service Layer                             │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │  │
│  │  │ AppService   │  │TunnelService │  │SystemService │        │  │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘        │  │
│  │         │                  │                  │                 │  │
│  │  ┌──────┴───────┐  ┌──────┴───────┐  ┌──────┴───────┐        │  │
│  │  │ComposeService│  │              │  │              │        │  │
│  │  └──────────────┘  │              │  │              │        │  │
│  └───────────────────┼──────────────┼──────────────────┼──────────┘  │
│                      │              │                  │              │
│  ┌───────────────────▼──────────────▼──────────────────▼───────────┐ │
│  │                    Infrastructure Layer                          │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │ │
│  │  │   Docker     │  │  Cloudflare  │  │   Database   │         │ │
│  │  │   Manager    │  │   Manager    │  │   (SQLite)   │         │ │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘         │ │
│  └─────────┼──────────────────┼──────────────────┼─────────────────┘ │
└────────────┼──────────────────┼──────────────────┼────────────────────┘
             │                  │                  │
┌────────────▼─────────┐ ┌──────▼──────────┐ ┌───▼──────────┐
│   Docker Engine      │ │ Cloudflare API  │ │  SQLite DB   │
│   - Containers       │ │ - Tunnels       │ │  - Apps      │
│   - Compose          │ │ - DNS Records   │ │  - Tunnels   │
│   - Networks         │ │ - Ingress Rules │ │  - Versions  │
└──────────────────────┘ └─────────────────┘ └──────────────┘
```

---

## Core Components

### 1. Frontend Layer (React + TypeScript)

#### Technology Stack
- **Framework**: React 18 with TypeScript
- **State Management**: 
  - Zustand for global state
  - TanStack Query (React Query) for server state
- **UI Components**: Radix UI primitives with custom styling
- **Styling**: TailwindCSS
- **Code Editor**: Monaco Editor (VS Code editor)
- **Icons**: Lucide Icons
- **Build Tool**: Vite

#### Key Features
- **Feature-Based Architecture**: Organized by features (dashboard, app-details, monitoring, etc.)
- **Shared Components**: Reusable UI components, utilities, and services
- **Theme Support**: Light/dark mode with system preference detection
- **Responsive Design**: Mobile, tablet, and desktop support
- **Real-time Updates**: Auto-refreshing metrics (every 10 seconds)

#### Features Structure
```
features/
├── app-details/      # Application detail page with logs, compose editor
├── cloudflare/       # Cloudflare tunnel and ingress management
├── create-app/       # Multi-step app creation workflow
├── dashboard/        # Main dashboard with app list
├── login/            # Authentication (if enabled)
├── monitoring/       # System and container monitoring
└── settings/         # Application settings
```

### 2. Backend Layer (Go)

#### HTTP Server (`internal/http/`)

**Server Components**:
- **Web Framework**: Gin (high-performance HTTP framework)
- **Authentication**: go-pkgz/auth with GitHub OAuth support
- **Middleware**:
  - CORS with configurable origins
  - Security headers (XSS, clickjacking protection, etc.)
  - Cache control for static assets
  - Request logging (structured logging with slog)

**Key Responsibilities**:
- Route registration and request handling
- Authentication and authorization
- Middleware pipeline management
- Static file serving (embedded frontend)

#### Service Layer (`internal/service/`)

The service layer implements business logic and orchestrates operations between different infrastructure components.

##### AppService
Manages the complete lifecycle of Docker Compose applications:
- **Create**: Parse compose files, inject Cloudflare tunnels, create app directories
- **Update**: Modify compose files, create version history
- **Delete**: Comprehensive cleanup (containers, networks, volumes, files, database records)
- **Start/Stop**: Control application lifecycle
- **Update Containers**: Zero-downtime updates with docker compose pull and rebuild
- **Repair**: Fix apps with missing Cloudflare tokens

##### TunnelService
Manages Cloudflare tunnel operations:
- Create and configure tunnels
- Update tunnel status
- Manage tunnel metadata
- Synchronize tunnel state

##### SystemService
Handles system-level operations:
- System resource monitoring (CPU, memory, disk)
- Container statistics collection
- Docker daemon health checks

##### ComposeService
Manages Docker Compose file operations:
- Version control for compose files
- Rollback capability
- Change tracking and auditing

#### Infrastructure Layer

##### Docker Manager (`internal/docker/`)

**Responsibilities**:
- Docker Compose operations (up, down, pull, restart)
- Container lifecycle management
- Log retrieval and streaming
- Network and volume management
- Statistics collection via Docker API

**Key Methods**:
- `CreateAppDirectory()`: Create app workspace and write compose file
- `StartApp()`: Start application containers with docker compose up
- `StopApp()`: Stop and remove containers with docker compose down
- `UpdateApp()`: Zero-downtime update (pull images, rebuild containers)
- `RestartCloudflared()`: Restart tunnel container to apply ingress changes
- `GetAppLogs()`: Fetch container logs
- `DeleteAppDirectory()`: Clean up app workspace

**Command Executor Pattern**: Abstracted command execution for testability

##### Cloudflare Manager (`internal/cloudflare/`)

**Responsibilities**:
- Cloudflare API interactions (tunnels, DNS, ingress)
- Tunnel creation and deletion
- Ingress rule management
- DNS record automation
- Credential management

**Key Components**:
- **HTTP Client**: Configurable HTTP client with retry logic
- **API Manager**: Core Cloudflare API operations
- **Tunnel Manager**: High-level tunnel orchestration with database integration

**Key Methods**:
- `CreateTunnel()`: Create a new Cloudflare tunnel
- `DeleteTunnel()`: Delete a tunnel and cleanup DNS records
- `CreateIngressConfiguration()`: Configure ingress rules for traffic routing
- `CreateDNSRecord()`: Create CNAME records pointing to tunnel
- `GetTunnelToken()`: Retrieve tunnel authentication token

##### Database (`internal/db/`)

**Database**: SQLite (modernc.org/sqlite - pure Go, no CGO)

**Key Features**:
- Foreign key constraints for referential integrity
- Automatic migrations on startup
- Single-user optimized schema
- JSON support for complex fields (ingress rules)

##### Configuration Manager (`internal/config/`)

**Configuration Sources**:
- Environment variables
- `.env` file (optional)
- Default values

**Configuration Domains**:
- Server settings (address, ports)
- Database path
- Apps directory
- Cloudflare credentials
- Authentication settings
- CORS allowed origins
- Auto-start behavior

---

## Data Flow

### 1. Application Creation Flow

```
┌──────────┐     ┌──────────┐     ┌──────────────┐     ┌──────────┐
│  User    │────▶│ Frontend │────▶│   Backend    │────▶│ Database │
│          │     │          │     │ AppService   │     │          │
└──────────┘     └──────────┘     └──────┬───────┘     └──────────┘
                                         │
                      ┌──────────────────┴────────────────┐
                      │                                    │
                      ▼                                    ▼
              ┌───────────────┐                  ┌────────────────┐
              │   Docker      │                  │  Cloudflare    │
              │   Manager     │                  │  API           │
              │               │                  │                │
              │ - Create Dir  │                  │ - Create Tunnel│
              │ - Write File  │                  │ - Create DNS   │
              │ - Up -d       │                  │ - Configure    │
              └───────────────┘                  └────────────────┘
```

**Steps**:
1. User submits compose file and app metadata
2. Backend parses and validates compose file
3. Create Cloudflare tunnel (if configured)
4. Inject cloudflared service into compose file
5. Create app directory and write compose file
6. Store app metadata in database
7. Create initial compose version record
8. Start containers with docker compose up
9. Apply ingress rules (if provided)
10. Restart cloudflared to load new configuration

### 2. Monitoring Data Flow

```
┌──────────┐     ┌──────────┐     ┌──────────────┐
│ Frontend │────▶│ Backend  │────▶│   Docker     │
│ (Polling)│     │ System   │     │   API        │
│          │◀────│ Service  │◀────│              │
└──────────┘     └──────────┘     └──────────────┘
     │
     │ Every 10 seconds
     │
     ▼
┌──────────────────────────────────────────┐
│  Display:                                │
│  - System metrics (CPU, memory, disk)    │
│  - Container list with stats             │
│  - Resource alerts                       │
└──────────────────────────────────────────┘
```

**Monitoring Flow**:
1. Frontend polls `/api/stats/system` and `/api/stats/containers`
2. Backend queries Docker API for container metrics
3. Backend uses gopsutil for system-level metrics
4. Data is formatted and returned as JSON
5. Frontend displays real-time metrics and alerts

### 3. Compose Version Control Flow

```
┌──────────┐     ┌──────────┐     ┌──────────────┐     ┌──────────┐
│  User    │────▶│ Frontend │────▶│   Backend    │────▶│ Database │
│ (Update  │     │          │     │ Compose      │     │ (Version │
│  Compose)│     │          │     │ Service      │     │  Table)  │
└──────────┘     └──────────┘     └──────┬───────┘     └──────────┘
                                         │
                                         │
                      ┌──────────────────┴────────────────┐
                      │                                    │
                      ▼                                    ▼
              ┌───────────────┐                  ┌────────────────┐
              │   Create      │                  │  Mark Current  │
              │   New Version │                  │  Version       │
              │   (v2, v3...) │                  │  (is_current)  │
              └───────────────┘                  └────────────────┘
```

### 4. Cloudflare Tunnel Flow

```
┌──────────┐     ┌──────────┐     ┌──────────────┐
│  User    │────▶│ Backend  │────▶│  Cloudflare  │
│          │     │ Tunnel   │     │  API         │
│          │     │ Manager  │     │              │
└──────────┘     └────┬─────┘     └──────┬───────┘
                      │                   │
                      │ 1. Create Tunnel  │
                      ├──────────────────▶│
                      │                   │
                      │ 2. Get Token      │
                      │◀──────────────────┤
                      │                   │
                      ▼                   │
              ┌───────────────┐           │
              │   Inject      │           │
              │  cloudflared  │           │
              │   Service     │           │
              └───────┬───────┘           │
                      │                   │
                      │ 3. Configure      │
                      │   Ingress Rules   │
                      ├──────────────────▶│
                      │                   │
                      │ 4. Create DNS     │
                      ├──────────────────▶│
                      │                   │
                      ▼                   ▼
              ┌───────────────────────────────┐
              │ Traffic flows through tunnel  │
              │ Public URL ──▶ Cloudflare ──▶│
              │ Internal Service               │
              └───────────────────────────────┘
```

---

## Technology Stack

### Backend

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Language** | Go 1.21+ | High-performance, compiled language with excellent concurrency |
| **Web Framework** | Gin | Fast HTTP router and middleware framework |
| **Database** | SQLite (modernc.org/sqlite) | Embedded database, pure Go (no CGO) |
| **Auth** | go-pkgz/auth | OAuth authentication with GitHub provider |
| **Docker API** | github.com/docker/docker | Native Docker client for container management |
| **System Metrics** | gopsutil | Cross-platform system and process utilities |
| **Config** | godotenv | Environment variable management |
| **Logging** | slog (standard library) | Structured logging |

### Frontend

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Framework** | React 18 | Component-based UI framework |
| **Language** | TypeScript | Type-safe JavaScript |
| **Build Tool** | Vite | Fast development server and build tool |
| **State Management** | Zustand | Lightweight state management |
| **Server State** | TanStack Query | Data fetching, caching, and synchronization |
| **Styling** | TailwindCSS | Utility-first CSS framework |
| **UI Components** | Radix UI | Unstyled, accessible component primitives |
| **Code Editor** | Monaco Editor | VS Code editor for compose files |
| **Icons** | Lucide Icons | Beautiful, consistent icon set |
| **HTTP Client** | Fetch API | Native browser API |

### Infrastructure

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Containerization** | Docker | Container runtime |
| **Orchestration** | Docker Compose | Multi-container application definition |
| **Tunneling** | Cloudflare Tunnels | Secure remote access without port forwarding |
| **Live Reload** | Air | Go application hot reload for development |

---

## Design Principles

### 1. Single-User Architecture

**Philosophy**: Designed for personal use, not multi-tenant SaaS.

**Implications**:
- No user isolation or row-level security
- Single settings record (global configuration)
- Optional authentication (recommended behind Cloudflare Zero Trust)
- Simplified access control model

**Security Model**: Authentication is a perimeter defense, not a data access control mechanism.

### 2. Service-Oriented Backend

**Service Layer Pattern**: Business logic is encapsulated in service objects that implement domain interfaces.

**Benefits**:
- Clear separation of concerns
- Testable business logic
- Easy to mock dependencies
- Promotes interface-based design

**Example**:
```go
type AppService interface {
    CreateApp(ctx context.Context, req CreateAppRequest) (*App, error)
    GetApp(ctx context.Context, appID string) (*App, error)
    UpdateApp(ctx context.Context, appID string, req UpdateAppRequest) (*App, error)
    DeleteApp(ctx context.Context, appID string) error
    // ... more methods
}
```

### 3. Feature-Based Frontend Organization

**Structure**: Code is organized by feature, not by type.

**Example**:
```
features/
├── app-details/
│   ├── components/
│   │   ├── AppActions.tsx
│   │   ├── LogViewer.tsx
│   │   └── ComposeEditor.tsx
│   └── index.tsx
└── monitoring/
    ├── components/
    │   ├── SystemOverview.tsx
    │   └── ContainersTable.tsx
    └── index.tsx
```

**Benefits**:
- Features are self-contained
- Easy to locate related code
- Clear boundaries between features
- Shared code explicitly lives in `shared/`

### 4. Comprehensive Error Handling

**Domain Errors**: Typed error handling with context.

```go
// internal/domain/errors.go
func WrapAppNotFound(appID string, err error) error
func WrapDatabaseOperation(operation string, err error) error
func WrapContainerOperationFailed(operation string, err error) error
```

**Structured Logging**: All operations are logged with context.

```go
s.logger.InfoContext(ctx, "creating app", "name", req.Name)
s.logger.ErrorContext(ctx, "failed to create app", "app", req.Name, "error", err)
```

### 5. Zero-Downtime Updates

**Strategy**: Use Docker Compose's built-in update mechanism.

**Process**:
1. Pull latest images (`docker compose pull`)
2. Recreate containers (`docker compose up -d --build`)
3. Docker Compose handles rolling updates
4. Old containers stay running until new ones are healthy

### 6. Automatic Versioning

**Compose File Versioning**: Every change to a compose file creates a new version.

**Version Metadata**:
- Version number (incremental)
- Change reason
- Changed by (if auth enabled)
- Timestamp
- Current version flag

**Rollback**: One-click rollback to any previous version.

### 7. Comprehensive Cleanup

**Cleanup Manager**: Centralized cleanup logic for application deletion.

**Cleanup Steps**:
1. Stop and remove Docker containers
2. Delete Docker networks
3. Remove Docker volumes (optional, configurable)
4. Delete Cloudflare tunnel
5. Remove DNS records
6. Delete app directory
7. Delete database records
8. Clean up compose versions

**Resilience**: Each step is independent; failures are logged but don't stop cleanup.

---

## API Architecture

### RESTful Design

All API endpoints follow REST conventions:

```
GET    /api/apps              # List all apps
POST   /api/apps              # Create new app
GET    /api/apps/:id          # Get app details
PUT    /api/apps/:id          # Update app
DELETE /api/apps/:id          # Delete app

POST   /api/apps/:id/start    # Start app
POST   /api/apps/:id/stop     # Stop app
POST   /api/apps/:id/update   # Update containers
POST   /api/apps/:id/repair   # Repair app

GET    /api/apps/:id/logs     # Get app logs

GET    /api/stats/system      # System metrics
GET    /api/stats/containers  # Container metrics

POST   /api/containers/:id/restart  # Restart container
POST   /api/containers/:id/stop     # Stop container
```

### Request/Response Format

**Request**:
```json
{
  "name": "my-app",
  "description": "My application",
  "compose_content": "version: '3.8'\nservices:\n...",
  "ingress_rules": [
    {
      "hostname": "myapp.example.com",
      "service": "http://app:3000"
    }
  ]
}
```

**Response (Success)**:
```json
{
  "id": "uuid-v4",
  "name": "my-app",
  "status": "running",
  "public_url": "https://myapp.example.com",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Response (Error)**:
```json
{
  "error": "failed to start app: container exited with code 1"
}
```

### Middleware Stack

1. **Security Headers**: X-Frame-Options, X-Content-Type-Options, etc.
2. **CORS**: Configurable origin whitelisting
3. **Cache Control**: Appropriate caching for static vs dynamic content
4. **Logger**: Structured request/response logging
5. **Auth** (optional): JWT token validation with GitHub OAuth

---

## Database Schema

### Tables

#### apps
```sql
CREATE TABLE apps (
    id TEXT PRIMARY KEY,                 -- UUID v4
    name TEXT NOT NULL UNIQUE,            -- App name
    description TEXT,                     -- Optional description
    compose_content TEXT NOT NULL,        -- Docker Compose YAML
    tunnel_token TEXT,                    -- Cloudflare tunnel token
    tunnel_id TEXT,                       -- Cloudflare tunnel ID
    tunnel_domain TEXT,                   -- Public domain
    public_url TEXT,                      -- Full public URL
    status TEXT NOT NULL DEFAULT 'stopped', -- App status
    error_message TEXT,                   -- Last error
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
)
```

#### cloudflare_tunnels
```sql
CREATE TABLE cloudflare_tunnels (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL UNIQUE,         -- Foreign key to apps
    tunnel_id TEXT NOT NULL,
    tunnel_name TEXT NOT NULL,
    tunnel_token TEXT NOT NULL,
    account_id TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    last_synced_at DATETIME,
    error_details TEXT,
    ingress_rules TEXT,                  -- JSON array
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
)
```

#### compose_versions
```sql
CREATE TABLE compose_versions (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL,
    version INTEGER NOT NULL,             -- Incremental version number
    compose_content TEXT NOT NULL,
    change_reason TEXT,
    changed_by TEXT,                      -- Username (if auth enabled)
    is_current INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    rolled_back_from INTEGER,             -- Previous version if rollback
    UNIQUE(app_id, version),
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
)
CREATE INDEX idx_compose_versions_app_id ON compose_versions(app_id);
CREATE INDEX idx_compose_versions_is_current ON compose_versions(app_id, is_current);
```

#### settings
```sql
CREATE TABLE settings (
    id TEXT PRIMARY KEY,
    cloudflare_api_token TEXT,
    cloudflare_account_id TEXT,
    auto_start_apps INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL
)
```

#### users
```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,              -- Not currently used (OAuth only)
    created_at DATETIME NOT NULL
)
```

### Relationships

```
apps (1) ──── (1) cloudflare_tunnels
  │
  └──── (∞) compose_versions
```

---

## Security Architecture

### Authentication Options

#### 1. Cloudflare Zero Trust (Recommended)

**Benefits**:
- No OAuth configuration needed
- Support for multiple identity providers
- Email-based access control
- Built-in 2FA support
- Enterprise-grade security

**Setup**: Deploy behind Cloudflare Access with authentication policy.

#### 2. GitHub OAuth

**Configuration**:
```env
AUTH_ENABLED=true
GITHUB_CLIENT_ID=your_client_id
GITHUB_CLIENT_SECRET=your_client_secret
GITHUB_ALLOWED_USERS=username1,username2
```

**Flow**:
1. User clicks "Login with GitHub"
2. Redirect to GitHub OAuth
3. GitHub returns user profile
4. Validate username against whitelist
5. Issue JWT token
6. Store JWT in secure cookie

### Security Middleware

1. **CORS**: Whitelist-based origin validation
2. **Security Headers**:
   - X-Frame-Options: DENY
   - X-Content-Type-Options: nosniff
   - X-XSS-Protection: 1; mode=block
   - Referrer-Policy: strict-origin-when-cross-origin
3. **HSTS** (if HTTPS): max-age=31536000; includeSubDomains
4. **JWT Validation**: Token expiry, signature verification, user whitelist

### Docker Socket Security

**Risk**: Docker socket access = root access.

**Mitigations**:
1. Run behind authentication (Cloudflare Zero Trust or GitHub OAuth)
2. Single-user design (no multi-tenant isolation)
3. Deploy in trusted network
4. Use firewall rules to restrict access

**Production Recommendation**: Deploy behind Cloudflare Zero Trust with email-based access control.

---

## Deployment Architecture

### Production Deployment

```
┌─────────────────────────────────────────────────────────┐
│                    Internet                             │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│             Cloudflare Tunnel (Optional)                │
│  - Secure ingress without port forwarding               │
│  - Zero Trust authentication                            │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│            Docker Host (Raspberry Pi / Server)          │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │  Selfhostly Container (Unified Image)              │ │
│  │  - Backend (Go)                                    │ │
│  │  - Frontend (Static files)                         │ │
│  │  - Port 8080                                       │ │
│  └────────────────────────────────────────────────────┘ │
│                        │                                 │
│  ┌─────────────────────┼──────────────────────────────┐ │
│  │         Docker Socket (/var/run/docker.sock)       │ │
│  └─────────────────────┼──────────────────────────────┘ │
│                        │                                 │
│  ┌─────────────────────▼──────────────────────────────┐ │
│  │        Application Containers                      │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────┐    │ │
│  │  │  App 1   │  │  App 2   │  │ cloudflared  │    │ │
│  │  └──────────┘  └──────────┘  └──────────────┘    │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │           Data Volume (./data)                     │ │
│  │  - SQLite database                                 │ │
│  │  - Application compose files                       │ │
│  └────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

### Development Deployment

```
┌────────────────────────────────────────────────────────┐
│         Local Development Machine                      │
│                                                         │
│  ┌──────────────────┐      ┌──────────────────┐       │
│  │   Frontend       │      │    Backend       │       │
│  │   (Vite)         │      │    (Air)         │       │
│  │   Port 5173      │      │    Port 8080     │       │
│  └────────┬─────────┘      └────────┬─────────┘       │
│           │                         │                  │
│           │  API Proxy              │                  │
│           └────────────────────────▶│                  │
│                                     │                  │
│  ┌──────────────────────────────────▼────────────────┐ │
│  │         Docker Socket (via host)                  │ │
│  └────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘
```

**Hot Reload**:
- Frontend: Vite HMR (Hot Module Replacement)
- Backend: Air (automatic rebuild on Go file changes)

### Docker Compose Profiles

**Development** (`docker-compose.dev.yml`):
- Backend service with Air
- Volume mounts for live code
- Debug logging

**Production** (`docker-compose.prod.yml`):
- Single unified container
- Embedded frontend
- Optimized build

**Tunnel Profile** (optional):
- Add `--profile tunnel` to include Cloudflare tunnel

---

## Conclusion

Selfhostly is a modern, well-architected platform for managing self-hosted applications. The system balances simplicity with powerful features, providing a comprehensive solution for single-user self-hosting scenarios. The architecture is designed to be maintainable, testable, and extensible, with clear separation of concerns and well-defined interfaces between components.

### Key Strengths

1. **Clean Architecture**: Clear layers (HTTP → Service → Infrastructure)
2. **Type Safety**: TypeScript frontend, Go backend
3. **Real-time Monitoring**: Container and system metrics
4. **Version Control**: Complete rollback capability
5. **Zero-Downtime**: Seamless container updates
6. **Cloudflare Integration**: Automatic tunnel and DNS management
7. **Developer Experience**: Hot reload, structured logging, comprehensive error handling

### Future Considerations

- Multi-user support with proper access control
- Kubernetes support alongside Docker Compose
- Enhanced observability (metrics, traces)
- Backup and restore functionality
- Application templates marketplace
- Plugin system for extensibility

---

**Document Version**: 1.0  
**Last Updated**: January 26, 2026  
**Maintained By**: Selfhostly Team
