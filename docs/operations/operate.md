# Operate

Running an install day to day: ship a new version, restart, roll back, back up, rotate secrets. Commands are in the
[command reference](../reference/selfhostlyctl.md). Problems: [troubleshooting.md](troubleshooting.md).

## Ship a new version

1. Push to `main`, then wait for **Build and Publish** to finish in the Actions tab (it publishes the new `:latest` images).
2. On the server: `selfhostlyctl upgrade --with-frontend`
3. Check: `selfhostlyctl doctor`. Log in and open an app.

`upgrade` saves a rollback point, pulls the new images, does a dry start with your real settings (a start that would fail is
refused before anything stops), copies the database, recreates the primary, waits until healthy, recreates the gateway,
and checks your apps are still up. **If the primary is not healthy it rolls back by itself.** It ends by listing recommended settings
(`SECURITY_MODE=enforce`, `ENCRYPT_SECRETS_AT_REST=true`) that your `.env` does not have. It only reports them.

| The release changed | Run |
|---|---|
| Backend or API only | `selfhostlyctl upgrade` |
| The UI (or you are not sure) | `selfhostlyctl upgrade --with-frontend` (safe every time) |
| A setting you want to change now | `selfhostlyctl upgrade --set KEY=VALUE` |
| The compose file (new service or mount) | [Update the compose file](#update-the-compose-file) |
| Nothing you need, just a restart | `selfhostlyctl upgrade --no-pull --restart` |
| You want to look first | add `--dry-run` to any of the above |

Edited `.env`? Use `upgrade`, not `docker compose restart`: restart keeps the old environment.

| If | Then |
|---|---|
| You pinned image digests (`pin-images`) | `latest` is not followed. Run `selfhostlyctl pin-images`, then `upgrade`. |
| `--set` says your compose file never reads the setting | Your file does not pass that variable to the server, so writing it would change nothing. Switch to the current compose file (below), or add `KEY: ${KEY:-}` to the service's environment. |
| The release adds settings | `upgrade` never edits `.env` for you. New settings have safe defaults. `selfhostlyctl bootstrap --print-plan` shows what you could add. |
| You run `upgrade` twice | Safe. When the images and settings are unchanged it restarts nothing and leaves no rollback point (`--restart` forces one). Otherwise `--rollback` returns to the state before the *latest* run. Older points: `--rollback-to <stamp>` (folders in `.upgrade/`, last five kept). |
| You want to keep a known-good point | `selfhostlyctl upgrade --keep` protects that run's point from pruning. Delete its folder in `.upgrade/` to release it. |
| Several Selfhostly containers run here | It asks which. In scripts pass `--container NAME`. |
| A new `selfhostlyctl` behaviour is needed | Run `selfhostlyctl self-update` (add `sudo` if it lives in `/usr/local/bin`). `--check` only reports. `upgrade` does not update the tool, but ends with a note when a newer one exists (silent offline or with `SELFHOSTLYCTL_NO_UPDATE_CHECK=1`). `selfhostlyctl version` shows what you have. |

## Update from the UI

Settings, Updates does what `selfhostlyctl upgrade` does, from the browser. It is off by default. Design and threat
model: [ui-updates.md](../design/ui-updates.md).

**Turn it on** in `.env`, then `selfhostlyctl upgrade --set UI_UPDATES_ENABLED=true`:

| Setting | Meaning |
|---|---|
| `UI_UPDATES_ENABLED=true` | Turns the feature on. Needs `AUTH_ENABLED=true` and the Docker socket (not the socket proxy). |
| `UPDATE_PUBLIC_KEY` | The base64 ed25519 key releases must be signed with. Empty uses the key built into the release. |
| `UPDATE_MANIFEST_URL`, `UPDATE_IMAGE_REPO_PREFIX`, `UPDATE_CHECK_INTERVAL_HOURS` | Where releases are published, which repository their images must live in, and how often to look. The defaults are right for the official releases. |

The compose file has to pass these on (the current `docker-compose.prod.yml` does). Until it does, `--set` refuses with
"your compose file never reads ...": update the compose file first (below).

**When updates appear.** Every push to main that changes the gateway, backend or frontend is published as a release,
numbered `1.0.<commits on main>`. Nobody tags anything. Your install checks every `UPDATE_CHECK_INTERVAL_HOURS` (6 by
default), or straight away with **Check for updates**. Installs from before this feature have no updater: move them once with
`selfhostlyctl upgrade`, and from then on they update from the UI. How releases are built and signed:
[ui-updates.md](../design/ui-updates.md#releases).

**Use it.** An "Update available" badge appears in the header. In Settings, Updates:

1. **Review** pulls the release's backend image by digest and checks it against your install: a running deployment, the
   compose file, and any new settings.
2. **Fill in** what the release needs. Values only you know are asked for. Secrets it can generate are listed and created
   for you. A compose file that is just an older stock file shows its diff and needs your approval.
3. **Update now** asks you to type the version. A short-lived updater container then runs the same steps as `upgrade`.
   The UI is unavailable for 10 to 30 seconds while the primary is replaced, then reconnects by itself.

If the new version does not become healthy, or `doctor` reports a failure after it starts, it rolls back by itself and the
screen says why. If the updater is killed halfway (a reboot, an out-of-memory kill), the screen shows the run as
interrupted after the next start and offers **Roll back** or **Review and retry**.

| The screen says | What to do |
|---|---|
| Blocked: a deployment is running | Wait for it to finish. The blocker clears by itself. |
| Blocked: your compose file has edits the release would drop | Merge by hand: `selfhostlyctl compose-diff` shows what would be lost. Then review again. |
| Blocked: needs values | Type them in the form. They are written to `.env`. |
| Blocked: socket proxy | Not supported yet. Update with `selfhostlyctl upgrade`. |
| Rolled back | The reason is on the screen. The old version runs. Fix it, or wait for the next release. |
| Interrupted | Roll back, or review and retry. A stale lock is cleared by the next run. |

The rollback button restores the state saved before the last update: images, `.env`, the compose file. It keeps the
database, because an older backend runs on a migrated database. Your apps are never touched.

Only a signed-in user can start an update. Node credentials cannot, and every action is in the activity log. Releases that
are not signed with the trusted key are refused, and so is any image outside the trusted repository.

## Coming from an install that predates `selfhostlyctl`

Your install keeps running and your apps are never touched. In the folder with your compose file, `.env` and `data/`:

1. Install the tool ([install.md](install.md#1-get-the-tool)) and keep copies: `cp <your compose file> docker-compose.before-upgrade.yml`
   and `cp .env .env.before-upgrade`.
2. Back up the database by hand once (the old build has no backup command): stop the primary, `cp -p data/selfhostly.db* <safe place>`,
   start it again.
3. `selfhostlyctl upgrade --dry-run`, then `selfhostlyctl upgrade`. It finds the compose file the install was started from, and
   an existing database starts in `warn` mode, so nothing you deployed is blocked.
4. Check `selfhostlyctl doctor`, then carry on with this page when you are ready: [update the compose
   file](#update-the-compose-file), then `selfhostlyctl doctor --audit-apps` and `selfhostlyctl upgrade --set SECURITY_MODE=enforce`
   (this only takes effect once your compose file passes the setting on), and optionally [encryption](#turn-on-encryption-at-rest-optional).

`scripts/test-upgrade-original.sh` tests exactly this path against a hand-edited original compose file.

## Update the compose file

Only when a release changes the file itself. Your current file is never overwritten. The new file ships inside
`selfhostlyctl`, so run `selfhostlyctl self-update` first, or `compose write` gives you the old one.

```bash
selfhostlyctl compose write                                                    # writes docker-compose.prod.yml beside yours
selfhostlyctl compose-diff docker-compose.yml --env-file .env --apply .env    # carry your edits into .env
selfhostlyctl compose-diff docker-compose.yml --env-file .env                  # must say: Nothing would be lost
selfhostlyctl upgrade --compose docker-compose.prod.yml --no-pull
```

Replace `docker-compose.yml` with the name of your current file. After this, `upgrade` finds the new file by itself. Do not
rename it: the running containers remember its path.

## What a restart touches

- **Apps: nothing.** Each app is its own compose project. `AUTO_START_APPS` is not used at boot.
- **UI and API:** down for 10 to 30 seconds. Public apps on their own tunnels stay up.
- **Logins:** survive, unless you change `JWT_SECRET`.
- **A running deployment job:** the one real hazard. It is drained for up to 30 seconds, then stopped. `upgrade` checks
  first (needs `sqlite3` on the host; otherwise check the UI shows nothing "updating").
- **Schedules:** a cron that fires while the backend is down is skipped. Avoid restarting near a scheduled time.
- **`docker compose down` on the platform stack** also removes `selfhostly-cloudflared` and drops the tunnel. Recreate only what you mean to.

## What a restart depends on

Each start logs one `effective configuration` line: every setting and where its value came from. `selfhostlyctl doctor`
prints the same table.

| State | Lives in | If missing or wrong |
|---|---|---|
| Settings | `.env` and the compose file | Shown, with sources, in the log and in `doctor`. |
| Security mode | `SECURITY_MODE`, else `data/security-mode` | First start decides (existing database: `warn`, fresh: `enforce`) and saves it, so it cannot drift. `.env` wins, if the compose file passes it on. |
| Encryption on or off | `ENCRYPT_SECRETS_AT_REST` | Always explicit. Changing the security mode never changes it. |
| Encryption key | `SETTINGS_ENCRYPTION_KEY` or `data/secrets.key` | Missing with encrypted data: the server refuses to start, changes nothing, and says why. |
| Node identity | The database's primary node (`NODE_ID` only if you pin one) | Unset: adopted from the database. Pinned but different: `doctor` fails before you restart. |
| Node keys | `data/node-api-key`, `data/registration-token` | The log says whether each came from the environment, a saved file, or was just generated. |
| Database | `data/selfhostly.db` | Backed up before every schema change: `data/selfhostly.db.bak-pre-migration-*` (last five kept). |
| Rollback points | `.upgrade/<time>/` | Made by `selfhostlyctl upgrade` (last five kept, plus any made with `--keep`). |

## Read `doctor`

`PASS` is fine. `NOTE` is expected for how this install is set up, or optional: nothing to do. `WARN` needs your attention
and is followed by the step that fixes it. `FAIL` will not work. The end of the report lists every WARN and FAIL under
"To do". What counts depends on the install: no login is a note on a secondary or in development, a warning on a server
others can reach. `doctor --audit-apps` checks each deployed app's compose file against the volume policy; only apps
registered in the database are judged.

## Roll back

| Problem | Do this |
|---|---|
| Last upgrade or setting | `selfhostlyctl upgrade --rollback` (restores images and `.env`) |
| The new compose file | `docker compose -f <your previous compose file> up -d --no-deps --force-recreate primary gateway` |
| Enforce blocked something | `selfhostlyctl upgrade --set SECURITY_MODE=warn` |
| Encryption, before an older image | `--set ENCRYPT_SECRETS_AT_REST=false`, restart once, then roll back the image |
| Damaged database | Stop the primary, copy a backup over `data/selfhostly.db`, delete `selfhostly.db-wal` and `-shm`, start |

An older backend runs normally on the migrated database: migrations only add tables.

## Back up

```bash
selfhostlyctl backup /path/to/private/backups        # online snapshot + keys + .env, mode 0600
```

Nightly: `0 3 * * *  cd /opt/selfhostly && selfhostlyctl backup /var/backups/selfhostly`. Keep it off the server: it
contains secrets, including the encryption key, so it is only as safe as where you store it. Apps' own files and data
live under `apps/` and are not included. Restore: [install.md](install.md#4-move-to-a-new-server).

## Turn on encryption at rest (optional)

It protects the stored secrets if someone gets a copy of the database file alone. It does not protect a backup archive,
which holds the key too.

1. `selfhostlyctl backup ~/selfhostly-backups`
2. `selfhostlyctl bootstrap --print-plan`; if it lists `SETTINGS_ENCRYPTION_KEY`, run `selfhostlyctl bootstrap`.
3. **Copy the key off the machine** (`SETTINGS_ENCRYPTION_KEY` in `.env`, or `data/secrets.key`). If it is lost, the
   encrypted secrets cannot be recovered.
4. `selfhostlyctl upgrade --set ENCRYPT_SECRETS_AT_REST=true` (if it says your compose file never reads it, [update the
   compose file](#update-the-compose-file) first)
5. `selfhostlyctl doctor` shows `N encrypted, 0 plaintext` and `all encrypted secrets are readable`.

Undo: `selfhostlyctl upgrade --set ENCRYPT_SECRETS_AT_REST=false` (before going back to an older image).

## Rotate secrets (separate from any upgrade)

- `JWT_SECRET`: same new value on gateway and primary, recreate both together. Everyone logs in again.
- `GATEWAY_API_KEY`: same value on gateway, primary and every directly connected secondary.
- End all sessions without changing the secret: `POST /api/security/revoke-sessions`.
