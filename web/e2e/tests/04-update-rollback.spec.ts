import { expect, test } from '@playwright/test'
import { BROKEN_VERSION, NEW_VERSION, appContainers, envGet, publish, unpublish, updateState } from '../helpers/harness'
import {
    UPDATE_TIMEOUT_MS,
    approveComposeIfShown,
    checkForUpdate,
    confirmAndApply,
    openUpdates,
    reviewUpdate,
} from '../helpers/updates-ui'

// A release whose primary never becomes healthy. The update must put the previous version back by itself and say why.

test.afterAll(() => {
    unpublish()
})

test('a release that does not start is rolled back and the old version keeps running', async ({ page, request }) => {
    const before = await updateState(request)
    test.skip(
        before.current_version !== NEW_VERSION,
        `needs the install on ${NEW_VERSION}, it is on ${before.current_version}`,
    )
    const imageBefore = envGet('BACKEND_IMAGE')
    const appsBefore = appContainers()

    publish(BROKEN_VERSION)
    await openUpdates(page)
    await checkForUpdate(page)
    await expect(page.getByTestId('update-available-version')).toContainText(BROKEN_VERSION)
    await reviewUpdate(page)
    await expect(page.getByTestId('update-plan-state')).toHaveText(/ready/i)
    await approveComposeIfShown(page)
    await confirmAndApply(page, BROKEN_VERSION)

    await expect(page.getByTestId('update-progress')).toBeVisible()
    await expect(page.getByTestId('update-result')).toHaveText(/rolled back/i, { timeout: UPDATE_TIMEOUT_MS })
    // the screen says why, not only that
    await expect(page.getByTestId('update-result')).toHaveText(/healthy|start|refus/i)

    await expect(page.getByTestId('update-current-version')).toContainText(NEW_VERSION)
    const after = await updateState(request)
    expect(after.current_version).toBe(NEW_VERSION)
    expect(after.run?.state).toBe('rolled_back')
    expect(after.run?.message ?? '').not.toBe('')

    // the settings and the apps are as they were, and the stack answers
    expect(envGet('BACKEND_IMAGE')).toBe(imageBefore)
    expect(appContainers()).toEqual(appsBefore)
    expect((await request.get('/api/health')).ok()).toBeTruthy()
})
