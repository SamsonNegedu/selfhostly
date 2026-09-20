import { RefreshCw, ShieldAlert } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent } from '@/shared/components/ui/Card'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { useToast } from '@/shared/components/ui/Toast'
import {
    useApplyUpdate,
    useCheckUpdate,
    usePlanUpdate,
    useRollbackUpdate,
    useUpdateStatus,
} from '@/shared/hooks/useUpdate'
import { describeError } from '@/shared/lib/errors'
import { formatUpdateTime, isRunActive, isRunEnded } from '@/shared/lib/update'
import type { UpdateRun } from '@/shared/types/api'
import UpdatePlanPanel from '../components/UpdatePlanPanel'
import UpdateResultCard from '../components/UpdateResultCard'
import UpdateRunPanel from '../components/UpdateRunPanel'

// What the server says when it will not do updates from the browser, and what to do about it.
const DISABLED_MESSAGES: Record<string, string> = {
    'not enabled':
        'Updates from this page are turned off. Set UI_UPDATES_ENABLED=true and restart Selfhostly to turn them on. You can always update with selfhostlyctl upgrade.',
    'no signing key':
        'This build has no release signing key, so it cannot check that an update is genuine. Set UPDATE_PUBLIC_KEY to the key your releases are signed with.',
    'no docker socket': 'The updater needs the Docker socket, and Selfhostly cannot reach it here.',
    'secondary node': 'This is a secondary node. Update it from the primary, or with selfhostlyctl upgrade.',
    'auth disabled': 'Sign-in is turned off. Updating Selfhostly needs a signed-in user, so turn sign-in on first.',
}

const NOT_FOUND_MESSAGE = 'NOT_FOUND'

// Whether Selfhostly is up to date, and the way to update it: check, review what would change, then update.
function UpdatesSection() {
    const { toast } = useToast()
    const { data, error, isLoading, refetch, reconnecting } = useUpdateStatus()
    const check = useCheckUpdate()
    const plan = usePlanUpdate()
    const apply = useApplyUpdate()
    const rollback = useRollbackUpdate()

    let content: React.ReactNode
    if (isLoading) {
        content = <Skeleton role="status" aria-label="Loading update status" className="h-32 rounded-xl" />
    } else if (!data) {
        content =
            error instanceof Error && error.message === NOT_FOUND_MESSAGE ? (
                <EmptyState
                    title="This server cannot update itself from here"
                    description="It is running a version from before updates in Settings. Update it once with selfhostlyctl upgrade, then this page works."
                />
            ) : (
                <ErrorState title="Could not load update status" error={error} onRetry={() => void refetch()} />
            )
    } else {
        const run: UpdateRun | null = data.run
        const running = isRunActive(run)
        const available = data.available
        const currentPlan = available && data.plan?.version === available.version ? data.plan : null
        const reviewing = currentPlan?.state === 'preparing' || plan.isPending
        const canReview = !currentPlan || currentPlan.state !== 'ready'
        const checked = formatUpdateTime(data.checked_at)

        const review = (version: string) => plan.mutate({ version })

        content = (
            <div className="flex flex-col gap-5">
                <Card>
                    <CardContent className="flex flex-wrap items-center gap-4 p-4">
                        <div className="min-w-0 flex-1">
                            <h2 className="font-semibold">Selfhostly version</h2>
                            <p className="text-[13px] text-muted-foreground">
                                Running{' '}
                                <span data-testid="update-current-version" className="font-mono">
                                    {data.current_version}
                                </span>
                                {checked && `. Last checked ${checked}.`}
                            </p>
                        </div>
                        <Button
                            variant="outline"
                            data-testid="update-check-button"
                            disabled={!data.enabled || running || check.isPending}
                            onClick={() =>
                                check.mutate(undefined, {
                                    onError: (failure) => toast.error('Could not check', describeError(failure)),
                                })
                            }
                        >
                            <RefreshCw className={check.isPending ? 'h-4 w-4 animate-spin' : 'h-4 w-4'} />
                            {check.isPending ? 'Checking...' : 'Check for updates'}
                        </Button>
                    </CardContent>
                </Card>

                {data.check_error && (
                    <p role="alert" className="text-[13px] text-status-err-fg">
                        Could not check for updates: {data.check_error}
                    </p>
                )}

                {!data.enabled && (
                    <Card>
                        <CardContent className="flex items-start gap-3 p-4">
                            <ShieldAlert aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-muted-foreground" />
                            <div>
                                <h2 className="font-semibold">Updates from this page are not available</h2>
                                <p className="text-[13px] text-muted-foreground">
                                    {DISABLED_MESSAGES[data.disabled_reason] ?? data.disabled_reason}
                                </p>
                            </div>
                        </CardContent>
                    </Card>
                )}

                {data.enabled && running && run && (
                    <Card>
                        <CardContent className="p-4">
                            <UpdateRunPanel run={run} reconnecting={reconnecting} />
                        </CardContent>
                    </Card>
                )}

                {data.enabled && run && isRunEnded(run) && (
                    <Card>
                        <CardContent className="p-4">
                            <UpdateResultCard
                                run={run}
                                locked={reviewing}
                                rollingBack={rollback.isPending}
                                onRollback={() =>
                                    rollback.mutate(undefined, {
                                        onError: (failure) =>
                                            toast.error('Could not start the rollback', describeError(failure)),
                                    })
                                }
                                onRetry={() => review(run.to_version)}
                            />
                        </CardContent>
                    </Card>
                )}

                {data.enabled && !running && available && (
                    <Card>
                        <CardContent className="flex flex-col gap-4 p-4">
                            <div className="flex flex-col gap-1">
                                <h2 className="font-semibold">
                                    Version{' '}
                                    <span data-testid="update-available-version" className="font-mono">
                                        {available.version}
                                    </span>{' '}
                                    is available
                                </h2>
                                {available.published_at && (
                                    <p className="text-[13px] text-muted-foreground">
                                        Published {formatUpdateTime(available.published_at)}.
                                    </p>
                                )}
                            </div>
                            {available.notes && (
                                <p className="whitespace-pre-wrap text-[13px] text-muted-foreground">
                                    {available.notes}
                                </p>
                            )}
                            {available.settings.length > 0 && (
                                <p className="text-[13px] text-muted-foreground">
                                    This release adds settings:{' '}
                                    <code className="font-mono">
                                        {available.settings
                                            .map((setting) => `${setting.key} (${setting.kind})`)
                                            .join(', ')}
                                    </code>
                                </p>
                            )}
                            {canReview && (
                                <div>
                                    <Button
                                        data-testid="update-review-button"
                                        onClick={() => review(available.version)}
                                        disabled={reviewing}
                                    >
                                        {reviewing ? 'Reviewing...' : currentPlan ? 'Review again' : 'Review update'}
                                    </Button>
                                </div>
                            )}
                            {plan.isError && (
                                <p role="alert" className="text-[13px] text-status-err-fg">
                                    {describeError(plan.error)}
                                </p>
                            )}
                            {currentPlan && (
                                <div className="border-t border-border pt-4">
                                    <UpdatePlanPanel
                                        key={currentPlan.version}
                                        plan={currentPlan}
                                        locked={running}
                                        starting={apply.isPending}
                                        startError={apply.error}
                                        onApply={(request) => apply.mutate(request)}
                                    />
                                </div>
                            )}
                        </CardContent>
                    </Card>
                )}

                {data.enabled && !available && !running && !data.check_error && (
                    <EmptyState
                        title="You are up to date"
                        description="No newer version is available. Selfhostly checks by itself every so often, and you can check now."
                    />
                )}
            </div>
        )
    }

    return <div data-testid="updates-section">{content}</div>
}

export default UpdatesSection
