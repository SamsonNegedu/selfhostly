#!/usr/bin/env bash
# Tests scripts/upgrade.sh against a throwaway stack (own project name, containers, network, temp
# directories, so it cannot touch a real install). It builds the OLD backend from OLD_REF and the
# NEW one from the working tree, then checks: a dry run changes nothing, a real upgrade migrates
# and keeps data and apps, a broken image is rolled back automatically, a --set change applies,
# and a --set that stops the server starting is reverted.
#
#   OLD_REF=HEAD ./scripts/test-upgrade.sh        (before committing, HEAD is the previous version)
#   SKIP_BUILD=1 ./scripts/test-upgrade.sh        reuse images from the last run
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
source "$ROOT/scripts/lib/ctl.sh"
OLD_REF="${OLD_REF:-HEAD}"
source "$ROOT/scripts/lib/old-ref.sh"
OLD_REF="$(resolve_old_ref "$OLD_REF")"
WORK="$(mktemp -d)"
PROJECT="selfhostly-upg"
LIVE_BACKEND="selfhostly-upg-backend:live"
GATEWAY="selfhostly-upg-gateway:live"
FAILURES=0
export SFH_ROLLBACK_PREFIX="selfhostly-upgtest-rollback"
# The server runs as root in the container and keeps its database owner-only, so the host user cannot
# write it (sqlite reports "attempt to write a readonly database"). Fall back to sudo when needed.
sql() { if [[ -w "$1" ]]; then sqlite3 "$@"; else sudo -n sqlite3 "$@"; fi; }
pass() { echo "  PASS  $*"; }
fail() { echo "  FAIL  $*"; FAILURES=$((FAILURES + 1)); }

COMPOSE=(docker compose -p "$PROJECT" --env-file "$WORK/.env" -f "$ROOT/docker-compose.prod.yml" -f "$WORK/override.yml")
UPGRADE=("$CTL" upgrade --project "$PROJECT" --env-file "$WORK/.env" --compose "$ROOT/docker-compose.prod.yml" --compose "$WORK/override.yml" --health-timeout 40 --yes)

cleanup() {
  "${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
  docker rm -f selfhostly-upg-app >/dev/null 2>&1 || true
  docker images --format '{{.Repository}}:{{.Tag}}' | grep -E "^${SFH_ROLLBACK_PREFIX}/" | xargs -r docker rmi -f >/dev/null 2>&1 || true
  rm -rf "$WORK"
}
trap cleanup EXIT

image_id()  { docker inspect --format '{{.Image}}' "$("${COMPOSE[@]}" ps -q "$1")"; }
healthy()   { [[ "$(docker inspect --format '{{.State.Health.Status}}' "$("${COMPOSE[@]}" ps -q "$1")")" == healthy ]]; }
env_line()  { grep -E "^$1=" "$WORK/.env" | tail -n1 || true; }

if [[ "${SKIP_BUILD:-0}" != 1 ]]; then
  echo "building the OLD backend from $OLD_REF"
  # git archive only reads the repository: nothing in the working tree or index is touched
  OLD="$WORK/old-src"; mkdir -p "$OLD"; git -C "$ROOT" archive "$OLD_REF" | tar -x -C "$OLD"
  docker build -q -f "$OLD/Dockerfile.backend" -t selfhostly-upg-backend:old "$OLD" >/dev/null
  rm -rf "$OLD"
  echo "building the NEW backend and gateway"
  docker build -q -f "$ROOT/Dockerfile.backend" -t selfhostly-upg-backend:new "$ROOT" >/dev/null
  docker build -q -f "$ROOT/Dockerfile.gateway" -t "$GATEWAY" "$ROOT" >/dev/null
fi
# a broken build: starts and exits immediately
printf 'FROM selfhostly-upg-backend:new\nCMD ["/bin/false"]\n' | docker build -q -t selfhostly-upg-backend:broken - >/dev/null

echo "creating a throwaway install"
(
  cd "$WORK"
  DATA_DIR="$WORK/data" APPS_DIR="$WORK/apps" "$CTL" bootstrap --auth github --domain upg.example.test --github-user smoke-user >/dev/null
  sed -i.bak -E '/^(DATA_DIR|APPS_DIR)=\.\//d' .env && rm -f .env.bak
  { echo "DATA_DIR=$WORK/data"; echo "APPS_DIR=$WORK/apps"; echo "PRIMARY_NODE_PORT=18192"; echo "GITHUB_CLIENT_ID=x"; echo "GITHUB_CLIENT_SECRET=y"
    echo "BACKEND_IMAGE=$LIVE_BACKEND"; echo "GATEWAY_IMAGE=$GATEWAY"; } >> .env
)
mkdir -p "$WORK/data" "$WORK/apps"; chmod 777 "$WORK/data" "$WORK/apps"
cat > "$WORK/override.yml" <<YAML
services:
  primary:
    container_name: upg-primary
    user: "0:0"
  gateway:
    container_name: upg-gateway
networks:
  selfhostly-network:
    name: upg-network
YAML

docker tag selfhostly-upg-backend:old "$LIVE_BACKEND"
echo "starting the OLD version and a stand-in app container"
"${COMPOSE[@]}" up -d --wait primary gateway >/dev/null
docker run -d --name selfhostly-upg-app alpine sleep 3600 >/dev/null
sql "$WORK/data/selfhostly.db" "INSERT INTO apps (id,name,description,compose_content,status,node_id) VALUES ('a1','kan','','x','running',(SELECT id FROM nodes LIMIT 1))"
OLD_ID="$(image_id primary)"
cd "$WORK"

echo; echo "T1 dry run changes nothing"
docker tag selfhostly-upg-backend:new "$LIVE_BACKEND"
"${UPGRADE[@]}" --dry-run >/dev/null 2>&1 && pass "dry run exits 0" || fail "dry run exits 0"
[[ "$(image_id primary)" == "$OLD_ID" ]] && pass "still running the old image" || fail "still running the old image"
[[ ! -d .upgrade/last ]] && [[ ! -f .upgrade/last ]] && pass "no state written" || fail "no state written"

echo; echo "T2 real upgrade (new image is tagged as live; --no-pull because it is a local image)"
if "${UPGRADE[@]}" --no-pull > upgrade1.log 2>&1; then pass "upgrade exits 0"; else fail "upgrade exits 0"; tail -20 upgrade1.log; fi
NEW_ID="$(image_id primary)"
[[ "$NEW_ID" != "$OLD_ID" ]] && pass "primary now runs the new image" || fail "primary now runs the new image"
healthy primary && healthy gateway && pass "primary and gateway healthy" || fail "primary and gateway healthy"
[[ "$(sql data/selfhostly.db 'SELECT COUNT(*) FROM apps WHERE name = "kan"')" == 1 ]] && pass "app record survived" || fail "app record survived"
[[ "$(sql data/selfhostly.db 'SELECT MAX(version) FROM schema_migrations')" -ge 1 ]] && pass "database migrated" || fail "database migrated"
ls .upgrade/*/backup/selfhostly.db >/dev/null 2>&1 && pass "database copy saved" || fail "database copy saved"
[[ "$(docker inspect --format '{{.State.Running}}' selfhostly-upg-app)" == true ]] && pass "stand-in app container untouched" || fail "stand-in app container untouched"
grep -q "every other container is still running" upgrade1.log && pass "script confirmed apps undisturbed" || fail "script confirmed apps undisturbed"
grep -q "dry start: ok" upgrade1.log && pass "dry start ran against the new image before anything stopped" || fail "dry start ran against the new image"
# capture first: with pipefail, grep -q closing the pipe early would fail the pipeline spuriously
PRIMARY_LOG="$(docker logs upg-primary 2>&1)"
EFFECTIVE="$(printf '%s\n' "$PRIMARY_LOG" | grep '"msg":"effective configuration"' || true)"
[[ -n "$EFFECTIVE" ]] && pass "startup log shows the effective configuration" || fail "startup log shows the effective configuration"
[[ "$EFFECTIVE" == *"security_mode"* && "$EFFECTIVE" == *"<- "* ]] && pass "with where each value came from" || fail "effective configuration lists sources"

echo; echo "T3 a broken image is rolled back automatically"
docker tag selfhostly-upg-backend:broken "$LIVE_BACKEND"
if "${UPGRADE[@]}" --no-pull > upgrade2.log 2>&1; then fail "broken upgrade must exit non-zero"; else pass "broken upgrade exits non-zero"; fi
[[ "$(image_id primary)" == "$NEW_ID" ]] && pass "rolled back to the previous image" || fail "rolled back to the previous image"
healthy primary && pass "primary healthy after rollback" || fail "primary healthy after rollback"
[[ "$(sql data/selfhostly.db 'SELECT COUNT(*) FROM apps')" == 1 ]] && pass "data intact after rollback" || fail "data intact after rollback"
docker tag selfhostly-upg-backend:new "$LIVE_BACKEND"

echo; echo "T4 --set applies a setting without pulling"
"${UPGRADE[@]}" --set AUTH_SESSION_HOURS=12 > upgrade3.log 2>&1 && pass "set exits 0" || { fail "set exits 0"; tail -15 upgrade3.log; }
[[ "$(env_line AUTH_SESSION_HOURS)" == "AUTH_SESSION_HOURS=12" ]] && pass "env file updated" || fail "env file updated"
docker exec upg-primary printenv AUTH_SESSION_HOURS 2>/dev/null | grep -qx 12 && pass "container picked up the new value" || fail "container picked up the new value"

echo; echo "T5 a --set that stops the server starting is reverted"
BEFORE_ENV="$(env_line GITHUB_ALLOWED_USERS)"
PRIMARY_BEFORE="$("${COMPOSE[@]}" ps -q primary)"
if "${UPGRADE[@]}" --set GITHUB_ALLOWED_USERS= > upgrade4.log 2>&1; then fail "bad setting must exit non-zero"; else pass "bad setting exits non-zero"; fi
grep -q "ABORTED before any change" upgrade4.log && pass "caught by the dry start, before anything was stopped" || fail "caught by the dry start"
[[ "$("${COMPOSE[@]}" ps -q primary)" == "$PRIMARY_BEFORE" ]] && pass "the running primary was never touched (zero downtime)" || fail "the running primary was never touched"
[[ "$(env_line GITHUB_ALLOWED_USERS)" == "$BEFORE_ENV" ]] && pass ".env restored" || fail ".env restored (got '$(env_line GITHUB_ALLOWED_USERS)')"
healthy primary && pass "primary healthy after the revert" || fail "primary healthy after the revert"

echo; echo "T5b node id drift is caught before a restart"
BEFORE_ID="$(env_line NODE_ID)"
if "${UPGRADE[@]}" --set NODE_ID=some-other-id > upgrade5.log 2>&1; then fail "node id drift must exit non-zero"; else pass "node id drift exits non-zero"; fi
grep -q "would use node id some-other-id" upgrade5.log && pass "doctor names the mismatch and the fix" || fail "doctor names the mismatch"
[[ "$(env_line NODE_ID)" == "$BEFORE_ID" ]] && pass ".env restored" || fail ".env restored"
healthy primary && pass "still healthy" || fail "still healthy"

echo; echo "T5c encryption is only ever explicit"
"${UPGRADE[@]}" --set ENCRYPT_SECRETS_AT_REST=true > up6.log 2>&1 && pass "turn encryption on" || { fail "turn encryption on"; tail -8 up6.log; }
[[ "$(sql data/selfhostly.db "SELECT COUNT(*) FROM nodes WHERE api_key LIKE 'enc:v1:%'")" -ge 1 ]] && pass "node keys are now encrypted" || fail "node keys are now encrypted"
"${UPGRADE[@]}" --set SECURITY_MODE=enforce > up7.log 2>&1 && pass "switch to enforce" || { fail "switch to enforce"; tail -8 up7.log; }
[[ "$(sql data/selfhostly.db "SELECT COUNT(*) FROM nodes WHERE api_key LIKE 'enc:v1:%'")" -ge 1 ]] && pass "changing the mode did not touch the encryption state" || fail "mode change altered encryption"
"${UPGRADE[@]}" --set ENCRYPT_SECRETS_AT_REST=false > up8.log 2>&1 && pass "turn encryption off" || { fail "turn encryption off"; tail -8 up8.log; }
[[ "$(sql data/selfhostly.db "SELECT COUNT(*) FROM nodes WHERE api_key LIKE 'enc:v1:%'")" == 0 ]] && pass "node keys converted back to plaintext (rollback-safe)" || fail "conversion back to plaintext"
healthy primary && pass "healthy" || fail "healthy"

echo; echo "T5d identity is adopted from the database, not from a compose default"
DB_ID="$(sql data/selfhostly.db 'SELECT id FROM nodes WHERE is_primary = 1')"
rm -f data/node-id
"${UPGRADE[@]}" --set NODE_ID= > up9.log 2>&1 && pass "restart with NODE_ID unset" || { fail "restart with NODE_ID unset"; tail -8 up9.log; }
[[ "$(sql data/selfhostly.db 'SELECT id FROM nodes WHERE is_primary = 1')" == "$DB_ID" ]] && pass "database identity unchanged" || fail "database identity unchanged"
LOGS="$(docker logs upg-primary 2>&1)"
EFF="$(printf '%s\n' "$LOGS" | grep '"msg":"effective configuration"' | tail -n1 || true)"
[[ "$EFF" == *"$DB_ID"* ]] && pass "the running node uses the database's ID" || fail "the running node uses the database's ID"
NODE_ID_FIELD="$(printf '%s' "$EFF" | python3 -c 'import sys,json; print(json.loads(sys.stdin.read()).get("node_id",""))')"
[[ "$NODE_ID_FIELD" == "$DB_ID"*"<- "* ]] && pass "the log line for node_id shows that ID and where it came from" || fail "node_id log line (got: $NODE_ID_FIELD)"
[[ "$(cat data/node-id 2>/dev/null)" == "$DB_ID" ]] && pass "the ID was saved so later starts agree" || fail "the ID was saved (got '$(cat data/node-id 2>/dev/null)')"
DOCTOR="$(docker exec upg-primary ./selfhostly doctor --offline 2>&1 || true)"
[[ "$DOCTOR" == *"node identity matches"* ]] && pass "doctor confirms the identity" || fail "doctor confirms the identity"
[[ "$DOCTOR" == *"owned by group"* ]] && pass "doctor states the docker socket facts" || fail "doctor states the docker socket facts"

echo; echo "T6 explicit --rollback"
"${UPGRADE[@]}" --rollback > rollback.log 2>&1 && pass "manual rollback exits 0" || { fail "manual rollback exits 0"; tail -10 rollback.log; }
healthy primary && pass "healthy after manual rollback" || fail "healthy after manual rollback"

echo
if [[ "$FAILURES" -eq 0 ]]; then echo "upgrade tests passed"; else echo "$FAILURES check(s) failed"; exit 1; fi
