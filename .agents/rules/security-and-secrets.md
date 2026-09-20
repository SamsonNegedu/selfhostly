# Security and Secrets

## Overview

This app holds credentials that control other machines and a Cloudflare account, and it can run
containers on the host. Treat secrets and the compose security checks as part of the product.

## Core Principles

**Do not read, print or paste env files.** `.env`, `.env.primary`, `.env.secondary`,
`.env.gateway` and any other `.env*` hold real tokens, OAuth secrets and keys. They are gitignored.
Do not `cat`, `grep` values from, or echo them. Read `env.example` for names and defaults. If a
secret is shown by accident, say so instead of repeating it.

**Never write a secret into code, docs, logs, fixtures or the audit log.** Use placeholders in
docs. Logs and audit entries name actions and targets, not values.

**API responses never contain secrets.** Node responses omit the API key (`toNodeResponse`), and
provider tokens are returned masked. Join tokens are shown once and only their hash is stored.
Node API keys are sealed at rest by `internal/secrets`.

**Do not weaken authentication or origin checks to make something easier to test.** `AUTH_ENABLED`,
`GATEWAY_API_KEY`, the origin guard, CORS and the security modes (`warn`, `enforce`) are deliberate.
A test that needs auth off uses the test server helpers, not a config change.

**Compose validation is a security boundary.** `internal/validation` stops a compose file from
mounting the host, using privileged mode and similar. Do not bypass or loosen it, and do not add a
template or example that would fail it.

**Node endpoints are validated.** Registration rejects addresses the network policy forbids
(`internal/netguard`, `node.ValidateEndpoint`). Do not add a path that makes outbound calls to a
user-supplied address without going through it.

**Destructive and outward-facing actions need confirmation in the UI.** Deleting an app, removing a
node and signing everyone out go through a confirmation, with the name typed for the destructive
ones.
