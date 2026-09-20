import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
    ALL_DAYS,
    buildCron,
    describeDays,
    formatTime,
    parseSimpleCron,
    parseTime,
    runningWindow,
    windowHours,
} from '../src/features/app-details/lib/schedule-cron.ts'

test('a simple cron expression is read as a time and its days', () => {
    assert.deepEqual(parseSimpleCron('0 8 * * 1-5'), { hour: 8, minute: 0, days: [1, 2, 3, 4, 5] })
    assert.deepEqual(parseSimpleCron('30 22 * * *'), { hour: 22, minute: 30, days: ALL_DAYS })
    assert.deepEqual(parseSimpleCron('0 8 * * 6,0'), { hour: 8, minute: 0, days: [0, 6] })
})

test('expressions the simple form cannot show are left for the text form', () => {
    for (const text of [
        '*/5 * * * *',
        '0 8 1 * *',
        '0 8 * 6 1',
        '0 25 * * *',
        '61 8 * * *',
        '0 8 * * 8',
        '',
        'not cron',
    ]) {
        assert.equal(parseSimpleCron(text), null, text)
    }
})

test('building then reading gives the same schedule back', () => {
    for (const days of [ALL_DAYS, [1, 2, 3, 4, 5], [0, 6], [1], [1, 3, 5], [0, 1, 2, 4]]) {
        const schedule = { hour: 7, minute: 45, days }
        assert.deepEqual(parseSimpleCron(buildCron(schedule)), schedule, JSON.stringify(days))
    }
})

test('days are written as ranges only for runs of three or more', () => {
    assert.equal(buildCron({ hour: 8, minute: 0, days: [1, 2, 3, 4, 5] }), '0 8 * * 1-5')
    assert.equal(buildCron({ hour: 8, minute: 0, days: [1, 2] }), '0 8 * * 1,2')
    assert.equal(buildCron({ hour: 8, minute: 0, days: ALL_DAYS }), '0 8 * * *')
    assert.equal(buildCron({ hour: 8, minute: 0, days: [5, 1, 3] }), '0 8 * * 1,3,5')
})

test('times are written and read as HH:MM', () => {
    assert.equal(formatTime({ hour: 8, minute: 5 }), '08:05')
    assert.deepEqual(parseTime('22:30'), { hour: 22, minute: 30 })
    assert.equal(parseTime('8:30'), null)
    assert.equal(parseTime(''), null)
})

test('the running window is one segment, or two when it runs past midnight', () => {
    assert.deepEqual(runningWindow({ hour: 8, minute: 0 }, { hour: 22, minute: 0 }), [{ from: 480, to: 1320 }])
    assert.deepEqual(runningWindow({ hour: 22, minute: 0 }, { hour: 6, minute: 0 }), [
        { from: 1320, to: 1440 },
        { from: 0, to: 360 },
    ])
    assert.deepEqual(runningWindow({ hour: 8, minute: 0 }, { hour: 8, minute: 0 }), [])
})

test('hours per day add up across midnight', () => {
    assert.equal(windowHours({ hour: 8, minute: 0 }, { hour: 22, minute: 0 }), 14)
    assert.equal(windowHours({ hour: 22, minute: 0 }, { hour: 6, minute: 0 }), 8)
    assert.equal(windowHours({ hour: 8, minute: 0 }, { hour: 8, minute: 30 }), 0.5)
})

test('days are described in words', () => {
    assert.equal(describeDays(ALL_DAYS), 'every day')
    assert.equal(describeDays([1, 2, 3, 4, 5]), 'Mon to Fri')
    assert.equal(describeDays([6, 0]), 'Sun and Sat')
    assert.equal(describeDays([1, 3, 5]), 'Mon, Wed, Fri')
})
