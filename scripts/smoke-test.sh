#!/usr/bin/env bash
# End-to-end check of the deployment path: bootstrap a throwaway install, bring up the gateway and
# primary from docker-compose.prod.yml, assert the security behaviour, restart the primary and
# assert nothing was lost. It uses its own project name, container names, network and temp
# directories, so it cannot collide with a real install on the same host.
#
#   ./scripts/smoke-test.sh                 # build local images, run everything
#   SKIP_BUILD=1 ./scripts/smoke-test.sh    # reuse selfhostly-{backend,gateway}:smoke
#   SMOKE_PRIMARY_USER=0:0 ./scripts/smoke-test.sh   # Docker Desktop/OrbStack, where the socket group differs
#   KEEP=1 ./scripts/smoke-test.sh          # leave the stack up for inspection
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
source "$ROOT/scripts/lib/ctl.sh"
WORK="$(mktemp -d)"
PROJECT="selfhostly-smoke"
COMPOSE=(docker compose -p "$PROJECT" --env-file "$WORK/.env" -f "$ROOT/docker-compose.prod.yml" -f "$WORK/override.yml")
FAILURES=0

pass() { echo "  PASS  $*"; }
fail() { echo "  FAIL  $*"; FAILURES=$((FAILURES + 1)); }
check() { local name="$1" out; shift; if out="$("$@" 2>&1)"; then pass "$name"; else fail "$name"; printf '%s\n' "$out" | tail -40 | grep -v "PASS" | sed 's/^/        /'; fi; }

cleanup() {
  if [[ "${KEEP:-0}" != 1 ]]; then
    "${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
    rm -rf "$WORK"
  else
    echo "KEEP=1: stack left running (project $PROJECT), files in $WORK"
  fi
}
trap cleanup EXIT

if [[ "${SKIP_BUILD:-0}" != 1 ]]; then
  echo "building images"
  docker build -q -f "$ROOT/Dockerfile.backend" -t selfhostly-backend:smoke "$ROOT" >/dev/null
  docker build -q -f "$ROOT/Dockerfile.gateway" -t selfhostly-gateway:smoke "$ROOT" >/dev/null
fi

echo "bootstrapping a throwaway install in $WORK"
(
  cd "$WORK"
  DATA_DIR="$WORK/data" APPS_DIR="$WORK/apps" "$CTL" bootstrap \
    --auth github --domain smoke.example.test --github-user smoke-user >/dev/null
  {
    echo "DATA_DIR=$WORK/data"
    echo "APPS_DIR=$WORK/apps"
    echo "GITHUB_CLIENT_ID=smoke-id"
    echo "GITHUB_CLIENT_SECRET=smoke-secret"
    echo "PRIMARY_NODE_PORT=18191"
    echo "BACKEND_IMAGE=selfhostly-backend:smoke"
    echo "GATEWAY_IMAGE=selfhostly-gateway:smoke"
  } >> .env
  # bootstrap wrote the relative defaults first; the absolute paths above must win
  sed -i.bak -E '/^(DATA_DIR|APPS_DIR)=\.\//d' .env && rm -f .env.bak
)
mkdir -p "$WORK/data" "$WORK/apps"
# the container user must be able to write the bind mounts
chmod 777 "$WORK/data" "$WORK/apps"

cat > "$WORK/override.yml" <<YAML
services:
  primary:
    container_name: smoke-primary
    ${SMOKE_PRIMARY_USER:+user: "$SMOKE_PRIMARY_USER"}
  gateway:
    container_name: smoke-gateway
networks:
  selfhostly-network:
    name: smoke-network
YAML

echo "starting primary and gateway"
"${COMPOSE[@]}" up -d --wait primary gateway

http_status() { # service url [header...]
  local svc="$1" url="$2"; shift 2
  local args=(); for h in "$@"; do args+=(--header "$h"); done
  "${COMPOSE[@]}" exec -T "$svc" wget -S -O /dev/null ${args[@]+"${args[@]}"} "$url" 2>&1 | grep -oE 'HTTP/[0-9.]+ [0-9]+' | tail -n1 | awk '{print $2}' || true
}

echo "assertions"
[[ "$(http_status gateway http://localhost:8080/api/health)" == 200 ]] && pass "gateway health" || fail "gateway health"
[[ "$(http_status primary http://localhost:8082/api/health)" == 200 ]] && pass "primary health" || fail "primary health"
[[ "$(http_status gateway http://localhost:8080/api/apps)" == 401 ]] && pass "API requires a login through the gateway" || fail "API requires a login through the gateway"
[[ "$(http_status gateway http://localhost:8080/api/apps 'X-Gateway-API-Key: guess' 'X-Node-ID: x' 'X-Node-API-Key: y')" == 401 ]] && pass "forged credential headers do not bypass the login" || fail "forged credential headers do not bypass the login"
[[ "$(http_status primary http://localhost:8082/api/nodes 'X-Node-ID: x' 'X-Node-API-Key: wrong')" == 401 ]] && pass "wrong node key rejected by the backend" || fail "wrong node key rejected by the backend"
check "root filesystem of the backend is read-only" bash -c "! ${COMPOSE[*]} exec -T primary sh -c 'touch /app/write-test'"
check "backend runs without extra capabilities" bash -c "${COMPOSE[*]} exec -T primary sh -c 'grep -q \"^CapEff:.*0000000000000000\" /proc/self/status'"
check "doctor passes" "${COMPOSE[@]}" exec -T primary ./selfhostly doctor --offline

BEFORE="$("${COMPOSE[@]}" exec -T primary ./selfhostly doctor --offline 2>&1 | grep -E 'node\(s\)' || true)"

echo "restarting the primary"
"${COMPOSE[@]}" restart primary >/dev/null
"${COMPOSE[@]}" up -d --wait primary >/dev/null
[[ "$(http_status primary http://localhost:8082/api/health)" == 200 ]] && pass "primary healthy after restart" || fail "primary healthy after restart"
AFTER="$("${COMPOSE[@]}" exec -T primary ./selfhostly doctor --offline 2>&1 | grep -E 'node\(s\)' || true)"
[[ -n "$BEFORE" && "$BEFORE" == "$AFTER" ]] && pass "node and app records survive a restart" || fail "node and app records survive a restart ($BEFORE vs $AFTER)"
check "no refusal to start in the logs" bash -c "! ${COMPOSE[*]} logs primary 2>&1 | grep -q 'refusing to start'"

echo
if [[ "$FAILURES" -eq 0 ]]; then echo "smoke test passed"; else echo "$FAILURES check(s) failed"; "${COMPOSE[@]}" logs --tail 40 primary gateway || true; exit 1; fi
