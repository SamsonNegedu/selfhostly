# Picks the "old" version for the upgrade tests.
#
# The old version is normally the commit before this one (a pull request's base, or the tip before a push).
# After a force push or an amend, that tip is not part of this history any more, and it can be a commit that
# never built. Comparing with it would test nothing useful, so use the last commit both share instead.
#
# usage: OLD_REF="$(resolve_old_ref "$OLD_REF")"
resolve_old_ref() {
  local ref="$1" base
  if [[ -n "$ref" ]] && git cat-file -e "$ref^{commit}" 2>/dev/null && git merge-base --is-ancestor "$ref" HEAD 2>/dev/null; then
    echo "$ref"
    return
  fi
  if base="$(git merge-base "$ref" HEAD 2>/dev/null)" && [[ -n "$base" ]]; then
    echo "old version ${ref:-<none>} is not in this history (force push or amend); using the last shared commit ${base:0:12}" >&2
    echo "$base"
    return
  fi
  echo "old version ${ref:-<none>} is not available; using the previous commit" >&2
  git rev-parse HEAD~1
}

# The old version when the caller did not name one. With uncommitted changes to tracked files, HEAD is what
# the install runs today. With a clean tree the change under test is already committed, so HEAD is the NEW
# version and comparing it with itself would test nothing: use the commit before it.
#
# usage: OLD_REF="$(default_old_ref "${OLD_REF:-}")"
default_old_ref() {
  if [[ -n "$1" ]]; then
    echo "$1"
    return
  fi
  if [[ -n "$(git status --porcelain --untracked-files=no 2>/dev/null)" ]]; then
    echo "OLD_REF not set: the tree has uncommitted changes, so HEAD is the old version" >&2
    echo HEAD
    return
  fi
  echo "OLD_REF not set: the tree is clean, so HEAD is the new version; using the previous commit (set OLD_REF to the deployed version to override)" >&2
  git rev-parse HEAD~1
}
