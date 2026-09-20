# Git and the Working Tree

## Overview

This repository routinely carries a large amount of uncommitted work (a redesign touching most of
`web/`, plus backend changes). That work exists nowhere else. A careless git command can destroy it.

## Core Principles

**Do not run commands that discard or hide uncommitted work.** That means `git stash`,
`git reset --hard`, `git checkout -- <path>` or `git checkout .`, `git restore` on modified files,
`git clean`, `git rebase` and `git worktree` moves, unless the user asked for that exact command.
If you need the old version of a file to compare, use `git show HEAD:<path>` or `git diff`.

**Do not commit, push, amend or create branches unless asked.** Report the changes and let the user
decide. When you do commit, follow the attribution and message rules in the session instructions.

**Leave the staging area alone.** Do not `git add` or `git rm --cached` as a side effect. When a
tracked file must go, delete it from the working tree and say so. The user stages what they want.

**Do not touch files you did not change for this task.** Many files show as modified. Leave
unrelated changes exactly as they are.

**Check before deleting.** Look at what a file is (`git ls-files`, `git log -1 -- <path>`) before
removing it, and list what you removed.

**Generated and local directories are not yours to clean.** `tmp/`, `tmp-gateway/`, `bin/`, `data/`,
`apps/` and `web/dist/` may hold the developer's data or builds. `make clean` removes some of them,
so do not run it unasked.
