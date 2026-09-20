# Frontend UI (web/)

## Overview

Rules for the React, TypeScript and Tailwind app in `web/`. The design tokens and primitives are
the design system. Work that bypasses them shows up as inconsistent screens, broken dark mode and
inaccessible controls.

## Core Principles

**Use the primitives in `web/src/shared/components/ui/`.** They are Radix based (Button, Field,
Input, Select, Tabs, Table, Sheet, Dialog, ConfirmationDialog, StatusPill, ProgressBar, Terminal,
EmptyState, ErrorState, Skeleton, YamlEditor and so on). Search there and in `web/src/shared/lib/`
before writing a new one. `/dev/ui` shows all of them in both themes.

**Colors come from tokens, never raw values.** Use `bg-card`, `text-muted-foreground`,
`border-border`, and the status tokens (`bg-status-ok`, `bg-status-warn-bg text-status-warn-fg`,
and so on). Do not use hex codes or palette classes such as `green-500` or `red-600` in features.
The tokens are defined in `web/src/styles/globals.css` and mapped in `web/tailwind.config.js`.

**The accent is indigo (`--primary`).** Use it for primary actions, the active tab, selected cards and the
focus ring, and never for state. Neutral grey is for everything else.

**Health is always color coded, and never by color alone.** Healthy, warning, critical and idle use
the status tokens and also say it in words (a pill, a label, an `sr-only` prefix). Thresholds for
CPU, memory and disk are in `shared/lib/thresholds.ts`. Do not invent a second set.

**Every screen has loading, empty and error states.** Use Skeleton, EmptyState and ErrorState. A
query that has no data yet must not render as "nothing found".

**Errors go through `describeError` and `fieldError`.** Never show a raw response body. The API
returns `code` and `field` for known failures (`ApiRequestError`); show field errors next to the
input.

**Data goes through `shared/services/api.ts` and the API client.** Add hooks there. Raw `fetch` is
for streaming only (logs, the event stream). Invalidate the right query keys after a mutation.

**Accessibility is part of done.**
- One `h1` per page and no skipped heading levels.
- Icon-only buttons have an `aria-label`.
- Every input has a label (use `Field`).
- Touch targets are at least 44px on phones.
- Dialogs are labelled and described, trap focus, and close on Escape.
- Do not put `overflow-auto` around inputs without padding for the focus ring.

**No card inside a card inside a card.** Sheet and dialog content is flat.

**Copy is plain language for the person running the server.** No internal vocabulary, no em
dashes (see `no-em-dashes.md`). Say what happened and what to do next.

**Dev-only routes are gated.** `/dev/ui` and `/dev/login` are behind `import.meta.env.DEV` and must
not reach the production bundle.

## Native controls

A native `select` is allowed where the list is very long and a native picker is the better
experience (the timezone list). Everywhere else use the Radix `Select`.

## Verify before saying done

Run `cd web && npx tsc --noEmit -p . && npm run build`. Then look at the screen in a browser in
dark and light, at 1440 and 390 wide, including the empty, loading and error states. See
`verification.md`.
