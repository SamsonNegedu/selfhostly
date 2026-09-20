import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
    blockedReason,
    buildChecks,
    nameError,
    portError,
    portSharedMessage,
    type NewAppRuleInput,
} from '../src/features/create-app/lib/new-app-rules.ts'

// A form that is ready to deploy. Each test changes only what it is about.
const ready: NewAppRuleInput = {
    mode: 'template',
    hasTemplate: true,
    content: 'services: {}',
    name: 'my-app',
    nameProblem: undefined,
    nameTaken: false,
    nodeId: 'n1',
    nodeName: 'Pi',
    portError: undefined,
    portShared: undefined,
    composeChecks: [],
    access: 'none',
    hasExposedPort: true,
    hostname: '',
}
const withInput = (patch: Partial<NewAppRuleInput>) => ({ ...ready, ...patch })
const ids = (input: NewAppRuleInput) => buildChecks(input).map((check) => `${check.id}:${check.level}`)

test('names: lowercase letters, numbers and hyphens, 3 to 63 characters', () => {
    assert.equal(nameError(''), undefined)
    assert.equal(nameError('my-app-2'), undefined)
    assert.match(nameError('My_App') ?? '', /lowercase/)
    assert.match(nameError('ab') ?? '', /at least 3/)
    assert.equal(nameError('abc'), undefined)
    assert.equal(nameError('a'.repeat(63)), undefined)
    assert.match(nameError('a'.repeat(64)) ?? '', /63/)
})

test('a port is only asked for from templates, and must be 1 to 65535', () => {
    assert.equal(portError('template', 8080), undefined)
    assert.equal(portError('template', 1), undefined)
    assert.equal(portError('template', 65535), undefined)
    for (const bad of [0, -1, 65536, 1.5, NaN])
        assert.match(portError('template', bad) ?? '', /1 to 65535/, String(bad))
    assert.equal(portError('paste', 0), undefined)
    assert.equal(portError('link', NaN), undefined)
})

test('a port another app already publishes is a warning that names the node', () => {
    const used = new Set([8080])
    assert.equal(portSharedMessage('template', 8080, false, used, 'Pi'), 'Another app on Pi already publishes 8080.')
    assert.equal(portSharedMessage('template', 9090, false, used, 'Pi'), undefined)
    assert.equal(portSharedMessage('template', 8080, true, used, 'Pi'), undefined)
    assert.equal(portSharedMessage('paste', 8080, false, used, 'Pi'), undefined)
})

test('a valid, free name and an online node give two passing checks', () => {
    assert.deepEqual(ids(ready), ['name:ok', 'node:ok'])
})

test('no name yet adds no name check', () => {
    assert.deepEqual(ids(withInput({ name: '' })), ['node:ok'])
})

test('a taken name is an error that says which node has it', () => {
    const [name] = buildChecks(withInput({ nameTaken: true }))
    assert.equal(name.level, 'err')
    assert.equal(name.title, 'Name already used')
    assert.equal(name.detail, 'Pi already has an app called my-app.')
})

test('an invalid name is an error carrying the reason', () => {
    const [name] = buildChecks(withInput({ name: 'Bad', nameProblem: 'Use lowercase letters.' }))
    assert.equal(name.title, 'Name is not valid')
    assert.equal(name.detail, 'Use lowercase letters.')
})

test('no online node is an error', () => {
    const node = buildChecks(withInput({ nodeId: '' })).find((check) => check.id === 'node')
    assert.equal(node?.level, 'err')
})

test('a shared port is a warning, and checks of the compose file are passed through', () => {
    const extra = { id: 'yaml', level: 'ok' as const, title: 'Valid YAML' }
    const list = ids(withInput({ portShared: 'shared', composeChecks: [extra] }))
    assert.deepEqual(list, ['name:ok', 'node:ok', 'port-shared:warn', 'yaml:ok'])
})

test('a Quick Tunnel needs a published port', () => {
    assert.ok(ids(withInput({ access: 'quick', hasExposedPort: false })).includes('quick:err'))
    assert.ok(!ids(withInput({ access: 'quick', hasExposedPort: true })).includes('quick:err'))
    assert.ok(!ids(withInput({ access: 'none', hasExposedPort: false })).includes('quick:err'))
})

test('your own domain needs a hostname', () => {
    assert.ok(ids(withInput({ access: 'custom', hostname: '   ' })).includes('host:err'))
    assert.ok(!ids(withInput({ access: 'custom', hostname: 'app.example.com' })).includes('host:err'))
})

test('a form with nothing wrong is not blocked', () => {
    assert.equal(blockedReason(ready, buildChecks(ready)), undefined)
})

test('the blocking reason follows the order things are fixed in', () => {
    const reason = (patch: Partial<NewAppRuleInput>) => {
        const input = withInput(patch)
        return blockedReason(input, buildChecks(input))
    }
    assert.equal(reason({ hasTemplate: false, content: '', name: '' }), 'Choose a template')
    assert.equal(reason({ mode: 'paste', content: ' ', name: '' }), 'Add a compose file')
    assert.equal(reason({ mode: 'link', content: '', name: '' }), 'Fetch a compose file first')
    assert.equal(reason({ name: '' }), 'Give the app a name')
    assert.equal(reason({ nameTaken: true }), 'Fix the problems in the checks')
    assert.equal(reason({ portError: 'bad port' }), 'Fix the problems in the checks')
    assert.equal(reason({ nodeId: '' }), 'Fix the problems in the checks')
})
