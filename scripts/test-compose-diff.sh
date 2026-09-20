#!/usr/bin/env bash
# Tests `selfhostlyctl compose-diff` against a copy of a real-world hand-edited production compose file.
# Needs the docker CLI but not the daemon (rendering a compose file is client-side).
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
T="$(mktemp -d)"; trap 'rm -rf "$T"' EXIT
FAILURES=0
pass() { echo "  PASS  $*"; }
fail() { echo "  FAIL  $*"; FAILURES=$((FAILURES + 1)); }
contains() { [[ "$1" == *"$2"* ]]; }
source "$ROOT/scripts/lib/ctl.sh"
export JWT_SECRET=x GATEWAY_API_KEY=y REGISTRATION_TOKEN=z

# A production file as people actually end up with it: values edited in place instead of set in .env
cat > "$T/live.yml" <<'YAML'
services:
  gateway:
    image: ghcr.io/samsonnegedu/selfhostly-gateway:latest
    environment:
      PRIMARY_BACKEND_URL: http://primary:8082
      GATEWAY_API_KEY: ${GATEWAY_API_KEY}
      LOG_JSON: "false"
      JWT_SECRET: ${JWT_SECRET}
    networks: [selfhostly-network]
  primary:
    image: ghcr.io/samsonnegedu/selfhostly-backend:latest
    user: 1000:984
    environment:
      SERVER_ADDRESS: ":8082"
      LOG_JSON: "false"
      ALLOWED_VOLUME_PATHS: /home/me/apps,/mnt/media
      NODE_ID: ${NODE_ID:-450359e5-52c3-47e8-a256-6ea537528a06}
      REGISTRATION_TOKEN: ${REGISTRATION_TOKEN}
      JWT_SECRET: ${JWT_SECRET}
      GATEWAY_API_KEY: ${GATEWAY_API_KEY}
    volumes:
      - ${DATA_DIR:-./data}:/app/data
      - ${APPS_DIR:-./apps}:/app/apps
      - /var/run/docker.sock:/var/run/docker.sock
    networks: [selfhostly-network]
networks:
  selfhostly-network:
    name: selfhostly-network
YAML

echo "a hand-edited file with values that can move to .env"
out="$("$CTL" compose-diff "$T/live.yml" 2>&1)"; rc=$?
contains "$out" "ALLOWED_VOLUME_PATHS=/home/me/apps,/mnt/media" && pass "finds a hard-coded volume whitelist" || fail "whitelist: $out"
contains "$out" "LOG_JSON=false" && pass "finds the hard-coded log format" || fail "log json"
contains "$out" "NODE_ID=450359e5" && pass "finds the pinned node id" || fail "node id"
[[ "$rc" == 1 ]] && pass "exits 1 while something would be lost" || fail "exit code $rc"
contains "$out" "WOULD BE LOST" && fail "these are carriable, not lost" || pass "and does not call them lost"

echo "--apply writes only what is missing and never overwrites"
printf 'JWT_SECRET=keep\nLOG_JSON=true\n' > "$T/.env"
out="$("$CTL" compose-diff "$T/live.yml" --apply "$T/.env" 2>&1)"
grep -qx "ALLOWED_VOLUME_PATHS=/home/me/apps,/mnt/media" "$T/.env" && pass "adds the missing line" || fail "apply: $(cat "$T/.env")"
grep -qx "LOG_JSON=true" "$T/.env" && ! grep -qx "LOG_JSON=false" "$T/.env" && pass "leaves a value that is already set alone" || fail "overwrote: $(cat "$T/.env")"
grep -qx "JWT_SECRET=keep" "$T/.env" && pass "leaves unrelated settings alone" || fail "unrelated"
[[ "$(stat -c %a "$T/.env" 2>/dev/null || stat -f %Lp "$T/.env")" == 600 ]] && pass "env file is mode 600" || fail "mode"
# LOG_JSON=true was already set to something else, so drift against the live "false" is real and must stay reported
out="$("$CTL" compose-diff "$T/live.yml" --env-file "$T/.env" 2>&1)"
contains "$out" "NEEDS A DECISION" && contains "$out" "LOG_JSON: your env file has 'true' but the old compose file ran with 'false'" && pass "a conflicting value is named as a decision, with both sides" || fail "conflict: $out"
printf 'JWT_SECRET=keep\n' > "$T/.env2"
"$CTL" compose-diff "$T/live.yml" --apply "$T/.env2" >/dev/null 2>&1
"$CTL" compose-diff "$T/live.yml" --env-file "$T/.env2" >/dev/null 2>&1; rc=$?
[[ "$rc" == 0 ]] && pass "after --apply on a clean env file nothing is left to carry (exit 0)" || fail "after apply exit $rc"

echo "things that cannot move to .env are reported as lost"
python3 - "$T/live.yml" "$T/custom.yml" <<'PY'
import sys
s=open(sys.argv[1]).read()
s=s.replace("      LOG_JSON: \"false\"\n      ALLOWED","      LOG_JSON: \"false\"\n      MY_CUSTOM_SETTING: hand-edited\n      ALLOWED")
s=s.replace("      - /var/run/docker.sock:/var/run/docker.sock","      - /var/run/docker.sock:/var/run/docker.sock\n      - /srv/extra:/extra")
s=s.replace("networks:\n  selfhostly-network:","  sidecar:\n    image: nginx\nnetworks:\n  selfhostly-network:",1)
open(sys.argv[2],"w").write(s)
PY
out="$("$CTL" compose-diff "$T/custom.yml" 2>&1)"; rc=$?
contains "$out" "MY_CUSTOM_SETTING = 'hand-edited' is not in the new file at all" && pass "a setting the new file does not know is reported with the exact line to add" || fail "custom env: $out"
contains "$out" "/srv/extra" && pass "an extra volume mount is reported" || fail "extra volume: $out"
contains "$out" "service 'sidecar'" && pass "an extra service is reported" || fail "extra service: $out"
[[ "$rc" == 1 ]] && pass "exits 1" || fail "exit $rc"

echo "a changed user id is carried through APP_UID / DOCKER_GID"
sed 's/user: 1000:984/user: 1001:999/' "$T/live.yml" > "$T/user.yml"
out="$("$CTL" compose-diff "$T/user.yml" 2>&1)"
contains "$out" "APP_UID=1001" && contains "$out" "DOCKER_GID=999" && pass "user 1001:999 becomes two .env lines" || fail "user: $out"

echo "secrets are masked in the output"
sed 's/GATEWAY_API_KEY: ${GATEWAY_API_KEY}/GATEWAY_API_KEY: hardcoded-super-secret/' "$T/live.yml" > "$T/secret.yml"
out="$("$CTL" compose-diff "$T/secret.yml" 2>&1)"
contains "$out" "hardcoded-super-secret" && fail "a secret leaked into the output" || pass "a hard-coded secret is not printed"
out="$("$CTL" compose-diff "$T/secret.yml" --show-secrets 2>&1)"
contains "$out" "hardcoded-super-secret" && pass "--show-secrets reveals it on request" || fail "show-secrets"

echo "the repository's own file"
out="$("$CTL" compose-diff "$ROOT/docker-compose.prod.yml" 2>&1)"; rc=$?
[[ "$rc" == 0 ]] && pass "a file compared with itself loses nothing" || fail "self-compare: $out"

echo
if [[ "$FAILURES" -eq 0 ]]; then echo "compose-diff tests passed"; else echo "$FAILURES check(s) failed"; exit 1; fi
