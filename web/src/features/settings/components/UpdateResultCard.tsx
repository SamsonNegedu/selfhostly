import { useState } from 'react'
import { RotateCcw } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { canRollBack, formatUpdateTime, isRollbackRun } from '@/shared/lib/update'
import type { StatusKind } from '@/shared/lib/status'
import type { UpdateRun, UpdateRunState } from '@/shared/types/api'
import UpdateSteps from './UpdateSteps'

interface Outcome {
    label: string
    kind: StatusKind
    title: (run: UpdateRun) => string
    detail: string
}

const OUTCOMES: Partial<Record<UpdateRunState, Outcome>> = {
    succeeded: {
        label: 'Succeeded',
        kind: 'ok',
        title: (run) => `Updated to ${run.to_version}`,
        detail: '',
    },
    rolled_back: {
        label: 'Rolled back',
        kind: 'warn',
        title: (run) => `The update to ${run.to_version} failed and was rolled back`,
        detail: 'You are back on the version you had. Nothing was lost.',
    },
    failed: {
        label: 'Failed',
        kind: 'err',
        title: (run) => `The update to ${run.to_version} failed`,
        detail: 'Check that Selfhostly is working. You can try to roll back to the version you had.',
    },
    interrupted: {
        label: 'Interrupted',
        kind: 'warn',
        title: (run) => `The update to ${run.to_version} was interrupted`,
        detail: 'The updater stopped before it finished. Roll back, or review the update and try again.',
    },
}

// A rollback that ended has its own words: "the update failed and was rolled back" would be wrong for it.
const ROLLBACK_OUTCOMES: Partial<Record<UpdateRunState, Pick<Outcome, 'title' | 'detail'>>> = {
    rolled_back: {
        title: (run) => (run.to_version ? `Rolled back to ${run.to_version}` : 'Rolled back'),
        detail: 'The images, settings and compose file are back as they were. Your apps were not touched.',
    },
    failed: {
        title: () => 'The rollback failed',
        detail: 'Check that Selfhostly is working, or restore from a backup: see the operations guide.',
    },
}

interface UpdateResultCardProps {
    run: UpdateRun
    // True while another request keeps the buttons from being used.
    locked: boolean
    rollingBack: boolean
    onRollback: () => void
    onRetry: () => void
}

// How the last update ended, with the way back where one exists.
function UpdateResultCard({ run, locked, rollingBack, onRollback, onRetry }: UpdateResultCardProps) {
    const [confirming, setConfirming] = useState(false)
    const base = OUTCOMES[run.state]
    if (!base) return null
    const outcome = isRollbackRun(run) ? { ...base, ...ROLLBACK_OUTCOMES[run.state] } : base
    const finished = formatUpdateTime(run.finished_at)

    return (
        <div data-testid="update-result" data-state={run.state} className="flex flex-col gap-3">
            <div className="flex flex-wrap items-center gap-3">
                <h2 className="text-base font-semibold">{outcome.title(run)}</h2>
                <StatusPill kind={outcome.kind}>{outcome.label}</StatusPill>
            </div>
            {finished && <p className="text-compact text-muted-foreground">Finished {finished}.</p>}
            {outcome.detail && <p className="text-compact text-muted-foreground">{outcome.detail}</p>}
            {run.message && <p className="text-compact">{run.message}</p>}
            {run.warnings.length > 0 && (
                <ul aria-label="Warnings" className="flex flex-col gap-1 text-compact text-status-warn-fg">
                    {run.warnings.map((warning) => (
                        <li key={warning}>{warning}</li>
                    ))}
                </ul>
            )}
            <UpdateSteps steps={run.steps} />
            {canRollBack(run) && (
                <div className="flex flex-wrap gap-2">
                    <Button
                        variant="outline"
                        data-testid="update-rollback-button"
                        onClick={() => setConfirming(true)}
                        disabled={locked || rollingBack}
                    >
                        <RotateCcw className="h-4 w-4" />
                        Roll back
                    </Button>
                    {run.state === 'interrupted' && (
                        <Button variant="outline" data-testid="update-retry-button" onClick={onRetry} disabled={locked}>
                            Review and retry
                        </Button>
                    )}
                </div>
            )}
            <ConfirmationDialog
                open={confirming}
                onOpenChange={setConfirming}
                title={`Roll back to ${run.from_version}?`}
                description="This restores the previous images, settings and compose file. This page and the API are unavailable for 10 to 30 seconds. Your apps keep running."
                confirmText="Roll back"
                isLoading={rollingBack}
                onConfirm={() => {
                    setConfirming(false)
                    onRollback()
                }}
            />
        </div>
    )
}

export default UpdateResultCard
