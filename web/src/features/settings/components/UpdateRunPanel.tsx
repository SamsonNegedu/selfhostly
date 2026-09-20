import { Loader2 } from 'lucide-react'
import { ProgressBar } from '@/shared/components/ui/ProgressBar'
import { runHeading, runPercent } from '@/shared/lib/update'
import type { UpdateRun } from '@/shared/types/api'
import UpdateSteps from './UpdateSteps'

interface UpdateRunPanelProps {
    run: UpdateRun
    reconnecting: boolean
}

// An update under way: which version, how far it is, and each step. It says so when the API is restarting.
function UpdateRunPanel({ run, reconnecting }: UpdateRunPanelProps) {
    return (
        <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
                <h2 className="flex items-center gap-2 text-base font-semibold">
                    <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin text-status-info-fg" />
                    {runHeading(run)}
                </h2>
                <p className="text-compact text-muted-foreground">
                    From {run.from_version}. Your apps keep running while this happens.
                </p>
            </div>
            <ProgressBar value={runPercent(run)} aria-label="Update progress" />
            {reconnecting && (
                <p role="status" className="text-compact text-muted-foreground">
                    Selfhostly is restarting. This page reconnects by itself.
                </p>
            )}
            {run.message && <p className="text-compact text-muted-foreground">{run.message}</p>}
            <UpdateSteps steps={run.steps} />
        </div>
    )
}

export default UpdateRunPanel
