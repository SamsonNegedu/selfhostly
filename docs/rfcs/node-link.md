# RFC: outbound-only node link

Status: implemented (see "What implementation showed"). Replaces the two-way "direct" node connectivity for new nodes;
direct nodes keep working.

## Problem

A secondary node today needs the primary and the gateway to reach it, and it needs to reach the primary:

- every machine must be routable from the primary and the gateway (a problem behind NAT or a home router);
- the secondary serves its API without user login and relies on network isolation for protection;
- traffic is plain HTTP with long-lived shared keys;
- registration and heartbeats cannot pass through the gateway (login required, credential headers stripped);
- the primary polls every node every 30 seconds, and every node also sends heartbeats.

## Decision

The secondary opens **one outbound WebSocket** to the primary and serves HTTP/2 **over that connection, in
reverse**: the primary is the HTTP/2 client, the secondary is the HTTP/2 server. The primary's existing node
client keeps speaking ordinary HTTP; only the connection it uses changes.

```
secondary --(outbound wss through Cloudflare + gateway)--> primary
primary   --(HTTP/2 requests over that same connection)--> secondary's loopback API
```

- The secondary's API listens on **loopback only**: no published port, nothing to firewall.
- A tunnel node is stored with the endpoint `tunnel://<node-id>`. A `RoundTripper` registered for that scheme
  sends the request down the node's session. All node client code is unchanged.
- The gateway forwards everything for tunnel nodes to the primary, which already routes to nodes.
- Liveness is the connection: a dropped connection marks the node offline at once. No heartbeat and no polling
  are needed for tunnel nodes.

## Why HTTP/2 in reverse, not a stream multiplexer

`golang.org/x/net/http2` is already in the module graph. It gives per-request streams, flow control (streaming
log responses work as they are) and PING keepalive, and the traffic is already HTTP, so there is no
stream-to-HTTP glue to write. The only new direct dependency is a WebSocket library
(`github.com/coder/websocket`), needed because Cloudflare proxies WebSocket upgrades but not arbitrary ones.

## Security

- The WebSocket carries the node's identity in headers over TLS (terminated at Cloudflare). A new node presents
  a single-use **join token**; a known node presents its node key, compared in constant time.
- Requests the primary sends over the link still carry the node key, and the secondary still verifies it, so
  loopback is defence in depth and not the only control.
- The connect endpoint is rate limited, and skips the gateway's user login (it has its own credentials).
  It never gets the gateway API key.
- One session per node: a new connection replaces the old one.
- Tunnel endpoints cannot be supplied through the node registration API (only the connect handler creates them),
  so a user cannot point one node's traffic at another node's session.

## Compatibility and migration

- Existing direct nodes are untouched. `selfhostly join --direct` keeps the old flow.
- `NODE_TRANSPORT=tunnel` selects the new mode on a secondary. No schema change.
- Heartbeat and auto-registration are not started in tunnel mode.

## Not in scope

Retiring direct mode; per-node rotating credentials; a UI for node links.

## What implementation showed

- **By-id operations are not forwarded by the primary today.** They run against the local database of
  whichever backend receives them, and the gateway sends them straight to the right node. A linked node has no
  address the gateway can reach, so the primary gained a small forwarding step: a request naming a linked node
  (`?node_id=`, or `node_id` in the body of `POST /api/apps`) is proxied down that node's link with the node's
  credentials.
- **The gateway now uses the standard library's reverse proxy.** The hand-rolled one could neither upgrade to
  WebSocket nor flush streaming responses. It also re-reads its node list immediately when a lookup fails,
  because a node that has just joined or reconnected was otherwise unroutable for up to a minute.
- **A pre-existing bug surfaced:** direct auto-registration ran its health check before saving the node, so
  every node was reported unreachable at registration. Fixed.
- **Link credentials use dedicated headers** (`X-Selfhostly-Node-*`), because the gateway removes `X-Node-*`
  from client requests.
- **Verified end to end** with a secondary on a Docker network that cannot reach the primary, whose only path is
  the gateway's published port (`scripts/test-join.sh`): it joins with one command, publishes no port, is
  controlled through the gateway, goes offline within seconds of stopping, and reconnects by itself after a node
  restart and after a primary restart.
