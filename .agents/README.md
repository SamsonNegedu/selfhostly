# Shared AI Agent Guides

These documents are the single source of truth for agent policies and reusable workflows in this
repository. They are tool-neutral. `.claude/` holds only a lightweight discovery adapter that points
back here.

When changing a rule, edit the document here. Do not copy its body into an adapter.

## Rules (`rules/`)

| File | Covers |
| --- | --- |
| `ai-agent-guardrails.md` | Scoping, verifying claims, confirming costs, surfacing conflicts |
| `backend-conventions.md` | Go layering, domain errors, audit actions, migrations, events |
| `frontend-ui.md` | Tokens, primitives, states, accessibility for `web/` |
| `security-and-secrets.md` | Env files, secrets in responses and logs, compose validation |
| `fixtures-and-docker-safety.md` | The dev fixtures use the real Docker daemon |
| `local-processes-and-ports.md` | Never kill by port, never touch Docker or OrbStack |
| `git-working-tree.md` | Large uncommitted work: no stash, reset, clean or staging side effects |
| `verification.md` | What "done" means and how to report it |
| `code-comments.md` | What belongs in a comment |
| `no-magic-values.md` | Named constants |
| `no-em-dashes.md` | Punctuation in prose and UI copy |
| `concise-responses.md` | How to write replies |
