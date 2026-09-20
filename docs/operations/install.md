# Install

A new server, extra machines, and moving to another server. Every command is in the
[command reference](../reference/selfhostlyctl.md). Keeping a running install up to date: [operate.md](operate.md).

## 1. Get the tool

1. Install `selfhostlyctl` (one file; linux and macOS, amd64 and arm64). CI publishes it on every push to `main`, and a
   `v*` tag makes a numbered release:
   ```bash
   curl -fsSL https://raw.githubusercontent.com/samsonnegedu/selfhostly/main/scripts/install.sh | sh
   ```
   Or build it from a clone: `make ctl` gives `bin/selfhostlyctl`.
2. Run it in the folder that holds your compose file and `.env`, or point at it with `--dir /opt/selfhostly`.

Every command re-checks the real state of the machine, so if you stop halfway, run the same command again. Commands that
talk to the running server find its container by probing what is running, never by name; with several candidates they
ask, and in scripts you pass `--container NAME`.

## 2. New server

Needs a 64-bit Linux machine (a Raspberry Pi 4 or 5 with the 64-bit OS works), a Cloudflare tunnel token, and for GitHub
login an OAuth app with callback `https://<your host>/auth/github/callback`.

1. Get the compose file (no clone needed):
   ```bash
   mkdir selfhostly && cd selfhostly
   selfhostlyctl compose write
   ```
2. Set up:
   ```bash
   selfhostlyctl setup
   ```
   It checks the machine (Docker, socket group, folder ownership, 64-bit CPU, disk, RAM, clock, Pi memory accounting),
   asks how people sign in, generates every secret into `.env`, starts the stack and runs `doctor`. Each problem it finds
   prints the command that fixes it, and `selfhostlyctl check --fix` runs them after asking.
3. Open the address it prints and log in.

By hand instead: `selfhostlyctl check --fix`, `selfhostlyctl bootstrap --auth github --domain <host> --github-user <you>`,
edit `.env` (`GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `TUNNEL_TOKEN`), then
`docker compose -f docker-compose.prod.yml up -d` and `selfhostlyctl doctor`.

Cloudflare Access instead of GitHub login: `--auth cloudflare --cf-team myteam.cloudflareaccess.com --cf-aud <AUD tag>`.
A fresh install starts in `enforce` and refuses to start with no login on a public address (override:
`ALLOW_UNAUTHENTICATED=true`). In scripts add `--non-interactive --yes` and the option for each answer.

Optional: pin exact image versions (`selfhostlyctl pin-images --version 1.4.0`; a git tag `v1.4.0` publishes image tags
`1.4.0` and `1.4`), and a Docker socket filter (`docker-compose.socket-proxy.yml`, test it first).

## 3. Add a secondary machine

The new machine connects **out** to your public address. No open port, firewall rule or VPN, and it works behind a home router.

1. On the primary: `selfhostlyctl join-token`. It prints the exact command for the new machine (token valid one hour,
   single use).
2. On the new machine (install the tool first, no clone needed): run the command it printed, for example
   ```bash
   selfhostlyctl compose write secondary
   selfhostlyctl join --primary-url https://selfhostly.example.com --token sfj_...
   ```
   It checks the machine, confirms it can reach the primary, starts the node and waits until the primary accepts it.

The node's API listens on loopback and publishes no port. It reconnects by itself after a network drop or a restart of
either side, and shows offline within seconds when the connection drops. Use an `https` address: the node key is sent in
the connection request. Design: [node-link RFC](../rfcs/node-link.md).

**Direct mode** (older: the primary calls the machine, which needs a reachable port on a private network):
`selfhostlyctl join-token --direct`, then `selfhostlyctl compose write secondary-direct` and
`join --direct --primary-url http://<primary>:8082 --token ...`. An existing direct node can switch to the outbound
connection by setting `NODE_TRANSPORT=tunnel` on it.

## 4. Move to a new server

1. Old server: `selfhostlyctl backup`, copy the archive over.
2. New server: install the tool and run `selfhostlyctl compose write`, then
   ```bash
   mkdir -p /tmp/restore && tar -xzf selfhostly-backup-<stamp>.tar.gz -C /tmp/restore
   mkdir -p data && cp /tmp/restore/data/* data/ && cp /tmp/restore/env .env && chmod 600 .env data/*
   ```
   Keep `secrets.key` with the database. If paths differ, edit `DATA_DIR` and `APPS_DIR` in `.env` (delete
   `HOST_APPS_DIR` to detect it).
3. Copy your apps' folders into `apps/`.
4. `docker compose -f docker-compose.prod.yml up -d`, then `selfhostlyctl doctor --audit-apps`.
5. Point the Cloudflare tunnel at the new server. Never run both servers against the same tunnel and database.
