# Test the deployment paths

| Script | Proves |
|---|---|
| `scripts/smoke-test.sh` | fresh install: login required, forged headers ignored, read-only root, restart keeps data |
| `scripts/test-upgrade.sh` | upgrade, automatic rollback, `--set` reverts, encryption cycle |
| `scripts/test-upgrade-original.sh` | upgrade from a hand-edited original compose file, detection of the live compose file |
| `scripts/test-join.sh` | a machine on an isolated network joins and is controlled through the gateway |
| `scripts/test-host-linux.sh`, `scripts/test-compose-diff.sh` | host checks (Pi simulated, real Linux sockets) and compose-diff |
| `make e2e-update` | the UI-driven update in a real browser, against production images built locally: see [below](#test-the-ui-driven-update) |
| `go test ./internal/ctl/...` | the CLI itself: upgrade order and rollback, bootstrap, backup, join-token, the generated reference |

The scripts need Docker and run on private networks under their own names, so they do not touch a real install.

## Test the UI-driven update

`make e2e-update` mirrors a deployment and drives the real UI. It builds the production images (`Dockerfile.backend`,
`Dockerfile.gateway`, `web/Dockerfile`) from your working tree, pushes them to a local registry in place of GHCR, and
runs `docker-compose.prod.yml` behind a reverse proxy in place of the Cloudflare tunnel. A signed release manifest is
served locally. Playwright then signs in, opens Settings, Updates and updates the install.

Change code, run it again, and the new images are what the update installs. No GitHub, GHCR or Cloudflare is used.

```bash
make e2e-update                      # purge, build, start, run every spec, tear down
OLD_REF=<commit> make e2e-update     # the installed version is built from this commit (it must contain the update feature)
SKIP_BUILD=1 make e2e-update         # reuse the images from the last run
KEEP=1 make e2e-update               # leave the stack up at http://localhost:45052 to look around
```

Each part also runs alone, which is how to iterate. `scripts/e2e/run.sh help` lists them:

```bash
scripts/e2e/run.sh build                       # images for 1.0.0 (installed), 1.1.0 (the update) and 1.2.0 (never healthy)
scripts/e2e/run.sh up                          # stack, proxy, manifest server, signed-in test session
scripts/e2e/run.sh publish 1.1.0               # also: --required KEY, --optional KEY, --generated KEY, --tamper
scripts/e2e/run.sh test tests/03-update-success.spec.ts
scripts/e2e/run.sh down                        # remove the stack; `purge` also removes the registry, images and files
```

| Spec | Proves |
|---|---|
| `01-baseline` | the stack is the production topology, the browser is signed in, and sign-in is still enforced |
| `02-update-blocked` | a tampered release, a missing required setting and a running job each stop the update, and an anonymous caller cannot reach it |
| `03-update-success` | badge, review, confirm, progress, reconnect, then the new version runs, the digest is pinned and the apps were not restarted |
| `04-update-rollback` | a release that never becomes healthy is rolled back with a reason, and the previous version, settings and apps are intact |

What is real and what stands in for it:

| Deployed | Here |
|---|---|
| GHCR | `e2e-registry` on `localhost:45050`, images referenced by digest |
| Cloudflare tunnel routing | `e2e-proxy` on `localhost:45052`: `/api`, `/auth`, `/avatar` to the gateway, the rest to the frontend |
| GitHub release manifest | `e2e-manifest` (nginx) serving `release.json` and `release.json.sig`, signed with a throwaway key |
| GitHub sign-in | a session cookie signed by `scripts/e2e/mint-session` with the install's `JWT_SECRET`. The server still checks signature, issuer and allowlist |

The primary runs as root here (`APP_UID=0`), because the Docker socket's group differs between Linux, Docker Desktop and OrbStack.

**One compose file, like a real install.** `scripts/e2e/composegen` renames the containers and the network in
`docker-compose.prod.yml` (`selfhostly-*` becomes `e2e-*`) and drops cloudflared. The release images embed that file and the
manifest signs its hash, so the update sees the compose file a real release carries. The installed copy lacks one line
(`UPDATE_CHECK_INTERVAL_HOURS`), as an older install does, so the review reports the compose file as `behind` with a one line
diff, and the update needs the approval a real one would.

**Safety.** Everything is named `e2e-*` and labelled `selfhostly.e2e=1` under the project `selfhostly-e2e`, on the network
`e2e-network`. Only those resources are ever removed. Ports are `45050` (registry), `45052` (UI) and `45053` (node
port), all on loopback, set in `scripts/e2e/env.sh` and overridable with `E2E_REGISTRY_PORT`, `E2E_UI_PORT`,
`E2E_NODE_PORT`. A port that something else holds is reported, never freed. Files live in `tmp/e2e/` (git-ignored).
The first run downloads base images (golang, alpine, node, registry, nginx) and a Playwright Chromium. Set
`E2E_BROWSER_CHANNEL=chrome` to use the Chrome you already have instead.

## Choosing the "old" version

The upgrade tests start from the version your install runs today. Set `OLD_REF` to that commit or tag.

- With no `OLD_REF` and uncommitted changes, they use `HEAD` as the old version.
- With no `OLD_REF` and a clean tree, the change is already committed, so they use the previous commit and say so.
- To start from an older deployed version, pass that commit. CI passes the pull request's base.

## Docker Desktop and OrbStack

The socket group differs from a Linux host's, so the smoke test needs `SMOKE_PRIMARY_USER=0:0`.

## Changing `selfhostlyctl`

After changing a command or flag, run `make docs`. A test fails if `docs/reference/selfhostlyctl.md` is out of date.
Day-to-day development commands are in [getting-started.md](getting-started.md).
