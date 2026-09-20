import assert from 'node:assert/strict'
import { test } from 'node:test'
import { buildActivities } from '../src/features/app-details/lib/activity.ts'

const app = (patch: Record<string, unknown> = {}) =>
    ({ status: 'idle', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-06-01T00:00:00Z', ...patch }) as never
const job = (patch: Record<string, unknown>) =>
    ({ id: 'j', type: 'app_update', status: 'completed', created_at: '2026-03-01T00:00:00Z', ...patch }) as never

test('an app with no jobs shows its creation, and how it is right now', () => {
    const running = buildActivities(app({ status: 'running' }), [])
    assert.deepEqual(
        running.map((entry) => entry.id),
        ['status-running', 'create'],
    )
    const stopped = buildActivities(app({ status: 'stopped' }), undefined)
    assert.deepEqual(
        stopped.map((entry) => entry.id),
        ['status-stopped', 'create'],
    )
    assert.equal(stopped[0].tone, 'idle')
    assert.equal(stopped[0].icon, 'pause')
})

test('other statuses add no "currently" entry', () => {
    assert.deepEqual(
        buildActivities(app({ status: 'error' }), []).map((entry) => entry.id),
        ['create'],
    )
})

test('a create job replaces the made-up creation entry', () => {
    const list = buildActivities(app(), [job({ id: 'c', type: 'app_create' })])
    assert.ok(!list.some((entry) => entry.id === 'create'))
    assert.equal(list[0].description, 'App creation completed')
})

test('entries are newest first, using when the job finished, then started, then was created', () => {
    const list = buildActivities(app(), [
        job({ id: 'old', type: 'app_create', created_at: '2026-02-01T00:00:00Z' }),
        job({ id: 'done', completed_at: '2026-05-01T00:00:00Z', created_at: '2026-01-15T00:00:00Z' }),
        job({ id: 'started', started_at: '2026-04-01T00:00:00Z', created_at: '2026-01-16T00:00:00Z' }),
    ])
    assert.deepEqual(
        list.map((entry) => entry.id),
        ['done', 'started', 'old'],
    )
})

test('a job reads by its type and how it went', () => {
    const one = (patch: Record<string, unknown>) => buildActivities(app(), [job({ type: 'app_create', ...patch })])[0]
    assert.equal(one({ status: 'completed' }).description, 'App creation completed')
    assert.equal(one({ status: 'failed' }).description, 'App creation failed')
    assert.equal(one({ status: 'running' }).description, 'App creation in progress')
    assert.equal(one({ status: 'pending' }).description, 'App creation started')
    assert.equal(buildActivities(app(), [job({ type: 'mystery' })])[0].description, 'mystery completed')
})

test('tone and icon follow the job', () => {
    const one = (patch: Record<string, unknown>) => buildActivities(app(), [job(patch)])[0]
    assert.deepEqual([one({ status: 'failed' }).tone, one({ status: 'failed' }).icon], ['err', 'alert'])
    assert.deepEqual([one({ status: 'running' }).tone, one({ status: 'running' }).icon], ['info', 'loading'])
    assert.equal(one({ status: 'pending' }).tone, 'warn')
    assert.deepEqual([one({ type: 'quick_tunnel' }).tone, one({ type: 'quick_tunnel' }).icon], ['ok', 'zap'])
    assert.equal(one({ type: 'tunnel_create' }).icon, 'globe')
    assert.equal(one({ type: 'mystery' }).icon, 'refresh')
})

test('a failed job shows its error, and a bare exit status points at the logs', () => {
    const one = (message: string) => buildActivities(app(), [job({ status: 'failed', error_message: message })])[0]
    assert.equal(one('port is already allocated').details, 'port is already allocated')
    assert.equal(one('exit status 1').details, 'Build or deployment failed - click to view logs')
    assert.equal(one('  exit status 137 ').details, 'Build or deployment failed - click to view logs')
    assert.equal(one('exit status 1: image not found').details, 'exit status 1: image not found')
})

test('other jobs show their progress message', () => {
    const entry = buildActivities(app(), [
        job({ status: 'running', progress_message: 'Pulling', error_message: 'x' }),
    ])[0]
    assert.equal(entry.details, 'Pulling')
})
