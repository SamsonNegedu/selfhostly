# Verification

## Overview

"Done" here means checked in the real system, not only compiled. These are the checks that fit this
repo and how to report them honestly.

## Core Principles

**Backend.** Run `go build ./...` and the tests for every package you changed, for example
`go test ./internal/http/ ./internal/service/ ./internal/db/`. The full `go test ./...` can be slow, so
prefer the changed packages, and say if you did not run everything. Add a test for new behavior
next to the code (`newTestServer` in `internal/http` is the usual harness). Run `gofmt` on files you
edit.

**Frontend.** Run `cd web && npx tsc --noEmit -p . && npm run lint && npm run build`. Lint is clean and
is a gate (`oxlint --deny-warnings`), so do not add warnings. A disabled rule needs a comment saying why.

**UI changes are checked in a browser.** Look at the screen in dark and light, at 1440, 768 and 390
wide, and in its empty, loading and error states. Check contrast (4.5:1 for text), tap targets (44px on
phones), and that nothing scrolls sideways. The fixtures in `docs/dev-fixtures/` give you data for each
state (see `fixtures-and-docker-safety.md` for what not to click).

**Prove the behavior, not the render.** If you added a mutation, call it and see the result. If you
added an endpoint, `curl` it. If you added a filter, use it. A screenshot of a form does not show
that saving works.

**Report exactly what you ran.** List the commands and what each showed. Say which states you could
not check (Docker, credentials, a second node) and why. Do not write "verified" for something you
only type-checked.

**Do not hide a failure.** If a test fails, say so with the output. If it failed before your change,
show that with a run that isolates it.

## Driving Chrome for checks

Call `tabs_context_mcp` first, work in a tab you create, close it at the end, and never trigger native alert,
confirm or prompt dialogs.

- **Viewports:** `resize_window` cannot be trusted and a window cannot be narrower than about 500px. Embed the
  app in a same-origin iframe of the exact size instead. Replace the page with
  `document.open(); document.write('<body style="margin:0"><iframe id="d" src="/apps" style="width:390px;height:844px;border:0"></iframe></body>'); document.close()`.
  Media queries follow the iframe width, and `document.getElementById('d').contentDocument` gives DOM access.
  Measure tap targets with `getBoundingClientRect`, since nothing emulates touch.
- **Timers:** the automation tab is hidden, so `setTimeout` is throttled and a script with several waits can hit
  the 45 second limit while still running. Wait with a `MessageChannel` yield, and split long checks into
  several calls. Polling that pauses in a hidden tab (React Query intervals) needs a nudge or a
  `refetchIntervalInBackground` setting to observe.
- **Rendering:** CSS transitions do not advance until a frame is painted, so a color read right after load can be
  the starting value. Re-read after a few ticks before chasing a contrast bug that is not there.
- **Menus and tabs:** Radix menus open on a `pointerdown` event and tabs on `mousedown`. Dispatch those, then
  read the result.
- **Focus:** `element.focus()` does not match `:focus-visible` in the hidden tab, and focus behavior in a frame
  that is not the focused window is not the real thing. Say so when you report it.
- **Phones:** below 640px the root font size is 14px, so `rem` sizes shrink. Size tap targets in `px` or with an
  explicit `min-h-[44px]`.
- **Tool output:** the JavaScript tool refuses output that looks like a query string. Return
  `JSON.stringify(...)` of short values.
- **Console:** `read_console_messages` floods with React Router warnings. Filter with a narrow pattern.
- **Dev overlay:** the Agentation dev tool injects buttons outside `#root`. Scope measurements to `main`.
