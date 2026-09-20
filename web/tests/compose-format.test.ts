import assert from 'node:assert/strict'
import { test } from 'node:test'
import { formatCompose } from '../src/features/app-details/lib/compose-format.ts'

test('port mappings and times stay quoted, so no parser reads them as base 60 numbers', () => {
    const out = formatCompose(
        'services:\n  a:\n    image: x\n    ports:\n      - "22:22"\n      - 8080:80/tcp\n      - "127.0.0.1:8080:80"\n',
    )
    assert.match(out, /- "22:22"/)
    assert.match(out, /- "8080:80\/tcp"/)
    assert.match(out, /- "127\.0\.0\.1:8080:80"/)
})

test('other strings are left plain', () => {
    const out = formatCompose(
        'services:\n  a:\n    image: nginx:latest\n    command: run --fast\n    volumes:\n      - ./data:/data\n',
    )
    assert.match(out, /image: nginx:latest/)
    assert.match(out, /command: run --fast/)
    assert.match(out, /- \.\/data:\/data/)
})

test('empty volumes, networks and services print as a bare name', () => {
    const out = formatCompose('services:\n  a:\n    image: x\nvolumes:\n  data:\n  cache: null\nnetworks:\n  net: {}\n')
    assert.match(out, /^ {2}data:$/m)
    assert.match(out, /^ {2}cache:$/m)
    assert.match(out, /^ {2}net:$/m)
    assert.doesNotMatch(out, /null|\{\}/)
})

test('a null that is not an empty definition stays null', () => {
    const out = formatCompose('services:\n  a:\n    image: x\n    environment:\n      KEY: null\n')
    assert.match(out, /KEY: null/)
})

test('indentation becomes two spaces and long lines are not wrapped', () => {
    const long = 'word '.repeat(60).trim()
    const out = formatCompose(`services:\n        a:\n                image: x\n                command: ${long}\n`)
    assert.match(out, /^ {2}a:$/m)
    assert.match(out, /^ {4}image: x$/m)
    assert.ok(out.includes(`command: ${long}`))
})

test('key order is kept', () => {
    const out = formatCompose('services:\n  a:\n    ports: []\n    image: x\n    restart: always\n')
    assert.ok(out.indexOf('ports') < out.indexOf('image') && out.indexOf('image') < out.indexOf('restart'))
})

test('formatting twice changes nothing more', () => {
    const once = formatCompose('services:\n  a:\n    image: x\n    ports:\n      - "22:22"\nvolumes:\n  data:\n')
    assert.equal(formatCompose(once), once)
})

test('a file that is not valid YAML throws the parser message', () => {
    assert.throws(() => formatCompose('services:\n  a: [\n'), /flow|bracket|\]/i)
})
