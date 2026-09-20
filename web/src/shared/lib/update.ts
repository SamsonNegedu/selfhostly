import type { CodeLine } from '@/shared/components/ui/CodeBlock'
import type {
    ApplyUpdateRequest,
    UpdatePlan,
    UpdateRun,
    UpdateRunState,
    UpdateStatus,
    UpdateStep,
} from '@/shared/types/api'

// The updater reports steps by name. Known names get a sentence a person can read, others are shown as sent.
const STEP_LABELS: Record<string, string> = {
    verify: 'Verify the release',
    rollback_point: 'Save a rollback point',
    pull: 'Pull the new images',
    configure: 'Apply settings and compose file',
    dry_start: 'Test the new version',
    database: 'Copy the database',
    primary: 'Restart the primary',
    gateway: 'Restart the gateway',
    frontend: 'Restart the web interface',
    apps: 'Check your apps',
}

const ACTIVE_RUN_STATES: readonly UpdateRunState[] = ['pending', 'running']
const FINISHED_RUN_STATES: readonly UpdateRunState[] = ['succeeded', 'rolled_back', 'failed']
const ROLLBACK_RUN_STATES: readonly UpdateRunState[] = ['succeeded', 'failed', 'interrupted']
const PERCENT = 100
// Shown before the updater has reported its steps, so the bar is never empty while work is under way.
const STARTING_PERCENT = 5

export const isRunActive = (run: UpdateRun | null | undefined): boolean =>
    !!run && ACTIVE_RUN_STATES.includes(run.state)

// Ended by itself, one way or the other. An interrupted run is not finished: nobody knows how far it got.
export const isRunFinished = (run: UpdateRun | null | undefined): boolean =>
    !!run && FINISHED_RUN_STATES.includes(run.state)

export const isRunEnded = (run: UpdateRun | null | undefined): boolean =>
    isRunFinished(run) || run?.state === 'interrupted'

export const canRollBack = (run: UpdateRun | null | undefined): boolean =>
    !!run && ROLLBACK_RUN_STATES.includes(run.state)

// True while something is happening that the page should keep polling for.
export const isUpdateBusy = (status: UpdateStatus | null | undefined): boolean =>
    !!status && (status.plan?.state === 'preparing' || isRunActive(status.run))

export const isRollbackRun = (run: UpdateRun | null | undefined): boolean => run?.kind === 'rollback'

// What the run is doing, in a sentence: an update names the version it goes to, a rollback the one it returns to.
export const runHeading = (run: UpdateRun): string => {
    if (isRollbackRun(run))
        return run.to_version ? `Rolling back Selfhostly to ${run.to_version}` : 'Rolling back Selfhostly'
    return `Updating Selfhostly to ${run.to_version}`
}

export const stepLabel = (name: string): string => STEP_LABELS[name] ?? name.replace(/_/g, ' ')

export function runPercent(run: UpdateRun): number {
    if (run.steps.length === 0) return STARTING_PERCENT
    const done = run.steps.filter((step: UpdateStep) => step.state === 'done').length
    return Math.max(STARTING_PERCENT, Math.round((done / run.steps.length) * PERCENT))
}

export function formatUpdateTime(iso: string): string {
    const date = new Date(iso)
    return Number.isNaN(date.getTime()) ? '' : date.toLocaleString()
}

const DIFF_FILE_HEADERS = ['--- ', '+++ ']

// Reads a unified diff into lines the code block can color. File headers and hunk markers stay as plain lines.
export function diffToLines(diff: string): CodeLine[] {
    if (!diff.trim()) return []
    return diff
        .replace(/\n$/, '')
        .split('\n')
        .map((line): CodeLine => {
            if (DIFF_FILE_HEADERS.some((header) => line.startsWith(header)) || line.startsWith('@@')) {
                return { text: line }
            }
            if (line.startsWith('+')) return { text: line.slice(1), mark: 'add' }
            if (line.startsWith('-')) return { text: line.slice(1), mark: 'del' }
            return { text: line.startsWith(' ') ? line.slice(1) : line }
        })
}

// A blocker the update form can clear. Every other blocker needs something done outside the page.
const RESOLVED_BY_INPUTS = 'missing_required'

// Whether a reviewed update can be started, given what has been typed and approved so far.
export function evaluatePlan(plan: UpdatePlan, values: Record<string, string>, approved: boolean, locked: boolean) {
    const { compose, settings } = plan
    const needsApproval = compose.state === 'behind'
    const hardBlockers = plan.blockers.filter((blocker) => blocker.code !== RESOLVED_BY_INPUTS)
    const inputsComplete = settings.required_missing.every((setting) => (values[setting.key] ?? '').trim() !== '')
    const startable =
        plan.state === 'ready' ||
        (plan.state === 'blocked' && settings.required_missing.length > 0 && hardBlockers.length === 0)
    const canApply = startable && hardBlockers.length === 0 && inputsComplete && (!needsApproval || approved) && !locked
    return { needsApproval, hardBlockers, inputsComplete, startable, canApply }
}

// The request that starts the update: the typed settings, and the token that approves the compose changes when
// the release changes the compose file.
export function buildApplyRequest(plan: UpdatePlan, values: Record<string, string>): ApplyUpdateRequest {
    const { compose, settings } = plan
    const inputs = Object.fromEntries(
        settings.required_missing.map((setting) => [setting.key, (values[setting.key] ?? '').trim()]),
    )
    return {
        version: plan.version,
        inputs,
        ...(compose.state === 'behind' && compose.approval_token ? { approve_compose: compose.approval_token } : {}),
    }
}
