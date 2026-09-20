#!/usr/bin/env bash
# End-to-end test of the UI-driven update, against a stack that mirrors a deployment: production images built
# locally, docker-compose.prod.yml, a local registry in place of GHCR, a reverse proxy in place of the Cloudflare
# tunnel, and a real browser (Playwright) that clicks through Settings, Updates.
#
#   make e2e-update                      build, start, run every spec, tear down
#   scripts/e2e/run.sh <command>         run one part, to iterate (see `help`)
#
# Everything created is named e2e-* and labelled selfhostly.e2e=1, and only those resources are ever removed.
# Names, ports and paths are in scripts/e2e/env.sh. Nothing here touches a real install.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
E2E="$ROOT/scripts/e2e"
# shellcheck source=scripts/e2e/env.sh
source "$E2E/env.sh"

log() { printf '==> %s\n' "$*"; }
die() { printf 'e2e: %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

compose() {
  docker compose -p "$E2E_PROJECT" --env-file "$INSTALL_DIR/.env" -f "$INSTALL_DIR/docker-compose.prod.yml" "$@"
}

env_get() { grep -E "^$1=" "$INSTALL_DIR/.env" 2>/dev/null | tail -n1 | cut -d= -f2- || true; }

# writes KEY=VALUE into the install's settings file, replacing an earlier line for the same key
set_env() {
  local file="$INSTALL_DIR/.env" tmp
  tmp="$(mktemp)"
  grep -v "^$1=" "$file" >"$tmp" || true
  printf '%s=%s\n' "$1" "$2" >>"$tmp"
  cat "$tmp" >"$file"
  rm -f "$tmp"
}

version_key() { echo "${1//./_}"; }

# the install's database is owned by the container user, which on some hosts is not the current user
sql() { if [[ -w "$1" ]]; then sqlite3 "$@"; else sudo -n sqlite3 "$@"; fi; }

wait_healthy() { # container [seconds]
  local name="$1" limit="${2:-120}" waited=0
  while [[ "$(docker inspect --format '{{.State.Health.Status}}' "$name" 2>/dev/null || true)" != healthy ]]; do
    sleep 2
    waited=$((waited + 2))
    [[ "$waited" -lt "$limit" ]] || die "$name did not become healthy in ${limit}s (docker logs $name)"
  done
}

wait_http() { # url [seconds]
  local url="$1" limit="${2:-60}" waited=0
  until curl -fsS -o /dev/null "$url" 2>/dev/null; do
    sleep 2
    waited=$((waited + 2))
    [[ "$waited" -lt "$limit" ]] || die "$url did not answer in ${limit}s"
  done
}

# a port that is taken by something that is not ours is reported, never freed
port_check() {
  local port="$1"
  (exec 3<>"/dev/tcp/127.0.0.1/$port") 2>/dev/null || return 0
  if docker ps --filter "label=$E2E_LABEL" --format '{{.Ports}}' | grep -q ":${port}->"; then
    return 0
  fi
  echo "port $port is in use by something that is not part of this harness:" >&2
  lsof -nP -iTCP:"$port" -sTCP:LISTEN >&2 2>/dev/null || true
  die "pick another port (E2E_REGISTRY_PORT, E2E_UI_PORT, E2E_NODE_PORT) or stop that program yourself"
}

# ---------- build -------------------------------------------------------------------------------------

registry_up() {
  port_check "$REGISTRY_PORT"
  if [[ "$(docker inspect --format '{{.State.Running}}' "$REGISTRY_NAME" 2>/dev/null || true)" == true ]]; then
    return
  fi
  docker rm -f "$REGISTRY_NAME" >/dev/null 2>&1 || true
  log "starting the local registry on $REGISTRY_HOST"
  docker volume create --label "$E2E_LABEL" "$REGISTRY_VOLUME" >/dev/null
  docker run -d --name "$REGISTRY_NAME" --label "$E2E_LABEL" \
    -p "127.0.0.1:${REGISTRY_PORT}:5000" -v "${REGISTRY_VOLUME}:/var/lib/registry" registry:2 >/dev/null
  wait_http "http://${REGISTRY_HOST}/v2/" 30
}

build_tools() {
  mkdir -p "$BIN_DIR"
  log "building the test tools"
  (cd "$ROOT" && go build -o "$BIN_DIR/releasetool" ./cmd/releasetool \
    && go build -o "$BIN_DIR/mint-session" ./scripts/e2e/mint-session \
    && go build -o "$BIN_DIR/composegen" ./scripts/e2e/composegen \
    && go build -o "$BIN_DIR/selfhostlyctl" ./cmd/selfhostlyctl)
}

# records the pushed digest of an image as <KIND>_<version>=<repo>@sha256:...
record_digest() { # kind version image-tag
  local ref digest
  digest="$(docker image inspect --format '{{range .RepoDigests}}{{println .}}{{end}}' "$3" | grep "^${REPO_PREFIX}" | head -n1)"
  [[ -n "$digest" ]] || die "no registry digest for $3 after the push"
  ref="$digest"
  grep -v "^$1_$(version_key "$2")=" "$DIGESTS_FILE" >"$DIGESTS_FILE.tmp" 2>/dev/null || true
  printf '%s_%s=%s\n' "$1" "$(version_key "$2")" "$ref" >>"$DIGESTS_FILE.tmp"
  mv "$DIGESTS_FILE.tmp" "$DIGESTS_FILE"
}

digest_ref() { grep -E "^$1_$(version_key "$2")=" "$DIGESTS_FILE" | cut -d= -f2- || true; }

push_image() { # kind version tag
  docker push -q "$3" >/dev/null
  record_digest "$1" "$2" "$3"
}

# stages a source tree for one version: the sources the Dockerfiles copy, with docker-compose.prod.yml renamed for
# the harness. The images embed that file (selfhostlyctl writes it during an update) and the manifest signs its
# hash, so an update sees the same compose file a real release would carry. Prints the stage directory.
stage_source() { # source-dir version
  local src="$1" version="$2" stage="$E2E_DIR/stage-$2" f
  rm -rf "$stage"
  mkdir -p "$stage"
  for f in Dockerfile.backend Dockerfile.gateway assets.go go.mod go.sum env.example .dockerignore \
    docker-compose.secondary.yml docker-compose.secondary-direct.yml docker-compose.socket-proxy.yml; do
    cp "$src/$f" "$stage/$f"
  done
  cp -R "$src/cmd" "$src/internal" "$stage/"
  tar -C "$src" --exclude=node_modules --exclude=dist --exclude=e2e -cf - web | tar -C "$stage" -xf -
  "$BIN_DIR/composegen" --in "$src/docker-compose.prod.yml" --out "$stage/docker-compose.prod.yml"
  cp "$stage/docker-compose.prod.yml" "$E2E_DIR/compose-$version.yml"
  echo "$stage"
}

# builds and pushes the three images of one version from a source tree. The label changes the image digest, so
# two versions built from the same code (the default) are still different releases.
build_set() { # source-dir version
  local version="$2" label="selfhostly.e2e.version=$2" stage
  log "building version $version from $1"
  stage="$(stage_source "$1" "$version")"
  docker build -q --build-arg "VERSION=$version" --label "$label" \
    -f "$stage/Dockerfile.backend" -t "${REPO_PREFIX}backend:$version" "$stage" >/dev/null
  docker build -q --label "$label" -f "$stage/Dockerfile.gateway" -t "${REPO_PREFIX}gateway:$version" "$stage" >/dev/null
  docker build -q --label "$label" -f "$stage/web/Dockerfile" -t "${REPO_PREFIX}frontend:$version" "$stage/web" >/dev/null
  push_image BACKEND "$version" "${REPO_PREFIX}backend:$version"
  push_image GATEWAY "$version" "${REPO_PREFIX}gateway:$version"
  push_image FRONTEND "$version" "${REPO_PREFIX}frontend:$version"
  rm -rf "$stage"
}

# a release whose primary starts and exits at once, so the health check never passes and the update must roll back.
# The updater runs `selfhostlyctl` from this image with an explicit command, so it is not affected.
build_broken() {
  local label="selfhostly.e2e.version=$V_BROKEN" stage
  log "building the broken version $V_BROKEN"
  stage="$(stage_source "$ROOT" "$V_BROKEN")"
  docker build -q --build-arg "VERSION=$V_BROKEN" --label "$label" \
    -f "$stage/Dockerfile.backend" -t "${REPO_PREFIX}backend:${V_BROKEN}-base" "$stage" >/dev/null
  rm -rf "$stage"
  printf 'FROM %s\nCMD ["/bin/false"]\n' "${REPO_PREFIX}backend:${V_BROKEN}-base" \
    | docker build -q --label "$label" -t "${REPO_PREFIX}backend:$V_BROKEN" - >/dev/null
  push_image BACKEND "$V_BROKEN" "${REPO_PREFIX}backend:$V_BROKEN"
  # the gateway and frontend of the broken release are the good ones
  printf 'GATEWAY_%s=%s\nFRONTEND_%s=%s\n' \
    "$(version_key "$V_BROKEN")" "$(digest_ref GATEWAY "$V_NEW")" \
    "$(version_key "$V_BROKEN")" "$(digest_ref FRONTEND "$V_NEW")" >>"$DIGESTS_FILE"
}

cmd_build() {
  have docker || die "docker is required"
  have go || die "go is required"
  mkdir -p "$E2E_DIR"
  : >"$DIGESTS_FILE"
  registry_up
  build_tools

  # OLD_REF is the version the install runs today. It must already contain the update feature, or there is no
  # Updates screen to click. Without it the working tree is both the old and the new release.
  local old_src="$ROOT"
  if [[ -n "${OLD_REF:-}" ]]; then
    old_src="$E2E_DIR/old-src"
    rm -rf "$old_src"
    mkdir -p "$old_src"
    # git archive only reads the repository: the working tree and index are not touched
    git -C "$ROOT" archive "$OLD_REF" | tar -x -C "$old_src"
    log "the installed version is built from $OLD_REF"
  fi
  build_set "$old_src" "$V_OLD"
  [[ "$old_src" == "$ROOT" ]] || rm -rf "$old_src"

  build_set "$ROOT" "$V_NEW"
  build_broken
  log "images are in the registry: $(wc -l <"$DIGESTS_FILE" | tr -d ' ') digests recorded"
}

# ---------- the install ---------------------------------------------------------------------------------

write_proxy_conf() {
  # Stands in for the Cloudflare ingress: /api, /auth and /avatar go to the gateway, everything else to the
  # frontend. The upstream names are variables resolved at request time, so a recreated container is found again.
  cat >"$E2E_DIR/proxy.conf" <<EOF
resolver 127.0.0.11 valid=2s;
server {
    listen 80;
    location ~ ^/(api|auth|avatar)(/|\$) {
        set \$gateway http://${GATEWAY_NAME}:8080;
        proxy_pass \$gateway;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host \$http_host;
        proxy_set_header X-Forwarded-Host \$http_host;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_buffering off;
        proxy_read_timeout 1h;
    }
    location / {
        set \$frontend http://${FRONTEND_NAME}:80;
        proxy_pass \$frontend;
        proxy_set_header Host \$http_host;
    }
}
EOF
}

ensure_keys() {
  mkdir -p "$KEYS_DIR"
  if [[ ! -s "$KEYS_DIR/private.key" ]]; then
    "$BIN_DIR/releasetool" keygen --out "$KEYS_DIR" >/dev/null
  fi
  # a second key that the install does not trust, for the tampered-release spec
  if [[ ! -s "$KEYS_DIR/other/private.key" ]]; then
    mkdir -p "$KEYS_DIR/other"
    "$BIN_DIR/releasetool" keygen --out "$KEYS_DIR/other" >/dev/null
  fi
}

write_env() {
  local file="$INSTALL_DIR/.env"
  if [[ ! -f "$file" ]]; then
    (cd "$INSTALL_DIR" && DATA_DIR="$INSTALL_DIR/data" APPS_DIR="$INSTALL_DIR/apps" "$BIN_DIR/selfhostlyctl" bootstrap \
      --auth github --domain e2e.example.test --github-user "$E2E_LOGIN" >/dev/null)
    # bootstrap wrote relative defaults first; the absolute paths below must win
    sed -i.bak -E '/^(DATA_DIR|APPS_DIR)=\.\//d' "$file" && rm -f "$file.bak"
  fi
  set_env DATA_DIR "$INSTALL_DIR/data"
  set_env APPS_DIR "$INSTALL_DIR/apps"
  # fake OAuth credentials: nobody signs in with GitHub here, the browser gets a signed session instead
  set_env GITHUB_CLIENT_ID e2e-client-id
  set_env GITHUB_CLIENT_SECRET e2e-client-secret
  set_env GITHUB_ALLOWED_USERS "$E2E_LOGIN"
  set_env AUTH_ENABLED true
  set_env AUTH_BASE_URL "$UI_URL"
  set_env AUTH_SECURE_COOKIE false
  set_env PUBLIC_HOSTS "localhost:${UI_PORT},localhost"
  set_env PRIMARY_NODE_PORT "$NODE_PORT"
  # the primary runs as root here: the Docker socket's group differs between Linux, Docker Desktop and OrbStack
  set_env APP_UID 0
  set_env DOCKER_GID 0
  set_env BACKEND_IMAGE "${REPO_PREFIX}backend:${V_OLD}"
  set_env GATEWAY_IMAGE "${REPO_PREFIX}gateway:${V_OLD}"
  set_env FRONTEND_IMAGE "${REPO_PREFIX}frontend:${V_OLD}"
  set_env UI_UPDATES_ENABLED true
  set_env UPDATE_MANIFEST_URL "$MANIFEST_URL_IN_STACK"
  set_env UPDATE_IMAGE_REPO_PREFIX "$REPO_PREFIX"
  if [[ -s "$KEYS_DIR/public.key" ]]; then
    set_env UPDATE_PUBLIC_KEY "$(tr -d '\n' <"$KEYS_DIR/public.key")"
  fi
}

cmd_up() {
  have docker || die "docker is required"
  [[ -s "$DIGESTS_FILE" ]] || die "no images yet: run '$0 build' first"
  port_check "$UI_PORT"
  port_check "$NODE_PORT"
  # the tools are small and change with the code, so they are rebuilt on every start
  build_tools
  ensure_keys

  mkdir -p "$INSTALL_DIR/data" "$INSTALL_DIR/apps" "$MANIFEST_DIR"
  # the container user must be able to write the bind mounts
  chmod 777 "$INSTALL_DIR/data" "$INSTALL_DIR/apps"
  if [[ ! -f "$INSTALL_DIR/docker-compose.prod.yml" ]]; then
    # An install made before this release: its compose file lacks the newest setting, as a real one does. The
    # update swaps in the release's file, which only adds that line, so it loses nothing and needs approval.
    grep -v "$AGED_COMPOSE_LINE" "$E2E_DIR/compose-$V_OLD.yml" >"$INSTALL_DIR/docker-compose.prod.yml"
  fi
  write_env
  write_proxy_conf

  log "starting primary, gateway and frontend (version $V_OLD)"
  compose up -d --wait primary gateway frontend

  # the manifest server and the proxy join the stack's network by name, as plain containers
  docker rm -f "$PROXY_NAME" "$MANIFEST_NAME" >/dev/null 2>&1 || true
  docker run -d --name "$MANIFEST_NAME" --label "$E2E_LABEL" --network "$E2E_NETWORK" \
    -v "$MANIFEST_DIR:/usr/share/nginx/html:ro" nginx:alpine >/dev/null
  docker run -d --name "$PROXY_NAME" --label "$E2E_LABEL" --network "$E2E_NETWORK" \
    -p "127.0.0.1:${UI_PORT}:80" -v "$E2E_DIR/proxy.conf:/etc/nginx/conf.d/default.conf:ro" nginx:alpine >/dev/null
  wait_http "$UI_URL/api/health" 60
  wait_http "$UI_URL/" 60

  "$BIN_DIR/mint-session" --secret "$(env_get JWT_SECRET)" --login "$E2E_LOGIN" --out "$SESSION_FILE"
  log "the stack is up at $UI_URL (signed-in session: $SESSION_FILE)"
}

# ---------- publishing a release ----------------------------------------------------------------------------

# usage: publish <version> [--required KEY] [--optional KEY] [--generated KEY] [--tamper]
cmd_publish() {
  local version="${1:-}"
  [[ -n "$version" ]] || die "usage: publish <version> [--required KEY] [--optional KEY] [--generated KEY] [--tamper]"
  shift
  local tamper=0 settings=()
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --required) settings+=(--setting "$2:required:false:Needed by release $version"); shift 2 ;;
      --optional) settings+=(--setting "$2:optional:false:Optional in release $version"); shift 2 ;;
      --generated) settings+=(--setting "$2:generated:true:Created by the update"); shift 2 ;;
      --tamper) tamper=1; shift ;;
      *) die "unknown publish option: $1" ;;
    esac
  done
  [[ -s "$DIGESTS_FILE" ]] || die "no images yet: run '$0 build' first"
  ensure_keys
  local backend gateway frontend
  backend="$(digest_ref BACKEND "$version")"
  gateway="$(digest_ref GATEWAY "$version")"
  frontend="$(digest_ref FRONTEND "$version")"
  [[ -n "$backend" && -n "$gateway" && -n "$frontend" ]] || die "no images for version $version (built: $V_OLD, $V_NEW, $V_BROKEN)"

  mkdir -p "$MANIFEST_DIR"
  local tmp="$MANIFEST_DIR/.release.json.tmp" sig="$MANIFEST_DIR/.release.json.sig.tmp"
  "$BIN_DIR/releasetool" manifest --version "$version" --backend "$backend" --gateway "$gateway" --frontend "$frontend" \
    --image-prefix "$REPO_PREFIX" --compose-file "$E2E_DIR/compose-$version.yml" \
    --notes "End-to-end test release $version" --min-from "$V_OLD" ${settings[@]+"${settings[@]}"} --out "$tmp"
  "$BIN_DIR/releasetool" sign --key "$KEYS_DIR/private.key" --in "$tmp" --out "$sig"
  if [[ "$tamper" == 1 ]]; then
    # a valid signature over different content: the notes change after signing
    perl -pi -e 's/End-to-end test release/Tampered after signing/' "$tmp"
  fi
  # the two files replace the old ones together, so the server never serves a mismatched pair for long
  mv "$sig" "$MANIFEST_DIR/release.json.sig"
  mv "$tmp" "$MANIFEST_DIR/release.json"
  if [[ "$tamper" == 1 ]]; then log "published $version (content changed after signing)"; else log "published $version"; fi
}

cmd_unpublish() {
  rm -f "$MANIFEST_DIR/release.json" "$MANIFEST_DIR/release.json.sig"
  log "no release is published"
}

# ---------- state for the specs -----------------------------------------------------------------------------

# A running job that started a moment ago counts as work in progress for the update's preflight. The worker only
# claims pending jobs, and only marks a running job failed after JobStaleThreshold (30 minutes), so the row stays.
# The database is written while the primary is stopped, because a live SQLite file cannot be written from the host
# reliably.
cmd_job() {
  local action="${1:-}" db="$INSTALL_DIR/data/selfhostly.db" job_id="e2e-blocking-job" app_id
  have sqlite3 || die "sqlite3 is required on the host for this command"
  case "$action" in
    on)
      docker stop "$PRIMARY_NAME" >/dev/null
      app_id="$(sql "$db" 'SELECT id FROM apps LIMIT 1')"
      [[ -n "$app_id" ]] || { docker start "$PRIMARY_NAME" >/dev/null; die "create an app first: a job belongs to one"; }
      sql "$db" "INSERT OR REPLACE INTO jobs (id, type, app_id, status, claimed_by, claimed_at, started_at, updated_at) VALUES ('$job_id', 'app_update', '$app_id', 'running', 'e2e-harness', datetime('now'), datetime('now'), datetime('now'))"
      ;;
    off)
      docker stop "$PRIMARY_NAME" >/dev/null
      sql "$db" "DELETE FROM jobs WHERE id = '$job_id'"
      ;;
    *) die "usage: job on|off" ;;
  esac
  docker start "$PRIMARY_NAME" >/dev/null
  wait_healthy "$PRIMARY_NAME" 120
}

# ids of the app containers the stack's primary created (their compose files live in the install's apps directory)
cmd_app_containers() {
  local id
  for id in $(docker ps -q --filter "label=com.docker.compose.project=$E2E_APP_PROJECT"); do
    docker inspect --format '{{.Id}} {{.State.StartedAt}}' "$id"
  done
}

cmd_env_get() { env_get "${1:?usage: env-get KEY}"; }

# the running image reference of a stack service, for asserting what an update did
cmd_image() { docker inspect --format '{{.Config.Image}}' "${1:?usage: image <container>}"; }

# ---------- tests -----------------------------------------------------------------------------------------

cmd_test() {
  [[ -s "$SESSION_FILE" ]] || die "the stack is not up: run '$0 up' first"
  # the browser tests are their own package (web/e2e), so the frontend image build never installs a test runner
  cd "$ROOT/web/e2e"
  if [[ ! -d node_modules/@playwright/test ]]; then
    log "installing the browser test runner"
    npm install --no-audit --no-fund
  fi
  # E2E_BROWSER_CHANNEL=chrome uses the Chrome that is already installed, so nothing is downloaded
  [[ -n "${E2E_BROWSER_CHANNEL:-}" ]] || npx playwright install chromium
  E2E_BASE_URL="$UI_URL" E2E_SESSION_FILE="$SESSION_FILE" E2E_RUN_SH="$E2E/run.sh" \
    E2E_OLD_VERSION="$V_OLD" E2E_NEW_VERSION="$V_NEW" E2E_BROKEN_VERSION="$V_BROKEN" \
    E2E_INSTALL_DIR="$INSTALL_DIR" E2E_PRIMARY="$PRIMARY_NAME" E2E_REGISTRY_HOST="$REGISTRY_HOST" \
    npx playwright test -c playwright.config.ts "$@"
}

# ---------- teardown --------------------------------------------------------------------------------------

# ids of every container this harness owns: labelled ones, ones started from the local registry (the updater), and
# the app containers its primary created. The registry itself is only included on request.
e2e_containers() {
  local with_registry="${1:-}" id name dir
  {
    docker ps -aq --filter "label=$E2E_LABEL"
    docker ps -a --format '{{.ID}} {{.Image}}' | awk -v p="${REGISTRY_HOST}/" 'index($2, p) == 1 { print $1 }'
    # the app the specs deploy, and the updater containers the primary started for this install (they run in it)
    docker ps -aq --filter "label=com.docker.compose.project=$E2E_APP_PROJECT"
    for id in $(docker ps -aq --filter "label=selfhostly.updater"); do
      dir="$(docker inspect --format '{{.Config.WorkingDir}}' "$id" 2>/dev/null || true)"
      if [[ "$dir" == "$INSTALL_DIR" ]]; then echo "$id"; fi
    done
  } | sort -u | while read -r id; do
    [[ -n "$id" ]] || continue
    name="$(docker inspect --format '{{.Name}}' "$id" 2>/dev/null || true)"
    if [[ "$with_registry" == --with-registry || "$name" != "/$REGISTRY_NAME" ]]; then echo "$id"; fi
  done
}

cmd_down() {
  have docker || return 0
  local ids projects p
  if [[ -f "$INSTALL_DIR/docker-compose.prod.yml" ]]; then
    compose down --remove-orphans >/dev/null 2>&1 || true
  fi
  ids="$(e2e_containers)"
  if [[ -n "$ids" ]]; then
    # networks created for the apps this stack deployed go with them
    projects="$(for id in $ids; do docker inspect --format '{{index .Config.Labels "com.docker.compose.project"}}' "$id" 2>/dev/null; done | sort -u)"
    # shellcheck disable=SC2086
    docker rm -f $ids >/dev/null 2>&1 || true
    for p in $projects; do
      [[ -n "$p" && "$p" != "$E2E_PROJECT" ]] || continue
      docker network ls -q --filter "label=com.docker.compose.project=$p" | xargs -r docker network rm >/dev/null 2>&1 || true
    done
  fi
  docker network rm "$E2E_NETWORK" >/dev/null 2>&1 || true
  log "the stack is down (the registry and built images are kept)"
}

# removes the rollback tags the updater made for this install: only stamps that exist in its own .upgrade folder
remove_rollback_tags() {
  [[ -d "$INSTALL_DIR/.upgrade" ]] || return 0
  local stamp ref
  for stamp in $(ls "$INSTALL_DIR/.upgrade" 2>/dev/null); do
    for ref in $(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E "^[^ ]*rollback/[a-z]+:${stamp}\$" || true); do
      docker rmi "$ref" >/dev/null 2>&1 || true
    done
  done
}

cmd_purge() {
  cmd_down
  have docker || return 0
  remove_rollback_tags
  docker rm -f "$REGISTRY_NAME" >/dev/null 2>&1 || true
  docker volume rm "$REGISTRY_VOLUME" >/dev/null 2>&1 || true
  # by image id, because an image pulled by digest has no tag to name it by
  docker images --format '{{.ID}} {{.Repository}}' | awk -v p="${REGISTRY_HOST}/selfhostly-" 'index($2, p) == 1 { print $1 }' | sort -u | xargs -r docker rmi -f >/dev/null 2>&1 || true
  docker image prune -f --filter "label=selfhostly.e2e.version" >/dev/null 2>&1 || true
  if [[ -d "$E2E_DIR" ]]; then
    # root-owned files from the containers may not be removable by this user; a throwaway container removes them
    rm -rf "$E2E_DIR" 2>/dev/null || docker run --rm --label "$E2E_LABEL" -v "$E2E_DIR:/x" alpine sh -c 'rm -rf /x/* /x/.[!.]*' >/dev/null 2>&1 || true
    rm -rf "$E2E_DIR" 2>/dev/null || true
  fi
  log "everything the harness created is removed"
}

cmd_status() {
  docker ps -a --filter "label=$E2E_LABEL" --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
  [[ -s "$DIGESTS_FILE" ]] && echo && cat "$DIGESTS_FILE" || true
}

cmd_logs() { docker logs --tail "${2:-80}" "${1:?usage: logs <container> [lines]}"; }

cmd_all() {
  # a run starts from nothing: the registry and images are kept only with SKIP_BUILD=1
  if [[ "${SKIP_BUILD:-0}" == 1 ]]; then cmd_down; else cmd_purge; fi
  if [[ "${KEEP:-0}" != 1 ]]; then
    trap 'cmd_down' EXIT
  else
    trap 'echo "KEEP=1: the stack is left running at $UI_URL. Remove it with: $E2E/run.sh down"' EXIT
  fi
  [[ "${SKIP_BUILD:-0}" == 1 ]] || cmd_build
  cmd_up
  cmd_test "$@"
}

help() {
  cat <<EOF
usage: scripts/e2e/run.sh <command>

  all [playwright args]     purge, build, up, run the specs, tear down (SKIP_BUILD=1 reuses images, KEEP=1 leaves the stack)
  build                     build old, new and broken images and push them to the local registry (OLD_REF=<commit>)
  up                        start the stack, the proxy and the manifest server, sign in a test session
  publish <version> [opts]  publish a signed release: $V_NEW, or $V_BROKEN (never healthy)
                            opts: --required KEY | --optional KEY | --generated KEY | --tamper
  unpublish                 publish nothing
  job on|off                add or remove a job that counts as work in progress
  app-containers            ids and start times of the apps the stack deployed
  env-get KEY | image NAME  read the install's settings, or a container's image
  test [playwright args]    run the specs against the running stack
  status | logs NAME        what is running, or a container's log
  down                      remove the stack (keeps the registry and images)
  purge                     remove everything the harness created
EOF
}

main() {
  command="${1:-all}"
  [[ $# -gt 0 ]] && shift
  case "$command" in
    all) cmd_all "$@" ;;
    build) cmd_build ;;
    up) cmd_up ;;
    publish) cmd_publish "$@" ;;
    unpublish) cmd_unpublish ;;
    job) cmd_job "$@" ;;
    app-containers) cmd_app_containers ;;
    env-get) cmd_env_get "$@" ;;
    image) cmd_image "$@" ;;
    test) cmd_test "$@" ;;
    status) cmd_status ;;
    logs) cmd_logs "$@" ;;
    down) cmd_down ;;
    purge) cmd_purge ;;
    help | -h | --help) help ;;
    *) help >&2; exit 2 ;;
  esac
}

# sourcing this file only defines the functions, which is how they are tested one by one
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then main "$@"; fi
