import { Link } from 'react-router-dom'
import {
    AlertTriangle,
    ExternalLink,
    Globe,
    Loader2,
    Lock,
    MoreHorizontal,
    Play,
    RefreshCw,
    Square,
    Trash2,
} from 'lucide-react'
import { AppTile } from '@/shared/components/ui/AppTile'
import { Button, buttonClasses } from '@/shared/components/ui/Button'
import { Card } from '@/shared/components/ui/Card'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/shared/components/ui/DropdownMenu'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { cn } from '@/shared/lib/utils'
import { formatBytes, formatPercent, formatUntil } from '@/shared/lib/format'
import { upcomingRuns } from '@/shared/lib/schedule'
import { ROUTES, appHref } from '@/shared/lib/routes'
import { appStatusMeta } from '@/shared/lib/status'
import { resourceTone } from '@/shared/lib/thresholds'
import { useScheduleNextRuns } from '@/shared/services/api'
import type { App } from '@/shared/types/api'
import type { FleetApp } from '../lib/fleet'
import type { AppMetrics } from '../hooks/useFleetMetrics'
import type { useFleetActions } from '../hooks/useFleetActions'

type FleetActions = ReturnType<typeof useFleetActions>

interface FleetAppCardProps {
    fleetApp: FleetApp
    metrics?: AppMetrics
    actions: FleetActions
}

const hostOf = (url: string) => {
    try {
        return new URL(url).host
    } catch {
        return url
    }
}

// Shows when the next automatic start or stop happens. Only asked for apps that have a schedule turned on.
function NextRun({ app }: { app: App }) {
    const { data } = useScheduleNextRuns(app.id, app.node_id)
    const [first, second] = upcomingRuns(data)
    if (!first) return null

    const label = (run: { action: string; at: string }) =>
        `${run.action === 'start' ? 'starts' : 'stops'} ${formatUntil(run.at)}`
    return (
        <p className="text-[12.5px] text-muted-foreground">
            {label(first).replace(/^./, (letter) => letter.toUpperCase())}
            {second ? `, then ${label(second)}` : ''}
        </p>
    )
}

const VALUE_TONE = { ok: '', warn: 'text-status-warn-fg', err: 'text-status-err-fg', info: '', idle: '' } as const

function Metric({ label, value, tone = 'ok' }: { label: string; value: string; tone?: keyof typeof VALUE_TONE }) {
    return (
        <div className="flex flex-col gap-0.5">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">{label}</span>
            <span className={cn('text-[13.5px] font-semibold', VALUE_TONE[tone])}>{value}</span>
        </div>
    )
}

// One app, in whichever state it is in. The card shows only what the API really reports.
function FleetAppCard({ fleetApp, metrics, actions }: FleetAppCardProps) {
    const { app, unreachable } = fleetApp
    const meta = unreachable ? ({ kind: 'warn', label: 'Unreachable' } as const) : appStatusMeta(app.status)
    const busy = actions.isBusy(app.id)
    const isFailed = !unreachable && app.status === 'error'
    const isStopped = !unreachable && app.status === 'stopped'
    const isRunning = !unreachable && app.status === 'running'
    const inProgress = !unreachable && (app.status === 'updating' || app.status === 'pending')

    return (
        <Card
            className={cn(
                'relative flex flex-col gap-3 p-4',
                isFailed && 'border-status-err/50',
                unreachable && 'opacity-90',
            )}
            data-app={app.name}
        >
            <div className="flex items-start gap-3">
                <AppTile name={app.name} size="md" />
                <div className="min-w-0 flex-1">
                    {/* The link stretches over the whole card so the card is one large tap target. The actions sit above it. */}
                    <Link
                        to={appHref(app)}
                        className="block truncate text-[15px] font-semibold hover:underline after:absolute after:inset-0 after:content-['']"
                    >
                        {app.name}
                    </Link>
                    <p className="truncate text-[13px] text-muted-foreground">{app.description || 'No description'}</p>
                </div>
                <StatusPill kind={meta.kind}>{meta.label}</StatusPill>
            </div>

            {app.public_url ? (
                <div className="flex items-center gap-1.5 text-[12.5px] text-status-info-fg">
                    <Globe className="h-3.5 w-3.5 shrink-0" />
                    <span className="truncate font-mono">{hostOf(app.public_url)}</span>
                    {app.tunnel_mode === 'quick' && (
                        <StatusPill kind="warn" size="sm">
                            Temporary
                        </StatusPill>
                    )}
                </div>
            ) : (
                <div className="flex items-center gap-1.5 text-[12.5px] text-muted-foreground">
                    <Lock className="h-3.5 w-3.5 shrink-0" />
                    {unreachable ? 'Node offline, status unknown' : 'LAN only'}
                </div>
            )}

            {isFailed && (
                <div className="flex items-start gap-2 rounded-lg bg-status-err-bg px-3 py-2 text-[12.5px] text-status-err-fg">
                    <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                    <span>{app.error_message || 'The app reported an error'}</span>
                </div>
            )}
            {unreachable && (
                <div className="rounded-lg bg-status-idle-bg px-3 py-2 text-[12.5px] text-status-idle-fg">
                    {fleetApp.node?.name ?? 'Its node'} is not answering. Actions return when it reconnects.
                </div>
            )}
            {isRunning && metrics && (
                <div className="grid grid-cols-3 gap-3">
                    <Metric
                        label="CPU"
                        value={formatPercent(metrics.cpuPercent)}
                        tone={resourceTone('cpu', metrics.cpuPercent)}
                    />
                    <Metric label="Memory" value={formatBytes(metrics.memoryBytes)} />
                    <Metric label="Containers" value={String(metrics.containers)} />
                </div>
            )}
            {!unreachable && app.schedule?.enabled && <NextRun app={app} />}

            <div className="relative z-10 mt-auto flex flex-wrap items-center gap-2 pt-1">
                {unreachable && (
                    <Link to={ROUTES.nodes} className={buttonClasses({ variant: 'outline' })}>
                        View node
                    </Link>
                )}
                {(isStopped || isFailed) && (
                    <Button onClick={() => actions.start(app)} disabled={busy}>
                        {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                        {isFailed ? 'Retry start' : 'Start'}
                    </Button>
                )}
                {isFailed && (
                    <Link to={appHref(app, 'logs')} className={buttonClasses({ variant: 'outline' })}>
                        View logs
                    </Link>
                )}
                {isStopped && (
                    <Link to={appHref(app, 'schedule')} className={buttonClasses({ variant: 'outline' })}>
                        Schedule
                    </Link>
                )}
                {inProgress && (
                    <Button variant="outline" disabled>
                        <Loader2 className="h-4 w-4 animate-spin" />
                        {app.status === 'updating' ? 'Updating' : 'Working'}
                    </Button>
                )}
                {isRunning && (
                    <Button variant="outline" onClick={() => actions.requestStop(app)} disabled={busy}>
                        {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Square className="h-4 w-4" />}
                        Stop
                    </Button>
                )}
                {isRunning && app.public_url && (
                    <a
                        href={app.public_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className={buttonClasses({ variant: 'outline' })}
                    >
                        <ExternalLink className="h-4 w-4" />
                        Open
                    </a>
                )}

                <div className="ml-auto">
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
        </Card>
    )
}

export default FleetAppCard
