# Backend gaps

Things the designs need, or that the fixtures exposed, that the backend does not do yet. The UI is
built from what the API returns today and shows an honest empty or unavailable state. Nothing here
is faked in shipped UI.

| Found in | Gap | What the UI does meanwhile |
| --- | --- | --- |
| F10 | `GET /api/system/stats?node_ids=a,b,c` leaves out a node that did not answer, with no per-node error, so the caller cannot tell an unreachable node from one it did not ask for. It should return the nodes that answered plus a per-node error for the rest. | Insights takes the reachable nodes' numbers and shows unreachable nodes from the node list's status, not from the stats response. |
| F10 | `GET /api/me` returned 503 while a stats request was pending. Not reproduced: with a hanging node selected, stats answered in about 3 s and `/api/nodes` stayed instant. `/api/me` itself was not tested because it only exists with login on. | None needed. |
| S5 | No endpoint reports host ports in use, or checks a compose file before deploy. Port conflicts are detected from the failure text, and free ports are checked only against other apps' compose files. | The guide says the suggested port is not used by other apps, never that it is free. |
| S7 | No environment storage, `.env` import endpoint or secret store. Variables live as plain text in each service's `environment` in the compose file, and versions keep old values. | The tab edits the compose file, masks values that look secret, and does not claim they are encrypted. There is no "unused" flag. |
| S12/S13 | No template catalog, no server-side Git import and no pre-deploy check endpoint. | Templates are static data in the UI, links are fetched by the browser (public files only), and checks run in the browser. |
| S17 | No metrics history endpoint (only the current reading), and no endpoint to adopt an unmanaged container as an app. | Charts are built from the readings taken while the page is open and say so. There is no range picker and no Adopt action. |
| S18/S19 | No endpoint to test tunnel provider credentials, no alert or notification settings, and no backup or export. The provider token is stored as saved text unless `ENCRYPT_SECRETS_AT_REST=true` (the API response always masks it). | Settings has no Test connection, Alerts or Backup section. The status card reflects whether credentials exist, not that they work. |

## Resolved

| Gap | What was done |
| --- | --- |
| S15: node registration returned a bare 500 for a duplicate ID or name, and waited on the node's health check | Conflicts are now 409 with `code` and `field` (`NODE_ID_TAKEN`, `NODE_NAME_TAKEN`, `PRIMARY_NODE_PROTECTED`, `NODE_HAS_APPS`), not found is 404 with `NODE_NOT_FOUND`. Registration answers at once with status `unknown` and checks the node in the background. The web client reads `code` and `field` and shows them next to the input. |
| F10: one unreachable node held up stats for the others | Stats no longer wait on a node that does not answer: nodes already known to be offline or unreachable are skipped, and one that hangs while still unknown is dropped after about 3 s. Checked with nodes at a black-hole address (packets dropped): the healthy node's stats came back in under 3 s. |
| F7: `GET /api/apps/:id/services` returned 500 for an app with no containers | No longer reproduces. Checked against a fresh server: it answers 200 with the compose file's service names for an app that was never started, is running or was stopped, and with `[]` when the app's folder is missing. |
| Audit log only had method and path | Each entry now has an action (`app.stop`), a target type, id and name, recorded before the handler runs so a deleted app is still named. Settings > Activity shows sentences. |
| Node cards had no check details | Nodes report `last_health_check`, `last_latency_ms` and `consecutive_failures`. |
| Polling only | `GET /api/events` is a server-sent stream of app, job and node changes (kind and id only, never values). The web app refreshes the matching queries when one arrives, and polling stays as the fallback. |

Still open: events come from this node's own database, so changes on a secondary node reach the primary's screen through polling.

