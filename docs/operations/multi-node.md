# Multiple nodes

One **primary** holds the database and the UI's API. **Secondary** nodes run apps on other machines and are controlled from the
same UI. Adding one is two commands ([install.md](install.md#3-add-a-secondary-machine)); this page explains how the nodes
relate and how to tell what is wrong. Design: [node-link RFC](../rfcs/node-link.md). The public entry point:
[gateway](gateway-deployment.md).

## How a secondary connects

| | Linked (default) | Direct (older) |
|---|---|---|
| Who connects | The secondary dials **out** to the primary's public address and keeps the connection open | The primary calls the secondary's address |
| What the secondary needs | Internet access. No open port, firewall rule or VPN | A reachable address and port on a private network |
| Recorded address | `tunnel://<node id>` | `http(s)://host:port` |
| Liveness | The open connection: offline within seconds of a drop, back by itself on reconnect | Polling by the primary (below) |

Existing direct nodes keep working. To switch one, set `NODE_TRANSPORT=tunnel` on it.

A secondary needs no Cloudflare or login settings of its own: it syncs settings from the primary and is only ever reached by
the primary and gateway.

## Authentication between nodes

- A node proves itself with its **id and API key** (`X-Node-ID`, `X-Node-API-Key`). The same routes accept a signed-in user or a
  node; a node's key can never mint join tokens, end sessions, read the audit log or open the change stream.
- A **new** node presents a single-use join token (`selfhostlyctl join-token`, valid one hour) or the shared registration token
  when it first registers, through `POST /api/nodes/register` or, for a linked node, `GET /api/nodes/connect`. Both are rate
  limited.
- Requests between nodes only go to `http(s)` addresses without credentials in the URL. Link-local and similar addresses are
  always refused; `NODE_ENDPOINT_ALLOWED_CIDRS` can restrict further ([security overview](../security/overview.md)).
- The gateway key (`GATEWAY_API_KEY`) is what lets the gateway call node management routes.
- Use `https` for the primary's address: the node key is sent in the connection request.

Users sign in as usual ([GitHub](../security/github-allowlist.md) or [Cloudflare Access](cloudflare-zero-trust.md)). Public
endpoints: `/api/health`, `/auth/*`, `/avatar/*`, `/api/nodes/register`, `/api/nodes/connect`.

## Health of direct nodes

The primary checks each direct node with `GET /api/health` (sending the node's credentials) every **30 seconds**, and backs off
while it keeps failing:

| Consecutive failures | Checked | Status |
|---|---|---|
| 0 to 2 | every 30 s | `online` (still) |
| 3 to 5 | every 2 min | `offline` from the 3rd |
| 6 to 9 | every 5 min | `offline`, `unreachable` from the 5th |
| 10 or more | every 15 min | `unreachable` |

A success resets the count. A circuit breaker also stops calls to a node that keeps failing, so a dead node costs little.
`POST /api/nodes/:id/check` (the UI's check button) tests a node right now and skips the wait.

A direct node also announces itself: two seconds after it starts it sends `POST /api/nodes/:id/heartbeat`, which marks it
`online` at once instead of waiting for the next check.

The primary's own node is always `online`.

## Troubleshooting

| Symptom | Check |
|---|---|
| A linked node shows offline | On the node: `docker logs selfhostly-node \| grep "node link"`. `node link up` means connected. `node link down, reconnecting` shows the last error: `401` the token or key is wrong, expired or already used; `409` the name is taken; a connection error means it cannot reach the primary's public address. It retries by itself. |
| Joining fails with "answered, but not like a Selfhostly primary" | A login page (Cloudflare Access) is in front of the primary. See [Cloudflare Access](cloudflare-zero-trust.md#adding-a-secondary-machine). |
| A direct node shows offline | From the primary: `curl http://<node>:8082/api/health` should answer `{"status":"healthy","service":"selfhostly"}`. If not: the node is down, its address is wrong, or a firewall blocks its port (8082 by default). |
| A direct node answers but is refused | The key the primary has for the node does not match the node's `NODE_API_KEY`. Update it in **Settings → Nodes → the node → Edit**. |
| Registration fails with `401` | The registration or join token is wrong, expired or already used. `selfhostlyctl join-token` makes a new one. |
| `409` on registering | A node with that name (or a different key for the same id) already exists. Choose another `--name`, or update the existing node's key. |
| Apps do not deploy to a node | Requests for a node must carry its `node_id`. Check the node is online, then look at the job log for the app. |
| A node does not appear in monitoring | Offline and unreachable nodes are left out ([monitoring](monitoring.md)). |
| Health checks seem not to run | The primary logs `background tasks started` at startup and a line per check at debug level (`APP_ENV=development`). |

To register a direct node by hand when it could not register itself: **Settings → Nodes → Register Node**, then enter the
node's id and API key exactly as the node has them (its startup log and `.env.node`), its name, and the address the primary
can reach it on, for example `http://192.168.1.50:8082`.
