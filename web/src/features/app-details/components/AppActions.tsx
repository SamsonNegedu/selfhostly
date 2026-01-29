import { Button } from '@/shared/components/ui/button'
import { Play, Square, RefreshCw, Trash2, Loader2, RotateCcw } from 'lucide-react'
import { TooltipProvider, TooltipContent, TooltipTrigger } from '@/shared/components/ui/tooltip'

interface AppActionsProps {
    appStatus: 'running' | 'stopped' | 'updating' | 'error'
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
    const isAnyActionPending = isStartPending || isStopPending || isUpdatePending || isDeletePending

    const isRunning = appStatus === 'running'
    const isStopped = appStatus === 'stopped'

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
                            className="h-9 px-2 sm:px-3 bg-green-600 hover:bg-green-700 text-white gap-1 sm:gap-1.5 text-xs sm:text-sm"
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
