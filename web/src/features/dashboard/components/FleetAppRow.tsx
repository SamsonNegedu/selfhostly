import { Link } from 'react-router-dom'
import { AlertTriangle, Loader2, Play } from 'lucide-react'
import { AppTile } from '@/shared/components/ui/AppTile'
import { Button } from '@/shared/components/ui/Button'
import { Card } from '@/shared/components/ui/Card'
import { DropdownMenuItem } from '@/shared/components/ui/DropdownMenu'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { formatBytes, formatPercent, hostOf } from '@/shared/lib/format'
import { appHref } from '@/shared/lib/routes'
import { appStatusMeta } from '@/shared/lib/status'
import { cn } from '@/shared/lib/utils'
import type { AppMetrics } from '../hooks/useFleetMetrics'
import type { useFleetActions } from '../hooks/useFleetActions'
import FleetAppMenu from './FleetAppMenu'
import type { FleetApp } from '../lib/fleet'

interface FleetAppRowProps {
    fleetApp: FleetApp
    metrics?: AppMetrics
    actions: ReturnType<typeof useFleetActions>
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
                        title={app.name}
                        className="block truncate text-title font-semibold after:absolute after:inset-0 after:content-['']"
                    >
                        {app.name}
                    </Link>
                    <p className="truncate text-compact text-muted-foreground" title={detail}>
                        {detail}
                    </p>
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
                            {busy ? (
                                <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                            ) : (
                                <Play className="h-4 w-4" />
                            )}
                        </Button>
                    )}
                    {inProgress && (
                        <Loader2
                            aria-hidden="true"
                            aria-label="Working"
                            className="mx-3 h-4 w-4 animate-spin text-muted-foreground"
                        />
                    )}
                    <FleetAppMenu
                        app={app}
                        actions={actions}
                        unreachable={unreachable}
                        busy={busy}
                        inProgress={inProgress}
                    >
                        {app.status === 'running' && !unreachable && (
                            <DropdownMenuItem onSelect={() => actions.requestStop(app)} disabled={busy}>
                                Stop
                            </DropdownMenuItem>
                        )}
                    </FleetAppMenu>
                </div>
            </div>
            {isFailed && (
                <div className="flex items-start gap-2 rounded-lg bg-status-err-bg px-3 py-2 text-compact text-status-err-fg">
                    <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                    <span className="line-clamp-2" title={app.error_message || undefined}>
                        {app.error_message || 'The app reported an error'}
                    </span>
                </div>
            )}
        </Card>
    )
}

export default FleetAppRow
