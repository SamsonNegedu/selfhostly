# Builds selfhostlyctl for the tests and sets CTL to it. Source this after ROOT is set.
CTL="${SELFHOSTLYCTL:-$ROOT/bin/selfhostlyctl}"
if [[ -z "${SELFHOSTLYCTL:-}" ]]; then
  (cd "$ROOT" && go build -o bin/selfhostlyctl ./cmd/selfhostlyctl) || { echo "could not build selfhostlyctl" >&2; exit 1; }
fi
