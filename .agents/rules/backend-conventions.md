# Backend Conventions (Go)

## Overview

Selfhostly's Go backend has strict layering and several places where a change must be repeated. Miss
one and the failure is silent: a missing audit action, an event that never fires, a migration that
breaks an upgrade.

## Core Principles

**Layering.** HTTP handlers in `internal/http/` are thin. They bind and validate input, call a
service in `internal/service/` through a port in `internal/domain/ports.go`, and answer. Business
rules live in services. SQL lives in `internal/db/`. Do not query the database from a handler.

**Errors are domain errors with codes.** Return `domain.Wrap...` errors (`WrapConflict`,
`WrapNodeNotFound`, `WrapValidationError`) from services and let `handleServiceError` map them to
404, 409, 400 or 500. Give the client a `code` and, for form errors, a `field`. Never put a
`Cause` or driver text in a response: `PublicMessage` exists so internals do not leak.

**Every new state-changing route gets an audit action.** Add it to `auditRoutes` in
`internal/http/audit_actions.go`. A test fails if an entry points at a route that does not exist.
The audit log stores the action and the target's name, never request bodies or values.

**Database changes are additive and re-runnable.** Add a version to `versionedMigrations` in
`internal/db/versioned.go`. Never drop or rewrite existing data. `ALTER TABLE ... ADD COLUMN`
tolerates being run again. Existing installs are upgraded in place and rolled back by keeping the
old binary, so an old binary must still work against the new schema.

**Writes to apps, jobs and nodes notify the browser.** New write paths on those tables go through
`internal/db` methods that call `db.changed(...)`. Events say what changed, never the values.

**Nodes and node_id.** App and tunnel routes need a `node_id` and are routed by
`resolveNodeMiddleware`. Anything that fans out to nodes must give each node its own timeout and
must not let one unreachable node hold up the rest.

**User-level administration is closed to node credentials.** Routes that mint tokens, revoke
sessions, read the audit log or open the event stream use `denyNodeAuthMiddleware`, and a test in
`security_test.go` lists them.

**No magic values.** Timeouts, limits, header names and codes are constants in
`internal/constants` or the package that owns them (see `no-magic-values.md`).

## When you add an endpoint

1. Port method, service, DB method.
2. Handler using `handleServiceError`.
3. Route in `routes.go` with the right auth middleware.
4. `auditRoutes` entry if it changes state.
5. Test through `newTestServer` in `internal/http`.
6. Web hook in the matching file under `web/src/shared/services/api/` and a type in `types/api.ts`.
7. Amend `AGENTS.md` or `docs/` if behavior a reader relies on changed.
