import assert from 'node:assert/strict'
import { test } from 'node:test'
import { buildApplyRequest, evaluatePlan } from '../src/shared/lib/update.ts'

const plan = (patch: Record<string, unknown> = {}) =>
    ({
        version: 'v2.0.0',
        state: 'ready',
        blockers: [],
        compose: { state: 'current', diff: '' },
        settings: { required_missing: [], generated: [], optional: [] },
        ...patch,
    }) as never
const required = (...keys: string[]) => ({
    required_missing: keys.map((key) => ({ key })),
    generated: [],
    optional: [],
})

test('a ready plan with nothing to fill in can be started', () => {
    const result = evaluatePlan(plan(), {}, false, false)
    assert.equal(result.startable, true)
    assert.equal(result.canApply, true)
})

test('it cannot be started while another run or request holds it', () => {
    assert.equal(evaluatePlan(plan(), {}, false, true).canApply, false)
})

test('preparing and failed plans cannot be started', () => {
    assert.equal(evaluatePlan(plan({ state: 'preparing' }), {}, false, false).startable, false)
    assert.equal(evaluatePlan(plan({ state: 'failed' }), {}, false, false).startable, false)
})

test('a compose file that is behind needs approval first', () => {
    const behind = plan({ compose: { state: 'behind', diff: '-a\n+b', approval_token: 't' } })
    assert.equal(evaluatePlan(behind, {}, false, false).needsApproval, true)
    assert.equal(evaluatePlan(behind, {}, false, false).canApply, false)
    assert.equal(evaluatePlan(behind, {}, true, false).canApply, true)
})

test('a customized compose file is not something approval can fix', () => {
    const customized = plan({ compose: { state: 'customized', diff: '' } })
    assert.equal(evaluatePlan(customized, {}, true, false).needsApproval, false)
})

test('missing settings block until every one has a value', () => {
    const blocked = plan({
        state: 'blocked',
        blockers: [{ code: 'missing_required', message: 'needs X' }],
        settings: required('A', 'B'),
    })
    const none = evaluatePlan(blocked, {}, false, false)
    assert.equal(none.startable, true)
    assert.equal(none.canApply, false)
    assert.deepEqual(none.hardBlockers, [])
    assert.equal(evaluatePlan(blocked, { A: 'x' }, false, false).canApply, false)
    assert.equal(evaluatePlan(blocked, { A: 'x', B: '  ' }, false, false).canApply, false)
    assert.equal(evaluatePlan(blocked, { A: 'x', B: 'y' }, false, false).canApply, true)
})

test('any other blocker stops the update, even with every setting filled in', () => {
    const blocked = plan({
        state: 'blocked',
        blockers: [
            { code: 'missing_required', message: 'needs X' },
            { code: 'disk_full', message: 'no room' },
        ],
        settings: required('A'),
    })
    const result = evaluatePlan(blocked, { A: 'x' }, false, false)
    assert.deepEqual(
        result.hardBlockers.map((blocker: { code: string }) => blocker.code),
        ['disk_full'],
    )
    assert.equal(result.startable, false)
    assert.equal(result.canApply, false)
})

test('a blocked plan with no missing settings cannot be started', () => {
    assert.equal(
        evaluatePlan(plan({ state: 'blocked', blockers: [{ code: 'x', message: 'm' }] }), {}, false, false).startable,
        false,
    )
})

test('the request carries the version and the trimmed settings', () => {
    const request = buildApplyRequest(plan({ settings: required('A', 'B') }), { A: '  one ', B: 'two' })
    assert.deepEqual(request, { version: 'v2.0.0', inputs: { A: 'one', B: 'two' } })
})

test('the request carries the approval token only when the compose file is behind', () => {
    const behind = plan({ compose: { state: 'behind', diff: '', approval_token: 'tok' } })
    assert.equal(buildApplyRequest(behind, {}).approve_compose, 'tok')
    const current = plan({ compose: { state: 'current', diff: '', approval_token: 'tok' } })
    assert.equal('approve_compose' in buildApplyRequest(current, {}), false)
    const noToken = plan({ compose: { state: 'behind', diff: '' } })
    assert.equal('approve_compose' in buildApplyRequest(noToken, {}), false)
})
