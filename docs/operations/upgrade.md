# Upgrade an existing install

For an install that is already running, including the original `docker-compose.prod.yml` with your
hand edits. Tested by `scripts/test-upgrade-original.sh`.

- Your apps are never touched. Only the backend and gateway restart (10 to 30 seconds without the UI).
- Logins survive as long as `JWT_SECRET` stays the same. Do not change it.
- An existing database starts in `warn` mode: it logs what it would block and blocks nothing.
- Existing secondary machines keep working unchanged.

## Prepare (5 minutes)

No git clone is needed. All you need in the folder is your live compose file (whatever it is called), `.env` and `data/`.

1. Install the tool on the server (one file, published by CI on every push to `main`):
   ```bash
   curl -fsSL https://raw.githubusercontent.com/samsonnegedu/selfhostly/main/scripts/install.sh | sh
   ```
2. Pick a quiet time: no deployment running, no schedule due in the next 10 minutes.
3. Get the new images: push to `main` and let CI publish them, or build on the host and add `--no-pull` below.
4. Keep copies of your live files (the examples call the compose file `docker-compose.yml`):
   ```bash
   cp docker-compose.yml docker-compose.before-upgrade.yml
   cp .env .env.before-upgrade && chmod 600 .env.before-upgrade
   ```
5. Back up the database by hand once (the running build has no backup command):
   ```bash
   docker compose -f docker-compose.yml stop primary
   cp -p data/selfhostly.db* ~/selfhostly-backup/
   docker compose -f docker-compose.yml start primary
   ```

## Phase 1: new images, your old configuration

6. Look first (changes nothing), then do it. The tool reads which compose file your install was started from out of
   Docker's own labels and says so, so there is no file or project name to give:
   ```bash
   selfhostlyctl upgrade --dry-run
   selfhostlyctl upgrade
   ```
7. Check:
   ```bash
   selfhostlyctl doctor
   selfhostlyctl status
   ```
   Expect: security mode `warn`, your volume whitelist counted, "node identity matches the database", every
   app container still up. Log in and open an app.

If the primary is not healthy it rolls back by itself. To go back yourself: `selfhostlyctl upgrade --rollback`.

Use the system for a few days, then see what enforce mode would block:
```bash
selfhostlyctl doctor --audit-apps
```

## Phase 2: the new compose file (one step at a time, check the UI after each)

8. Write the compose file that matches this version of the tool (it never overwrites an existing file):
   ```bash
   selfhostlyctl compose write
   ```
9. Carry your hand edits into `.env`, and confirm nothing is lost:
   ```bash
   selfhostlyctl compose-diff docker-compose.yml --env-file .env --apply .env
   selfhostlyctl compose-diff docker-compose.yml --env-file .env     # must say: Nothing would be lost
   ```
   Anything listed as "would be lost" comes with the exact change to make in the new file.
10. Switch to the new file:
    ```bash
    selfhostlyctl upgrade --compose docker-compose.prod.yml --no-pull
    ```
    It publishes the primary on `127.0.0.1:8082` only. If 8082 is taken, set `PRIMARY_NODE_PORT` in `.env` first.
    To go back: `docker compose -f docker-compose.yml up -d --no-deps --force-recreate primary gateway`
    From now on `selfhostlyctl upgrade` finds the new file by itself.
11. Add the new settings (safe to repeat; never overwrites a value):
    ```bash
    selfhostlyctl bootstrap --print-plan
    selfhostlyctl bootstrap --auth github --domain <your hostname> --github-user <your login>
    selfhostlyctl upgrade --no-pull
    ```
12. Turn on enforcement once the watch period showed nothing you need (this does not turn on encryption):
    ```bash
    selfhostlyctl upgrade --set SECURITY_MODE=enforce
    ```
13. Optional, encrypt stored secrets. First copy `SETTINGS_ENCRYPTION_KEY` (in `.env`) off the machine:
    ```bash
    selfhostlyctl upgrade --set ENCRYPT_SECRETS_AT_REST=true
    ```
    Before going back to an older image, set it to `false` and restart once.

A setting that stops the server starting is refused before anything stops, or reverted automatically.

## If something goes wrong

| Problem | Do this |
|---|---|
| The upgrade script failed | Nothing to do: it rolled back and printed why. |
| A setting broke startup | `selfhostlyctl upgrade --rollback` |
| The new compose file misbehaves | `docker compose -f docker-compose.yml up -d --no-deps --force-recreate primary gateway` |
| Enforce blocked an app | `selfhostlyctl upgrade --set SECURITY_MODE=warn`, then fix or allow the path in `ALLOWED_VOLUME_PATHS` |
| Database damaged | Stop the primary, copy your backup over `data/selfhostly.db`, delete `selfhostly.db-wal` and `-shm`, start it. Automatic copies: `data/selfhostly.db.bak-pre-migration-*` |

More: [safe-restart.md](safe-restart.md).

## Add a machine

On the primary run `selfhostlyctl join-token`, then run the one command it prints on the new machine.
The new machine connects out to your public address: no open port, no VPN. Existing machines need nothing.
Details: [deploy.md](deploy.md#add-a-secondary-node).

## Not tested

- Your real host (the test used `user: 0:0`; your `1000:984` is checked by `doctor`, which names the right `DOCKER_GID` if wrong).
- A real Cloudflare Tunnel for the new node connection, a Raspberry Pi, and GitHub login against a live OAuth app.

The closest rehearsal is Phase 1 on a copy of your data on another machine.
