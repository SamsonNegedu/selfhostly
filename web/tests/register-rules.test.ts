import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
    findConflicts,
    findJoinedNode,
    isRegistrationValid,
    joinSettings,
} from '../src/features/nodes/lib/register-rules.ts'

const node = (id: string, name: string) => ({ id, name }) as never
const nodes = [node('primary', 'Pi Primary'), node('nas', 'Garage NAS')]
const form = (patch: Record<string, string> = {}) =>
    ({ id: 'new', name: 'New box', api_endpoint: 'http://10.0.0.5:8080', api_key: 'k', ...patch }) as never

test('an ID is taken when a node already has it, ignoring spaces around it', () => {
    assert.equal(findConflicts(form({ id: 'nas' }), nodes).idTaken, true)
    assert.equal(findConflicts(form({ id: '  nas ' }), nodes).idTaken, true)
    assert.equal(findConflicts(form({ id: 'NAS' }), nodes).idTaken, false)
    assert.equal(findConflicts(form({ id: '' }), nodes).idTaken, false)
})

test('a name is taken when a node has it, whatever the case', () => {
    assert.equal(findConflicts(form({ name: 'garage nas' }), nodes).nameTaken, true)
    assert.equal(findConflicts(form({ name: ' PI PRIMARY ' }), nodes).nameTaken, true)
    assert.equal(findConflicts(form({ name: 'Other' }), nodes).nameTaken, false)
    assert.equal(findConflicts(form({ name: '  ' }), nodes).nameTaken, false)
})

test('a registration is valid only when every field is filled and nothing clashes', () => {
    assert.equal(isRegistrationValid(form(), nodes), true)
    for (const field of ['id', 'name', 'api_endpoint', 'api_key']) {
        assert.equal(isRegistrationValid(form({ [field]: '   ' }), nodes), false, field)
    }
    assert.equal(isRegistrationValid(form({ id: 'nas' }), nodes), false)
    assert.equal(isRegistrationValid(form({ name: 'garage nas' }), nodes), false)
})

test('the settings for the new machine list the primary, the token and its own address', () => {
    const text = joinSettings(' https://home.example.com ', ' http://10.0.0.5:8080 ', 'tok123')
    assert.deepEqual(text.split('\n'), [
        'NODE_IS_PRIMARY=false',
        'PRIMARY_NODE_URL=https://home.example.com',
        'REGISTRATION_TOKEN=tok123',
        'NODE_API_ENDPOINT=http://10.0.0.5:8080',
    ])
})

test('until the machine address is known, a placeholder shows in its place', () => {
    assert.match(joinSettings('https://a', '  ', 't'), /NODE_API_ENDPOINT=http:\/\/<this-machine-address>:8080/)
})

test('the machine that joined is the first one that was not there when the token was made', () => {
    const known = new Set(['primary', 'nas'])
    assert.equal(findJoinedNode(nodes, known), undefined)
    assert.equal(findJoinedNode([...nodes, node('fresh', 'Fresh')], known)?.id, 'fresh')
    assert.equal(findJoinedNode(undefined, known), undefined)
})
