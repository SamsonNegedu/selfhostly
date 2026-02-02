import { useState, useEffect, useRef } from 'react'
import { Button } from '@/shared/components/ui/button'
import { Play, Square, RefreshCw, Trash2, Loader2, RotateCcw } from 'lucide-react'
import { TooltipProvider, TooltipContent, TooltipTrigger } from '@/shared/components/ui/tooltip'
import { useJobPolling } from '@/shared/hooks/useJobPolling'
import { useAppJobs, useQueryClient } from '@/shared/services/api'
import { JobProgress } from '@/shared/components/ui/JobProgress'
import { useToast } from '@/shared/components/ui/Toast'

interface AppActionsProps {
    appId: string
    nodeId: string
    appStatus: 'running' | 'stopped' | 'updating' | 'error' | 'pending'
    isStartPending: boolean
    isStopPending: boolean
    isUpdatePending: boolean
    isDeletePending: boolean
    isRefreshing?: boolean
    onStart: () => void
    onStop: () => void
    onUpdate: () => void
    onDelete: () => void
    onRefresh?: () => void
}

export function AppActions({
    appId,
    nodeId,
    appStatus,
    isStartPending,
    isStopPending,
    isUpdatePending,
    isDeletePending,
    isRefreshing = false,
    onStart,
    onStop,
    onUpdate,
    onDelete,
    onRefresh
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
    const activeJob = jobs?.find(j => j.status === 'pending' || j.status === 'running')
    const hasActiveJobs = !!activeJob

    // Poll the active job if one exists
    const { data: polledJob } = useJobPolling(
        activeJob?.id || activeJobId,
        nodeId,
        !!(activeJob?.id || activeJobId)
    )

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
            if (currentJob.type === 'tunnel_create' || currentJob.type === 'tunnel_delete' || currentJob.type === 'quick_tunnel') {
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
            if (currentJob.type === 'tunnel_create' || currentJob.type === 'tunnel_delete' || currentJob.type === 'quick_tunnel') {
                queryClient.invalidateQueries({ queryKey: ['tunnels', 'app', appId, nodeId] })
                queryClient.refetchQueries({ queryKey: ['tunnels', 'app', appId, nodeId], type: 'active' })
            }
            
            // Keep job displayed so user can see error details
        }
        
        // Clean up old processed IDs to prevent memory leak (keep last 10)
        if (processedJobIdsRef.current.size > 10) {
            const ids = Array.from(processedJobIdsRef.current)
            processedJobIdsRef.current.clear()
            ids.slice(-10).forEach(id => processedJobIdsRef.current.add(id))
        }
    }, [currentJob?.id, currentJob?.status, currentJob?.type, appId, nodeId, toast, queryClient])

    const isAnyActionPending = isStartPending || isStopPending || isUpdatePending || isDeletePending

    const isRunning = appStatus === 'running'
    const isStopped = appStatus === 'stopped'
    const hasActiveJob = !!(currentJob && (currentJob.status === 'pending' || currentJob.status === 'running'))

    // Show job progress if there's an active job
    if (hasActiveJob && currentJob) {
        return (
            <div className="flex-1">
                <JobProgress job={currentJob} compact />
            </div>
        )
    }

    return (
        <div className="flex items-center gap-1 sm:gap-1.5 flex-wrap sm:flex-nowrap">
            {/* Refresh Button */}
            {onRefresh && (
                <TooltipProvider>
                    <TooltipTrigger asChild>
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={onRefresh}
                            disabled={isRefreshing}
                            className="h-8 w-8 sm:h-9 sm:w-9 text-muted-foreground hover:text-foreground"
                        >
                            <RefreshCw className={`h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Refresh</p>
                    </TooltipContent>
                </TooltipProvider>
            )}

            {/* Divider */}
            {onRefresh && <div className="h-6 w-px bg-border mx-1" />}

            {/* Start Button - shown when stopped */}
            {isStopped && (
                <TooltipProvider>
                    <TooltipTrigger asChild>
                        <Button
                            variant="default"
                            size="sm"
                            onClick={onStart}
                            disabled={isAnyActionPending}
                            className="h-9 px-2 sm:px-3 bg-green-600 hover:bg-green-700 dark:bg-green-500 dark:hover:bg-green-400 text-white gap-1 sm:gap-1.5 text-xs sm:text-sm"
                        >
                            {isStartPending ? (
                                <Loader2 className="h-4 w-4 animate-spin" />
                            ) : (
                                <Play className="h-4 w-4" />
                            )}
                            <span className="hidden xs:inline">Start</span>
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Start application</p>
                    </TooltipContent>
                </TooltipProvider>
            )}

            {/* Stop Button - shown when running */}
            {isRunning && (
                <TooltipProvider>
                    <TooltipTrigger asChild>
                        <Button
                            variant="secondary"
                            size="sm"
                            onClick={onStop}
                            disabled={isAnyActionPending}
                            className="h-9 px-2 sm:px-3 gap-1 sm:gap-1.5 text-xs sm:text-sm"
                        >
                            {isStopPending ? (
                                <Loader2 className="h-4 w-4 animate-spin" />
                            ) : (
                                <Square className="h-4 w-4" />
                            )}
                            <span className="hidden xs:inline">Stop</span>
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Stop application</p>
                    </TooltipContent>
                </TooltipProvider>
            )}

            {/* Update Button */}
            <TooltipProvider>
                <TooltipTrigger asChild>
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={onUpdate}
                        disabled={isAnyActionPending}
                        className="h-9 px-2 sm:px-3 gap-1 sm:gap-1.5 text-xs sm:text-sm"
                    >
                        {isUpdatePending ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                            <RotateCcw className="h-4 w-4" />
                        )}
                        <span className="hidden xs:inline">Update</span>
                    </Button>
                </TooltipTrigger>
                <TooltipContent>
                    <p>Pull latest images & restart</p>
                </TooltipContent>
            </TooltipProvider>

            {/* Delete Button */}
            <TooltipProvider>
                <TooltipTrigger asChild>
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={onDelete}
                        disabled={isAnyActionPending}
                        className="h-9 w-9 text-destructive hover:text-destructive hover:bg-destructive/10"
                    >
                        {isDeletePending ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                            <Trash2 className="h-4 w-4" />
                        )}
                    </Button>
                </TooltipTrigger>
                <TooltipContent>
                    <p>Delete application</p>
                </TooltipContent>
            </TooltipProvider>
        </div>
    )
}
