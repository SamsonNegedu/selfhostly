import { Play, Square } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { formatUntil } from '@/shared/lib/format'
import { formatRunTime, upcomingRuns } from '@/shared/lib/schedule'
import { cn } from '@/shared/lib/utils'

interface NextRunsCardProps {
    enabled: boolean
    timezone: string
    runs: ReturnType<typeof upcomingRuns>
    // The times on the left could not be understood, so there is nothing to show.
    hasError: boolean
}

// The next few times the schedule will start or stop the app, on the schedule's own clock.
function NextRunsCard({ enabled, timezone, runs: upcoming, hasError }: NextRunsCardProps) {
    return (
        <Card aria-label="Next runs" role="region">
            <CardHeader>
                <CardTitle className="text-base">Next runs</CardTitle>
                <p className="text-compact text-muted-foreground">in {timezone.replace(/_/g, ' ')}</p>
            </CardHeader>
            <CardContent>
                {!enabled ? (
                    <p className="text-sm text-muted-foreground">The schedule is paused, so nothing will run.</p>
                ) : upcoming.length > 0 ? (
                    <ol className="flex flex-col">
                        {upcoming.map((run) => (
                            <li
                                key={`${run.action}-${run.at}`}
                                className="flex items-center gap-3 border-b border-border py-2.5 first:pt-0 last:border-b-0 last:pb-0"
                            >
                                <span
                                    aria-hidden="true"
                                    className={cn(
                                        'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
                                        run.action === 'start'
                                            ? 'bg-status-ok-bg text-status-ok-fg'
                                            : 'bg-status-idle-bg text-status-idle-fg',
                                    )}
                                >
                                    {run.action === 'start' ? (
                                        <Play className="h-4 w-4" />
                                    ) : (
                                        <Square className="h-4 w-4" />
                                    )}
                                </span>
                                <div className="min-w-0 flex-1">
                                    <p className="text-sm font-medium">{run.action === 'start' ? 'Starts' : 'Stops'}</p>
                                    <p className="text-compact text-muted-foreground">
                                        {formatRunTime(run.at, timezone)}
                                    </p>
                                </div>
                                <span className="shrink-0 text-compact tabular-nums text-muted-foreground">
                                    {formatUntil(run.at)}
                                </span>
                            </li>
                        ))}
                    </ol>
                ) : (
                    <p className="text-sm text-muted-foreground">
                        {hasError
                            ? 'Fix the problem on the left to see the next runs.'
                            : 'Set a time to see the next runs.'}
                    </p>
                )}
            </CardContent>
        </Card>
    )
}

export default NextRunsCard
