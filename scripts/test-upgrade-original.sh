#!/usr/bin/env bash
# Tests the upgrade path for an install that runs the ORIGINAL production compose file (with its hand
# edits), which is what real installs look like: not the repository's current file.
#
#   Phase A  new images run on the OLD compose file and the old .env, with no configuration change
#   Phase B  compose-diff reports what the new compose file would lose, --apply carries it into .env,
#            the new compose file is swapped in, and everything the old file hard-coded still applies
#   Undo     going back to the old compose file works
#
# Everything runs under its own names on a private network. Test-only substitutions in the fixture:
# local image tags, container/network names, and user 0:0 (this machine's Docker socket group differs
# from a Linux host's).
#
#   OLD_REF=<commit> ./scripts/test-upgrade-original.sh    the version the install runs today
#   With no OLD_REF it uses HEAD while the tree has uncommitted changes, and the previous commit once the
#   tree is clean. To test against an older deployed version, pass that commit (CI passes the pull request's base).
#   SKIP_BUILD=1 ./scripts/test-upgrade-original.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
source "$ROOT/scripts/lib/ctl.sh"
source "$ROOT/scripts/lib/old-ref.sh"
OLD_REF="$(resolve_old_ref "$(default_old_ref "${OLD_REF:-}")")"
WORK="$(mktemp -d)"
PROJECT="selfhostly-orig"
FAILURES=0
export SFH_ROLLBACK_PREFIX="selfhostly-origtest-rollback"
# The server runs as root in the container and keeps its database owner-only, so the host user cannot
# write it (sqlite reports "attempt to write a readonly database"). Fall back to sudo when needed.
sql() { if [[ -w "$1" ]]; then sqlite3 "$@"; else sudo -n sqlite3 "$@"; fi; }
pass() { echo "  PASS  $*"; }
fail() { echo "  FAIL  $*"; FAILURES=$((FAILURES + 1)); }
contains() { [[ "$1" == *"$2"* ]]; }

LIVE="$WORK/live/docker-compose.yml"
NEW="$WORK/live/docker-compose.new.yml"
ENVF="$WORK/live/.env"
DC() { docker compose -p "$PROJECT" --env-file "$ENVF" -f "$1" "${@:2}"; }

cleanup() {
  DC "$LIVE" down -v --remove-orphans >/dev/null 2>&1 || true
  docker network rm orig-network >/dev/null 2>&1 || true
  docker images --format '{{.Repository}}:{{.Tag}}' | grep -E "^${SFH_ROLLBACK_PREFIX}/" | xargs -r docker rmi -f >/dev/null 2>&1 || true
  rm -rf "$WORK"
}
trap cleanup EXIT

if [[ "${SKIP_BUILD:-0}" != 1 ]]; then
  echo "building the OLD images from $OLD_REF and the NEW ones from the working tree"
  # git archive only reads the repository: nothing in the working tree or index is touched
  OLD="$WORK/old-src"; mkdir -p "$OLD"; git -C "$ROOT" archive "$OLD_REF" | tar -x -C "$OLD"
  docker build -q -f "$OLD/Dockerfile.backend" -t selfhostly-orig-backend:old "$OLD" >/dev/null
  docker build -q -f "$OLD/Dockerfile.gateway" -t selfhostly-orig-gateway:old "$OLD" >/dev/null
  rm -rf "$OLD"
  docker build -q -f "$ROOT/Dockerfile.backend" -t selfhostly-orig-backend:new "$ROOT" >/dev/null
  docker build -q -f "$ROOT/Dockerfile.gateway" -t selfhostly-orig-gateway:new "$ROOT" >/dev/null
fi

mkdir -p "$WORK/live/data" "$WORK/live/apps"; chmod 777 "$WORK/live/data" "$WORK/live/apps"

# test-only substitutions, applied identically to the old and the new file so they compare like for like
adapt() {
  sed -e 's#ghcr.io/samsonnegedu/selfhostly-gateway:[A-Za-z0-9._${}:-]*#selfhostly-orig-gateway:live#' \
      -e 's#ghcr.io/samsonnegedu/selfhostly-backend:[A-Za-z0-9._${}:-]*#selfhostly-orig-backend:live#' \
      -e 's#container_name: selfhostly-#container_name: orig-#' \
      -e 's#name: selfhostly-network#name: orig-network#' \
      -e 's#user: 1000:984#user: "0:0"#' \
      -e 's#user: \${APP_UID:-1000}:\${DOCKER_GID:-984}#user: "0:0"#' "$1"
}
adapt "$ROOT/scripts/fixtures/original-prod-compose.yml" > "$LIVE"
# the new file's image lines are `${GATEWAY_IMAGE:-ghcr.io/...:${SELFHOSTLY_VERSION:-latest}}`
python3 - "$ROOT/docker-compose.prod.yml" "$NEW" <<'PY'
import re,sys
s=open(sys.argv[1]).read()
s=re.sub(r'\$\{GATEWAY_IMAGE:-[^\n]*\}\}', 'selfhostly-orig-gateway:live', s)
s=re.sub(r'\$\{BACKEND_IMAGE:-[^\n]*\}\}', 'selfhostly-orig-backend:live', s)
s=s.replace('container_name: selfhostly-','container_name: orig-').replace('name: selfhostly-network','name: orig-network')
s=re.sub(r'user: \$\{APP_UID:-1000\}:\$\{DOCKER_GID:-984\}','user: "0:0"',s)
open(sys.argv[2],'w').write(s)
PY
cat > "$ENVF" <<EOF
DATA_DIR=$WORK/live/data
APPS_DIR=$WORK/live/apps
JWT_SECRET=$(openssl rand -hex 32)
GATEWAY_API_KEY=$(openssl rand -hex 24)
REGISTRATION_TOKEN=$(openssl rand -hex 24)
AUTH_ENABLED=true
GITHUB_CLIENT_ID=test-id
GITHUB_CLIENT_SECRET=test-secret
GITHUB_ALLOWED_USERS=test-user
PRIMARY_NODE_PORT=18193
EOF

# login stays ON in this stack; reads go through the gateway key, like the gateway itself does
GW_KEY="$(grep '^GATEWAY_API_KEY=' "$ENVF" | cut -d= -f2)"
api() { docker exec orig-primary wget -qO- --header "X-Gateway-API-Key: $GW_KEY" "$1" 2>/dev/null; }
# Reads the database itself (a live SQLite file read across the Docker VM boundary is unreliable), so
# the primary is stopped for the read and started again afterwards.
app_count() {
  local f="$1"
  DC "$f" stop primary >/dev/null
  local n; n="$(sql "$WORK/live/data/selfhostly.db" 'SELECT COUNT(*) FROM apps WHERE name = "kan"')"
  DC "$f" start primary >/dev/null
  for _ in $(seq 1 30); do healthy orig-primary && break; sleep 2; done
  echo "$n"
}
healthy() { [[ "$(docker inspect --format '{{.State.Health.Status}}' "$1" 2>/dev/null)" == healthy ]]; }
image_id() { docker inspect --format '{{.Image}}' "$1"; }

echo "the existing install: OLD images on the ORIGINAL compose file"
docker tag selfhostly-orig-backend:old selfhostly-orig-backend:live
docker tag selfhostly-orig-gateway:old selfhostly-orig-gateway:live
DC "$LIVE" up -d --wait primary gateway >/dev/null
OLD_PRIMARY_ID="$(image_id orig-primary)"
NODE_ID_BEFORE="$(api http://localhost:8082/api/node/info | python3 -c 'import sys,json; print(json.load(sys.stdin)["id"])')"
# give it some state to lose: an app record (written while the primary is stopped, for a clean read)
DC "$LIVE" stop primary >/dev/null
sql "$WORK/live/data/selfhostly.db" "INSERT INTO apps (id,name,description,compose_content,status,node_id) VALUES ('a1','kan','','x','running','$NODE_ID_BEFORE')"
DC "$LIVE" start primary >/dev/null
for _ in $(seq 1 30); do healthy orig-primary && break; sleep 2; done
[[ "$NODE_ID_BEFORE" == 450359e5-52c3-47e8-a256-6ea537528a06 ]] && pass "the old install runs with the original compose file's fixed node id" || fail "node id before: $NODE_ID_BEFORE"
LOGS_OLD="$(docker logs orig-primary 2>&1 | sed -n 1,5p)"
[[ "$LOGS_OLD" != *'{"time"'* ]] && pass "and writes plain-text logs (LOG_JSON=false, hard-coded in that file)" || fail "old logs are JSON"

echo
echo "Phase A: new images, OLD compose file, OLD .env, nothing else changed"
docker tag selfhostly-orig-backend:new selfhostly-orig-backend:live
docker tag selfhostly-orig-gateway:new selfhostly-orig-gateway:live
cd "$WORK/live"
# no --compose and no --project: a real install is upgraded by finding the file it was started from.
# --container only says which install when other Selfhostly stacks happen to run on this machine too.
if "$CTL" upgrade --container orig-primary --env-file "$ENVF" --no-pull --yes --health-timeout 60 > "$WORK/phaseA.log" 2>&1; then
  pass "upgrade exits 0 with no compose file or project given"
else fail "upgrade exits 0"; tail -20 "$WORK/phaseA.log"; fi
grep -q "using the compose file your running install was started from: $LIVE" "$WORK/phaseA.log" && pass "it found the live compose file from Docker's own record" || fail "live compose file not detected: $(head -3 "$WORK/phaseA.log")"
[[ "$(image_id orig-primary)" != "$OLD_PRIMARY_ID" ]] && pass "the primary runs the new image" || fail "primary still on the old image"
healthy orig-primary && healthy orig-gateway && pass "primary and gateway are healthy" || fail "not healthy"
DOC="$(docker exec orig-primary ./selfhostly doctor --offline --skip-db-integrity 2>&1 || true)"
contains "$DOC" "security mode               warn" && pass "an existing database starts in warn mode (nothing is blocked)" || fail "mode: $(printf '%s' "$DOC" | grep 'security mode')"
contains "$DOC" "extra allowed volume paths  1" && pass "the volume whitelist hard-coded in the old file still applies" || fail "whitelist: $(printf '%s' "$DOC" | grep 'allowed volume')"
contains "$DOC" "node identity matches the database" && pass "the node identity is unchanged" || fail "identity: $(printf '%s' "$DOC" | grep -i identity)"
N="$(app_count "$LIVE")"; [[ "$N" == 1 ]] && pass "the existing app record survived" || fail "app record lost (count=$N; API says: $(api http://localhost:8082/api/apps | cut -c1-160))"
LOGS_NEW="$(docker logs orig-primary 2>&1)"
[[ "$LOGS_NEW" != *'{"time"'* ]] && pass "logs are still plain text, as the old file configured" || fail "new logs are JSON"
contains "$LOGS_NEW" "applied database migration" && pass "the database was migrated" || fail "no migration logged"
ls "$WORK/live/data/"selfhostly.db.bak-pre-migration-* >/dev/null 2>&1 && pass "with an automatic backup taken first" || fail "no pre-migration backup"

echo
echo "A setting the original compose file never reads is refused instead of silently ignored"
ENV_BEFORE="$(cat "$ENVF")"
if "$CTL" upgrade --container orig-primary --env-file "$ENVF" --no-pull --yes --set SECURITY_MODE=enforce > "$WORK/unread.log" 2>&1; then
  fail "--set on a setting the file never reads must not report success"
else pass "--set SECURITY_MODE=enforce is refused"; fi
grep -q "never reads SECURITY_MODE" "$WORK/unread.log" && grep -q "Nothing was changed" "$WORK/unread.log" && pass "and it says why and what to do" || fail "unhelpful refusal: $(tail -5 "$WORK/unread.log")"
[[ "$(cat "$ENVF")" == "$ENV_BEFORE" ]] && pass "and the settings file is untouched" || fail "the settings file was edited"

echo
echo "Phase B: what would the new compose file lose?"
cp "$LIVE" "$WORK/live/docker-compose.yml.before-upgrade"
"$CTL" compose-diff "$WORK/live/docker-compose.yml.before-upgrade" "$NEW" --env-file "$ENVF" > "$WORK/diff1.log" 2>&1 || true
D1="$(cat "$WORK/diff1.log")"
contains "$D1" "ALLOWED_VOLUME_PATHS=/home/user/selfhosted" && contains "$D1" "LOG_JSON=false" && pass "compose-diff finds the two hand-edits (whitelist, log format)" || fail "diff: $D1"
contains "$D1" "WOULD BE LOST" && fail "nothing here should be unrecoverable: $D1" || pass "and nothing that cannot be carried through .env"
"$CTL" compose-diff "$WORK/live/docker-compose.yml.before-upgrade" "$NEW" --env-file "$ENVF" --apply "$ENVF" >/dev/null 2>&1 || true
grep -qx "ALLOWED_VOLUME_PATHS=/home/user/selfhosted" "$ENVF" && grep -qx "LOG_JSON=false" "$ENVF" && pass "--apply wrote them into .env" || fail ".env: $(cat "$ENVF")"
if "$CTL" compose-diff "$WORK/live/docker-compose.yml.before-upgrade" "$NEW" --env-file "$ENVF" >/dev/null 2>&1; then
  pass "and a second run says nothing would be lost (exit 0)"; else fail "still reports drift after --apply"; fi

echo
echo "Phase B: swap in the new compose file"
cp "$NEW" "$LIVE"
if "$CTL" upgrade --project "$PROJECT" --compose "$LIVE" --env-file "$ENVF" --no-pull --yes --health-timeout 60 > "$WORK/phaseB.log" 2>&1; then
  pass "upgrade.sh applies the new compose file (exit 0)"; else fail "apply new compose"; tail -20 "$WORK/phaseB.log"; fi
healthy orig-primary && healthy orig-gateway && pass "primary and gateway are healthy on the new file" || fail "not healthy after the swap"
docker exec orig-primary sh -c 'touch /app/write-test' >/dev/null 2>&1 && fail "the root filesystem is writable" || pass "the new file's hardening is in effect (read-only root filesystem)"
DOC="$(docker exec orig-primary ./selfhostly doctor --offline --skip-db-integrity 2>&1 || true)"
contains "$DOC" "extra allowed volume paths  1" && pass "the volume whitelist now comes from .env and still applies" || fail "whitelist lost: $(printf '%s' "$DOC" | grep 'allowed volume')"
contains "$DOC" "node identity matches the database" && pass "node identity unchanged" || fail "identity changed"
LOGS_B="$(docker logs orig-primary 2>&1 | tail -20)"
[[ "$LOGS_B" != *'{"time"'* ]] && pass "logs are still plain text (LOG_JSON carried through .env)" || fail "logs became JSON"
N="$(app_count "$LIVE")"; [[ "$N" == 1 ]] && pass "the app record still exists" || fail "app record lost (count=$N)"
"$CTL" upgrade --container orig-primary --project "$PROJECT" --compose "$LIVE" --env-file "$ENVF" --no-pull --yes --health-timeout 60 --set AUTH_SESSION_HOURS=12 > "$WORK/set-new.log" 2>&1 \
  && [[ "$(docker exec orig-primary printenv AUTH_SESSION_HOURS)" == 12 ]] && pass "on the new file the same kind of --set takes effect inside the container" || fail "--set on the new file: $(tail -5 "$WORK/set-new.log")"

echo
echo "Undo: back to the original compose file"
cp "$WORK/live/docker-compose.yml.before-upgrade" "$LIVE"
DC "$LIVE" up -d --no-deps --force-recreate primary >/dev/null 2>&1
for _ in $(seq 1 30); do healthy orig-primary && break; sleep 2; done
healthy orig-primary && pass "the original file runs again on the new image (nothing was left incompatible)" || fail "did not come back on the original file"

echo
if [[ "$FAILURES" -eq 0 ]]; then echo "upgrade-from-original tests passed"; else echo "$FAILURES check(s) failed"; exit 1; fi
