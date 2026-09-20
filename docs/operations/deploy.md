# Deploy

New server, extra machines, backups. To upgrade a running install: [upgrade.md](upgrade.md).

## Get the tool

1. Install `selfhostlyctl` (one file; linux and macOS, amd64 and arm64; CI publishes it on every push to `main`, and a `v*` tag makes a numbered release):
   ```bash
   curl -fsSL https://raw.githubusercontent.com/samsonnegedu/selfhostly/main/scripts/install.sh | sh
   ```
   Or build it from a clone: `make ctl` gives `bin/selfhostlyctl`.
2. Run it inside the cloned folder (where `docker-compose.prod.yml` is), or point at it with `--dir /opt/selfhostly`.

## The commands

```bash
selfhostlyctl setup          # a new primary server, step by step
selfhostlyctl check --fix    # is this machine ready? repair what is not
selfhostlyctl join-token     # on the primary: one-time token for a new machine
selfhostlyctl join ...       # on the new machine: connect it as a secondary
selfhostlyctl upgrade        # safe upgrade or restart (see safe-restart.md)
selfhostlyctl backup | doctor | status
selfhostlyctl compose write  # write the compose file that matches this version (no git clone needed)
```

Commands that talk to the running server (`doctor`, `join-token`, `backup`, `status`) find its container by probing what is
running, never by name. With several candidates they ask which one; in scripts pass `--container NAME`.

Each step re-checks the real state of the machine, so if you stop halfway, run the same command again.
`check` covers Docker and Compose, permission to use Docker and the socket group (`DOCKER_GID`), folder ownership,
64-bit CPU, disk, RAM, clock sync and, on a Raspberry Pi, container memory accounting. Each problem prints the
command that fixes it; `--fix` runs them after asking. In scripts add `--non-interactive --yes` and the option for each answer (`selfhostlyctl setup --help`).

## New server

Needs a 64-bit Linux machine (a Raspberry Pi 4 or 5 with the 64-bit OS works), a Cloudflare tunnel token, and for
GitHub login an OAuth app with callback `https://<your host>/auth/github/callback`.

1. Clone and run setup:
   ```bash
   git clone <repo> selfhostly && cd selfhostly
   selfhostlyctl setup
   ```
   It checks the machine, asks how people sign in, generates every secret into `.env`, starts the stack and runs `doctor`.
2. By hand instead:
   ```bash
   selfhostlyctl check --fix
   selfhostlyctl bootstrap --auth github --domain selfhostly.example.com --github-user your-login
   $EDITOR .env        # GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET, TUNNEL_TOKEN
   docker compose -f docker-compose.prod.yml up -d
   docker exec selfhostly-primary ./selfhostly doctor
   ```

Cloudflare Access instead of GitHub login: `--auth cloudflare --cf-team myteam.cloudflareaccess.com --cf-aud <AUD tag>`.
A fresh install starts in `enforce` and refuses to start with no login on a public address (override: `ALLOW_UNAUTHENTICATED=true`).

Optional: pin exact image versions (`selfhostlyctl pin-images --version 1.4.0`; a git tag `v1.4.0` publishes
image tags `1.4.0` and `1.4`), and a Docker socket filter (`docker-compose.socket-proxy.yml`, test it first).

## Add a secondary node

The new machine connects **out** to your public address. No open port, firewall rule or VPN, and it works behind a home router.

1. On the primary:
   ```bash
   selfhostlyctl join-token
   ```
   It prints the exact command for the new machine (token valid one hour, single use).
2. On the new machine:
   ```bash
   git clone <repo> selfhostly && cd selfhostly
   selfhostlyctl join --primary-url https://selfhostly.example.com --token sfj_...
   ```
   It checks the machine, confirms it can reach the primary, starts the node from `docker-compose.secondary.yml`
   and waits until the primary accepts it.

The node's API listens on loopback and publishes no port. It reconnects by itself after a network drop or a restart of
either side, and shows offline within seconds when the connection drops. Use an `https` address: the node key is sent
in the connection request.

**Direct mode** (older: the primary calls the machine, which needs a reachable port on a private network):
`selfhostlyctl join-token --direct`, then `join --direct --primary-url http://<primary>:8082 --token ...`.
An existing direct node can switch to the outbound connection by setting `NODE_TRANSPORT=tunnel` on it.

## Back up

```bash
selfhostlyctl backup /path/to/private/backups        # online snapshot + keys + .env, mode 0600
```
Nightly: `0 3 * * *  cd /opt/selfhostly && selfhostlyctl backup /var/backups/selfhostly`. Keep it off the server:
it contains secrets. Apps' own files and data live under `apps/` and are not included.

## Move to a new server

1. Old server: `selfhostlyctl backup`, copy the archive over.
2. New server: clone the repo, then
   ```bash
   mkdir -p /tmp/restore && tar -xzf selfhostly-backup-<stamp>.tar.gz -C /tmp/restore
   mkdir -p data && cp /tmp/restore/data/* data/ && cp /tmp/restore/env .env && chmod 600 .env data/*
   ```
   Keep `secrets.key` with the database. If paths differ, edit `DATA_DIR` and `APPS_DIR` in `.env` (delete `HOST_APPS_DIR` to detect it).
3. Copy your apps' folders into `apps/`.
4. `docker compose -f docker-compose.prod.yml up -d`, then `docker exec selfhostly-primary ./selfhostly doctor --audit-apps`.
5. Point the Cloudflare tunnel at the new server. Never run both servers against the same tunnel and database.

## Test the deployment paths

| Script | Proves |
|---|---|
| `scripts/smoke-test.sh` | fresh install: login required, forged headers ignored, read-only root, restart keeps data |
| `scripts/test-upgrade.sh` | upgrade, automatic rollback, `--set` reverts, encryption cycle |
| `scripts/test-upgrade-original.sh` | upgrade from a hand-edited original compose file |
| `scripts/test-join.sh` | a machine on an isolated network joins and is controlled through the gateway |
| `scripts/test-host-linux.sh`, `scripts/test-compose-diff.sh` | host checks (Pi simulated, real Linux sockets) and compose-diff |
| `go test ./internal/ctl/...` | the CLI itself: upgrade order and rollback, bootstrap, backup, join-token |

They run under their own names on private networks and need Docker. Local development: `make dev`; the Go entry point is
the package `./cmd/server` (`go run ./cmd/server`).
