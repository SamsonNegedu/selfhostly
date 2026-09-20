# Test Fixtures and Docker Safety

## Overview

`docs/dev-fixtures/` runs an isolated backend with sample data, which is the normal way to look
at the UI. Isolated means its database and ports. It does not mean isolated from the machine: the
backend talks to the developer's real Docker daemon. Insights lists their real containers, and any
start, update or create runs real `docker compose`.

## Core Principles

**Assume every container action is real.** Do not click or call Start, Update, Stop, Retry start,
Apply and retry, Create app, a container Restart, Stop or Delete, or a compose Save and redeploy
against the fixtures without asking. They can pull images, use disk and bandwidth, and change or
remove containers that belong to other projects. Opening a menu or a confirmation dialog and
cancelling is fine.

**Never act on containers you did not create.** Containers that Insights shows as "Not managed by
Selfhostly" belong to the developer. Do not stop, restart or delete them, and do not remove them
with `docker` commands.

**Use the fixture ports and database.** Backend on 8090, web on 5183, data in `tmp/dev-fixtures/`. Set
by `docs/dev-fixtures/env.fixtures` and `dev-server.mjs`. See `local-processes-and-ports.md`.

**Leave the fixture data as you found it.** If a test creates a node, app, schedule or setting,
remove or restore it afterwards (`sqlite3 tmp/dev-fixtures/data/fixtures.db`), and record the change.
Reseed with `docs/dev-fixtures/seed.sql` only when asked.

**Never put real credentials in a fixture.** No real Cloudflare token, no GitHub OAuth secret. A
fake token saved during a test must be cleared before finishing.

**Say what could not be verified without Docker.** Live container metrics, streaming deploy logs,
real tunnels and multi-node behavior need real infrastructure. Report them as unverified.

## Multi-node reminders

- Fixture secondary nodes (`node-nas`, `node-vps`) are unreachable on purpose. Selecting them in
  the scope switcher fans out to nodes that will not answer, which is a useful test and a slow one.
- A node registered with an unreachable address becomes `unreachable` after its first check. Remove
  test nodes when finished.
