import assert from 'node:assert/strict'
import { test } from 'node:test'
import { isProviderReady, parseProviderConfig } from '../src/features/settings/lib/provider-config.ts'

test('nothing saved gives an empty form', () => {
    assert.deepEqual(parseProviderConfig(''), { config: {}, masked: {} })
    assert.deepEqual(parseProviderConfig(undefined), { config: {}, masked: {} })
    assert.deepEqual(parseProviderConfig(null), { config: {}, masked: {} })
})

test('text that is not JSON gives an empty form, not an error', () => {
    assert.deepEqual(parseProviderConfig('{not json'), { config: {}, masked: {} })
})

test('a masked token is kept as a hint and blanked in the form', () => {
    const { config, masked } = parseProviderConfig(
        JSON.stringify({ cloudflare: { api_token: 'abcd****wxyz', account_id: 'acc1' } }),
    )
    assert.deepEqual(masked, { cloudflare: 'abcd****wxyz' })
    assert.deepEqual(config, { cloudflare: { api_token: '', account_id: 'acc1' } })
})

test('a token with no mask is left as it is', () => {
    const { config, masked } = parseProviderConfig(
        JSON.stringify({ cloudflare: { api_token: 'plain', account_id: 'a' } }),
    )
    assert.deepEqual(masked, {})
    assert.equal(config.cloudflare?.api_token, 'plain')
})

test('each provider is handled on its own', () => {
    const { config, masked } = parseProviderConfig(
        JSON.stringify({ cloudflare: { api_token: 'x****y' }, other: { account_id: 'o' } }),
    )
    assert.deepEqual(Object.keys(masked), ['cloudflare'])
    assert.deepEqual(config.other, { account_id: 'o' })
})

test('Cloudflare needs an account ID and either a typed or a saved token', () => {
    assert.equal(isProviderReady('cloudflare', {}, {}), false)
    assert.equal(isProviderReady('cloudflare', { api_token: 't' }, {}), false)
    assert.equal(isProviderReady('cloudflare', { account_id: 'a' }, {}), false)
    assert.equal(isProviderReady('cloudflare', { api_token: 't', account_id: 'a' }, {}), true)
    assert.equal(isProviderReady('cloudflare', { account_id: 'a' }, { cloudflare: 'x****y' }), true)
    assert.equal(isProviderReady('cloudflare', { api_token: 't', account_id: '' }, {}), false)
})

test('other providers need nothing typed here', () => {
    assert.equal(isProviderReady('other', {}, {}), true)
})
