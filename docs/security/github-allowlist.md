# GitHub allow-list

With GitHub login (`AUTH_ENABLED=true`), only the GitHub accounts you list can sign in. Anyone else can complete GitHub's
sign-in but is refused by Selfhostly. The alternative is [Cloudflare Access](../operations/cloudflare-zero-trust.md).

## Configure

```env
AUTH_ENABLED=true
GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...
GITHUB_ALLOWED_USERS=alice,bob
AUTH_BASE_URL=https://selfhostly.example.com
AUTH_SECURE_COOKIE=true
```

Create the OAuth app at https://github.com/settings/developers with callback URL `https://<your host>/auth/github/callback`.
`selfhostlyctl setup` asks for all of this and writes it for you.

`GITHUB_ALLOWED_USERS` is a comma-separated list of GitHub **logins** (the name in `github.com/<login>`). Spaces around commas
are ignored; do not use `@`, quotes or spaces inside a login. Change the list with
`selfhostlyctl upgrade --set GITHUB_ALLOWED_USERS=alice,bob` (it restarts the server). A person you remove is refused on their
next request, because the list is checked on every request, not only at sign-in.

## What is matched

Each login is turned into the stable user ID that GitHub login produces, and that ID is what is compared:

- casing does not matter (`Alice` and `alice` are the same person);
- a person cannot get in by setting their profile display name to an allowed login;
- if someone renames their GitHub account, the old entry stops matching: put the new login in the list;
- an entry may also be a ready-made `github_<hash>` ID.

At startup the server also looks up each login on GitHub to add its canonical casing. If it cannot reach GitHub (offline, or
rate limited) it says so and matches the text you typed; `selfhostlyctl doctor` shows this as a note.

## Safe by default

An empty `GITHUB_ALLOWED_USERS` rejects every login. In `enforce` mode the server refuses to start in that state, and
`selfhostlyctl doctor` reports it. Everyone on the list has full control of everything: add only people you trust completely.

## Something is wrong

| Symptom | Cause and fix |
|---|---|
| The sign-in page says "This GitHub account is not allowed" | The account is not on the list, or is listed with a typo. Check the login in the URL of the person's GitHub profile, and that the list has no `@`, quotes or inner spaces. |
| Everyone is rejected | The list is empty or `AUTH_ENABLED` is off. Run `selfhostlyctl doctor`. |
| You need to allow someone and only have the log | The server logs `Unauthorized user attempted access` with the person's `user_id` (`github_…`). Add that exact ID to `GITHUB_ALLOWED_USERS` to allow that account. |
| GitHub says the redirect URI is not valid | The OAuth app's callback URL must be exactly `https://<your host>/auth/github/callback`, and `AUTH_BASE_URL` must be the same host. |
| Sign-in loops or the cookie is not kept | `AUTH_SECURE_COOKIE=true` needs an `https` address, and `PUBLIC_HOSTS` on the gateway must list the hostname you type. |
| You want everyone signed out | `POST /api/security/revoke-sessions` ends every session at once ([operate.md](../operations/operate.md#rotate-secrets-separate-from-any-upgrade)). |
