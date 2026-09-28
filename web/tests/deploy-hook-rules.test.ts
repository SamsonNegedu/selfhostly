import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
    GITHUB_ACTIONS_DEPLOY_STEP,
    looksUnreachableFromHostedCI,
} from '../src/features/app-details/lib/deploy-hook-rules.ts'

test('the workflow step reads the instance, app id and node id from variables, never bakes them in', () => {
    assert.match(
        GITHUB_ACTIONS_DEPLOY_STEP,
        /-X POST "\$\{\{ vars\.SELFHOSTLY_URL \}\}\/api\/apps\/\$\{\{ vars\.SELFHOSTLY_APP_ID \}\}\/deploy-trigger\?node_id=\$\{\{ vars\.SELFHOSTLY_NODE_ID \}\}"/,
    )
    assert.match(GITHUB_ACTIONS_DEPLOY_STEP, /Authorization: Bearer \$\{\{ secrets\.SELFHOSTLY_DEPLOY_TOKEN \}\}/)
    assert.doesNotMatch(GITHUB_ACTIONS_DEPLOY_STEP, /sfd_/) // never bakes in an actual token
    assert.doesNotMatch(GITHUB_ACTIONS_DEPLOY_STEP, /:\/\/(?!\$)/) // no literal scheme://host baked in either
})

test('the step retries and fails the job on a bad response, instead of failing silently', () => {
    assert.match(GITHUB_ACTIONS_DEPLOY_STEP, /--fail/)
    assert.match(GITHUB_ACTIONS_DEPLOY_STEP, /--retry 3/)
})

test('loopback and private-network addresses are flagged as unreachable from hosted runners', () => {
    for (const hostname of [
        'localhost',
        'raspberrypi.local',
        '127.0.0.1',
        '10.0.0.5',
        '192.168.1.50',
        '172.16.0.1',
        '172.31.255.255',
    ]) {
        assert.equal(looksUnreachableFromHostedCI(hostname), true, hostname)
    }
})

test('a public domain is not flagged', () => {
    for (const hostname of ['selfhostly.example.com', 'my-tunnel.trycloudflare.com', '203.0.113.5']) {
        assert.equal(looksUnreachableFromHostedCI(hostname), false, hostname)
    }
    // 172.32.x and 172.15.x are outside the private 172.16-172.31 range
    assert.equal(looksUnreachableFromHostedCI('172.32.0.1'), false)
    assert.equal(looksUnreachableFromHostedCI('172.15.0.1'), false)
})
