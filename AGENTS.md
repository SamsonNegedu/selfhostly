# AGENTS.md

Selfhostly is a web platform for managing self-hosted Docker Compose apps across one or more
Raspberry Pi or Linux nodes. Go backend (`cmd/server`, `cmd/gateway`, `internal/`), React and
TypeScript frontend (`web/`).

This file is a map, not a manual. It points to the source of truth instead of restating it, so it
should rarely need editing. If you find yourself adding a directory listing, a command list or a
feature description here, link to where it already lives instead.

## Rules

Read `.agents/README.md`. The rule files in `.agents/rules/` are authoritative and each one says when
it applies. Load the ones relevant to the task. Never restate a rule here.

## Where things live

| Need | Look at |
| --- | --- |
| Commands (run, test, logs) | `make help` |
| Local setup and workflow | `docs/development/getting-started.md` |
| Security model, blocked compose configs, `SECURITY_MODE` | `docs/security/overview.md` and `internal/validation/` |
| Deploy and restart | `docs/operations/install.md`, `docs/operations/operate.md`, `docs/reference/selfhostlyctl.md` |
| Updating from the UI (signed releases, the updater container) | `docs/design/ui-updates.md` |
| Multi-node and gateway | `docs/operations/multi-node.md`, `docs/operations/gateway-deployment.md` |
| UI decisions | `docs/design/ui.md` |
| Missing backend features | `docs/design/backend-gaps.md` |
| Live updates, audit log, error codes | `docs/design/realtime-and-audit.md` |
| UI without real infrastructure | `docs/dev-fixtures/README.md` |
| Config and env vars | `internal/config/` and `env.example` |
| Design tokens | `web/src/styles/globals.css` |

## Non-obvious facts

- The server entry point is a multi-file package: `go build ./cmd/server`, not `main.go`.
- Single-user deployments only. Authorization uses the stable GitHub user ID, never the display name.
- Any new policy that could block an already-running app must follow `SECURITY_MODE`
  (`enforce` or `warn`) and be covered by `selfhostly doctor`.
- SQLite via `modernc.org/sqlite` (pure Go, no CGO). Keep it that way.
- Changes should work across nodes, including when a node is down.

## Working here

- Follow the patterns in the code next to your change before inventing new ones.
- Add tests with the change. Bug fixes start with a failing test.
- Update the doc that owns a behavior when you change it. Do not add it here.
- Never bypass compose security validation. New validations need tests and a `docs/security/overview.md` entry.
