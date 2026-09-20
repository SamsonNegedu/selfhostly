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
import { isRunActive, isRunEnded } from '@/shared/lib/update'
import type { UpdateRun } from '@/shared/types/api'
import AvailableUpdateCard from '../components/AvailableUpdateCard'
import UpdateResultCard from '../components/UpdateResultCard'
import UpdateRunPanel from '../components/UpdateRunPanel'
import UpdatesDisabledCard from '../components/UpdatesDisabledCard'
import UpdateVersionCard from '../components/UpdateVersionCard'

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

        const review = (version: string) => plan.mutate({ version })

        content = (
            <div className="flex flex-col gap-5">
                <UpdateVersionCard
                    status={data}
                    running={running}
                    checking={check.isPending}
                    onCheck={() =>
                        check.mutate(undefined, {
                            onError: (failure) => toast.error('Could not check', describeError(failure)),
                        })
                    }
                />

                {data.check_error && (
                    <p role="alert" className="text-compact text-status-err-fg">
                        Could not check for updates: {data.check_error}
                    </p>
                )}

                {!data.enabled && <UpdatesDisabledCard reason={data.disabled_reason} />}

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
                    <AvailableUpdateCard
                        available={available}
                        currentPlan={currentPlan}
                        canReview={canReview}
                        reviewing={reviewing}
                        onReview={() => review(available.version)}
                        reviewError={plan.isError ? plan.error : null}
                        running={running}
                        starting={apply.isPending}
                        startError={apply.error}
                        onApply={(request) => apply.mutate(request)}
                    />
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
