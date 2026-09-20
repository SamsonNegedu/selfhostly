# Live updates, audit log and error codes

How the API tells the web app what changed, who did what, and why a request failed.

## Change stream

`GET /api/events` is a server-sent stream. One `change` event is written whenever an app, job or node
changes, and it carries only the `kind` and `id`, never values. The browser refreshes the matching queries
(`web/src/shared/hooks/useEventStream.ts`), so the normal permission checks still apply to the data itself.

- Events come from a bus in `internal/events`, fed by the database's change notifier, so every code path
  that writes gets one without each handler having to remember.
- A keepalive comment every 25 seconds stops proxies from closing a quiet connection.
- Node credentials are refused, so a secondary node cannot listen to the primary.
- Polling stays as the fallback. When the stream reconnects the browser refreshes everything, because
  changes may have been missed while it was down.
- The stream reflects this node's own database. Changes on a secondary node reach the primary's screen by
  polling.

## Audit log

Each state-changing request is recorded with the actor, method, path, result, and an action such as
`app.stop` with a target type, id and name.

- Actions are declared in `internal/http/audit_actions.go`. A route that is not listed is still recorded,
  only without an action, so a new route is never silently unlogged.
- The target is worked out before the handler runs, so an app that the request deletes is still named.
  Handlers that create things call `setAuditTarget`, since the new id is not in the URL.
- Names are labels such as an app name, never values from the request body.
- Settings > Activity reads it as sentences. The list endpoint always returns an array, never `null`.

## Error codes

Errors that the client can act on carry a `code`, and a `field` when one input is at fault. Conflicts are
HTTP 409, missing things are 404. The codes are in `internal/domain/errors.go`, for example
`NODE_ID_TAKEN`, `NODE_NAME_TAKEN`, `PRIMARY_NODE_PROTECTED`, `NODE_HAS_APPS`, `CURRENT_NODE` and
`NODE_ONLINE`. The web client keeps them on `ApiRequestError`, and `describeError` and `fieldError` turn them
into text and inline input errors. Never show a raw response body to the user.

## Schema changes

Database changes are additive versioned migrations in `internal/db/versioned.go`. A backup is written before
any pending migration runs on a database that already holds data, and re-running one is safe, so an older
binary keeps working against a newer schema.
