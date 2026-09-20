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

**Type sizes come from the scale, never `text-[13px]`.** Use `text-caption` (11px), `text-detail` (12px),
`text-compact` (13px), `text-body` (14px), `text-title` (15px) or `text-heading` (16px), defined in
`web/tailwind.config.js`. They are in px on purpose, so phones (which shrink the root size) keep the same
reading size. `npm run build` runs `npm run check:tokens`, which fails on an arbitrary text size, a palette
color, `bg-black` or `bg-white`, a hex color, or `h-screen` in `web/src`. Use `bg-scrim` for the dim behind a
dialog or sheet, and `h-dvh` or `min-h-dvh` instead of `h-screen` so a phone's browser bar does not cover the
bottom of the page.

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

**Data goes through `shared/services/api/` and the API client.** Add hooks to the file for their area (apps, tunnels, nodes and so on); `index.ts` re-exports them. Raw `fetch` is
for streaming only (logs, the event stream). Invalidate the right query keys after a mutation.

**Accessibility is part of done.**
- One `h1` per page and no skipped heading levels.
- Icon-only buttons have an `aria-label`.
- Every input has a label (use `Field`).
- Touch targets are at least 44px on phones.
- Dialogs are labelled and described, trap focus, and close on Escape.
- Do not put `overflow-auto` around inputs without padding for the focus ring.

**A bar pinned to the bottom of a page is an `ActionBar`.** Sticky offsets are measured inside the scroll
container's padding, and `main` already keeps the tab bar's height clear there, so a bar must not add the
tab bar's height again. `ActionBar` (the `sticky-action-bar` class in `globals.css`) gets this right. Do not
hand-write `sticky bottom-[...]`.

**Unsaved edits are guarded.** An editor with a draft renders `<UnsavedChangesGuard when={dirty} />`. It asks
before the person navigates away and uses the browser's own prompt when the tab is closed. Tabs of the app
page are in the address, so switching tabs counts as leaving.

**Every page has a title and one `h1`.** `MainLayout` sets the tab title from the breadcrumbs and moves focus
to the page after a page change. A page that renders outside the layout (the sign-in page, the server-down
screen) calls `useDocumentTitle` itself and passes `headingLevel={1}` to `ErrorState` or `EmptyState`.

**Overlays animate with the keyframes in `tailwind.config.js`.** Use `animate-overlay-in`,
`animate-dialog-in`, `animate-popover-in` and the sheet ones with their `-out` pairs. There is no
`tailwindcss-animate`, so `animate-in` and `fade-in-0` do nothing. Reduced motion is handled once in
`globals.css`, so components do not need `motion-reduce:` classes.

**Keep screens thin.** State goes in a hook (`hooks/`), rules that need no React go in `lib/` as plain
functions, and each section of a long screen is its own component. A function past about 150 lines is a
sign to split it. Pure `lib/` code gets a test in `web/tests/` (`npm test`, which runs on Node's built-in
test runner, so there is nothing to install).

**No card inside a card inside a card.** Sheet and dialog content is flat.

**Copy is plain language for the person running the server.** No internal vocabulary, no em
dashes (see `no-em-dashes.md`). Say what happened and what to do next.

**Dev-only routes are gated.** `/dev/ui` and `/dev/login` are behind `import.meta.env.DEV` and must
not reach the production bundle.

## Native controls

A native `select` is allowed where the list is very long and a native picker is the better
experience (the timezone list). Everywhere else use the Radix `Select`.

## Verify before saying done

Run `cd web && npx tsc --noEmit -p . && npm run lint && npm test && npm run build`. Then look at the screen in a browser in
dark and light, at 1440 and 390 wide, including the empty, loading and error states. See
`verification.md`.
