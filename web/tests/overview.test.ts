import assert from 'node:assert/strict'
import { test } from 'node:test'
import { readComposeResources } from '../src/features/app-details/lib/compose-resources.ts'
import { formatDayAgo, hostOf } from '../src/shared/lib/format.ts'

test('networks and volumes are the names under the top-level sections', () => {
    const out = readComposeResources(
        'version: "3"\nservices:\n  a:\n    image: x\nnetworks:\n  front:\n  back:\n    driver: bridge\nvolumes:\n  data:\n  cache: {}\n',
    )
    assert.deepEqual(out.networks, ['front', 'back'])
    assert.deepEqual(out.volumes, ['data', 'cache'])
})

test('comments and blank lines are ignored', () => {
    const out = readComposeResources('# top\nservices:\n  a: {}\n\nnetworks:\n  # first\n  net:\n')
    assert.deepEqual(out.networks, ['net'])
})

test('a repeated name is listed once', () => {
    assert.deepEqual(readComposeResources('networks:\n  n:\n  n:\n').networks, ['n'])
})

test('with no named volumes, bind mounts are counted instead', () => {
    const many = readComposeResources(
        'services:\n  a:\n    volumes:\n      - /srv/a:/a\n      - ~/b:/b\n      - "./c:/c"\n',
    )
    assert.deepEqual(many.volumes, ['2 bind mounts'])
    const one = readComposeResources('services:\n  a:\n    volumes:\n      - /srv/a:/a\n')
    assert.deepEqual(one.volumes, ['1 bind mount'])
})

test('named volumes win over counting bind mounts', () => {
    const out = readComposeResources('services:\n  a:\n    volumes:\n      - /srv/a:/a\nvolumes:\n  data:\n')
    assert.deepEqual(out.volumes, ['data'])
})

test('an empty file has nothing', () => {
    assert.deepEqual(readComposeResources(''), { networks: [], volumes: [] })
})

test('a date reads as today, yesterday, days ago, then the date', () => {
    const now = new Date('2026-09-20T12:00:00Z')
    const ago = (days: number) => new Date(now.getTime() - days * 86400000 - 1000).toISOString()
    assert.equal(formatDayAgo(new Date(now.getTime() - 3600000).toISOString(), now), 'Today')
    assert.equal(formatDayAgo(ago(1), now), 'Yesterday')
    assert.equal(formatDayAgo(ago(3), now), '3 days ago')
    assert.equal(formatDayAgo(ago(6), now), '6 days ago')
    const old = ago(10)
    assert.equal(
        formatDayAgo(old, now),
        new Date(old).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }),
    )
    assert.match(formatDayAgo(old, now), /^Sep (9|10|11), 2026$/)
})

test('the host of an address, or the text itself when it is not a URL', () => {
    assert.equal(hostOf('https://app.example.com/path?x=1'), 'app.example.com')
    assert.equal(hostOf('http://localhost:8080'), 'localhost:8080')
    assert.equal(hostOf('not a url'), 'not a url')
})
