#!/usr/bin/env bash
# End-to-end test of adding a machine to a cluster.
#
# Scenario A (the default way): the secondary sits on a SEPARATE Docker network that cannot reach the
# primary at all. Its only path is the gateway's published port, exactly like a machine at another
# site reaching the primary through its public address. It joins with one command, publishes no
# port, and the primary still controls it, through the connection the secondary opened.
#
# Scenario B (compatibility): the older direct mode, where the primary calls the secondary.
#
# The test stack runs with login disabled so the test can call the API. To keep that safe, neither the
# test primary nor the test nodes get the host's Docker socket, so nothing that reaches them can start a
# container on the machine running the test. Published ports are loopback-only and unusual.
#
# Everything runs on private test networks; nothing touches a real install. The CLI runs from a
# scratch copy of the scripts so it cannot edit this checkout's .env files.
#
#   ./scripts/test-join.sh                 build the images, run everything
#   SKIP_BUILD=1 ./scripts/test-join.sh    reuse selfhostly-join-{backend,gateway}:test
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
source "$ROOT/scripts/lib/ctl.sh"
WORK="$(mktemp -d)"
BACKEND="selfhostly-join-backend:test"
GATEWAY="selfhostly-join-gateway:test"
NET_A="jn-net-a"   # primary and gateway
NET_B="jn-net-b"   # the isolated secondary
FAILURES=0
pass() { echo "  PASS  $*"; }
fail() { echo "  FAIL  $*"; FAILURES=$((FAILURES + 1)); }

PRIMARY=(docker compose -p selfhostly-jn --env-file "$WORK/primary/.env" -f "$ROOT/docker-compose.prod.yml" -f "$WORK/primary-override.yml")
NODE_DOWN() { ( cd "$WORK/cli" 2>/dev/null && docker compose -p selfhostly-node --env-file .env.node -f docker-compose.secondary.yml down -v >/dev/null 2>&1 ) || true; }
cleanup() {
  NODE_DOWN
  "${PRIMARY[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
  docker network rm "$NET_A" "$NET_B" >/dev/null 2>&1 || true
  rm -rf "$WORK"
}
trap cleanup EXIT

if [[ "${SKIP_BUILD:-0}" != 1 ]]; then
  echo "building the images"
  docker build -q -f "$ROOT/Dockerfile.backend" -t "$BACKEND" "$ROOT" >/dev/null
  docker build -q -f "$ROOT/Dockerfile.gateway" -t "$GATEWAY" "$ROOT" >/dev/null
fi

mkdir -p "$WORK/primary" "$WORK/cli" "$WORK/node-data" "$WORK/node-apps"
cp -r "$ROOT/scripts" "$ROOT/docker-compose.prod.yml" "$ROOT/docker-compose.secondary.yml" "$ROOT/docker-compose.secondary-direct.yml" "$WORK/cli/"

echo "starting a primary and a gateway (the gateway's port is the only way in)"
(
  cd "$WORK/primary"
  DATA_DIR="$WORK/primary/data" APPS_DIR="$WORK/primary/apps" "$CTL" bootstrap --quiet --auth none >/dev/null
  sed -i.bak -E '/^(DATA_DIR|APPS_DIR)=\.\//d' .env && rm -f .env.bak
  { echo "DATA_DIR=$WORK/primary/data"; echo "APPS_DIR=$WORK/primary/apps"
    echo "BACKEND_IMAGE=$BACKEND"; echo "GATEWAY_IMAGE=$GATEWAY"; echo "PRIMARY_NODE_PORT=18194"; } >> .env
)
mkdir -p "$WORK/primary/data" "$WORK/primary/apps"; chmod 777 "$WORK/primary/data" "$WORK/primary/apps" "$WORK/node-data" "$WORK/node-apps"
cat > "$WORK/primary-override.yml" <<YAML
services:
  primary:
    container_name: jn-primary
    user: "0:0"
    volumes: !override
      - $WORK/primary/data:/app/data
      - $WORK/primary/apps:/app/apps
  gateway:
    container_name: jn-gateway
    ports:
      - "127.0.0.1:18080:8080"
networks:
  selfhostly-network:
    name: $NET_A
YAML
"${PRIMARY[@]}" up -d --wait primary gateway >/dev/null

api()  { docker exec jn-primary wget -qO- "$1" 2>/dev/null; }
gwapi(){ docker exec jn-gateway wget -qO- "$1" 2>/dev/null; }
code() { # container url [header...]  -> HTTP status
  local c="$1" url="$2"; shift 2
  local args=(); for h in "$@"; do args+=(--header "$h"); done
  docker exec "$c" wget -S -O /dev/null ${args[@]+"${args[@]}"} "$url" 2>&1 | grep -oE 'HTTP/[0-9.]+ [0-9]+' | tail -n1 | awk '{print $2}' || true
}
node_field() { # name field -> value from the primary's node list
  api http://localhost:8082/api/nodes | python3 -c "
import sys,json
for n in json.load(sys.stdin):
    if n['name']=='$1': print(n['$2']); break" 2>/dev/null || true
}
wait_node_status() { # name status seconds
  local s=0
  while [[ $s -lt "$3" ]]; do
    [[ "$(node_field "$1" status)" == "$2" ]] && return 0
    sleep 2; s=$((s + 2))
  done
  return 1
}

[[ "$(code jn-primary http://localhost:8082/api/health)" == 200 && "$(code jn-gateway http://localhost:8080/api/health)" == 200 ]] && pass "primary and gateway are up" || fail "primary and gateway are up"

echo
echo "A. a secondary that can only reach the gateway's public port"
cd "$WORK/cli"
TOKEN_OUT="$(SFH_PRIMARY_CONTAINER=jn-primary "$CTL" join-token --primary-url http://host.docker.internal:18080 2>&1)" \
  || { echo "  FAIL  join-token exited non-zero:"; printf '%s\n' "$TOKEN_OUT" | sed 's/^/        /'; exit 1; }
TOKEN="$(printf '%s\n' "$TOKEN_OUT" | grep -o 'sfj_[A-Za-z0-9_-]*' | head -n1)"
[[ -n "$TOKEN" ]] && pass "join-token printed a token" || fail "no token: $TOKEN_OUT"
[[ "$TOKEN_OUT" == *"join --primary-url http://host.docker.internal:18080 --token $TOKEN"* && "$TOKEN_OUT" != *"--direct"* ]] && pass "and the one command to run on the new machine (no port to open)" || fail "join command: $TOKEN_OUT"

cat > "$WORK/node-override.yml" <<YAML
services:
  node:
    extra_hosts:
      - "host.docker.internal:host-gateway"
    volumes: !override
      - $WORK/node-data:/app/data
      - $WORK/node-apps:/app/apps
YAML
{ echo "SELFHOSTLY_NETWORK=$NET_B"; echo "BACKEND_IMAGE=$BACKEND"; echo "DATA_DIR=$WORK/node-data"; echo "APPS_DIR=$WORK/node-apps"; echo "APP_UID=0"; } > .env.node
export SFH_NODE_EXTRA_COMPOSE="$WORK/node-override.yml"
if "$CTL" join --non-interactive --yes --skip-host-checks --skip-reach-check \
     --primary-url http://host.docker.internal:18080 --token "$TOKEN" --name sec1 > "$WORK/join.log" 2>&1; then
  pass "join exits 0"
else fail "join exits 0"; tail -15 "$WORK/join.log"; docker logs selfhostly-node 2>&1 | tail -5 | cut -c1-240 | sed 's/^/        /'; fi
grep -qE "✓.* connected to the primary$" "$WORK/join.log" && pass "the CLI confirmed the connection" || fail "connection not confirmed"

[[ -z "$(docker port selfhostly-node 2>/dev/null)" ]] && pass "the secondary publishes no port" || fail "the secondary published: $(docker port selfhostly-node)"
NODE_NETS="$(docker inspect --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}' selfhostly-node)"
[[ "$NODE_NETS" == *"$NET_B"* && "$NODE_NETS" != *"$NET_A"* ]] && pass "and it is on a different network from the primary (it cannot be reached, only connect out)" || fail "networks: $NODE_NETS"
[[ "$(docker exec selfhostly-node wget -qO- http://127.0.0.1:8082/api/health 2>/dev/null)" == *healthy* ]] && pass "its API answers on loopback" || fail "loopback health"

wait_node_status sec1 online 60 && pass "the primary shows the node online" || fail "node not online: $(node_field sec1 status)"
NID="$(node_field sec1 id)"
[[ "$(node_field sec1 api_endpoint)" == "tunnel://$NID" ]] && pass "recorded as a linked node (tunnel://<id>)" || fail "endpoint: $(node_field sec1 api_endpoint)"

INFO="$(api "http://localhost:8082/api/node/info?node_id=$NID")"
[[ "$INFO" == *'"name":"sec1"'* ]] && pass "the primary controls it: a request for that node is answered by the secondary" || fail "forward via primary: $INFO"
INFO="$(gwapi "http://localhost:8080/api/node/info?node_id=$NID")"
[[ "$INFO" == *'"name":"sec1"'* ]] && pass "and the same through the gateway" || fail "forward via gateway: $INFO"
[[ "$(code jn-gateway "http://localhost:8080/api/apps/no-such-app?node_id=$NID")" == 404 ]] && pass "a by-id request is routed to the secondary and answered there (404 from its own database)" || fail "by-id routing: $(code jn-gateway "http://localhost:8080/api/apps/no-such-app?node_id=$NID")"

echo "the link heals by itself"
docker stop selfhostly-node >/dev/null
wait_node_status sec1 offline 30 && pass "stopping the node marks it offline within seconds (no polling delay)" || fail "still $(node_field sec1 status)"
[[ "$(code jn-primary "http://localhost:8082/api/apps/x?node_id=$NID")" == 503 ]] && pass "a request for it now gets 503, not a hang" || fail "expected 503"
docker start selfhostly-node >/dev/null
wait_node_status sec1 online 60 && pass "starting it again reconnects it with no action from anyone" || fail "did not reconnect"
"${PRIMARY[@]}" restart primary >/dev/null
sleep 3
wait_node_status sec1 online 120 && pass "restarting the primary: the secondary reconnects by itself" || fail "did not reconnect after a primary restart"

echo "credentials"
CODE="$(code jn-gateway http://localhost:8080/api/nodes/connect "X-Selfhostly-Node-Id: intruder1" "X-Selfhostly-Node-Key: 0123456789abcdef0123" "X-Selfhostly-Join-Token: $TOKEN")"
[[ "$CODE" == 401 ]] && pass "a spent token is refused, and the refusal comes from the primary through the gateway" || fail "spent token got: $CODE"
[[ "$(code jn-gateway http://localhost:8080/api/nodes/connect "X-Selfhostly-Node-Id: sec1" "X-Selfhostly-Node-Key: wrong-key-wrong-key-wrong")" == 401 ]] && pass "a known node with the wrong key is refused" || fail "wrong key accepted"
NODE_LOG="$(docker logs selfhostly-node 2>&1)"
[[ "$NODE_LOG" != *"the primary refused"* ]] && pass "the secondary's own connection was never refused" || fail "refusals in the node log"

echo "a wrong token is diagnosed"
# link attempts are rate limited per client address; let the earlier connections leave the window so
# a 429 cannot mask the token diagnosis
sleep 65
NODE_DOWN
rm -rf "$WORK/node-data" "$WORK/node-apps"; mkdir -p "$WORK/node-data" "$WORK/node-apps"; chmod 777 "$WORK/node-data" "$WORK/node-apps"
if "$CTL" join --non-interactive --yes --skip-host-checks --skip-reach-check \
     --primary-url http://host.docker.internal:18080 --token sfj_not-a-real-token --name sec2 > "$WORK/join2.log" 2>&1; then
  fail "a wrong token must not succeed"
else pass "join with a wrong token exits non-zero"; fi
grep -q "token is wrong, expired or already used" "$WORK/join2.log" && pass "and says what to do about it" || fail "no diagnosis: $(tail -5 "$WORK/join2.log")"
[[ -z "$(node_field sec2 id)" ]] && pass "and nothing was registered" || fail "sec2 was registered"

echo
echo "B. compatibility: the older direct mode still works"
NODE_DOWN
rm -rf "$WORK/node-data" "$WORK/node-apps"; mkdir -p "$WORK/node-data" "$WORK/node-apps"; chmod 777 "$WORK/node-data" "$WORK/node-apps"
DTOKEN="$(SFH_PRIMARY_CONTAINER=jn-primary "$CTL" join-token --direct --primary-url http://jn-primary:8082 2>&1 | grep -o 'sfj_[A-Za-z0-9_-]*' | head -n1)"
cat > "$WORK/node-direct-override.yml" <<YAML
services:
  node:
    volumes: !override
      - $WORK/node-data:/app/data
      - $WORK/node-apps:/app/apps
networks:
  selfhostly-network: !override
    name: $NET_A
    external: true
YAML
{ echo "SELFHOSTLY_NETWORK=$NET_A"; echo "BACKEND_IMAGE=$BACKEND"; echo "DATA_DIR=$WORK/node-data"; echo "APPS_DIR=$WORK/node-apps"; echo "APP_UID=0"; echo "NODE_PORT=18083"; echo "NODE_BIND=127.0.0.1"; } > .env.node
export SFH_NODE_EXTRA_COMPOSE="$WORK/node-direct-override.yml"
if "$CTL" join --direct --non-interactive --yes --skip-host-checks --skip-reach-check \
     --primary-url http://jn-primary:8082 --token "$DTOKEN" --name sec3 --endpoint http://selfhostly-node:8082 --port 18083 > "$WORK/join3.log" 2>&1; then
  pass "direct join exits 0"
else fail "direct join exits 0"; tail -12 "$WORK/join3.log"; fi
grep -qE "✓.* registered with the primary$" "$WORK/join3.log" && pass "registered over plain HTTP as before" || fail "direct registration not confirmed"
wait_node_status sec3 online 60 && pass "and the primary reaches it directly (online)" || fail "direct node status: $(node_field sec3 status)"
[[ "$(node_field sec3 api_endpoint)" == "http://selfhostly-node:8082" ]] && pass "recorded with its own address, not a tunnel" || fail "endpoint: $(node_field sec3 api_endpoint)"

echo
if [[ "$FAILURES" -eq 0 ]]; then echo "join tests passed"; else echo "$FAILURES check(s) failed"; exit 1; fi
