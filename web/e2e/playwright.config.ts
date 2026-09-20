import { defineConfig, devices } from '@playwright/test'

// Set by scripts/e2e/run.sh, which owns the stack these specs drive.
const baseURL = process.env.E2E_BASE_URL ?? 'http://localhost:45052'
const sessionFile = process.env.E2E_SESSION_FILE

const ACTION_TIMEOUT_MS = 15_000
// An update pulls, recreates three containers and waits for health, so a spec that runs one needs minutes.
const SPEC_TIMEOUT_MS = 6 * 60_000
const EXPECT_TIMEOUT_MS = 20_000

export default defineConfig({
    testDir: './tests',
    outputDir: '../../tmp/e2e/playwright-results',
    // The specs change one shared install, so they run one at a time and in file order.
    fullyParallel: false,
    workers: 1,
    retries: 0,
    timeout: SPEC_TIMEOUT_MS,
    expect: { timeout: EXPECT_TIMEOUT_MS },
    reporter: [['list'], ['html', { outputFolder: '../../tmp/e2e/playwright-report', open: 'never' }]],
    use: {
        baseURL,
        storageState: sessionFile,
        actionTimeout: ACTION_TIMEOUT_MS,
        trace: 'retain-on-failure',
        screenshot: 'only-on-failure',
        video: 'retain-on-failure',
    },
    // E2E_BROWSER_CHANNEL=chrome runs the tests in the Chrome that is already installed, so no browser is downloaded
    projects: [
        {
            name: 'chromium',
            use: { ...devices['Desktop Chrome'], channel: process.env.E2E_BROWSER_CHANNEL || undefined },
        },
    ],
})
