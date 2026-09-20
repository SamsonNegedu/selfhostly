# Code Comments

## Overview

What belongs in a source comment in this repo (Go and TypeScript). Comments describe what the code
is responsible for and the non-obvious why. They are not a changelog or a link to a ticket.

## Core Principles

**Explain the current responsibility and the reason a reader cannot derive from the code.** History
(who asked, when, under which ticket) belongs in the commit message and pull request.

**Never reference an issue or ticket ID in a comment.** Not Linear, Jira, or a bare GitHub `#123`.
If the reasoning matters, write the reasoning. If it is only "this is what issue X asked for", cut
the comment.

**A pointer to a version-controlled doc is fine** (`docs/design/backend-gaps.md`,
`docs/design/architecture.md`). Still summarize the why inline.

**Forward-looking intent is worth keeping.** "Temporary: delete once the API returns history"
tells a reader something the code cannot.

**Comment the surprising thing.** Ordering that matters, a workaround for a library or browser
quirk, a security reason, a timeout that looks arbitrary.

## Examples

**Bad:** a ticket ID as the whole justification.

```go
// SEC-142: added the check for the audit log
if !isMutating(method) { return }
```

**Good:** the constraint it encodes.

```go
// Reads are not audited: the log records changes, and a page that polls would bury them.
if !isMutating(method) { return }
```

**Bad:** describes the line.

```ts
// set loading to true
setLoading(true)
```

**Good:** explains why.

```ts
// Radix restores focus to the menu trigger after it closes. A dialog opened from the menu must
// keep focus, so restoring is skipped while one is open.
```
