# Cloudflare Access

Cloudflare Access puts a login in front of Selfhostly at Cloudflare's edge: a request that is not signed in never reaches
your server. It is the alternative to GitHub login (`selfhostlyctl setup` offers both). Choose it if you already use
Cloudflare Tunnels and want its identity providers, MFA and access logs. Free for up to 50 users.

The server does not just trust that Cloudflare was in front: Access adds a signed token (`Cf-Access-Jwt-Assertion`) to each
request, and the server verifies its signature, expiry, issuer and audience. A request that bypasses Cloudflare has no valid
token and is refused with `401`.

## Set it up

You need a Cloudflare account, a domain on Cloudflare, and a Tunnel that already points your hostname at Selfhostly.

1. **Enable Zero Trust** in the Cloudflare dashboard (choose a team name; free plan is fine).
2. **Create the application:** Zero Trust → Access → Applications → Add an application → Self-hosted. Application domain:
   your hostname, for example `selfhostly.example.com`. Session duration: 24 hours is a good default.
3. **Add a policy** (Action: Allow) that includes the people who may use it, by email, email domain, or identity-provider
   group. Add a login method under Settings → Authentication (One-time PIN by email needs no setup).
4. **Copy two values** you need next:
   - the **team domain**, `<team>.cloudflareaccess.com` (Settings → Custom pages, or the login page address);
   - the application's **Audience (AUD) tag**, on the application's Overview tab.
5. **Tell Selfhostly.** New install: `selfhostlyctl setup` and choose Cloudflare Access, or
   `selfhostlyctl bootstrap --auth cloudflare --cf-team <team>.cloudflareaccess.com --cf-aud <AUD tag>`. Existing install:
   put the two values in `.env` and apply them:
   ```env
   AUTH_ENABLED=false
   CF_ACCESS_TEAM_DOMAIN=<team>.cloudflareaccess.com
   CF_ACCESS_AUD=<AUD tag>
   CLOUDFLARE_API_TOKEN=...      # only for managing app tunnels
   CLOUDFLARE_ACCOUNT_ID=...
   ```
   ```bash
   selfhostlyctl upgrade --set CF_ACCESS_TEAM_DOMAIN=<team>.cloudflareaccess.com --set CF_ACCESS_AUD=<AUD tag>
   ```
   (`upgrade` refuses to write a setting your compose file never reads; see [operate.md](operate.md#update-the-compose-file).)
6. **Check:** `selfhostlyctl doctor` says `Cloudflare Access verification configured for <team domain>`. Open your hostname:
   Cloudflare asks you to sign in, then Selfhostly loads.

`AUTH_ENABLED=false` on its own is not enough. Without the two `CF_ACCESS_*` values the server has no way to verify anyone:
on a fresh install it refuses to start, and on an existing install (`warn` mode) it warns that the API is open.

## Adding a secondary machine

A secondary connects **out** to your public hostname (`/api/nodes/connect`). If Access protects that hostname, Cloudflare
answers the node with a login page and it can never connect. `selfhostlyctl join` detects this: it reports that the address
answered "but not like a Selfhostly primary".

Let those two paths through Access without a login. They authenticate themselves (a single-use join token or the node's key,
rate limited), and they are the only ones:

1. In the application, add a second **policy with Action: Bypass** for the paths `/api/nodes/connect` and `/api/health`
   (use a path-scoped application for each if your plan asks for that).
2. Run `selfhostlyctl join` again.

Everything else on the hostname stays behind the login. I have not tested this against a live Cloudflare account, so check
that a signed-out request to any other path still gets the Access login.

## Policy examples

| Who | Include rule |
|---|---|
| Only you | Emails: `you@example.com` |
| Several people | Emails: `a@example.com`, `b@example.com` |
| A whole domain | Emails ending in: `@example.com` |
| Extra safety | Emails plus **Require** IP ranges, or a country |

## Troubleshooting

| Symptom | Check |
|---|---|
| Access Denied | Your email is in the policy, the action is Allow, the session has not expired. Sign out and in. |
| Redirect loop | The application domain must match the tunnel hostname, and its DNS record must be proxied (orange cloud). Clear cookies. |
| `401 Authentication required` after signing in | `CF_ACCESS_AUD` or `CF_ACCESS_TEAM_DOMAIN` is wrong. The server log says `Cloudflare Access verification failed` with the reason. |
| Can't reach the login page | The domain is proxied, the tunnel is connected, DNS has propagated. |
| A machine will not join | See [Adding a secondary machine](#adding-a-secondary-machine). |

### A script or style file returns 404 (blank page)

**Cause:** the `path` of a tunnel ingress rule is a regular expression, and it matches anywhere in the URL, not only at
the start. A rule such as `/api/*` also matches `/assets/api-abc123.js` and sends it to the gateway, which answers 404. The
web build names its files so none starts with `api` or `auth`, but a rule that is too loose can still catch other paths.

**Fix:** anchor the rules that go to the gateway, and put the frontend rule last:

| Path regex | Service |
|---|---|
| `^/api(/\|$)` | gateway |
| `^/auth(/\|$)` | gateway |
| `^/avatar(/\|$)` | gateway |
| (no path) | frontend |

## Moving from GitHub login

Set up Access first and confirm you can sign in through it. Then set `AUTH_ENABLED=false`, add the two `CF_ACCESS_*` values,
and remove `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET`, in one `selfhostlyctl upgrade --set ...` so it restarts once.
GitHub login stays available if you prefer it: [github-allowlist.md](../security/github-allowlist.md).

More: [Cloudflare Access documentation](https://developers.cloudflare.com/cloudflare-one/policies/access/),
[security overview](../security/overview.md).
