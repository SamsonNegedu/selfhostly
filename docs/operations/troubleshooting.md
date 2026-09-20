# Troubleshooting

Start with `selfhostlyctl status` (what is running) and `selfhostlyctl doctor` (a read-only check of the install; how to
read it: [operate.md](operate.md#read-doctor)). For the server's own log, run
`docker compose -f docker-compose.prod.yml logs -f primary` in the install folder. For restarts and rollbacks see
[operate.md](operate.md).

## After an upgrade or a setting change

| Symptom | Fix |
|---|---|
| `refusing to start` in the log | The message names the setting. On a fresh install set auth, or `ALLOW_UNAUTHENTICATED=true`. |
| `stored secrets are encrypted but no key is available` | Restore `secrets.key` from backup or set `SETTINGS_ENCRYPTION_KEY`. Nothing was changed. |
| `--set` says the compose file never reads the setting | Switch to the current compose file, or add `KEY: ${KEY:-}` to the service's environment ([operate.md](operate.md#update-the-compose-file)). |
| Login fails after pinning the gateway | `PUBLIC_HOSTS` must list the hostname users type. |
| An app cannot start after enforce | `doctor --audit-apps` names the path: move it into the app folder or add it to `ALLOWED_VOLUME_PATHS`. |
| Container buttons say "not part of an app" | That container is not from an app folder. Intended in enforce mode. |
| `doctor`: host path not detected | Set `HOST_APPS_DIR` to the host path of the apps directory. |
| `doctor`: not in the Docker socket group | Set `DOCKER_GID=<the number it shows>` in `.env` and recreate the primary. |
| `doctor` lists folders as "not deployed apps" | They are in the apps folder but not registered in the database (copies, backups, hand-run stacks), so they are not checked. |
| An app rebuild fails after moving its data into its folder | The data is now in the Docker build context. Add its folder to a `.dockerignore` next to the app's compose file. |

## The app will not start

- Docker must be running: `docker ps`.
- `.env` must exist in the install folder, or pass `--dir` or `--env-file`. `selfhostlyctl setup` writes it; `env.example`
  lists every setting.
- Port 8080 must be free, or change `SERVER_ADDRESS`.
- Read why it stopped: `selfhostlyctl doctor`, then the primary's log (see the top of this page). To recreate it with your
  current settings, run `selfhostlyctl upgrade --no-pull`.

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
  (`data/selfhostly.db.bak-*`, see [operate.md](operate.md#roll-back)).

## Cloudflare tunnel not working

- `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` must be set. Create the token at
  https://dash.cloudflare.com/profile/api-tokens with `Cloudflare Tunnel:Edit` and `Zone:DNS:Edit`. The
  account ID is on any zone's overview page.
- Read the tunnel logs on the app's Access tab.
- Each route must point at a `service:port` that the tunnel can reach.
- A script or style file returning 404 is a routing-rule problem, see
  [cloudflare-zero-trust.md](cloudflare-zero-trust.md#a-script-or-style-file-returns-404-blank-page).
