# Compose versions and rollback

Every change to an app's compose file is kept as a numbered version, so you can see what changed and go back.

## Behaviour

- Creating an app saves **version 1** ("Initial version").
- Editing the compose file saves the next version, but only if the content actually changed.
- **Rolling back never rewrites history.** It saves a *new* version whose content is the old one, marks it current, updates
  the app record and writes the file to the app's folder. `rolled_back_from` on that new version holds the version number
  that was current when the rollback happened.
- Exactly one version is current at a time.
- **A rollback does not restart the app.** It changes the stored and on-disk compose file; the containers keep running the old
  configuration until you update or restart the app from the UI.
- The compose policy is checked when an app is deployed, not when a version is saved, so a rolled-back version that the
  current policy would block is refused at the next deploy ([volume paths](../security/volume-allowlist.md)).
- `changed_by` is the signed-in user's display name. It is attribution for people reading the history, not proof of identity
  (the audit log records the request itself).
- Deleting an app deletes its versions (foreign key with cascade).

## Where it lives

| What | Where |
|---|---|
| Table `compose_versions` | `internal/db/db.go` (`id`, `app_id`, `version`, `compose_content`, `change_reason`, `changed_by`, `is_current`, `created_at`, `rolled_back_from`; unique on `app_id, version`) |
| Saving versions on create and update | `internal/service/app_service.go` |
| Listing, reading and rolling back | `internal/service/compose_service.go` (port `domain.ComposeService`) |
| HTTP handlers | `internal/http/app.go`, routes in `internal/http/routes.go` |
| The history UI | `web/src/features/app-details/` |

## API

All three need a login and a `node_id` (the app's node); the rollback is audited as `app.restore_version`.

| Request | Returns |
|---|---|
| `GET /api/apps/{id}/compose/versions` | every version, newest first |
| `GET /api/apps/{id}/compose/versions/{version}` | one version |
| `POST /api/apps/{id}/compose/rollback/{version}` with optional `{"change_reason": "..."}` | the new version |

## Troubleshooting

| Symptom | Check |
|---|---|
| No new version after an edit | The content did not change (whitespace-only edits that leave the text identical do not count), or the save failed: see the server log. |
| Rolled back but the app behaves the same | The rollback does not restart the app. Update or restart it from the UI. |
| A deploy is refused after a rollback | The old version breaks the current compose policy. `selfhostlyctl doctor --audit-apps` names the mount. |
| Rollback returns an error | The target version must exist for that app, and the app folder must be writable. |
