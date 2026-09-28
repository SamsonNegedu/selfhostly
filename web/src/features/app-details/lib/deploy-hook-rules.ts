// The GitHub Actions step for a deploy hook. It reads the instance's address, this app's id and the
// node it runs on from repository variables rather than baking them into the workflow file: the
// same step then works unedited if any of that ever changes (a new domain, a new tunnel, moving the
// app to a different node), and the file committed to the repo does not carry instance-specific
// data. node_id is required the same way it is on every other by-id request this app makes - a
// multi-node install's gateway routes purely on it, so a hook for an app on a secondary node has no
// other way to reach the node that actually holds it. Any CI that can run curl works the same way -
// this is one example to paste, not the only way to call the endpoint.
export const GITHUB_ACTIONS_DEPLOY_STEP = [
    '- name: Trigger Selfhostly deploy hook',
    '  run: |',
    '    curl --fail --silent --show-error \\',
    '      --retry 3 --retry-delay 5 \\',
    '      -X POST "${{ vars.SELFHOSTLY_URL }}/api/apps/${{ vars.SELFHOSTLY_APP_ID }}/deploy-trigger?node_id=${{ vars.SELFHOSTLY_NODE_ID }}" \\',
    '      -H "Authorization: Bearer ${{ secrets.SELFHOSTLY_DEPLOY_TOKEN }}"',
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
