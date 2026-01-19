import { Button } from '@/shared/components/ui/button'
import { Play, Pause, RotateCcw, Trash2, Loader2 } from 'lucide-react'
import { TooltipProvider, TooltipContent, TooltipTrigger } from '@/shared/components/ui/tooltip'

interface AppActionsProps {
    appStatus: 'running' | 'stopped' | 'updating' | 'error'
    isStartPending: boolean
    isStopPending: boolean
    isUpdatePending: boolean
    isDeletePending: boolean
    onStart: () => void
    onStop: () => void
    onUpdate: () => void
    onDelete: () => void
}

export function AppActions({
    appStatus,
    isStartPending,
    isStopPending,
    isUpdatePending,
    isDeletePending,
    onStart,
    onStop,
    onUpdate,
    onDelete
}: AppActionsProps) {
    const getActiveActions = () => {
        if (appStatus === 'running') {
            return { showStop: true, showStart: false }
        } else if (appStatus === 'stopped') {
            return { showStop: false, showStart: true }
        }
        return { showStop: true, showStart: true }
    }

    const { showStop, showStart } = getActiveActions()

    return (
        <div className="flex gap-2">
            {showStart && (
                <TooltipProvider>
                    <div className="relative group">
                        <TooltipTrigger asChild>
                            <Button
                                variant="default"
                                size="icon"
                                onClick={onStart}
                                disabled={isStartPending || isUpdatePending}
                                className="bg-green-600 hover:bg-green-700 text-white"
                                title="Start App"
                            >
                                {isStartPending ? (
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                ) : (
                                    <Play className="h-4 w-4" />
                                )}
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent>
                            <p>Start App</p>
                        </TooltipContent>
                    </div>
                </TooltipProvider>
            )}

            {showStop && (
                <TooltipProvider>
                    <div className="relative group">
                        <TooltipTrigger asChild>
                            <Button
                                variant="outline"
                                size="icon"
                                onClick={onStop}
                                disabled={isStopPending || isUpdatePending}
                                className="bg-red-600 hover:bg-red-700 text-white border-red-600 hover:border-red-700"
                                title="Stop App"
                            >
                                {isStopPending ? (
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                ) : (
                                    <Pause className="h-4 w-4" />
                                )}
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent>
                            <p>Stop App</p>
                        </TooltipContent>
                    </div>
                </TooltipProvider>
            )}

            <TooltipProvider>
                <div className="relative group">
                    <TooltipTrigger asChild>
                        <Button
                            variant="outline"
                            size="icon"
                            onClick={onUpdate}
                            disabled={isUpdatePending || isStartPending || isStopPending}
                            className="bg-blue-600 hover:bg-blue-700 text-white border-blue-600 hover:border-blue-700"
                            title="Update Containers"
                        >
                            {isUpdatePending ? (
                                <Loader2 className="h-4 w-4 animate-spin" />
                            ) : (
                                <RotateCcw className="h-4 w-4" />
                            )}
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Update Containers</p>
                    </TooltipContent>
                </div>
            </TooltipProvider>

            <TooltipProvider>
                <div className="relative group">
                    <TooltipTrigger asChild>
                        <Button
                            variant="outline"
                            size="icon"
                            onClick={onDelete}
                            disabled={isDeletePending || isUpdatePending || isStartPending || isStopPending}
                            className="text-destructive hover:text-destructive hover:bg-destructive hover:text-destructive-foreground"
                            title="Delete App"
                        >
                            {isDeletePending ? (
                                <Loader2 className="h-4 w-4 animate-spin" />
                            ) : (
                                <Trash2 className="h-4 w-4" />
                            )}
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Delete App</p>
                    </TooltipContent>
                </div>
            </TooltipProvider>
        </div>
    )
}
