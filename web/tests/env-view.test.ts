import assert from 'node:assert/strict'
import { test } from 'node:test'
import { readEnv, type EnvEntry } from '../src/features/app-details/lib/compose-env.ts'
import { envChecks, envFlag, MASK, rawEnvLines, rowId } from '../src/features/app-details/lib/env-view.ts'

const entry = (service: string, key: string, value: string): EnvEntry => ({ service, key, value }) as EnvEntry

test('a variable is identified by its service and name', () => {
    assert.equal(rowId(entry('web', 'PORT', '1')), 'web:PORT')
    assert.notEqual(rowId(entry('web', 'PORT', '1')), rowId(entry('db', 'PORT', '1')))
})

test('a variable is new, changed, or unchanged against what is saved', () => {
    const saved = [entry('web', 'PORT', '80')]
    assert.equal(envFlag(saved, entry('web', 'HOST', 'x')), 'new')
    assert.equal(envFlag(saved, entry('db', 'PORT', '80')), 'new')
    assert.equal(envFlag(saved, entry('web', 'PORT', '81')), 'changed')
    assert.equal(envFlag(saved, entry('web', 'PORT', '80')), null)
})

const compose =
    'services:\n  web:\n    image: x\n    environment:\n      PORT: "80"\n      API_TOKEN: abc\n  db:\n    image: y\n    environment:\n      MODE: ""\n'

test('the checks count variables and services, with singular and plural wording', () => {
    const many = envChecks(readEnv(compose), compose)
    assert.equal(many[0].title, '3 variables across 2 services')

    const one = 'services:\n  web:\n    image: x\n    environment:\n      PORT: "80"\n'
    assert.equal(envChecks(readEnv(one), one)[0].title, '1 variable across 1 service')
})

test('empty values are listed as a warning', () => {
    const checks = envChecks(readEnv(compose), compose)
    const empty = checks.find((check) => check.id === 'empty')
    assert.equal(empty?.level, 'warn')
    assert.equal(empty?.detail, 'MODE')
})

test('a reference with no value and no default is a warning', () => {
    const text = 'services:\n  web:\n    image: ${IMAGE}\n    environment:\n      PORT: "80"\n'
    const unresolved = envChecks(readEnv(text), text).find((check) => check.id === 'unresolved')
    assert.equal(unresolved?.level, 'warn')
    assert.match(unresolved?.detail ?? '', /\$\{IMAGE\}/)
})

test('a file that is not valid YAML gives a single error check', () => {
    const bad = 'services:\n  web: [\n'
    const checks = envChecks(readEnv(bad), bad)
    assert.equal(checks.length, 1)
    assert.equal(checks[0].id, 'yaml')
    assert.equal(checks[0].level, 'err')
})

test('the raw view groups by service and hides secrets until revealed', () => {
    const { entries } = readEnv(compose)
    const hidden = rawEnvLines(['web', 'db'], entries, new Set()).map((line) => line.text)
    assert.deepEqual(hidden, ['# web', 'PORT=80', `API_TOKEN=${MASK}`, '', '# db', 'MODE='])

    const shown = rawEnvLines(['web', 'db'], entries, new Set(['web:API_TOKEN'])).map((line) => line.text)
    assert.ok(shown.includes('API_TOKEN=abc'))
})

test('a service with no variables is left out of the raw view', () => {
    const { entries } = readEnv(compose)
    const lines = rawEnvLines(['cache', 'db'], entries, new Set()).map((line) => line.text)
    assert.deepEqual(lines, ['# db', 'MODE='])
})
