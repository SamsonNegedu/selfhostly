import { Calendar, Play, Square } from 'lucide-react'
import { Link } from 'react-router-dom'
import { formatUntil } from '@/shared/lib/format'
import { appHref } from '@/shared/lib/routes'
import { formatRunTime, upcomingRuns } from '@/shared/lib/schedule'
import { useScheduleNextRuns } from '@/shared/services/api'
import type { App } from '@/shared/types/api'
import OverviewCard from './OverviewCard'

// How many upcoming runs the Overview lists. The Schedule tab lists more.
const OVERVIEW_RUNS = 3

// The next few times the app will start or stop by itself, and a link to the schedule.
function ScheduleCard({ app }: { app: App }) {
    const { data: nextRuns } = useScheduleNextRuns(app.id, app.node_id || '')
    const upcoming = upcomingRuns(nextRuns)

    return (
        <OverviewCard
            icon={<Calendar className="h-4 w-4 text-muted-foreground" />}
            title="Schedule"
            contentClassName="flex flex-col gap-3"
        >
            {app.schedule?.enabled && nextRuns ? (
                upcoming.length > 0 ? (
                    <ol className="flex flex-col gap-2.5">
                        {upcoming.slice(0, OVERVIEW_RUNS).map((run) => (
                            <li
                                key={`${run.action}-${run.at}`}
                                className="flex items-center justify-between gap-3 text-sm"
                            >
                                <span className="flex items-center gap-2">
                                    {run.action === 'start' ? (
                                        <Play aria-hidden="true" className="h-4 w-4 text-status-ok-fg" />
                                    ) : (
                                        <Square aria-hidden="true" className="h-4 w-4 text-muted-foreground" />
                                    )}
                                    <span>
                                        <span className="font-medium">
                                            {run.action === 'start' ? 'Starts' : 'Stops'}
                                        </span>{' '}
                                        <span className="text-muted-foreground">
                                            {formatRunTime(run.at, app.schedule?.timezone)}
                                        </span>
                                    </span>
                                </span>
                                <span className="shrink-0 text-compact tabular-nums text-muted-foreground">
                                    {formatUntil(run.at)}
                                </span>
                            </li>
                        ))}
                    </ol>
                ) : (
                    <p className="text-sm text-muted-foreground">No upcoming scheduled actions</p>
                )
            ) : (
                <p className="text-sm text-muted-foreground">
                    Runs all the time. Set a schedule to start and stop it automatically.
                </p>
            )}
            <Link
                to={appHref(app, 'schedule')}
                className="inline-flex min-h-[44px] items-center text-sm font-medium hover:underline md:min-h-0"
            >
                {app.schedule?.enabled ? 'Edit schedule' : 'Set a schedule'}
            </Link>
        </OverviewCard>
    )
}

export default ScheduleCard
