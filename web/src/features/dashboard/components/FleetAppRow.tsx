import { Link } from 'react-router-dom'
import { AlertTriangle, Loader2, MoreHorizontal, Play, RefreshCw, Trash2 } from 'lucide-react'
import { AppTile } from '@/shared/components/ui/AppTile'
import { Button } from '@/shared/components/ui/Button'
import { Card } from '@/shared/components/ui/Card'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/shared/components/ui/DropdownMenu'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { formatBytes, formatPercent } from '@/shared/lib/format'
import { appHref } from '@/shared/lib/routes'
import { appStatusMeta } from '@/shared/lib/status'
import { cn } from '@/shared/lib/utils'
import type { AppMetrics } from '../hooks/useFleetMetrics'
import type { useFleetActions } from '../hooks/useFleetActions'
import type { FleetApp } from '../lib/fleet'

interface FleetAppRowProps {
    fleetApp: FleetApp
    metrics?: AppMetrics
    actions: ReturnType<typeof useFleetActions>
}

const hostOf = (url: string) => {
    try {
        return new URL(url).host
    } catch {
        return url
    }
}

// The phone version of an app on Fleet: one compact row, so more apps fit on the screen. Tapping the row opens
// the app, and the one action that is likely needed sits at the end.
function FleetAppRow({ fleetApp, metrics, actions }: FleetAppRowProps) {
    const { app, unreachable } = fleetApp
    const meta = unreachable ? ({ kind: 'warn', label: 'Unreachable' } as const) : appStatusMeta(app.status)
    const busy = actions.isBusy(app.id)
    const isFailed = !unreachable && app.status === 'error'
    const canStart = !unreachable && (app.status === 'stopped' || isFailed)
    const inProgress = !unreachable && (app.status === 'updating' || app.status === 'pending')

    const detail = unreachable
        ? 'Node offline'
        : app.status === 'running' && metrics
          ? `${formatPercent(metrics.cpuPercent)} CPU · ${formatBytes(metrics.memoryBytes)}`
          : app.public_url
            ? hostOf(app.public_url)
            : 'LAN only'

    return (
        <Card
            className={cn('relative flex flex-col gap-2 p-3', isFailed && 'border-status-err/50')}
            data-app={app.name}
        >
            <div className="flex items-center gap-3">
                <AppTile name={app.name} size="md" />
                <div className="min-w-0 flex-1">
                    <Link
                        to={appHref(app)}
                        className="block truncate text-[15px] font-semibold after:absolute after:inset-0 after:content-['']"
                    >
                        {app.name}
                    </Link>
                    <p className="truncate text-[13px] text-muted-foreground">{detail}</p>
                </div>
                <StatusPill kind={meta.kind} size="sm">
                    {meta.label}
                </StatusPill>
                <div className="relative z-10 flex items-center">
                    {canStart && (
                        <Button
                            variant="ghost"
                            size="icon"
                            aria-label={`${isFailed ? 'Retry start' : 'Start'} ${app.name}`}
                            onClick={() => actions.start(app)}
                            disabled={busy}
                        >
                            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                        </Button>
                    )}
                    {inProgress && (
                        <Loader2 aria-label="Working" className="mx-3 h-4 w-4 animate-spin text-muted-foreground" />
                    )}
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon" aria-label={`More actions for ${app.name}`}>
                                <MoreHorizontal className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-44">
                            <DropdownMenuItem asChild>
                                <Link to={appHref(app)}>Details</Link>
                            </DropdownMenuItem>
                            {app.status === 'running' && !unreachable && (
                                <DropdownMenuItem onSelect={() => actions.requestStop(app)} disabled={busy}>
                                    Stop
                                </DropdownMenuItem>
                            )}
                            {!unreachable && (
                                <DropdownMenuItem
                                    onSelect={() => actions.requestUpdate(app)}
                                    disabled={busy || inProgress}
                                >
                                    <RefreshCw className="mr-2 h-4 w-4" />
                                    Update
                                </DropdownMenuItem>
                            )}
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                                onSelect={() => actions.requestDelete(app)}
                                disabled={busy}
                                className="text-destructive focus:text-destructive"
                            >
                                <Trash2 className="mr-2 h-4 w-4" />
                                Delete
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </div>
            {isFailed && (
                <div className="flex items-start gap-2 rounded-lg bg-status-err-bg px-3 py-2 text-[12.5px] text-status-err-fg">
                    <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                    <span className="line-clamp-2">{app.error_message || 'The app reported an error'}</span>
                </div>
            )}
        </Card>
    )
}

export default FleetAppRow
