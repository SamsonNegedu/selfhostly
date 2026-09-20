import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent } from '@/shared/components/ui/Card'
import { describeError } from '@/shared/lib/errors'
import { formatUpdateTime } from '@/shared/lib/update'
import type { ApplyUpdateRequest, UpdatePlan, UpdateStatus } from '@/shared/types/api'
import UpdatePlanPanel from './UpdatePlanPanel'

interface AvailableUpdateCardProps {
    available: NonNullable<UpdateStatus['available']>
    // The review of this exact version, if there is one.
    currentPlan: UpdatePlan | null
    canReview: boolean
    reviewing: boolean
    onReview: () => void
    // Why the last review request failed, or null.
    reviewError: unknown | null
    running: boolean
    starting: boolean
    startError: unknown
    onApply: (request: ApplyUpdateRequest) => void
}

// A newer version: what it is, the button to review it, and once reviewed, the plan to start it from.
function AvailableUpdateCard({
    available,
    currentPlan,
    canReview,
    reviewing,
    onReview,
    reviewError,
    running,
    starting,
    startError,
    onApply,
}: AvailableUpdateCardProps) {
    return (
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
                        <p className="text-compact text-muted-foreground">
                            Published {formatUpdateTime(available.published_at)}.
                        </p>
                    )}
                </div>
                {available.notes && (
                    <p className="whitespace-pre-wrap text-compact text-muted-foreground">{available.notes}</p>
                )}
                {available.settings.length > 0 && (
                    <p className="text-compact text-muted-foreground">
                        This release adds settings:{' '}
                        <code className="font-mono">
                            {available.settings.map((setting) => `${setting.key} (${setting.kind})`).join(', ')}
                        </code>
                    </p>
                )}
                {canReview && (
                    <div>
                        <Button data-testid="update-review-button" onClick={() => onReview()} disabled={reviewing}>
                            {reviewing ? 'Reviewing...' : currentPlan ? 'Review again' : 'Review update'}
                        </Button>
                    </div>
                )}
                {reviewError !== null && (
                    <p role="alert" className="text-compact text-status-err-fg">
                        {describeError(reviewError)}
                    </p>
                )}
                {currentPlan && (
                    <div className="border-t border-border pt-4">
                        <UpdatePlanPanel
                            key={currentPlan.version}
                            plan={currentPlan}
                            locked={running}
                            starting={starting}
                            startError={startError}
                            onApply={onApply}
                        />
                    </div>
                )}
            </CardContent>
        </Card>
    )
}

export default AvailableUpdateCard
