import { expect, test } from '@playwright/test'
import { APP_DISPLAY_NAME, OLD_VERSION, ensureApp, runningImage } from '../helpers/harness'

// The stack is the production topology behind a proxy, and the browser holds a signed session. These checks say
// the environment is what the update specs assume, so a failure later is about the update and not the setup.

test('the UI loads through the proxy for a signed-in user', async ({ page, request }) => {
    const me = await request.get('/api/me')
    expect(me.ok()).toBeTruthy()
    expect((await me.json()).name).toBe('e2e-user')

    await page.goto('/apps')
    await expect(page.getByRole('heading', { name: 'Fleet' })).toBeVisible()
})

test('sign-in is still enforced: a request without the session is refused', async ({ playwright, baseURL }) => {
    const anonymous = await playwright.request.newContext({ baseURL, storageState: { cookies: [], origins: [] } })
    const apps = await anonymous.get('/api/apps')
    expect(apps.status()).toBe(401)
    await anonymous.dispose()
})

test('an app deployed through the API appears in the fleet', async ({ page, request }) => {
    await ensureApp(request)
    await page.goto('/apps')
    await expect(page.getByText(APP_DISPLAY_NAME).first()).toBeVisible()
})

test('the install starts on the old version', () => {
    expect(runningImage('e2e-primary')).toContain(`:${OLD_VERSION}`)
})
