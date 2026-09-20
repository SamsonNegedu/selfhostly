import { Link } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { ProgressBar } from '@/shared/components/ui/ProgressBar'
import { useReloadAfterUpdate, useUpdateStatus } from '@/shared/hooks/useUpdate'
import { ROUTES } from '@/shared/lib/routes'
import { isRunActive, runHeading, runPercent, stepLabel } from '@/shared/lib/update'

const UPDATES_ADDRESS = `${ROUTES.settings}?section=updates`

// A slim bar at the top of every page while Selfhostly updates itself, and while the API is restarting under it.
// It also reloads the page once after an update this tab watched, so the new web interface is the one on screen.
function UpdateProgress() {
    const { data, reconnecting } = useUpdateStatus()
    useReloadAfterUpdate()

    const run = data?.run
    const active = run && isRunActive(run) ? run : null
    if (!active && !reconnecting) return null

    return (
        <section
            aria-label="Update in progress"
            data-testid="update-progress"
            className="border-b border-border bg-card"
        >
            <div className="container mx-auto flex flex-wrap items-center gap-x-4 gap-y-2 px-3 py-2.5 sm:px-12">
                <div className="flex min-w-[160px] items-center gap-2 text-sm" role="status">
                    <Loader2 aria-hidden="true" className="h-4 w-4 shrink-0 animate-spin text-status-info-fg" />
                    {reconnecting ? (
                        <span data-testid="update-reconnecting">
                            Reconnecting to Selfhostly. It is restarting and this takes a few seconds.
                        </span>
                    ) : (
                        <span>
                            {active ? runHeading(active) : 'Updating Selfhostly'}: {stepLabel(active?.phase ?? '')}
                        </span>
                    )}
                </div>
                {active && (
                    <div className="min-w-[200px] flex-1">
                        <ProgressBar value={runPercent(active)} aria-label="Update progress" />
                    </div>
                )}
                <Link
                    to={UPDATES_ADDRESS}
                    className="inline-flex min-h-[44px] items-center text-compact font-medium hover:underline md:min-h-0"
                >
                    Details
                </Link>
            </div>
        </section>
    )
}

export default UpdateProgress
