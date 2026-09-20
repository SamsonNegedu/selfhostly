import { execFileSync } from 'node:child_process'
import { expect, type APIRequestContext } from '@playwright/test'

// What scripts/e2e/run.sh passes in. The specs never build or start anything themselves: they ask the harness.
const runSh = process.env.E2E_RUN_SH ?? ''
export const OLD_VERSION = process.env.E2E_OLD_VERSION ?? '1.0.0'
export const NEW_VERSION = process.env.E2E_NEW_VERSION ?? '1.1.0'
export const BROKEN_VERSION = process.env.E2E_BROKEN_VERSION ?? '1.2.0'
export const REGISTRY_HOST = process.env.E2E_REGISTRY_HOST ?? ''

const HARNESS_TIMEOUT_MS = 3 * 60_000

/** Runs one harness command and returns what it printed. Throws with the harness output when it fails. */
export function harness(...args: string[]): string {
    if (!runSh) throw new Error('E2E_RUN_SH is not set: run the specs with scripts/e2e/run.sh')
    try {
        return execFileSync(runSh, args, { encoding: 'utf8', timeout: HARNESS_TIMEOUT_MS }).trim()
    } catch (error) {
        const failure = error as { stdout?: string; stderr?: string; message: string }
        throw new Error(
            `run.sh ${args.join(' ')} failed:\n${failure.stdout ?? ''}${failure.stderr ?? ''}${failure.message}`,
        )
    }
}

export const publish = (version: string, ...options: string[]) => harness('publish', version, ...options)
export const unpublish = () => harness('unpublish')
export const envGet = (key: string) => harness('env-get', key)
export const runningImage = (container: string) => harness('image', container)

/** id and start time of each app container the stack deployed. Equal before and after means they were not restarted. */
export const appContainers = (): string[] => {
    const out = harness('app-containers')
    return out === '' ? [] : out.split('\n').sort()
}

const APP_NAME = 'e2e-web'
const APP_COMPOSE = `services:
  web:
    image: nginx:alpine
    restart: unless-stopped
`
const APP_TIMEOUT_MS = 2 * 60_000
const POLL_INTERVAL_MS = 2_000

/** The primary's node id, needed to address an app. */
async function localNodeId(request: APIRequestContext): Promise<string> {
    const response = await request.get('/api/node/info')
    expect(response.ok()).toBeTruthy()
    return (await response.json()).id as string
}

/** Deploys a tiny app through the real API and waits until it runs, so an update has an app to leave alone. */
export async function ensureApp(request: APIRequestContext): Promise<void> {
    const nodeId = await localNodeId(request)
    const listApps = async () => {
        const response = await request.get('/api/apps')
        expect(response.ok()).toBeTruthy()
        return ((await response.json()) ?? []) as Array<{ id: string; name: string; status: string }>
    }

    let app = (await listApps()).find((candidate) => candidate.name === APP_NAME)
    if (!app) {
        const created = await request.post('/api/apps', {
            data: { name: APP_NAME, description: 'end to end test app', compose_content: APP_COMPOSE, node_id: nodeId },
        })
        expect(created.status(), await created.text()).toBe(201)
        app = (await listApps()).find((candidate) => candidate.name === APP_NAME)
    }
    // a new app is created stopped
    if (app && app.status !== 'running') {
        const started = await request.post(`/api/apps/${app.id}/start?node_id=${nodeId}`)
        expect(started.ok(), await started.text()).toBeTruthy()
    }
    await expect
        .poll(async () => (await listApps()).find((candidate) => candidate.name === APP_NAME)?.status, {
            timeout: APP_TIMEOUT_MS,
            intervals: [POLL_INTERVAL_MS],
        })
        .toBe('running')
}

export const APP_DISPLAY_NAME = APP_NAME

/** The update state as the API reports it, for asserting what the screen should also show. */
export async function updateState(request: APIRequestContext) {
    const response = await request.get('/api/system/update')
    expect(response.ok()).toBeTruthy()
    return (await response.json()) as {
        enabled: boolean
        current_version: string
        available: { version: string } | null
        plan: { state: string; blockers: Array<{ code: string }> } | null
        run: { state: string; phase: string; message: string; to_version: string } | null
    }
}
