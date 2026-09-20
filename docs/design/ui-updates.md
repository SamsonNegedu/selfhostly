# UI-driven updates

Selfhostly can update itself from Settings -> Updates. The primary cannot restart itself, so it starts a short-lived
updater container that runs the same steps as `selfhostlyctl upgrade` and outlives the primary.

Off by default: set `UI_UPDATES_ENABLED=true`. Only a signed-in user can use it. Node credentials and the gateway key
cannot.

## Flow

1. **Release (CI).** Every push to main that changes the gateway, backend or frontend is a release: CI builds the images and
   publishes a signed `release.json` (see [Releases](#releases)). Nobody tags anything.
2. **Check.** The primary fetches `release.json` and `release.json.sig`, verifies the signature, and shows "Update available".
3. **Review.** The primary pulls the target backend image by digest and runs `selfhostlyctl update-plan` in it, with the
   project directory mounted read-only. The plan lists blockers, the compose diff and the settings the release needs.
4. **Inputs.** Required settings are entered in the UI. Generated secrets are listed. The compose diff is approved.
5. **Apply.** The primary takes a lock, writes the status file and starts the updater container.
6. **Updater.** Steps, in order: `verify` the manifest, `rollback_point` (images, `.env`, compose file, database copy taken later),
   `configure` (pin the new digests, write settings, swap the compose file), `pull`, `dry_start`, `database`, `primary`, `gateway`,
   `frontend`, `apps`. Any failure rolls back.
7. **Result.** The UI reconnects and shows success, or "rolled back" with the reason. An unfinished run is detected on the
   next start and offers rollback or retry.

## Trust

- `release.json` is signed with an ed25519 key. The public key is `UPDATE_PUBLIC_KEY` (base64), else the one built in
  (`internal/update/pubkey.go`). With no key, the feature reports itself as unavailable.
- Image references in the manifest must be `<UPDATE_IMAGE_REPO_PREFIX>(backend|gateway|frontend)@sha256:<64 hex>`.
  The prefix defaults to `ghcr.io/samsonnegedu/selfhostly-`. The API never accepts an image or version from the browser
  beyond choosing the version the manifest names.
- The updater re-verifies the manifest itself, so a change between the primary's check and the run is caught.
- The signing key lives only in CI (`RELEASE_SIGNING_KEY`). `cmd/releasetool` generates keys, builds and signs manifests.

## Releases

Automatic, from `.github/workflows/build-and-publish.yml`. A push to main that changes `cmd/gateway`, `internal/gateway`,
`cmd/server`, `internal/`, `web/`, a Dockerfile or `go.mod`/`go.sum` is a release. Docs-only pushes and pull requests are not.

- **Version:** `<RELEASE_SERIES>.<commits on main>`, for example `1.0.68`. `RELEASE_SERIES` is set in the `check-changes` job.
  Change it (to `1.1`, `2.0`) to start a new series; the count keeps the number growing. The backend image is built with this
  number stamped in, which is how an install knows what it runs. Builds from pull requests and other branches report `dev`.
- **Images:** the backend is rebuilt for every release. A gateway or frontend that did not change keeps the image of the last
  release (what `:latest` points at).
- **Manifest:** the `release-manifest` job reads the image digests, adds `release/manifest-inputs.json` (edit it when a release
  needs a new setting or drops support for old installs), signs it with the `RELEASE_SIGNING_KEY` secret and creates the GitHub
  release `v<version>` marked latest, with `release.json` and `release.json.sig` attached. The notes are the commit subjects
  of the push. `UPDATE_MANIFEST_URL` points at `releases/latest/download/release.json`, which is therefore always the newest.
- **Safety check:** before publishing, the job verifies the signature against the key built into the image
  (`internal/update/pubkey.go`). A secret that does not match fails the job instead of publishing something no install trusts.
- **One at a time:** releases queue, so a newer push is never overtaken by an older one.
- If `RELEASE_SIGNING_KEY` is missing the job warns and publishes nothing, so installs are never offered the release.
- Existing installs from before this feature have no updater. Move them once with `selfhostlyctl upgrade`.

## release.json

```json
{
  "schema": 1,
  "version": "1.4.0",
  "published_at": "2026-09-20T10:00:00Z",
  "notes": "plain text release notes",
  "min_from_version": "1.0.0",
  "images": {
    "backend": "ghcr.io/samsonnegedu/selfhostly-backend@sha256:...",
    "gateway": "ghcr.io/samsonnegedu/selfhostly-gateway@sha256:...",
    "frontend": "ghcr.io/samsonnegedu/selfhostly-frontend@sha256:..."
  },
  "compose": { "sha256": "<sha256 of the release's docker-compose.prod.yml>" },
  "settings": [
    { "key": "NEW_KEY", "kind": "optional | generated | required", "description": "text", "secret": false }
  ]
}
```

## API

All under `/api/system/update`, signed-in admin (allowlisted) only. State changing routes are audited (`update.check`,
`update.plan`, `update.apply`, `update.rollback`).

| Route | Purpose |
|---|---|
| `GET /api/system/update` | Current state. Cheap, safe to poll. |
| `POST /api/system/update/check` | Fetch and verify the manifest now. |
| `POST /api/system/update/plan` | Start the review (pull and plan) for a version the manifest names. |
| `POST /api/system/update/apply` | Start the updater with the version, the inputs and the compose approval token from the plan. |
| `POST /api/system/update/rollback` | Restore the last rollback point. |

The three `POST`s that start work return 202. The UI then polls `GET`. Its response has `enabled`, `disabled_reason`,
`current_version`, `checked_at`, and `available`, `plan` and `run`, each `null` when absent. The shapes are `UpdateStatus` in
`web/src/shared/types/api.ts` and `Plan` and `Run` in `internal/update`.

- `compose.state` in the plan is `current` (file already matches the release), `behind` (the swap loses nothing, carry-over
  lines go to `.env`, needs the approval token) or `customized` (the swap would lose hand edits: blocker
  `compose_customized`, merge by hand).
- A blocker stops the update until it clears. Its `code` says why, for example `job_running`, `missing_required`,
  `unsupported_from` or `release_mismatch` (the compose file in the release image is not the one the manifest was signed with).
- A rollback run (`kind: rollback`) has three steps (`primary`, `gateway`, `frontend`) and ends as `rolled_back`.

## Status file

`<data dir>/updates/status.json` holds the `run`. It is written atomically by the updater. The primary reads it. On start, a
`pending` or `running` state whose updater container is gone becomes `interrupted`.

## Failure handling

- The updater saves a rollback point first: images, `.env`, compose files. The database copy is taken just before the primary
  stops. A rollback keeps the database, since an older backend runs on a migrated one.
- Any failed step after that rolls back: the primary cannot be recreated, does not become healthy or refuses to start, the
  gateway or frontend fails, `doctor` reports a failure, or the database copy fails. A failure before anything running was
  touched (a pull, a missing value, an unapproved compose change) restores `.env` and the compose file and restarts nothing.
- A rollback that cannot restore an old image says so and reports the run as failed, never as rolled back.
- The updater keeps a lock in `.upgrade/.lock` that names its owner. A lock whose container or process is gone is removed by
  the next run. A lock with no owner is never guessed at.

## Not covered

- The docker socket proxy overlay (`docker-compose.socket-proxy.yml`): the updater needs the raw socket.
  The review reports the blocker `socket_proxy`.
- Secondary nodes are not updated from the UI. Each secondary is updated with `selfhostlyctl upgrade` on its machine. A
  primary that is ahead of a secondary keeps working with it: migrations only add, and the node API only grows.
- Key rotation. Installs only trust the key built into their image (or `UPDATE_PUBLIC_KEY`). To change it, ship a release
  signed with the old key that carries the new key in `internal/update/pubkey.go`, then switch the `RELEASE_SIGNING_KEY`
  secret. Switching the secret first makes every release fail the CI check in [Releases](#releases) and installs keep the old key.
