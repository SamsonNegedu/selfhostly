import { useState } from 'react'
import { Download, Loader2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { describeError } from '@/shared/lib/errors'
import { buildApplyRequest, evaluatePlan } from '@/shared/lib/update'
import type { StatusKind } from '@/shared/lib/status'
import type { ApplyUpdateRequest, UpdatePlan, UpdatePlanState } from '@/shared/types/api'
import ComposePlanSection from './ComposePlanSection'
import ConfirmUpdateDialog from './ConfirmUpdateDialog'
import PlanBlockerList from './PlanBlockerList'
import RequiredSettingsSection from './RequiredSettingsSection'

const PLAN_STATE_META: Record<UpdatePlanState, { label: string; kind: StatusKind }> = {
    preparing: { label: 'Preparing', kind: 'info' },
    ready: { label: 'Ready', kind: 'ok' },
    blocked: { label: 'Blocked', kind: 'warn' },
    failed: { label: 'Failed', kind: 'err' },
}

interface UpdatePlanPanelProps {
    plan: UpdatePlan
    // True while another run or request keeps the update from starting.
    locked: boolean
    starting: boolean
    startError: unknown
    onApply: (request: ApplyUpdateRequest) => void
}

// The result of reviewing a release: what stops it, what changes in the compose file, which settings it needs.
// Start it from here once everything is answered. Keyed by version, so a new review starts with an empty form.
function UpdatePlanPanel({ plan, locked, starting, startError, onApply }: UpdatePlanPanelProps) {
    const [values, setValues] = useState<Record<string, string>>({})
    const [approved, setApproved] = useState(false)
    const [confirming, setConfirming] = useState(false)

    const meta = PLAN_STATE_META[plan.state]
    const { compose, settings } = plan
    const { hardBlockers, startable, canApply } = evaluatePlan(plan, values, approved, locked)

    const start = () => {
        setConfirming(false)
        onApply(buildApplyRequest(plan, values))
    }

    return (
        <div className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center gap-3">
                <h3 className="text-sm font-semibold">Review of {plan.version}</h3>
                <StatusPill kind={meta.kind} data-testid="update-plan-state" data-state={plan.state}>
                    {meta.label}
                </StatusPill>
            </div>

            {plan.state === 'preparing' && (
                <p role="status" className="flex items-center gap-2 text-compact text-muted-foreground">
                    <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                    Pulling the new version and checking your install. This can take a minute.
                </p>
            )}

            {plan.state === 'failed' && (
                <p role="alert" className="text-compact text-status-err-fg">
                    {plan.error || 'The review could not finish.'}
                </p>
            )}

            {plan.blockers.length > 0 && <PlanBlockerList blockers={plan.blockers} />}

            {plan.state !== 'preparing' && plan.state !== 'failed' && (
                <ComposePlanSection compose={compose} approved={approved} onApprovedChange={setApproved} />
            )}

            {settings.required_missing.length > 0 && (
                <RequiredSettingsSection
                    required={settings.required_missing}
                    values={values}
                    onChange={(key, value) => setValues({ ...values, [key]: value })}
                />
            )}

            {settings.generated.length > 0 && (
                <p className="text-compact text-muted-foreground">
                    Created for you and saved in your settings file:{' '}
                    <code className="font-mono">{settings.generated.join(', ')}</code>
                </p>
            )}
            {settings.optional.length > 0 && (
                <p className="text-compact text-muted-foreground">
                    New optional settings, left at their defaults:{' '}
                    <code className="font-mono">{settings.optional.join(', ')}</code>
                </p>
            )}

            {startError !== null && startError !== undefined && (
                <p role="alert" className="text-compact text-status-err-fg">
                    {describeError(startError)}
                </p>
            )}

            {startable && (
                <div className="flex flex-wrap items-center gap-3">
                    <Button data-testid="update-apply-button" onClick={() => setConfirming(true)} disabled={!canApply}>
                        <Download className="h-4 w-4" />
                        Update now
                    </Button>
                    {!canApply && !locked && (
                        <p className="text-compact text-muted-foreground">
                            {hardBlockers.length > 0
                                ? 'Resolve what stops this update first.'
                                : 'Fill in every setting and approve the compose changes to continue.'}
                        </p>
                    )}
                </div>
            )}

            <ConfirmUpdateDialog
                open={confirming}
                onOpenChange={setConfirming}
                version={plan.version}
                isLoading={starting}
                onConfirm={start}
            />
        </div>
    )
}

export default UpdatePlanPanel
