import { Link } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { JobProgress } from '@/shared/components/ui/JobProgress'
import { useActiveJobs } from '@/shared/hooks/useActiveJobs'
import { appHref } from '@/shared/lib/routes'
import type { Job } from '@/shared/types/api'

const VERBS: Partial<Record<Job['type'], string>> = {
    app_create: 'Deploying',
    app_update: 'Updating',
    app_start: 'Starting',
}

// A slim bar at the top of every page while anything is deploying, updating or starting. It links to the app
// and shows the same progress as the app's page.
function JobBanner() {
    const active = useActiveJobs()
    if (active.length === 0) return null

    return (
        <section aria-label="Work in progress" className="border-b border-border bg-card">
            <ul className="container mx-auto flex flex-col divide-y divide-border px-3 sm:px-12">
                {active.map(({ app, job }) => (
                    <li key={app.id} className="flex flex-wrap items-center gap-x-4 gap-y-2 py-2.5">
                        <div className="flex min-w-[160px] items-center gap-2 text-sm">
                            <Loader2 aria-hidden="true" className="h-4 w-4 shrink-0 animate-spin text-status-info-fg" />
                            <span>
                                {(job && VERBS[job.type]) ?? 'Working on'}{' '}
                                <Link
                                    to={appHref(app)}
                                    className="inline-flex min-h-[44px] items-center font-semibold hover:underline md:min-h-0"
                                >
                                    {app.name}
                                </Link>
                            </span>
                        </div>
                        {job && (
                            <div className="min-w-[200px] flex-1">
                                <JobProgress job={job} compact />
                            </div>
                        )}
                    </li>
                ))}
            </ul>
        </section>
    )
}

export default JobBanner
