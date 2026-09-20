import { expect, test } from '@playwright/test'
import { NEW_VERSION, ensureApp, harness, publish, unpublish, updateState } from '../helpers/harness'
import { checkForUpdate, openUpdates, reviewUpdate } from '../helpers/updates-ui'

// Things that must stop an update before anything is changed. Each ends with the old version still running.

const REQUIRED_KEY = 'E2E_NEW_REQUIRED'
const REQUIRED_VALUE = 'a-value-only-the-operator-knows'

test.describe.configure({ mode: 'serial' })

test.beforeAll(async ({ request }) => {
    await ensureApp(request)
})

test.afterAll(() => {
    unpublish()
})

test('a request without the session cannot read or start an update', async ({ playwright, baseURL }) => {
    const anonymous = await playwright.request.newContext({ baseURL, storageState: { cookies: [], origins: [] } })
    expect((await anonymous.get('/api/system/update')).status()).toBe(401)
    const apply = await anonymous.post('/api/system/update/apply', { data: { version: NEW_VERSION } })
    expect(apply.status()).toBe(401)
    await anonymous.dispose()
})

test('a release signed with the wrong content is refused and nothing is planned', async ({ page, request }) => {
    publish(NEW_VERSION, '--tamper')
    await openUpdates(page)
    await checkForUpdate(page)

    await expect
        .poll(async () => (await request.get('/api/system/update').then((r) => r.json())).check_error, {
            intervals: [1_000],
        })
        .toMatch(/signature/i)
    const state = await updateState(request)
    expect(state.available).toBeNull()
    expect(state.plan).toBeNull()
    await expect(page.getByTestId('update-available-version')).toHaveCount(0)

    // the API will not plan a version that was never verified, even when asked for it by name
    const plan = await request.post('/api/system/update/plan', { data: { version: NEW_VERSION } })
    expect(plan.ok()).toBeFalsy()
})

test('a setting the release requires blocks the update until it is filled in', async ({ page }) => {
    publish(NEW_VERSION, '--required', REQUIRED_KEY)
    await openUpdates(page)
    await checkForUpdate(page)
    await expect(page.getByTestId('update-available-version')).toContainText(NEW_VERSION)
    await reviewUpdate(page)

    const input = page.getByTestId(`update-input-${REQUIRED_KEY}`)
    await expect(input).toBeVisible()
    await expect(page.getByTestId('update-apply-button')).toBeDisabled()
    await input.fill(REQUIRED_VALUE)
    await expect(page.getByTestId('update-apply-button')).toBeEnabled()
    // nothing was applied: the value is not written until the update runs
    expect(harness('env-get', REQUIRED_KEY)).toBe('')
})

test('a running deployment job blocks the update', async ({ page, request }) => {
    publish(NEW_VERSION)
    harness('job', 'on')
    try {
        await openUpdates(page)
        await checkForUpdate(page)
        await reviewUpdate(page)

        await expect(page.getByTestId('update-blocker').first()).toBeVisible()
        await expect(page.getByTestId('update-apply-button')).toBeDisabled()
        const state = await updateState(request)
        expect(state.plan?.blockers.map((blocker) => blocker.code)).toContain('job_running')
    } finally {
        harness('job', 'off')
    }
})
