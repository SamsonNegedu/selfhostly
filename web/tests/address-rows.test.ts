import assert from 'node:assert/strict'
import { test } from 'node:test'
import { buildAddressRows, statusFor } from '../src/features/cloudflare/lib/address-rows.ts'

// Only the fields these functions read.
const app = (patch: Record<string, unknown>) =>
    ({ id: 'a', name: 'app', public_url: 'https://x.example.com', ...patch }) as never
const tunnel = (patch: Record<string, unknown>) =>
    ({ app_id: 'a', status: 'active', is_active: true, ...patch }) as never

test('a Quick Tunnel is always temporary, whatever its tunnel says', () => {
    assert.deepEqual(statusFor(app({ tunnel_mode: 'quick' }), tunnel({ status: 'error' })), {
        kind: 'warn',
        label: 'Temporary',
    })
})

test('a custom domain with no tunnel record is unknown', () => {
    assert.deepEqual(statusFor(app({ tunnel_mode: 'custom' })), { kind: 'idle', label: 'Unknown' })
})

test('a tunnel in error is an error, and only an active running tunnel is active', () => {
    const custom = app({ tunnel_mode: 'custom' })
    assert.deepEqual(statusFor(custom, tunnel({ status: 'error' })), { kind: 'err', label: 'Error' })
    assert.deepEqual(statusFor(custom, tunnel({})), { kind: 'ok', label: 'Active' })
    assert.deepEqual(statusFor(custom, tunnel({ is_active: false })), { kind: 'idle', label: 'Inactive' })
    assert.deepEqual(statusFor(custom, tunnel({ status: 'degraded' })), { kind: 'idle', label: 'Inactive' })
})

test('rows keep only apps with a public address, matched to their tunnel and sorted by name', () => {
    const rows = buildAddressRows(
        [
            app({ id: 'b', name: 'zeta', tunnel_mode: 'custom' }),
            app({ id: 'c', name: 'private', public_url: '' }),
            app({ id: 'a', name: 'alpha', tunnel_mode: 'quick' }),
        ],
        [tunnel({ app_id: 'b' })],
    )
    assert.deepEqual(
        rows.map((row) => row.app.name),
        ['alpha', 'zeta'],
    )
    assert.equal(rows[0].kind, 'quick')
    assert.equal(rows[1].kind, 'custom')
    assert.ok(rows[1].tunnel)
    assert.equal(rows[0].tunnel, undefined)
    assert.equal(rows[1].status.label, 'Active')
})

test('no apps gives no rows', () => {
    assert.deepEqual(buildAddressRows([], []), [])
})
