# Gateway

The gateway is the single public entry point. This page explains what it does and how requests reach nodes. To install or
update it, see [install.md](install.md) and [operate.md](operate.md); the compose file is `docker-compose.prod.yml`.

```
Browser → Cloudflare Tunnel → Gateway :8080 → Primary :8082 (Docker network)
                                            → Direct secondaries (Docker/private network)
                                            → Linked secondaries, through the primary
```

## What each part does

| Part | Role |
|---|---|
| Gateway | Public port 8080. Checks the user's login, forwards each request, keeps a registry of nodes. Nothing else is public. |
| Primary | The database, the UI's API, user login, Docker on its own machine, and the coordinator of secondaries. Published only on `127.0.0.1:${PRIMARY_NODE_PORT:-8082}` on the host; the gateway reaches it over the Docker network. |
| Secondary | Runs Docker for its own machine. It normally connects **out** to the primary ([node link](../rfcs/node-link.md)) and publishes no port. |

## How a request is routed

1. The gateway verifies the user's session token (HS256, with the expected issuer) unless `AUTH_ENABLED=false`. Requests
   for a node's outbound link (`/api/nodes/connect`) skip this, because they carry the node's own credentials.
2. Requests with no `node_id` go to the primary.
3. Requests for a node go to that node:
   - a **direct** node is called by the gateway at its registered address;
   - a **linked** node has no address the gateway can reach, so the request goes to the primary, which forwards it down the
     node's connection using the node's own credentials. The user's cookies and tokens are removed and never reach the node.
4. A node id the gateway does not know triggers an immediate refresh of its registry from the primary (at most every 3
   seconds), so a node that just joined is routable at once. Otherwise the registry refreshes every
   `GATEWAY_REGISTRY_TTL_SEC`.
5. The gateway adds `X-Gateway-API-Key` (node management endpoints always require `GATEWAY_API_KEY`, whatever the user's
   login) and `X-Forwarded-Host` (so OAuth redirects use the public hostname).

Responses stream and WebSocket upgrades pass through.

## Settings that matter

| Setting | Meaning |
|---|---|
| `GATEWAY_API_KEY` | Secret shared by the gateway, the primary and every directly connected secondary. Required. |
| `JWT_SECRET` | Must be the same on the gateway and the primary. Changing it logs everyone out. |
| `PUBLIC_HOSTS` | The hostnames users type. OAuth redirects and cookies only ever use these: a request that names another host (forged `X-Forwarded-Host`, `Referer` or cookie) is answered as the first one, and logged as a warning. If login fails after pinning, check this. |
| `PRIMARY_BACKEND_URL` | Where the gateway reaches the primary. Set by the compose file. |
| `GATEWAY_REGISTRY_TTL_SEC` | How often the node registry refreshes on its own. |

Every setting and its default: [security overview](../security/overview.md).

## Keep it safe

- Keep `GATEWAY_API_KEY` secret: it grants node management.
- Never expose the primary or a direct node to the internet. Use the Cloudflare Tunnel (TLS at its edge) in front of the
  gateway.
- To publish the primary on a private address for direct-mode secondaries, set `PRIMARY_NODE_BIND` to that address only
  ([install.md](install.md#3-add-a-secondary-machine)).

## Logs

`docker compose -f docker-compose.prod.yml logs -f gateway`. The routing lines (`gateway: incoming request`,
`gateway: routing request`, `router: resolved by node_id`, `router: primary-only route`) are debug level, which is only on
with `APP_ENV=development`.

## Troubleshooting

| Symptom | Look at |
|---|---|
| `gateway: upstream request failed` | Is the primary healthy (`selfhostlyctl status`)? Is `PRIMARY_BACKEND_URL` right? |
| `router: node not found node_id=…` | The node has not registered yet (check the primary's logs), or it restarted with a new id. A linked node shows offline within seconds when its connection drops. |
| `gateway: auth required` | Session missing or expired, or `JWT_SECRET` differs between gateway and primary. |
| Login fails after pinning, or `forwarded host is not a configured public host` | `PUBLIC_HOSTS` must list the hostname users type. |

There is a single gateway container; the compose file gives it a fixed container name, so it is not designed to be scaled
by replicas.
