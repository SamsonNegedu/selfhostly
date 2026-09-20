import assert from 'node:assert/strict'
import { test } from 'node:test'
import { filterContainers } from '../src/features/monitoring/lib/container-filter.ts'

const container = (name: string, app_name: string, state: string) => ({ name, app_name, state }) as never
const all = [
    container('web-1', 'Website', 'running'),
    container('db-1', 'Website', 'stopped'),
    container('cache', 'Redis', 'running'),
]
const names = (list: { name: string }[]) => list.map((item) => item.name)

test('with no filter and no search, everything is shown', () => {
    assert.deepEqual(names(filterContainers(all, 'all', '')), ['web-1', 'db-1', 'cache'])
})

test('the state filter keeps only that state', () => {
    assert.deepEqual(names(filterContainers(all, 'running', '')), ['web-1', 'cache'])
    assert.deepEqual(names(filterContainers(all, 'stopped', '')), ['db-1'])
})

test('the search matches the container name or its app name, ignoring case', () => {
    assert.deepEqual(names(filterContainers(all, 'all', 'WEB')), ['web-1', 'db-1'])
    assert.deepEqual(names(filterContainers(all, 'all', 'redis')), ['cache'])
    assert.deepEqual(names(filterContainers(all, 'all', 'db')), ['db-1'])
})

test('spaces around the search are ignored, and a blank search matches everything', () => {
    assert.deepEqual(names(filterContainers(all, 'all', '  cache ')), ['cache'])
    assert.equal(filterContainers(all, 'all', '   ').length, 3)
})

test('the state and the search both have to match', () => {
    assert.deepEqual(names(filterContainers(all, 'running', 'website')), ['web-1'])
    assert.deepEqual(filterContainers(all, 'stopped', 'redis'), [])
})
