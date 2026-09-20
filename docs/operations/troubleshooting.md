# Troubleshooting

Start with the logs: `make logs`, or `docker compose -f docker-compose.prod.yml logs -f`. Run
`selfhostlyctl doctor` for a read-only check of the install. For restarts and rollbacks see
[safe-restart.md](safe-restart.md).

## The app will not start

- Docker must be running: `docker ps`.
- `.env` must exist with correct values (`cp env.example .env`).
- Port 8080 must be free, or change `SERVER_ADDRESS`.
- Rebuild: `make prod`.

## Cannot reach it from the network

- `SERVER_ADDRESS` must listen on more than loopback, for example `0.0.0.0:8080`.
- The firewall must allow the port. For remote access a Cloudflare tunnel is safer than opening a port.

## No containers in Insights

- Apps must be deployed and have running containers: `docker ps`.
- The backend needs access to the Docker socket, `/var/run/docker.sock` (or the socket proxy).

## Numbers do not update

- The browser console shows API errors, and you must be signed in.
- Updates pause while the tab is hidden.

## Database errors

- The directory of `DATABASE_PATH` must exist and be writable.
- Do not delete the database to fix an error: it holds your apps. Restore a backup instead
  (`data/selfhostly.db.bak-*`, see [safe-restart.md](safe-restart.md)).

## Cloudflare tunnel not working

- `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` must be set. Create the token at
  https://dash.cloudflare.com/profile/api-tokens with `Cloudflare Tunnel:Edit` and `Zone:DNS:Edit`. The
  account ID is on any zone's overview page.
- Read the tunnel logs on the app's Access tab.
- Each route must point at a `service:port` that the tunnel can reach.
- A script or style file returning 404 is a routing-rule problem, see
  [cloudflare-zero-trust.md](cloudflare-zero-trust.md#a-script-or-style-file-returns-404-blank-page).
