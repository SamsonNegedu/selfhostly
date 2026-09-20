import { expect, type Page } from '@playwright/test'

// Pulling images, recreating three containers and waiting for health is slow, and a rollback adds another round.
export const REVIEW_TIMEOUT_MS = 4 * 60_000
export const UPDATE_TIMEOUT_MS = 5 * 60_000

export const SETTINGS_UPDATES_URL = '/settings?section=updates'

/** Opens Settings, Updates as a signed-in user. */
export async function openUpdates(page: Page) {
    await page.goto(SETTINGS_UPDATES_URL)
    await expect(page.getByTestId('updates-section')).toBeVisible()
}

/** Asks the primary to fetch the manifest now, then waits for the screen to show the outcome. */
export async function checkForUpdate(page: Page) {
    await page.getByTestId('update-check-button').click()
}

/** Starts the review (pull and plan) and waits for it to finish, whatever it decided. */
export async function reviewUpdate(page: Page) {
    await page.getByTestId('update-review-button').click()
    await expect(page.getByTestId('update-plan-state')).toHaveText(/ready|blocked|failed/i, {
        timeout: REVIEW_TIMEOUT_MS,
    })
}

/** Approves the compose change when the release has one, which the plan shows as a diff. */
export async function approveComposeIfShown(page: Page) {
    const approve = page.getByTestId('update-approve-compose')
    if (await approve.isVisible()) await approve.check()
}

/** Confirms by typing the version, then starts the update. */
export async function confirmAndApply(page: Page, version: string) {
    await page.getByTestId('update-apply-button').click()
    await page.getByTestId('update-confirm-input').fill(version)
    await page.getByTestId('update-confirm-button').click()
}
