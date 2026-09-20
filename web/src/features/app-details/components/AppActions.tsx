import { useState, useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import { ExternalLink, Loader2, MoreHorizontal, Play, RefreshCw, RotateCcw, Square, Trash2 } from 'lucide-react'
import { Button, buttonClasses } from '@/shared/components/ui/Button'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/shared/components/ui/DropdownMenu'
import { useJobPolling } from '@/shared/hooks/useJobPolling'
import { useAppJobs, useQueryClient } from '@/shared/services/api'
import { JobProgress } from '@/shared/components/ui/JobProgress'
import { useToast } from '@/shared/components/ui/Toast'

interface AppActionsProps {
    appId: string
    nodeId: string
    appStatus: 'running' | 'stopped' | 'updating' | 'error' | 'pending'
    publicUrl?: string
    // True while a start, stop, update or delete is in flight.
    isBusy: boolean
    isRefreshing?: boolean
    onStart: () => void
    onStop: () => void
    onUpdate: () => void
    onDelete: () => void
    onRefresh?: () => void
    // On a phone the actions sit in a bar above the tab bar, so they stay in reach while the page scrolls.
    sticky?: boolean
}

export function AppActions({
    appId,
    nodeId,
    appStatus,
    publicUrl,
    isBusy,
    isRefreshing = false,
    onStart,
    onStop,
    onUpdate,
    onDelete,
    onRefresh,
    sticky = false,
}: AppActionsProps) {
    const queryClient = useQueryClient()
    const { toast } = useToast()
    const [activeJobId, setActiveJobId] = useState<string | null>(null)
    // Track processed job IDs to prevent duplicate processing
    const processedJobIdsRef = useRef<Set<string>>(new Set())
    // Track if there were active jobs in the previous render
    const prevHadActiveJobsRef = useRef<boolean>(false)

    // Get recent jobs for this app
    const { data: jobs } = useAppJobs(appId, nodeId)

    // Find the most recent active job (pending or running)
    const activeJob = jobs?.find((j) => j.status === 'pending' || j.status === 'running')
    const hasActiveJobs = !!activeJob

    // Poll the active job if one exists
    const { data: polledJob } = useJobPolling(activeJob?.id || activeJobId, nodeId, !!(activeJob?.id || activeJobId))

    // Use the polled job data if available, otherwise use the job from the list
    const currentJob = polledJob || activeJob

    // Smart refresh: detect when previously active jobs are no longer available
    useEffect(() => {
        const hadActiveJobs = prevHadActiveJobsRef.current
        prevHadActiveJobsRef.current = hasActiveJobs

        // If we had active jobs before, but now we don't, trigger a refresh
        // This handles cases where jobs completed/failed and are no longer in the list
        if (hadActiveJobs && !hasActiveJobs && jobs !== undefined) {
            console.log('[AppActions] Smart refresh: active jobs cleared, refreshing app data')

            // Invalidate and refetch app data
            queryClient.invalidateQueries({ queryKey: ['app', appId, nodeId] })
            queryClient.invalidateQueries({ queryKey: ['app', appId] })
            queryClient.refetchQueries({ queryKey: ['app', appId, nodeId], type: 'active' })

            // Also invalidate tunnels in case the job affected tunnel state
            queryClient.invalidateQueries({ queryKey: ['tunnels', 'app', appId, nodeId] })
            queryClient.refetchQueries({ queryKey: ['tunnels', 'app', appId, nodeId], type: 'active' })
        }
    }, [hasActiveJobs, jobs, appId, nodeId, queryClient])

    // Handle job completion/failure - only process once per job
    useEffect(() => {
        if (!currentJob?.id) return

        const jobId = currentJob.id
        const status = currentJob.status

        // Skip if we've already processed this job
        if (processedJobIdsRef.current.has(jobId)) {
            return
        }

        // Only process completed or failed jobs
        if (status !== 'completed' && status !== 'failed') {
            return
        }

        // Mark as processed immediately to prevent duplicate processing
        processedJobIdsRef.current.add(jobId)

        if (status === 'completed') {
            toast.success('Operation completed', currentJob.progress_message || 'Operation completed successfully')

            // Invalidate queries to mark them as stale
            // This ensures queries will refetch when they become enabled
            queryClient.invalidateQueries({ queryKey: ['app', appId, nodeId] })
            queryClient.invalidateQueries({ queryKey: ['app', appId] }) // Also invalidate without nodeId for compatibility
            queryClient.invalidateQueries({ queryKey: ['apps'] })

            // For app_create jobs, ensure queries refetch when they become enabled
            // The invalidation above will mark queries as stale, and they'll refetch when enabled

            // Refetch active queries immediately
            queryClient.refetchQueries({ queryKey: ['app', appId, nodeId], type: 'active' })
            queryClient.refetchQueries({ queryKey: ['app', appId], type: 'active' })

            // If it's a tunnel job, also refresh tunnel data
            if (
                currentJob.type === 'tunnel_create' ||
                currentJob.type === 'tunnel_delete' ||
                currentJob.type === 'quick_tunnel'
            ) {
                queryClient.invalidateQueries({ queryKey: ['tunnels', 'app', appId, nodeId] })
                queryClient.invalidateQueries({ queryKey: ['tunnels', 'list'] })
                queryClient.refetchQueries({ queryKey: ['tunnels', 'app', appId, nodeId], type: 'active' })
            }

            // Clear active job
            setActiveJobId(null)
        } else if (status === 'failed') {
            toast.error('Operation failed', currentJob.error_message || 'Operation failed')

            // Refresh app data - invalidate both patterns for compatibility
            queryClient.invalidateQueries({ queryKey: ['app', appId, nodeId] })
            queryClient.invalidateQueries({ queryKey: ['app', appId] }) // Also invalidate without nodeId for compatibility

            // Force immediate refetch of active app queries
            queryClient.refetchQueries({ queryKey: ['app', appId, nodeId], type: 'active' })

            // If it's a tunnel job, also refresh tunnel data (even on failure to update status)
            if (
                currentJob.type === 'tunnel_create' ||
                currentJob.type === 'tunnel_delete' ||
                currentJob.type === 'quick_tunnel'
            ) {
                queryClient.invalidateQueries({ queryKey: ['tunnels', 'app', appId, nodeId] })
                queryClient.refetchQueries({ queryKey: ['tunnels', 'app', appId, nodeId], type: 'active' })
            }

            // Keep job displayed so user can see error details
        }

        // Clean up old processed IDs to prevent memory leak (keep last 10)
        if (processedJobIdsRef.current.size > 10) {
            const ids = Array.from(processedJobIdsRef.current)
            processedJobIdsRef.current.clear()
            ids.slice(-10).forEach((id) => processedJobIdsRef.current.add(id))
        }
        // Each job is handled once (processedJobIdsRef), when its id or status changes. Its message fields are read
        // at that moment and must not retrigger this.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [currentJob?.id, currentJob?.status, currentJob?.type, appId, nodeId, toast, queryClient])

    const isRunning = appStatus === 'running'
    const canStart = appStatus === 'stopped' || appStatus === 'error'
    const hasActiveJob = !!(currentJob && (currentJob.status === 'pending' || currentJob.status === 'running'))
    const disabled = isBusy || hasActiveJob || appStatus === 'updating'

    const startButton = canStart && (
        <Button onClick={onStart} disabled={disabled} className={sticky ? 'flex-1' : undefined}>
            {isBusy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
            {appStatus === 'error' ? 'Retry start' : 'Start'}
        </Button>
    )
    const stopButton = isRunning && (
        <Button variant="outline" onClick={onStop} disabled={disabled} className={sticky ? 'flex-1' : undefined}>
            {isBusy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Square className="h-4 w-4" />}
            Stop
        </Button>
    )
    const updateButton = (
        <Button
            variant="outline"
            onClick={onUpdate}
            disabled={disabled}
            title="Pull the latest images and restart"
            className={sticky && !canStart && !isRunning ? 'flex-1' : undefined}
        >
            <RotateCcw className="h-4 w-4" />
            Update
        </Button>
    )
    const openLink = isRunning && publicUrl && (
        <a
            href={publicUrl}
            target="_blank"
            rel="noopener noreferrer"
            aria-label={sticky ? 'Open the app' : undefined}
            className={buttonClasses({ variant: 'outline', size: sticky ? 'icon' : 'default' })}
        >
            <ExternalLink className="h-4 w-4" />
            {!sticky && 'Open'}
        </a>
    )
    const menu = (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" aria-label="More actions">
                    <MoreHorizontal className="h-4 w-4" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44">
                {onRefresh && (
                    <DropdownMenuItem onSelect={onRefresh} disabled={isRefreshing}>
                        <RefreshCw className={`mr-2 h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
                        Refresh
                    </DropdownMenuItem>
                )}
                <DropdownMenuSeparator />
                <DropdownMenuItem
                    onSelect={onDelete}
                    disabled={disabled}
                    className="text-destructive focus:text-destructive"
                >
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete
                </DropdownMenuItem>
            </DropdownMenuContent>
        </DropdownMenu>
    )

    if (sticky) {
        return createPortal(
            <div
                role="region"
                aria-label="App actions"
                className="fixed inset-x-0 bottom-[var(--mobile-nav-h)] z-30 flex flex-col gap-2 border-t border-border bg-card p-3"
            >
                {hasActiveJob && currentJob && <JobProgress job={currentJob} compact />}
                <div className="flex items-center gap-2">
                    {startButton}
                    {stopButton}
                    {updateButton}
                    {openLink}
                    {menu}
                </div>
            </div>,
            document.body,
        )
    }

    return (
        <div className="flex flex-col gap-2 md:items-end">
            {hasActiveJob && currentJob && (
                <div className="w-full max-w-md">
                    <JobProgress job={currentJob} compact />
                </div>
            )}

            <div className="flex flex-wrap items-center gap-2">
                {startButton}
                {stopButton}
                {updateButton}
                {openLink}
                {menu}
            </div>
        </div>
    )
}
