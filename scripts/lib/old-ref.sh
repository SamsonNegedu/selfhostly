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
