import assert from 'node:assert/strict'
import { test } from 'node:test'
import { actionCopy, groupContainers } from '../src/features/monitoring/lib/container-groups.ts'

const container = (patch: Record<string, unknown>) =>
    ({ id: 'c', name: 'c1', state: 'running', is_managed: true, node_id: 'n1', app_name: 'web', ...patch }) as never
const app = (name: string, node_id: string) => ({ id: `${node_id}-${name}`, name, node_id }) as never

test('containers of the same app on the same node share a group', () => {
    const groups = groupContainers([container({ id: '1' }), container({ id: '2', name: 'c2' })], [app('web', 'n1')])
    assert.equal(groups.length, 1)
    assert.equal(groups[0].containers.length, 2)
    assert.equal(groups[0].name, 'web')
    assert.ok(groups[0].app)
})

test('the same app name on two nodes is two groups, each linked to its own app', () => {
    const groups = groupContainers(
        [container({ node_id: 'n1' }), container({ node_id: 'n2' })],
        [app('web', 'n1'), app('web', 'n2')],
    )
    assert.equal(groups.length, 2)
    assert.deepEqual(groups.map((group) => (group.app as unknown as { node_id: string }).node_id).sort(), ['n1', 'n2'])
})

test('containers Selfhostly did not start are kept in one group, after the apps', () => {
    const groups = groupContainers(
        [
            container({ id: 'x', is_managed: false, app_name: '' }),
            container({ id: 'y', is_managed: false, app_name: '', node_id: 'n2' }),
            container({ id: 'z', app_name: 'zeta' }),
            container({ id: 'a', app_name: 'alpha' }),
        ],
        [],
    )
    assert.deepEqual(
        groups.map((group) => group.name),
        ['alpha', 'zeta', 'Not managed by Selfhostly'],
    )
    assert.equal(groups[2].containers.length, 2)
    assert.equal(groups[2].managed, false)
    assert.equal(groups[2].app, undefined)
})

test('a managed container whose app is not in the list still gets a group, without a link', () => {
    const [group] = groupContainers([container({ app_name: 'ghost' })], [])
    assert.equal(group.name, 'ghost')
    assert.equal(group.app, undefined)
})

test('restart says start for a stopped container', () => {
    const stopped = actionCopy('restart', container({ state: 'stopped', name: 'db' }))
    assert.equal(stopped.title, 'Start db?')
    assert.equal(stopped.text, 'Start')
    const running = actionCopy('restart', container({ name: 'db' }))
    assert.equal(running.title, 'Restart db?')
    assert.equal(running.text, 'Restart')
})

test('stop and delete confirmations name the container', () => {
    assert.equal(actionCopy('stop', container({ name: 'db' })).title, 'Stop db?')
    assert.equal(actionCopy('delete', container({ name: 'db' })).title, 'Delete db?')
    assert.equal(actionCopy('delete', container({ name: 'db' })).text, 'Delete container')
})

test('deleting a managed container warns that it can break its app', () => {
    assert.match(actionCopy('delete', container({ app_name: 'web' })).body, /belongs to web/)
    assert.doesNotMatch(actionCopy('delete', container({ is_managed: false })).body, /belongs to/)
})
