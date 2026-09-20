# Restart safely

Reference for restarting or changing a running install. Upgrading from an older version: [upgrade.md](upgrade.md).

## Restart or change a setting

```bash
selfhostlyctl upgrade --dry-run                     # look first, changes nothing
selfhostlyctl upgrade                               # pull new images, recreate primary then gateway
selfhostlyctl upgrade --set SECURITY_MODE=enforce   # change one setting and apply it
selfhostlyctl upgrade --rollback                    # go back to what it saved before the last run
```

It refuses to run during a deployment, saves the running images, does a dry start of the new image with your real
settings (a start that would fail is refused before anything stops), copies the database, recreates the primary,
waits for it to be healthy, recreates the gateway, and checks your apps are still up. **If the primary is not
healthy it rolls back by itself.** Edited `.env`? Use this command, not `docker compose restart`: restart keeps the
old environment.

## What a restart touches

- **Apps: nothing.** Each app is its own compose project. `AUTO_START_APPS` is not used at boot.
- **UI and API:** down for 10 to 30 seconds. Public apps on their own tunnels stay up.
- **Logins:** survive, unless you change `JWT_SECRET`.
- **A running deployment job:** the one real hazard. It is drained for up to 30 seconds, then stopped.
  The script checks first (needs `sqlite3` on the host; otherwise check the UI shows nothing "updating").
- **Schedules:** a cron that fires while the backend is down is skipped. Avoid restarting near a scheduled time.
- **`docker compose down` on the platform stack** also removes `selfhostly-cloudflared` and drops the tunnel. Recreate only what you mean to.

## What a restart depends on

Each start logs one `effective configuration` line: every setting and where its value came from.
`docker exec selfhostly-primary ./selfhostly doctor` prints the same table.

| State | Lives in | If missing or wrong |
|---|---|---|
| Settings | `.env` and the compose file | Shown, with sources, in the log and in `doctor`. |
| Security mode | `SECURITY_MODE`, else `data/security-mode` | First start decides (existing database: `warn`, fresh: `enforce`) and saves it, so it cannot drift. `.env` wins. |
| Encryption on or off | `ENCRYPT_SECRETS_AT_REST` | Always explicit. Changing the security mode never changes it. |
| Encryption key | `SETTINGS_ENCRYPTION_KEY` or `data/secrets.key` | Missing with encrypted data: the server refuses to start, changes nothing, and says why. |
| Node identity | The database's primary node (`NODE_ID` only if you pin one) | Unset: adopted from the database. Pinned but different: `doctor` fails before you restart. |
| Node keys | `data/node-api-key`, `data/registration-token` | The log says whether each came from the environment, a saved file, or was just generated. |
| Database | `data/selfhostly.db` | Backed up before every schema change: `data/selfhostly.db.bak-pre-migration-*` (last five kept). |
| Rollback points | `.upgrade/<time>/` | Made by `selfhostlyctl upgrade` (last five kept). |

## Roll back

| Problem | Do this |
|---|---|
| Last upgrade or setting | `selfhostlyctl upgrade --rollback` (restores images and `.env`) |
| The new compose file | `docker compose -f <your previous compose file> up -d --no-deps --force-recreate primary gateway` |
| Enforce blocked something | `selfhostlyctl upgrade --set SECURITY_MODE=warn` |
| Encryption, before an older image | `--set ENCRYPT_SECRETS_AT_REST=false`, restart once, then roll back the image |
| Damaged database | Stop the primary, copy a backup over `data/selfhostly.db`, delete `selfhostly.db-wal` and `-shm`, start |

An older backend runs normally on the migrated database: migrations only add tables.

## Rotate secrets (separate from any upgrade)

- `JWT_SECRET`: same new value on gateway and primary, recreate both together. Everyone logs in again.
- `GATEWAY_API_KEY`: same value on gateway, primary and every directly connected secondary.
- End all sessions without changing the secret: `POST /api/security/revoke-sessions`.

## Reading `doctor`

`PASS` is fine. `NOTE` is expected for how this install is set up, or optional: nothing to do. `WARN` needs your attention and
is followed by the step that fixes it. `FAIL` will not work. The end of the report lists every WARN and FAIL under "To do".
What counts depends on the install: no login is a note on a secondary or in development, a warning on a server others can reach.

## If something is stuck

| Symptom | Fix |
|---|---|
| `refusing to start` in the log | The message names the setting. On a fresh install set auth, or `ALLOW_UNAUTHENTICATED=true`. |
| `stored secrets are encrypted but no key is available` | Restore `secrets.key` from backup or set `SETTINGS_ENCRYPTION_KEY`. Nothing was changed. |
| Login fails after pinning the gateway | `PUBLIC_HOSTS` must list the hostname users type. |
| An app cannot start after enforce | `doctor --audit-apps` names the path: move it into the app folder or add it to `ALLOWED_VOLUME_PATHS`. |
| Container buttons say "not part of an app" | That container is not from an app folder. Intended in enforce mode. |
| `doctor`: host path not detected | Set `HOST_APPS_DIR` to the host path of the apps directory. |
| `doctor`: not in the Docker socket group | Set `DOCKER_GID=<the number it shows>` in `.env` and recreate the primary. |
