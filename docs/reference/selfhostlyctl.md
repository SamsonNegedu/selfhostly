# selfhostlyctl reference

<!-- Generated from the command definitions by `make docs`. Do not edit by hand. -->

Set up, join, upgrade and check a Selfhostly install. Every command explains itself with `--help`.

| Command | What it does |
|---|---|
| [`selfhostlyctl backup`](#selfhostlyctl-backup) | Back up the database, keys and settings into one private archive |
| [`selfhostlyctl bootstrap`](#selfhostlyctl-bootstrap) | Fill in every setting a new install needs (never overwrites a value) |
| [`selfhostlyctl check`](#selfhostlyctl-check) | Is this machine ready? Says what is wrong and how to fix it |
| [`selfhostlyctl compose write`](#selfhostlyctl-compose-write) | Write a shipped compose file to disk (no git clone needed) |
| [`selfhostlyctl compose-diff`](#selfhostlyctl-compose-diff) | Show what replacing your compose file would lose, and carry it over through .env |
| [`selfhostlyctl doctor`](#selfhostlyctl-doctor) | Run the server's own health checks inside the running container |
| [`selfhostlyctl join`](#selfhostlyctl-join) | Connect this machine to a Selfhostly primary as a secondary node |
| [`selfhostlyctl join-token`](#selfhostlyctl-join-token) | On the primary: create a one-time token and print the command for the new machine |
| [`selfhostlyctl pin-images`](#selfhostlyctl-pin-images) | Pin the image tags to immutable digests in the settings file |
| [`selfhostlyctl self-update`](#selfhostlyctl-self-update) | Update selfhostlyctl itself to the newest release |
| [`selfhostlyctl setup`](#selfhostlyctl-setup) | Set up a new primary server, step by step |
| [`selfhostlyctl status`](#selfhostlyctl-status) | Show the containers of this Selfhostly install |
| [`selfhostlyctl upgrade`](#selfhostlyctl-upgrade) | Upgrade or restart safely: rollback point, dry start, health checks, automatic rollback |
| [`selfhostlyctl version`](#selfhostlyctl-version) | Print the selfhostlyctl version |

## Options for every command

| Option | Meaning |
|---|---|
| `--container string` | the Selfhostly container to use (default: found by probing what is running) |
| `--dir string` | the Selfhostly folder (where docker-compose.prod.yml and .env are) (default .) |
| `--dry-run` | show what would change and change nothing |
| `--env-file string` | settings file, relative to --dir (default .env) |
| `--non-interactive` | never ask; use defaults and flags |
| `--skip-host-checks` | trust that this machine is ready (for automation) |
| `-y, --yes` | answer yes to every question |

## selfhostlyctl backup

Back up the database, keys and settings into one private archive

```
selfhostlyctl backup [destination-dir]
```

```
Takes an online database snapshot (no downtime), adds the secrets key, node identity and settings file,
and writes one archive with mode 0600. It contains secrets: keep it somewhere private, off this machine.
```

## selfhostlyctl bootstrap

Fill in every setting a new install needs (never overwrites a value)

```
selfhostlyctl bootstrap [flags]
```

| Option | Meaning |
|---|---|
| `--auth string` | github, cloudflare or none (default github) |
| `--cf-aud string` | Cloudflare Access application audience tag |
| `--cf-team string` | Cloudflare Access team domain |
| `--domain string` | public hostname (sets AUTH_BASE_URL, PUBLIC_HOSTS, secure cookies) |
| `--github-user stringArray` | allowed GitHub login (repeatable) |
| `--print-plan` | show what would be added without writing |
| `--quiet` | print a one-line summary |

## selfhostlyctl check

Is this machine ready? Says what is wrong and how to fix it

```
selfhostlyctl check [flags]
```

| Option | Meaning |
|---|---|
| `--fix` | offer to run the repairs (asks before each one) |
| `--port int` | with secondary-direct: the API port to check is free |
| `--role string` | what the machine will run: primary, secondary or secondary-direct (default primary) |
| `--skip-network` | do not probe the internet |

## selfhostlyctl compose write

Write a shipped compose file to disk (no git clone needed)

```
selfhostlyctl compose write [prod|secondary|secondary-direct|socket-proxy] [flags]
```

```
Writes the compose file that matches this version of selfhostlyctl. Nothing is overwritten unless you pass
--force, so your hand-edited file is safe. Default: prod, written as docker-compose.prod.yml.
```

| Option | Meaning |
|---|---|
| `--force` | replace the file if it exists |
| `--out string` | where to write it (default: the standard name in --dir) |

## selfhostlyctl compose-diff

Show what replacing your compose file would lose, and carry it over through .env

```
selfhostlyctl compose-diff OLD.yml [NEW.yml] [flags]
```

```
People edit their production compose file by hand (a hard-coded volume path, a user id, a log setting).
Replacing the file drops those edits silently. This renders both files with the same settings and
reports every difference in what would run. Values the new file reads from the env file can be
written there with --apply, which only adds missing lines and never overwrites one.
Secrets are masked. Exit status: 0 nothing lost, 1 needs attention.
```

| Option | Meaning |
|---|---|
| `--apply string` | add the carry-over lines that are missing to this env file |
| `--show-secrets` | do not mask secret-looking values |

## selfhostlyctl doctor

Run the server's own health checks inside the running container

```
selfhostlyctl doctor [flags]
```

## selfhostlyctl join

Connect this machine to a Selfhostly primary as a secondary node

```
selfhostlyctl join [flags]
```

| Option | Meaning |
|---|---|
| `--direct` | older mode: the primary calls this machine |
| `--endpoint string` | direct mode: the address the primary uses to reach this machine |
| `--name string` | name for this machine (default: its hostname) |
| `--port int` | direct mode: this node's API port (default 8082) |
| `--primary-url string` | address of the primary (printed by join-token) |
| `--skip-reach-check` | do not check that the primary is reachable |
| `--token string` | one-time join token (starts with sfj_) |

## selfhostlyctl join-token

On the primary: create a one-time token and print the command for the new machine

```
selfhostlyctl join-token [flags]
```

| Option | Meaning |
|---|---|
| `--direct` | older mode: the primary calls the machine (needs a reachable port) |
| `--primary-url string` | address the new machine should use for the primary |

## selfhostlyctl pin-images

Pin the image tags to immutable digests in the settings file

```
selfhostlyctl pin-images [flags]
```

```
Resolves the image tags docker-compose.prod.yml uses to digests and writes them into the settings file
(GATEWAY_IMAGE, BACKEND_IMAGE, FRONTEND_IMAGE, CLOUDFLARED_IMAGE). A new server then pulls exactly the
bytes you tested. To go back to tags, delete the *_IMAGE lines. Use --dry-run to only print them.
```

| Option | Meaning |
|---|---|
| `--version string` | image version (default: SELFHOSTLY_VERSION from the environment or settings file, else latest) |

## selfhostlyctl self-update

Update selfhostlyctl itself to the newest release

```
selfhostlyctl self-update [flags]
```

```
Downloads the selfhostlyctl release for this machine, checks its checksum and replaces this file.
With no --version it takes the newest numbered release, or the newest main build if there is no numbered release yet.
It does not touch your install: use "selfhostlyctl upgrade" for that. If the file is in a folder you cannot write to
(for example /usr/local/bin), run it with sudo.
```

| Option | Meaning |
|---|---|
| `--check` | only say whether a newer build is available |
| `--version string` | latest, edge (the newest main), or a release such as v1.4.0 (default latest) |

## selfhostlyctl setup

Set up a new primary server, step by step

```
selfhostlyctl setup [flags]
```

| Option | Meaning |
|---|---|
| `--auth string` | github, cloudflare or none |
| `--cf-aud string` | Cloudflare Access application audience tag |
| `--cf-team string` | Cloudflare Access team domain |
| `--domain string` | public hostname |
| `--github-client-id string` | GitHub OAuth client ID |
| `--github-client-secret string` | GitHub OAuth client secret |
| `--github-user stringArray` | allowed GitHub login (repeatable) |
| `--no-start` | configure only; do not start anything |
| `--tunnel-token string` | Cloudflare Tunnel token |

## selfhostlyctl status

Show the containers of this Selfhostly install

```
selfhostlyctl status
```

```
Finds the running Selfhostly backend by probing containers (never by name) and shows every container of
its compose project. If no backend is found it shows all running containers.
```

## selfhostlyctl upgrade

Upgrade or restart safely: rollback point, dry start, health checks, automatic rollback

```
selfhostlyctl upgrade [flags]
```

```
Steps, in order:
  1. check nothing is deploying and that the compose file renders
  2. save the running images under a rollback tag, so a pull cannot lose them
  3. pull new images (unless you only changed settings), then DRY START: run the new image's doctor in a
     throwaway container with your real settings and data, so a start that would fail is caught first
  4. stop the primary briefly and copy the database
  5. recreate the primary, wait until healthy, run doctor
  6. recreate the gateway and wait
  7. confirm every other container that was running still is
If the primary does not become healthy it rolls back by itself. Your apps are never touched.
```

| Option | Meaning |
|---|---|
| `--compose stringArray` | compose file (repeatable; default docker-compose.prod.yml) |
| `--force` | continue even if a deployment is running or the dry start found problems |
| `--health-timeout int` | seconds to wait for health (default 90) |
| `--no-pull` | do not pull (use images already present) |
| `--no-rollback` | leave a failed upgrade in place for inspection |
| `--project string` | compose project name (default: detected) |
| `--pull` | pull images even when --set is used |
| `--rollback` | restore the state saved by the last run |
| `--rollback-to string` | with --rollback: restore this saved state instead of the last |
| `--set stringArray` | write KEY=VALUE into the settings file and apply it (repeatable) |
| `--with-frontend` | also recreate the frontend |

## selfhostlyctl version

Print the selfhostlyctl version

```
selfhostlyctl version
```
