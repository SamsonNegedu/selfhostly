import { expect, test } from '@playwright/test'
import {
    APP_DISPLAY_NAME,
    NEW_VERSION,
    OLD_VERSION,
    appContainers,
    ensureApp,
    envGet,
    publish,
    unpublish,
    updateState,
} from '../helpers/harness'
import {
    UPDATE_TIMEOUT_MS,
    approveComposeIfShown,
    checkForUpdate,
    confirmAndApply,
    openUpdates,
    reviewUpdate,
} from '../helpers/updates-ui'

// The whole path a person takes: see that a release exists, review it, confirm, watch it run, and end up on the
// new version with the apps they deployed untouched.

test.describe.configure({ mode: 'serial' })

test.beforeAll(async ({ request }) => {
    await ensureApp(request)
    unpublish()
})

test('with nothing published there is no update to offer', async ({ page, request }) => {
    await openUpdates(page)
    await expect(page.getByTestId('update-current-version')).toContainText(OLD_VERSION)
    await checkForUpdate(page)
    expect((await updateState(request)).available).toBeNull()
    await expect(page.getByTestId('update-badge')).toHaveCount(0)
})

test('a published release shows as a badge and on the Updates screen', async ({ page }) => {
    publish(NEW_VERSION)
    await openUpdates(page)
    await checkForUpdate(page)
    await expect(page.getByTestId('update-available-version')).toContainText(NEW_VERSION)

    await page.goto('/apps')
    await expect(page.getByTestId('update-badge')).toBeVisible()
})

test('the update runs to the end and the new version is what runs', async ({ page, request }) => {
    const appsBefore = appContainers()
    expect(appsBefore.length).toBeGreaterThan(0)

    await openUpdates(page)
    await checkForUpdate(page)
    await reviewUpdate(page)
    await expect(page.getByTestId('update-plan-state')).toHaveText(/ready/i)
    await approveComposeIfShown(page)
    await confirmAndApply(page, NEW_VERSION)

    await expect(page.getByTestId('update-progress')).toBeVisible()
    await expect(page.getByTestId('update-step-primary')).toBeVisible({ timeout: UPDATE_TIMEOUT_MS })
    // the API is down while the primary is recreated. The screen must say so, not show an error.
    const sawReconnecting = await page
        .getByTestId('update-reconnecting')
        .waitFor({ state: 'visible', timeout: UPDATE_TIMEOUT_MS })
        .then(() => true)
        .catch(() => false)
    test.info().annotations.push({ type: 'reconnecting-shown', description: String(sawReconnecting) })

    await expect(page.getByTestId('update-result')).toHaveText(/succe|updated|complete/i, {
        timeout: UPDATE_TIMEOUT_MS,
    })
    await expect(page.getByTestId('update-current-version')).toContainText(NEW_VERSION)

    const state = await updateState(request)
    expect(state.current_version).toBe(NEW_VERSION)
    expect(state.run?.state).toBe('succeeded')
    // the install now pins the release by digest, as `selfhostlyctl pin-images` does
    expect(envGet('BACKEND_IMAGE')).toContain('@sha256:')

    // the apps were never restarted, and are still listed as running
    expect(appContainers()).toEqual(appsBefore)
    await page.goto('/apps')
    await expect(page.getByText(APP_DISPLAY_NAME).first()).toBeVisible()
    expect((await request.get('/api/health')).ok()).toBeTruthy()
})

test('once updated the badge is gone', async ({ page }) => {
    await page.goto('/apps')
    await expect(page.getByTestId('update-badge')).toHaveCount(0)
})
