# Architecture

How Selfhostly is put together and why. For running it see [install](../operations/install.md) and
[operate](../operations/operate.md); for the security model see [security overview](../security/overview.md).

## What it is

A web app for managing Docker Compose applications on machines you own, with optional Cloudflare Tunnel publishing. One
**primary** node holds the database and the UI's API. Optional **secondary** nodes run apps on other machines and are
controlled from the same UI. Versions of Go and the web dependencies are in `go.mod` and `web/package.json`.

It is built for one owner or a small trusted group (an allow-list of people), not for multi-tenant use: there is no
per-user data isolation. Authentication is the perimeter, and the compose policy limits what an app may ask of the host.

## The pieces

```
Browser → Cloudflare Tunnel → Gateway :8080 ─┬→ Primary :8082 ── docker compose ── your apps
                                             │      │  └── SQLite (data/)
                                             │      └──── Cloudflare API (tunnels, DNS)
                                             └→ Secondary nodes (direct), or via the primary (linked)
Secondary ──────── outbound WebSocket link ───────────────→ Primary
```

| Piece | Role |
|---|---|
| Frontend (`web/`) | React single-page app: dashboard, app details, create app, monitoring, nodes, settings, login. |
| Gateway (`cmd/gateway`, `internal/gateway`) | Public entry point: verifies the session, routes each request, keeps a node registry. See [gateway](../operations/gateway-deployment.md). |
| Server (`cmd/server`) | The backend, running as primary or secondary. Also carries the `doctor`, `backup` and `join-token` subcommands. |
| `selfhostlyctl` (`cmd/selfhostlyctl`, `internal/ctl`) | The operator's command line: setup, join, upgrade, backup, checks. It runs on the host and talks to Docker; it is not part of the server. |

## Backend layers

Handlers stay thin: they bind and validate input, call a service through a port, and answer. Business rules live in
services, and SQL lives in `internal/db`.

```
internal/http      handlers, middleware, routes         (Gin)
internal/domain    ports (interfaces), request types, coded errors
internal/service   business rules: apps, tunnels, compose versions, nodes, node link, system, schedules, security
internal/db        SQLite access, versioned migrations, change notifications
internal/docker    runs `docker compose` and `docker` (command executor, so it can be faked in tests)
internal/cloudflare, internal/tunnel   Cloudflare API and the provider-agnostic tunnel layer
```

Supporting packages:

| Package | Purpose |
|---|---|
| `internal/config`, `internal/constants` | Settings with their sources, startup validation, every shared constant |
| `internal/auth` | GitHub allow-list (by stable user ID), Cloudflare Access token verification |
| `internal/validation` | The compose policy (what an app may mount and ask for) and input validation |
| `internal/secrets` | Encryption of stored secrets at rest |
| `internal/netguard`, `internal/node` | Safe outbound calls and the client for talking to other nodes |
| `internal/nodelink` | The outbound WebSocket link a secondary keeps open to the primary |
| `internal/routing` | Decides which node a request is for and whether it runs here |
| `internal/jobs`, `internal/scheduler` | Background worker for slow operations, and cron-style app start/stop schedules |
| `internal/events` | The change stream that tells the browser what changed ([live updates](realtime-and-audit.md)) |
| `internal/cleanup` | Deleting an app: containers, tunnel, DNS, folder, records |
| `internal/system` | CPU, memory, disk and container statistics |

Errors are domain errors with codes (`internal/domain`); handlers turn them into HTTP statuses and never expose internals.
Every state-changing route has an audit action. Database changes are additive and versioned so an older binary still runs
against a newer schema.

## Nodes

Each request that names a node is routed by `node_id`: to this machine, to a direct node's address, or, for a node that
connects out, down that node's link through the primary using the node's own credentials. Fan-out operations (system stats
across all nodes) query nodes in parallel with a timeout each, so one unreachable node does not hold up the rest.
Design: [node link RFC](../rfcs/node-link.md); operation: [multi-node](../operations/multi-node.md).

## How the important flows work

**Deploying an app.** The UI posts the compose file. The server checks it against the compose policy, creates the
tunnel if one is wanted and adds the `cloudflared` service, writes the app folder, records the app and its first compose
version, and hands the slow part (pull and start) to the background worker as a job. The UI follows the job's log lines while
it runs. Each app is its own compose project, so restarting the platform never restarts apps.

**Compose versions.** Every change to an app's compose file creates a numbered version with the reason and who changed it. A
rollback creates a new version whose content is the old one, so history is never rewritten
([details](compose-versioning.md)).

**Monitoring.** The browser polls `GET /api/system/stats`, the server collects local numbers and asks the other nodes in
parallel, and it returns one entry per reachable node ([details](../operations/monitoring.md)).

**Deleting an app.** Independent steps (stop containers, remove networks and volumes, delete the tunnel and DNS records,
remove the folder and records). A failing step is logged and the rest still run.

**Starting up.** The server resolves its settings and where each came from, checks them (refusing unsafe ones in `enforce`
mode), backs up the database before applying any pending schema migration, then starts the job worker and the health checks.
`selfhostlyctl doctor` reproduces this without changing anything.

## Where state lives

| State | Location |
|---|---|
| Apps, nodes, jobs and their logs, schedules, compose versions, tunnels, settings, audit log, join tokens, security state | SQLite, `data/selfhostly.db` (tables in `internal/db/db.go` and `internal/db/versioned.go`) |
| Encryption key, node identity and keys, saved security mode | small files beside the database in `data/` |
| Each app's compose file and files | `apps/<name>/` |
| Settings | `.env` and the compose file's environment |

## Design choices

1. **Thin handlers, services behind ports.** Testable business rules and swappable infrastructure (the command executor and
   HTTP clients are faked in tests).
2. **The host is reached through the `docker compose` command line**, not the Docker SDK: the same commands you would type,
   behaving the same on a Pi as on a laptop.
3. **Apps are separate compose projects.** The platform can restart, upgrade or fail without touching them.
4. **Safe by default, gradual to adopt.** New installs enforce the compose policy; an existing install starts in `warn` and
   only logs what it would block.
5. **Every change is explainable.** Settings show their source, state-changing requests are audited, and the change stream
   carries only what changed, never values.
6. **Feature-based frontend.** `web/src/features/<feature>/` holds a feature's components; shared code lives in
   `web/src/shared/`.

## Frontend

React 18, TypeScript, Vite, TanStack Query for server data, Zustand for local state, Tailwind and Radix for the interface,
CodeMirror 6 for the compose editor. Features: `app-details`, `cloudflare`, `create-app`, `dashboard`, `login`,
`monitoring`, `nodes`, `settings`. Design tokens and components: [ui.md](ui.md).
