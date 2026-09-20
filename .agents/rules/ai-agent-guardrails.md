# AI Agent Guardrails

## Overview

How an agent should scope, verify and confirm work that crosses parts of Selfhostly (Go backend, React
frontend, multi-node behavior, Docker, Cloudflare). These are the habits that prevent confident but
wrong work. Project-specific rules live in the other files in this directory.

## Core Principles

**Check the coupling before claiming reuse is safe.** "This should reuse the node client" is a
hypothesis, not a plan. Read what the code does end to end (what tables it writes, what network
calls it makes, which nodes it reaches) and report plainly if the coupling is tighter than assumed.

**Anything with a real external cost needs explicit confirmation first, however small.** That
includes paid API calls, sending messages, creating Cloudflare tunnels or DNS records with real
credentials, and pulling large container images. Name the action and the rough cost, then ask.

**When a request conflicts with a decision already recorded or shipped, say so before building.**
State the existing decision and where it lives (code, `docs/`, `docs/design/ui.md`), state
what the new ask implies, and ask which is intended. Do not silently reinterpret the old decision.

**Update the source-of-truth doc in the same pass as the code.** If a doc describes the old
behavior (`AGENTS.md`, `README.md`, `docs/`, `docs/design/`), amend it with the change.

**Verify claims against the working tree, not against a tracker or a summary.** A tracker row, a
TODO or an earlier message says what was intended. Read the code and run it before saying something
is built or fixed.

**A mock is a spec for the product, not for the engineering.** Labels in a design that explain
internal steps are not user-facing copy. Ask who the audience of each piece of content is.

**Search for an existing primitive before building one.** Look in `web/src/shared/components/ui/`
first, and for shared helpers in `web/src/shared/lib/` and `internal/`. A hand-rolled duplicate
drifts from the rest of the app.

**Test with the real volume and the real states.** A design showing three apps says nothing about
thirty. Check empty, loading, error, offline-node and failed-app states, not only the happy path.

**Say what you did not verify.** If a state needs Docker, credentials or a second node that you
could not run, list it as unverified in the same message that reports the work.

## Anti-patterns

- Stating "this reuses X" before reading X.
- Calling a Cloudflare or GitHub API with real credentials to "see if it works".
- Marking a screen done because it typechecks.
- Leaving `AGENTS.md` describing commands or structure that no longer exist.
- Copying an engineering label from a mock into the UI.
