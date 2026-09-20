# Development fixtures

An isolated backend and sample data for building and testing the web UI without real infrastructure. It has
its own database, apps directory and ports. Nothing here touches `.env`, `./data/selfhostly.db` or a Cloudflare
account.

**It does use your real Docker daemon.** The Insights page lists your real containers, and starting, updating or
creating an app here runs real `docker compose`. Read `.agents/rules/fixtures-and-docker-safety.md` before
clicking anything that starts, stops or deletes.

## Ports

Backend on **8090** and web on **5183**, chosen so they never collide with your own stack on 8080 or 5173. If a
port is busy, look at what owns it and use another. Never kill a process by port
(see `.agents/rules/local-processes-and-ports.md`).

## Run the backend

From the repo root:

```bash
mkdir -p tmp/dev-fixtures/data tmp/dev-fixtures/apps
make backend NOAIR=1 ENV_FILE=docs/dev-fixtures/env.fixtures
```

`tmp/` is git-ignored. When the log says "Scheduler started", seed it once:

```bash
sqlite3 tmp/dev-fixtures/data/fixtures.db < docs/dev-fixtures/seed.sql
curl -s localhost:8090/api/apps | head -c 300
```

The seed adds two secondary nodes (both unreachable), eight apps (running, stopped with a schedule, failed with a
port conflict, updating, quick tunnel, custom tunnel, LAN only, and apps on unreachable nodes), tunnels, compose
versions and a failed job with logs. No containers are started.

## Run the frontend

```bash
node docs/dev-fixtures/dev-server.mjs      # http://localhost:5183, proxies /api to :8090
```

It starts Vite with a polling file watcher, because some sandboxes miss file-system events and Vite then serves
stale modules. On a normal machine `cd web && VITE_API_BASE=http://localhost:8090 npm run dev -- --port 5183`
works too.

## Test modes

- **Empty:** stop the backend, delete `tmp/dev-fixtures/data/fixtures.db*`, start it again, and do not seed.
- **Loading:** `DELAY_MS=2500 node docs/dev-fixtures/delay-proxy.mjs` listens on 8091 and forwards to 8090 slowly.
  Run the frontend with `VITE_API_BASE=http://localhost:8091`.
- **Error:** stop the backend and reload. The UI must show a designed error state, not raw JSON.

## Gotchas

- Text columns on `apps` must not be NULL or `GET /api/apps` returns a 500. The seed already handles this.
- `npm ci` needs the npm registry. Without it the frontend cannot be built.
- Leave the data as you found it: remove any node, app or setting a test created.
