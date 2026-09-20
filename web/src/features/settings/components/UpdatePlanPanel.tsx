import { useMemo, useState } from 'react'
import { Download, Loader2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Checkbox } from '@/shared/components/ui/Checkbox'
import { CodeBlock } from '@/shared/components/ui/CodeBlock'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { describeError } from '@/shared/lib/errors'
import { diffToLines } from '@/shared/lib/update'
import type { StatusKind } from '@/shared/lib/status'
import type { ApplyUpdateRequest, UpdatePlan, UpdatePlanState } from '@/shared/types/api'
import ConfirmUpdateDialog from './ConfirmUpdateDialog'

const PLAN_STATE_META: Record<UpdatePlanState, { label: string; kind: StatusKind }> = {
    preparing: { label: 'Preparing', kind: 'info' },
    ready: { label: 'Ready', kind: 'ok' },
    blocked: { label: 'Blocked', kind: 'warn' },
    failed: { label: 'Failed', kind: 'err' },
}

// A blocker the form below can clear. Every other blocker needs something done outside this page.
const RESOLVED_BY_INPUTS = 'missing_required'

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
    const diffLines = useMemo(() => diffToLines(compose.diff), [compose.diff])
    const needsApproval = compose.state === 'behind'
    const hardBlockers = plan.blockers.filter((blocker) => blocker.code !== RESOLVED_BY_INPUTS)
    const inputsComplete = settings.required_missing.every((setting) => (values[setting.key] ?? '').trim() !== '')
    const startable =
        plan.state === 'ready' ||
        (plan.state === 'blocked' && settings.required_missing.length > 0 && hardBlockers.length === 0)
    const canApply = startable && hardBlockers.length === 0 && inputsComplete && (!needsApproval || approved) && !locked

    const start = () => {
        setConfirming(false)
        const inputs = Object.fromEntries(
            settings.required_missing.map((setting) => [setting.key, (values[setting.key] ?? '').trim()]),
        )
        onApply({
            version: plan.version,
            inputs,
            ...(needsApproval && compose.approval_token ? { approve_compose: compose.approval_token } : {}),
        })
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
                <p role="status" className="flex items-center gap-2 text-[13px] text-muted-foreground">
                    <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                    Pulling the new version and checking your install. This can take a minute.
                </p>
            )}

            {plan.state === 'failed' && (
                <p role="alert" className="text-[13px] text-status-err-fg">
                    {plan.error || 'The review could not finish.'}
                </p>
            )}

            {plan.blockers.length > 0 && (
                <ul aria-label="What stops this update" className="flex flex-col gap-2">
                    {plan.blockers.map((blocker) => (
                        <li
                            key={`${blocker.code}-${blocker.message}`}
                            data-testid="update-blocker"
                            data-code={blocker.code}
                            className="rounded-lg bg-status-warn-bg px-3 py-2 text-[13px] text-status-warn-fg"
                        >
                            {blocker.message}
                        </li>
                    ))}
                </ul>
            )}

            {plan.state !== 'preparing' && plan.state !== 'failed' && (
                <section aria-label="Compose file" className="flex flex-col gap-2">
                    <h4 className="text-[13px] font-semibold">Your compose file</h4>
                    {compose.state === 'current' && (
                        <p className="text-[13px] text-muted-foreground">Your compose file is already up to date.</p>
                    )}
                    {compose.state === 'behind' && (
                        <p className="text-[13px] text-muted-foreground">
                            This release changes the compose file. Nothing you set is lost. Review the changes and
                            approve them to continue.
                        </p>
                    )}
                    {compose.state === 'customized' && (
                        <p className="text-[13px] text-status-warn-fg">
                            Your compose file has edits this release would lose. Merge them by hand first, then review
                            again.
                        </p>
                    )}
                    {compose.report && (
                        <p className="whitespace-pre-wrap rounded-lg bg-muted px-3 py-2 font-mono text-xs">
                            {compose.report}
                        </p>
                    )}
                    {diffLines.length > 0 && (
                        <div data-testid="update-compose-diff">
                            <CodeBlock lines={diffLines} language="text" aria-label="Compose file changes" />
                        </div>
                    )}
                    {needsApproval && (
                        <label className="flex min-h-[44px] items-center gap-2.5 text-[13px] sm:min-h-0">
                            <Checkbox
                                data-testid="update-approve-compose"
                                checked={approved}
                                onCheckedChange={setApproved}
                            />
                            I have reviewed the compose file changes
                        </label>
                    )}
                </section>
            )}

            {settings.required_missing.length > 0 && (
                <section aria-label="Settings this release needs" className="flex flex-col gap-3">
                    <h4 className="text-[13px] font-semibold">Settings this release needs</h4>
                    {settings.required_missing.map((setting) => (
                        <Field key={setting.key} label={setting.key} hint={setting.description}>
                            <Input
                                data-testid={`update-input-${setting.key}`}
                                type={setting.secret ? 'password' : 'text'}
                                value={values[setting.key] ?? ''}
                                onChange={(event) => setValues({ ...values, [setting.key]: event.target.value })}
                                autoComplete="off"
                                spellCheck={false}
                            />
                        </Field>
                    ))}
                </section>
            )}

            {settings.generated.length > 0 && (
                <p className="text-[13px] text-muted-foreground">
                    Created for you and saved in your settings file:{' '}
                    <code className="font-mono">{settings.generated.join(', ')}</code>
                </p>
            )}
            {settings.optional.length > 0 && (
                <p className="text-[13px] text-muted-foreground">
                    New optional settings, left at their defaults:{' '}
                    <code className="font-mono">{settings.optional.join(', ')}</code>
                </p>
            )}

            {startError !== null && startError !== undefined && (
                <p role="alert" className="text-[13px] text-status-err-fg">
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
                        <p className="text-[13px] text-muted-foreground">
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
