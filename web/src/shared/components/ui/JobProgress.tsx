import { AlertCircle, CheckCircle2, Clock, Loader2 } from 'lucide-react'
import { ProgressBar } from '@/shared/components/ui/ProgressBar'
import { formatAgo } from '@/shared/lib/attention'
import { cn } from '@/shared/lib/utils'
import type { Job } from '@/shared/types/api'

interface JobProgressProps {
    job: Job
    compact?: boolean
}

const DEFAULT_MESSAGES: Record<Job['status'], string> = {
    pending: 'Waiting to start',
    running: 'Working on it',
    completed: 'Done',
    failed: 'Failed',
}

const ICON_CLASSES: Record<Job['status'], string> = {
    pending: 'text-muted-foreground',
    running: 'text-status-info-fg',
    completed: 'text-status-ok-fg',
    failed: 'text-status-err-fg',
}

function StatusIcon({ status }: { status: Job['status'] }) {
    const className = cn('h-4 w-4 shrink-0', ICON_CLASSES[status])
    if (status === 'running') return <Loader2 aria-hidden="true" className={cn(className, 'animate-spin')} />
    if (status === 'completed') return <CheckCircle2 aria-hidden="true" className={className} />
    if (status === 'failed') return <AlertCircle aria-hidden="true" className={className} />
    return <Clock aria-hidden="true" className={className} />
}

// The state of a background job such as a deploy or an update. Running jobs show a bar and a percentage, and a
// failed one says why in words.
export function JobProgress({ job, compact = false }: JobProgressProps) {
    const active = job.status === 'pending' || job.status === 'running'
    const message = job.progress_message || DEFAULT_MESSAGES[job.status]
    const when =
        job.status === 'completed' && job.completed_at
            ? `Finished ${formatAgo(job.completed_at)}`
            : job.status === 'failed' && job.completed_at
              ? `Failed ${formatAgo(job.completed_at)}`
              : job.status === 'running' && job.started_at
                ? `Started ${formatAgo(job.started_at)}`
                : ''

    return (
        <div className="flex flex-col gap-2" data-job-status={job.status}>
            <div className="flex items-center justify-between gap-3">
                <div className="flex min-w-0 items-center gap-2">
                    <StatusIcon status={job.status} />
                    <div className="min-w-0">
                        <p className="truncate text-sm font-medium">{message}</p>
                        {!compact && when && <p className="text-xs text-muted-foreground">{when}</p>}
                    </div>
                </div>
                {active && <span className="text-sm tabular-nums text-muted-foreground">{job.progress}%</span>}
            </div>

            {active && (
                <ProgressBar value={job.progress} tone="info" aria-label={`${message}, ${job.progress} percent`} />
            )}

            {job.status === 'failed' && job.error_message && (
                <div
                    role="alert"
                    className="flex items-start gap-2 rounded-lg bg-status-err-bg px-3 py-2 text-sm text-status-err-fg"
                >
                    <AlertCircle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0" />
                    <span>{job.error_message}</span>
                </div>
            )}
        </div>
    )
}
