# Test the deployment paths

| Script | Proves |
|---|---|
| `scripts/smoke-test.sh` | fresh install: login required, forged headers ignored, read-only root, restart keeps data |
| `scripts/test-upgrade.sh` | upgrade, automatic rollback, `--set` reverts, encryption cycle |
| `scripts/test-upgrade-original.sh` | upgrade from a hand-edited original compose file, detection of the live compose file |
| `scripts/test-join.sh` | a machine on an isolated network joins and is controlled through the gateway |
| `scripts/test-host-linux.sh`, `scripts/test-compose-diff.sh` | host checks (Pi simulated, real Linux sockets) and compose-diff |
| `go test ./internal/ctl/...` | the CLI itself: upgrade order and rollback, bootstrap, backup, join-token, the generated reference |

The scripts need Docker and run on private networks under their own names, so they do not touch a real install.

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
