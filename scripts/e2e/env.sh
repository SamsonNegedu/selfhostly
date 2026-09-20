# Names, ports and paths of the end-to-end stack, in one place. Sourced by run.sh.
#
# Everything the harness creates carries the `e2e` prefix and the label selfhostly.e2e=1, and cleanup removes only
# resources that match. The ports are high and unusual so they cannot collide with a developer's stack.

E2E_LABEL="selfhostly.e2e=1"
E2E_PROJECT="selfhostly-e2e"
E2E_NETWORK="e2e-network"

# host ports, all on loopback
REGISTRY_PORT="${E2E_REGISTRY_PORT:-45050}"
UI_PORT="${E2E_UI_PORT:-45052}"
NODE_PORT="${E2E_NODE_PORT:-45053}"

REGISTRY_NAME="e2e-registry"
REGISTRY_VOLUME="e2e-registry-data"
PROXY_NAME="e2e-proxy"
MANIFEST_NAME="e2e-manifest"
PRIMARY_NAME="e2e-primary"
GATEWAY_NAME="e2e-gateway"
FRONTEND_NAME="e2e-frontend"

REGISTRY_HOST="localhost:${REGISTRY_PORT}"
REPO_PREFIX="${REGISTRY_HOST}/selfhostly-"

# the login the test browser signs in as (it is the only entry in GITHUB_ALLOWED_USERS)
E2E_LOGIN="e2e-user"

# release versions: the install starts on OLD, the update goes to NEW, and BROKEN starts but never becomes healthy
V_OLD="1.0.0"
V_NEW="1.1.0"
V_BROKEN="1.2.0"

# where the harness keeps its files (tmp/ is git-ignored)
E2E_DIR="$ROOT/tmp/e2e"
INSTALL_DIR="$E2E_DIR/install"
KEYS_DIR="$E2E_DIR/keys"
MANIFEST_DIR="$E2E_DIR/manifest"
BIN_DIR="$E2E_DIR/bin"
DIGESTS_FILE="$E2E_DIR/digests.env"
SESSION_FILE="$E2E_DIR/session.json"
# the line the "older" install lacks in its compose file: the newest setting the release passes through
AGED_COMPOSE_LINE="UPDATE_CHECK_INTERVAL_HOURS"

UI_URL="http://localhost:${UI_PORT}"
# how the primary reaches the manifest server: by name, on the stack's network
MANIFEST_URL_IN_STACK="http://${MANIFEST_NAME}/release.json"

# The test app the specs deploy is named e2e-web, so its compose project is too. The primary runs compose inside its
# own container, so the project's folder label holds a container path (/app/apps/e2e-web), not a host path: the name is
# what tells this app from anyone else's.
E2E_APP_PROJECT="e2e-web"
