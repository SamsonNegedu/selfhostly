# Deploy hooks

A deploy hook closes the loop between an app's own repository and Selfhostly, without Selfhostly ever knowing that
repository, or any Git host, exists. A hook is a per-app secret token; POSTing it to `/api/apps/:id/deploy-trigger`
runs the exact same pull-and-restart that clicking Update in the dashboard runs. Wire it into a CI pipeline's last
step (a GitHub Actions workflow, a GitLab job, a plain cron with curl, anything that can make an HTTPS request) and a
push to `main` reaches the running app with nobody clicking anything.

This is deliberately not the same thing as [UI-driven updates](ui-updates.md), which update Selfhostly itself from a
signed release manifest. A deploy hook updates a user's own app, from whatever image its compose file already points
at.

## Why no GitHub integration

Earlier designs considered Selfhostly linking a GitHub repo, watching it, or receiving GitHub's own webhooks.
None of that is here. The trigger is a bearer token a pipeline presents; what the token authorizes is fixed
(`docker compose pull` + `up -d` for the one app it belongs to) and never depends on anything the caller says about
itself. This keeps the trust surface identical to the dashboard's own Update button: a hook grants no capability
a signed-in user did not already have, and Selfhostly does not need OAuth scopes, a GitHub App, or a registry API
client to make it work.

The compose file's image reference does not change when a hook fires: `docker compose pull` only picks up a newer
digest under the tag the compose file already names. This only closes the loop for apps whose image is built to a
mutable tag CI keeps pushing to (`:latest`, `:main`, ...). A pinned, immutable tag needs its compose file edited by
hand, the same as it always did; deploy hooks do not rewrite compose content.

## Data model

`app_deploy_hooks` (`internal/db/deploy_hooks.go`): `id`, `app_id`, `name`, `source_kind`, `token_hash`, `created_at`,
`last_used_at`, `last_used_ip`. Several hooks can exist per app, each independently named, triggered and revoked,
the same as Vercel or Netlify's deploy hooks: one for a `main` branch pipeline and a separate one for a manual
re-trigger do not have to share a token or a revoke button.

Only the hash is stored (`sha256`, the same scheme as join tokens in `internal/db/secure.go`). The plaintext token is
returned exactly once, in the response to creating the hook, and cannot be recovered afterwards; losing it means
rotating (revoke and make a new one).

## Source kinds: the extensible part

Every hook has a `source_kind`, today always `generic`: nothing beyond the token match is required. The verifier
registry in `internal/service/deploy_hook_service.go` is where a future kind that needs more would live, for
example one that also checks a caller's identity:

```go
var deploySourceVerifiers = map[string]func(ctx context.Context, hook *db.DeployHook, callerIP string) error{
    // "github_actions": verifyGitHubActionsOIDC,
}
```

A kind with no entry gets the shared check only. Adding a kind is a new map entry and a new file, not a schema
change or a new route. A kind whose entire transport differs from "present a bearer token in a header" (an inbound
webhook with a signed body, say) gets its own route rather than being folded into this one, since dispatching
unrelated request shapes through a single handler is where "generic" stops being simple.

## Security properties

- **Same blast radius as the manual Update button.** The trigger route calls the identical
  `AppService.UpdateAppContainersAsync` the dashboard's Update button calls. A hook cannot change a compose file,
  read a secret, or act on any app but the one it was made for.
- **Scoped per app.** `ConsumeDeployHookToken` matches a token against one `app_id`; a token made for one app is
  rejected outright against another, even if it were somehow guessed.
- **Rate limited per app**, not per caller IP - a CI runner's address is not a stable identity to key on, but an
  app's own budget (`constants.DeployTriggerRateLimitAttempts` per `DeployTriggerRateLimitWindow`) still bounds how
  hard its hooks can be brute-forced or misfired, from any source.
- **Independently revocable.** Revoking one hook never affects another hook on the same app.
- **Audited and attributed.** Every trigger, successful or not, is recorded (`app.deploy_trigger`); the actor reads
  `deploy-hook:<name>` rather than a user id, since there is no session on this request, and `last_used_at` /
  `last_used_ip` are shown per hook in the dashboard so a hook firing from an unexpected place is visible without
  reading the audit log.
- **No new inbound trust.** The route is mounted outside the session/node auth group (`internal/http/routes.go`,
  alongside node registration) because it carries its own credential instead of a session, not because it skips
  authentication - the token is checked on every request, same as a node's API key is checked on every node route.
- **Managing hooks allows node auth, deliberately.** Creating, listing and revoking hooks is reachable with a node's
  own credentials, the same as `start`/`stop`/`update`/`delete` on that node's apps already are. This is not a gap:
  a node's credentials already give it full control over apps hosted on it, so this adds no capability beyond what
  it already has, and denying node auth here would break the only way it can be reached for an app on a linked
  (tunnel) secondary - `forwardToLinkedNode` re-signs a forwarded request with the target's node credentials, since
  the secondary has no way to see the original session.

## API

| Route | Auth | Purpose |
|---|---|---|
| `POST /api/apps/:id/deploy-hooks` | session | Create a hook, named. Returns the plaintext token once. |
| `GET /api/apps/:id/deploy-hooks` | session | List an app's hooks. Never returns a token. |
| `DELETE /api/apps/:id/deploy-hooks/:hookId` | session | Revoke one hook immediately. |
| `POST /api/apps/:id/deploy-trigger` | hook token (`Authorization: Bearer <token>`) | Trigger the same job the Update button starts. 202 with `job_id`, or the id of an already-active job if one is running. |

## Example: GitHub Actions

The last step of a build-and-push job, after it pushes the app's image to its `:latest` tag:

```yaml
- name: Trigger Selfhostly deploy hook
  env:
    SELFHOSTLY_URL: ${{ vars.SELFHOSTLY_URL }}
    SELFHOSTLY_APP_ID: ${{ vars.SELFHOSTLY_APP_ID }}
    SELFHOSTLY_NODE_ID: ${{ vars.SELFHOSTLY_NODE_ID }}
    SELFHOSTLY_DEPLOY_TOKEN: ${{ secrets.SELFHOSTLY_DEPLOY_TOKEN }}
  run: |
    if [ -z "$SELFHOSTLY_URL" ] || [ -z "$SELFHOSTLY_APP_ID" ] || [ -z "$SELFHOSTLY_NODE_ID" ] || [ -z "$SELFHOSTLY_DEPLOY_TOKEN" ]; then
      echo "Missing Selfhostly configuration - check the repository variables and secret are set." >&2
      exit 1
    fi
    curl --fail --silent --show-error \
      --retry 3 --retry-delay 5 \
      -X POST "${SELFHOSTLY_URL}/api/apps/${SELFHOSTLY_APP_ID}/deploy-trigger?node_id=${SELFHOSTLY_NODE_ID}" \
      -H "Authorization: Bearer ${SELFHOSTLY_DEPLOY_TOKEN}"
```

`SELFHOSTLY_URL`, `SELFHOSTLY_APP_ID` and `SELFHOSTLY_NODE_ID` are plain repo variables, deliberately not baked into
the workflow file: the same step then needs no edit if any of them ever changes (a new domain, a new tunnel, moving
the app to a different node), and nothing instance-specific ends up committed to the repo. `SELFHOSTLY_DEPLOY_TOKEN`
is the one-time token from the app's Deploy tab, stored as a repo secret. Pulling all four into `env:` once, then
checking them before `curl` runs, turns a variable or secret nobody set into a clear failure message instead of a
curl error against a malformed URL. `--fail` turns a bad or revoked token into a failed workflow step instead of a
silent no-op; `--retry` covers a briefly-unreachable instance (a home-hosted primary restarting for its own update,
a network blip), not anything token-related.

The dashboard's Deploy tab shows this exact step (unfilled) plus the three variable values ready to copy, and warns
when the address it was opened through is a loopback or private-network one (`localhost`, `192.168.x.x`, ...): that
address is real on the machine running the browser, but GitHub's hosted runners cannot reach it, only a self-hosted
runner on the same network could. Copying such an address into `SELFHOSTLY_URL` produces a workflow that fails
every time, not one that fails to update.

## Multi-node routing

`node_id` is required for the same reason every other by-id route in this API requires it
(`internal/gateway/router.go`): the gateway that fronts a multi-node install cannot know which node holds a given
app id on its own, so it routes purely on the query parameter, straight to that node's own address when it is
directly reachable or to the primary when it is a linked (tunnel) node - identical to how the dashboard's own calls
are routed. `/api/apps/:id/deploy-trigger` is also added to the gateway's auth-skip list
(`internal/gateway/auth.go`): a deploy hook's bearer token is not a JWT, and the gateway would otherwise try to
validate it as one and reject it before the backend ever saw it.

For an app on a linked node, the primary does a second hop: `forwardDeployTriggerToLinkedNode`
(`internal/http/node_link.go`) relays the request down that node's connection, the same job `forwardToLinkedNode`
does for every session-authed by-id route - with one deliberate difference. `forwardToLinkedNode` strips
`Authorization` before forwarding, because for a session-authed request that header holds the *user's* credential,
which a linked node has no business seeing; the primary re-signs the forwarded request with the node's own
credentials instead. The deploy-trigger forward keeps `Authorization` exactly as it came in, because there it holds
the *deploy hook's* credential, the only one the request has, and it must reach the linked node's own token check
unchanged - node credentials are not involved on this route at all.

## Not covered

- Cryptographic image provenance (a cosign/sigstore signature, a SLSA attestation) is not checked before a pull.
  Trust is the registry plus the digest `docker compose pull` resolves, the same level of trust the platform's own
  images get in [UI-driven updates](ui-updates.md#trust). Verifying a signature would be a `source_kind` addition,
  not a redesign.
- A hook cannot be scoped to a caller IP or CIDR range. The schema has room to add this later (a nullable column on
  `app_deploy_hooks`, checked in a verifier) but nothing does today.
- Zero-touch app creation from a repository (declaring a brand-new app from a build with no app to point a hook at
  yet) is a separate, larger feature and is not part of this one.

If the instance is behind [Cloudflare Access](../operations/cloudflare-zero-trust.md) rather than (or alongside) GitHub
login, the exemption above only covers the gateway's own check - Access runs earlier, at Cloudflare's edge, and gates
every path by default. It needs its own Bypass policy for the trigger path; see
[Deploy hooks](../operations/cloudflare-zero-trust.md#deploy-hooks) in that doc.
