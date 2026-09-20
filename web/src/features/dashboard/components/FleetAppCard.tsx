import { Link } from 'react-router-dom'
import { AlertTriangle, Globe, Lock } from 'lucide-react'
import { AppTile } from '@/shared/components/ui/AppTile'
import { Card } from '@/shared/components/ui/Card'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { cn } from '@/shared/lib/utils'
import { formatBytes, formatPercent, formatUntil, hostOf } from '@/shared/lib/format'
import { upcomingRuns } from '@/shared/lib/schedule'
import { appHref } from '@/shared/lib/routes'
import { appStatusMeta } from '@/shared/lib/status'
import { resourceTone } from '@/shared/lib/thresholds'
import { useScheduleNextRuns } from '@/shared/services/api'
import type { App } from '@/shared/types/api'
import FleetAppCardActions from './FleetAppCardActions'
import type { FleetApp } from '../lib/fleet'
import type { AppMetrics } from '../hooks/useFleetMetrics'
import type { useFleetActions } from '../hooks/useFleetActions'

type FleetActions = ReturnType<typeof useFleetActions>

interface FleetAppCardProps {
    fleetApp: FleetApp
    metrics?: AppMetrics
    actions: FleetActions
}

// Shows when the next automatic start or stop happens. Only asked for apps that have a schedule turned on.
function NextRun({ app }: { app: App }) {
    const { data } = useScheduleNextRuns(app.id, app.node_id)
    const [first, second] = upcomingRuns(data)
    if (!first) return null

    const label = (run: { action: string; at: string }) =>
        `${run.action === 'start' ? 'starts' : 'stops'} ${formatUntil(run.at)}`
    return (
        <p className="text-compact text-muted-foreground">
            {label(first).replace(/^./, (letter) => letter.toUpperCase())}
            {second ? `, then ${label(second)}` : ''}
        </p>
    )
}

const VALUE_TONE = { ok: '', warn: 'text-status-warn-fg', err: 'text-status-err-fg', info: '', idle: '' } as const

function Metric({ label, value, tone = 'ok' }: { label: string; value: string; tone?: keyof typeof VALUE_TONE }) {
    return (
        <div className="flex flex-col gap-0.5">
            <span className="text-caption font-semibold uppercase tracking-wider text-muted-foreground">{label}</span>
            <span className={cn('text-compact font-semibold', VALUE_TONE[tone])}>{value}</span>
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
                        title={app.name}
                        className="block truncate text-title font-semibold hover:underline after:absolute after:inset-0 after:content-['']"
                    >
                        {app.name}
                    </Link>
                    <p className="truncate text-compact text-muted-foreground" title={app.description || undefined}>
                        {app.description || 'No description'}
                    </p>
                </div>
                <StatusPill kind={meta.kind}>{meta.label}</StatusPill>
            </div>

            {app.public_url ? (
                <div className="flex items-center gap-1.5 text-compact text-status-info-fg">
                    <Globe className="h-3.5 w-3.5 shrink-0" />
                    <span className="truncate font-mono" title={app.public_url}>
                        {hostOf(app.public_url)}
                    </span>
                    {app.tunnel_mode === 'quick' && (
                        <StatusPill kind="warn" size="sm">
                            Temporary
                        </StatusPill>
                    )}
                </div>
            ) : (
                <div className="flex items-center gap-1.5 text-compact text-muted-foreground">
                    <Lock className="h-3.5 w-3.5 shrink-0" />
                    {unreachable ? 'Node offline, status unknown' : 'LAN only'}
                </div>
            )}

            {isFailed && (
                <div className="flex items-start gap-2 rounded-lg bg-status-err-bg px-3 py-2 text-compact text-status-err-fg">
                    <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                    <span>{app.error_message || 'The app reported an error'}</span>
                </div>
            )}
            {unreachable && (
                <div className="rounded-lg bg-status-idle-bg px-3 py-2 text-compact text-status-idle-fg">
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

            <FleetAppCardActions
                app={app}
                actions={actions}
                unreachable={unreachable}
                busy={busy}
                isFailed={isFailed}
                isStopped={isStopped}
                isRunning={isRunning}
                inProgress={inProgress}
            />
        </Card>
    )
}

export default FleetAppCard
