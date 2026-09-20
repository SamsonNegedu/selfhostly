# Security model and configuration

Selfhostly is a single-user platform that can start containers on your host. A logged-in user can
create arbitrary containers, so **authentication and the compose policy are the security boundary**.
This page says what is enforced, how to configure it, and what is still your responsibility.

## Modes

`SECURITY_MODE` is `enforce` or `warn`.

- `enforce`: policy violations are blocked, and the server refuses to start with unsafe settings.
- `warn`: the same checks run, but anything that could stop an already-running app is only logged.

If unset: a **fresh install enforces**, an **existing database warns** until you opt in, and
`APP_ENV=development` warns. The first start saves the decision to `data/security-mode`, so it cannot
change on a later start by itself; `SECURITY_MODE` in `.env` overrides the file. Some checks are blocked in every mode because no legitimate app needs
them (see "Compose policy"). `selfhostly doctor --audit-apps` lists what enforce mode would block.

## What is enforced

| Area | Behaviour |
|---|---|
| Who may log in | GitHub allow-list is matched on the stable user ID derived from the immutable login, never the display name (which any GitHub user can set to someone else's login). `GITHUB_ALLOWED_USERS` stays a list of logins; canonical casing is resolved on GitHub at startup. Entries may also be `github_<hash>` IDs. |
| No-auth deployments | With `AUTH_ENABLED=false` and Cloudflare Access configured (`CF_ACCESS_TEAM_DOMAIN`, `CF_ACCESS_AUD`) every request must carry a valid signed `Cf-Access-Jwt-Assertion` (RS256, issuer, audience, expiry checked). |
| Startup | Enforce mode refuses to start: no auth on a non-loopback primary, weak `JWT_SECRET`, missing OAuth credentials, empty allow-list. Secondaries only get a warning (see limitations). |
| Cross-site requests | Cookies are `SameSite=Lax`. State-changing requests with an `Origin` from another site, or a body that is not `application/json`, are rejected. |
| Sessions | Token lifetime 1h (refreshed while the session cookie lives, `AUTH_SESSION_HOURS`, default 24). `POST /api/security/revoke-sessions` invalidates every token issued before now. |
| Gateway | Verifies HS256 only, issuer `selfhostly`, expiry required. Strips client-supplied `X-Gateway-API-Key`, `X-Node-ID`, `X-Node-API-Key`. `PUBLIC_HOSTS` pins the host used for OAuth redirects and cookies. |
| Node credentials | Constant-time comparison. Node and gateway keys cannot mint join tokens, revoke sessions or read the audit log. |
| Node link | A secondary's outbound WebSocket carries its identity in dedicated headers over TLS (terminated at Cloudflare). A new node needs a single-use join token; a known node proves itself with its key (constant time). The endpoint is rate limited and skips the gateway's user login. Requests the primary forwards over the link carry the node's credentials; the user's cookies, tokens and Origin are removed. A tunnel endpoint cannot be supplied through the registration API, so one node's traffic cannot be pointed at another node's link. |
| Node registration | Rate limited (10 per minute per client). Accepts the shared `REGISTRATION_TOKEN` or a single-use join token. Re-registering with a different API key is refused. |
| Node endpoints | `http(s)` only, no credentials in the URL. Link-local, unspecified and multicast addresses are always refused, checked again at connect time (defeats DNS rebinding). No redirects are followed. Optional `NODE_ENDPOINT_ALLOWED_CIDRS`. |
| Container actions | Restart, stop and delete only act on containers created from an app folder (compose working directory under the host apps directory) or labelled `com.selfhostly.managed=true`. Not the platform, not unrelated workloads. |
| Visibility | Every start logs `effective configuration` (each setting and where it came from); `selfhostly doctor` shows the same and checks the node ID against the database. |
| Secrets at rest | Node API keys, provider tokens and the per-app Cloudflare tunnel tokens can be AES-256-GCM encrypted (`ENCRYPT_SECRETS_AT_REST`). Reading accepts both forms, so apps saved before the flag was turned on keep working. The flag is the desired state of every stored secret: at startup, values are converted in either direction (existing plain text is encrypted on the first start with the flag on). A value that cannot be decrypted is an error for that app or node, never an empty value. |
| Compose interpolation | `docker compose` runs without the platform's own secrets in its environment, so app files cannot read them through `${VAR}`. |
| Audit | Every state-changing API request is recorded (actor by stable ID, method, path, status, client IP) for 90 days: `GET /api/security/audit`. |
| Headers | API responses send a restrictive CSP, `nosniff`, frame denial, referrer and permissions policies. The frontend container sends the same, with its CSP in report-only mode (below). |
| Request size | 10 MB cap on every state-changing body. |

## Compose policy

Checked on create, update, and again on the exact file on disk immediately before every `docker compose`
run (so a file edited by hand, or stored before the policy existed, is still checked).

Blocked in every mode: privileged, devices, `device_cgroup_rules`, `network_mode`/`pid`/`ipc`/`uts: host`,
custom `cgroup`, `cgroup_parent`, `userns_mode`, a non-default `runtime`, `group_add` of root or
docker, dangerous capabilities (`SYS_ADMIN`, `NET_ADMIN`, `BPF`, and others), disabled AppArmor,
SELinux, seccomp or `systempaths`, `include`, `extends`, `env_file` and `build` contexts outside the
app folder, and bind mounts (short form, long form, `driver_opts` binds, secrets and configs files)
of the Docker socket, `/`, `/etc`, `/root`, `/proc`, `/sys`, `/dev`, `/boot`, `/run`, `/var`, `/usr`
and similar, anything that resolves through a symlink out of the apps directory, or a relative path
that climbs out of the app folder.

Blocked in enforce mode, logged in warn mode: bind mounts outside the app folder and
`ALLOWED_VOLUME_PATHS`, interpolated (`${VAR}`) or `~` bind sources that cannot be verified, and
joining another container's namespaces (`network_mode: container:...`).

## Configuration reference

All optional unless marked. Defaults shown are for the primary node.

| Variable | Default | Purpose |
|---|---|---|
| `SECURITY_MODE` | see Modes | `enforce` or `warn` |
| `AUTH_ENABLED` | `false` | GitHub OAuth login |
| `JWT_SECRET` | required with auth | 32+ characters |
| `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `GITHUB_ALLOWED_USERS` | required with auth | OAuth app and allow-list of logins |
| `AUTH_BASE_URL` | `NODE_API_ENDPOINT` | Public URL for OAuth callbacks |
| `AUTH_SECURE_COOKIE` | `true` if `AUTH_BASE_URL` is https | Send session cookie over https only |
| `AUTH_SESSION_HOURS` | `24` | Browser session lifetime |
| `CF_ACCESS_TEAM_DOMAIN`, `CF_ACCESS_AUD` | unset | Verify Cloudflare Access tokens (with `AUTH_ENABLED=false`) |
| `ALLOW_UNAUTHENTICATED` | `false` | Acknowledge running open on a non-loopback address |
| `PUBLIC_HOSTS` (gateway) | unset | Comma-separated hostnames users reach the site on |
| `HOST_APPS_DIR` | detected | Host path behind `APPS_DIR` |
| `ALLOWED_VOLUME_PATHS` | empty | Host paths apps may bind-mount besides their own folder |
| `NODE_ENDPOINT_ALLOWED_CIDRS` | empty | Restrict node endpoints to these ranges |
| `NODE_ENDPOINT_ALLOW_LOOPBACK` | `false` | Allow loopback node endpoints (always allowed in warn mode and development) |
| `ENCRYPT_SECRETS_AT_REST` | `false` (never implied by the mode) | Encrypt stored node keys, provider tokens and per-app tunnel tokens. `selfhostlyctl bootstrap` sets it `true` for new installs |
| `SETTINGS_ENCRYPTION_KEY` | `data/secrets.key` (created) | Key for the above. Back it up off the server |
| `NODE_ID` | primary: the ID in the database; else generated once and saved | Pin only on purpose. An explicit value that disagrees with the database is reported by `doctor` |
| `NODE_API_KEY`, `REGISTRATION_TOKEN` | generated once and saved in the data directory | Node shared secrets |
| `APP_UID`, `DOCKER_GID` (compose) | `1000` / `984` | User and socket group the backend container runs as. Compose cannot detect them: `selfhostlyctl bootstrap` sets them for new installs, and `doctor` names the right `DOCKER_GID` if the process is not in the socket's group |
| `GATEWAY_API_KEY` | required on the gateway | Shared with backends |

## Commands and endpoints

- `selfhostly doctor [--audit-apps] [--offline]`: read-only preflight, non-zero exit on failures.
- `selfhostly backup [--dir]`: online database snapshot.
- `POST /api/nodes/join-tokens`: single-use token for a new secondary (user only).
- `POST /api/security/revoke-sessions`, `GET /api/security/audit?limit=100` (user only).

## Findings addressed and where

| Finding | Fix |
|---|---|
| Allow-list matched display name | `internal/auth/allowlist.go`, `internal/http/server.go` |
| Compose validator bypasses (`/run`, `/var`, long form, `driver_opts`, namespaces, `include`) | `internal/validation/policy.go`, deploy-time guard in `internal/docker/manager.go` |
| Auth off by default | `Config.ValidateStartup`, Cloudflare Access verification (`internal/auth/cfaccess.go`) |
| CSRF disabled | Origin, content-type and `SameSite` in `internal/http/security.go` |
| Gateway trusts host headers | `PUBLIC_HOSTS`, `internal/gateway/config.go` |
| Weak gateway JWT check | `internal/gateway/auth.go` |
| Client-supplied credential headers forwarded | `internal/gateway/proxy.go` |
| Timing, plaintext, registration abuse | constant-time compares, `internal/secrets`, rate limit, join tokens |
| Node endpoint SSRF | `internal/netguard`, `internal/node/policy.go` |
| Any container could be stopped | `authorizeContainer` in `internal/service/system_service.go` |
| Secrets visible to compose interpolation | `internal/docker/command_executor.go` |
| Hardening, pinning, socket proxy | `docker-compose.prod.yml`, `docker-compose.socket-proxy.yml`, `selfhostlyctl pin-images` |

## Known limitations

Read these; they are real.

- **The compose policy is a blocklist plus a bind-mount allow-list, not a sandbox.** A user who can
  create apps can still run containers that use the network, consume resources, and read anything
  under their own folder and `ALLOWED_VOLUME_PATHS`. Mounting the Docker socket is blocked, but the
  platform itself holds it, so any bug in the platform is host root. The socket proxy narrows the API
  but does not remove that.
- **Directly connected secondaries serve their API without user login** and trust the gateway and network
  (the gateway forwards user requests to them without a key). Keep them on a private network. Nodes that
  connect over the outbound link (the default for new nodes) do not have this problem: their API listens on
  loopback only and every request from the primary carries the node key.
- **Publishing the primary's port for secondaries** (`PRIMARY_NODE_BIND`) exposes its API on that
  address. Login still protects the user API, but bind it to a private LAN or VPN address only.
- **App environment variables are plain text.** They live in each service's `environment` in the compose file and
  in every stored version of it, so anyone with the database or a backup can read them. The UI masks values
  that look secret but does not encrypt them. There is no secret store yet (see `docs/design/backend-gaps.md`).
- **Database backups are not encrypted.** Backups written before a migration are mode 0600 and hold whatever the
  database holds: secrets stored with `ENCRYPT_SECRETS_AT_REST` stay encrypted, everything else is plain.
- **Nothing alerts on security events** such as a node joining, sessions being revoked or the compose policy
  blocking a deploy. They are in the audit log, but you have to look.
- **The tunnel provider credentials are not tested.** A revoked token looks healthy until an operation fails.
- **`golang-jwt/jwt` v3** is a legacy line pulled in by the auth library; the gateway pins the
  algorithm and claims, but migrating to v5 depends on `go-pkgz/auth`.
- **The frontend CSP is report-only.** The editor (CodeMirror) and the fonts are bundled, so nothing loads from
  a CDN any more and the policy allows only the site's own origin. Watch the browser console for violations
  on a real deployment, then rename the header to `Content-Security-Policy` in `web/public/serve.json`.
- **Where the UI is:** join tokens are made on the Nodes > Add node page, and Settings has Sign out everywhere
  (session revocation) and an Activity list (the audit log). All three are also API endpoints.
- **Requests through the gateway to `/api/nodes*` are authenticated by the gateway's key** after it
  has verified the login, so the audit record for those shows actor `gateway`, not the user.
- **GitHub Actions are pinned to major tags.** Dependabot (`.github/dependabot.yml`) is configured to
  propose SHA pins; accept those.
