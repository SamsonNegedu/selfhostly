// The GitHub Actions step for a deploy hook. It reads the instance's address, this app's id and the
// node it runs on from repository variables rather than baking them into the workflow file: the
// same step then works unedited if any of that ever changes (a new domain, a new tunnel, moving the
// app to a different node), and the file committed to the repo does not carry instance-specific
// data. node_id is required the same way it is on every other by-id request this app makes - a
// multi-node install's gateway routes purely on it, so a hook for an app on a secondary node has no
// other way to reach the node that actually holds it. The four ${{ }} values are pulled into env: once
// and validated before curl runs, so a variable or secret that was never set fails with a clear
// message instead of a curl error against a malformed URL. Any CI that can run curl works the same
// way - this is one example to paste, not the only way to call the endpoint.
export const GITHUB_ACTIONS_DEPLOY_STEP = [
    '- name: Trigger Selfhostly deploy hook',
    '  env:',
    '    SELFHOSTLY_URL: ${{ vars.SELFHOSTLY_URL }}',
    '    SELFHOSTLY_APP_ID: ${{ vars.SELFHOSTLY_APP_ID }}',
    '    SELFHOSTLY_NODE_ID: ${{ vars.SELFHOSTLY_NODE_ID }}',
    '    SELFHOSTLY_DEPLOY_TOKEN: ${{ secrets.SELFHOSTLY_DEPLOY_TOKEN }}',
    '  run: |',
    '    if [ -z "$SELFHOSTLY_URL" ] || [ -z "$SELFHOSTLY_APP_ID" ] || [ -z "$SELFHOSTLY_NODE_ID" ] || [ -z "$SELFHOSTLY_DEPLOY_TOKEN" ]; then',
    '      echo "Missing Selfhostly configuration - check the repository variables and secret are set." >&2',
    '      exit 1',
    '    fi',
    '    curl --fail --silent --show-error \\',
    '      --retry 3 --retry-delay 5 \\',
    '      -X POST "${SELFHOSTLY_URL}/api/apps/${SELFHOSTLY_APP_ID}/deploy-trigger?node_id=${SELFHOSTLY_NODE_ID}" \\',
    '      -H "Authorization: Bearer ${SELFHOSTLY_DEPLOY_TOKEN}"',
].join('\n')

// Loopback and private-network addresses: only reachable on this network, never from GitHub's
// hosted runners (a self-hosted runner on the same network is the exception). Flags a dashboard
// viewed through a LAN address or localhost, so that address is not copied into SELFHOSTLY_URL where
// it would silently never work from CI.
export function looksUnreachableFromHostedCI(hostname: string): boolean {
    const h = hostname.toLowerCase()
    if (h === 'localhost' || h.endsWith('.local')) return true
    if (h === '::1' || h.startsWith('127.')) return true
    if (h.startsWith('10.')) return true
    if (h.startsWith('192.168.')) return true
    if (/^172\.(1[6-9]|2\d|3[01])\./.test(h)) return true
    return false
}
