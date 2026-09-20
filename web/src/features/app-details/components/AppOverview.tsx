import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Activity, Calendar, Globe, HardDrive, Layers, Loader2, Play, RefreshCw, Square } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { useFleetMetrics } from '@/features/dashboard/hooks/useFleetMetrics'
import { formatBytes, formatPercent, formatUntil } from '@/shared/lib/format'
import { formatRunTime, upcomingRuns } from '@/shared/lib/schedule'
import { appHref } from '@/shared/lib/routes'
import ActivityTimeline from './ActivityTimeline'
import FailedStartGuide from './FailedStartGuide'
import { useAppServices, useNodes, useRestartAppService, useScheduleNextRuns } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import type { App } from '@/shared/types/api'

// How many upcoming runs the Overview lists. The Schedule tab lists more.
const OVERVIEW_RUNS = 3

interface AppOverviewProps {
    app: App
}

interface ComposeInfo {
    networks: string[]
    volumes: string[]
}

function AppOverview({ app }: AppOverviewProps) {
    // Get services from backend endpoint for consistency with LogViewer
    const { data: services = [] } = useAppServices(app.id, app.node_id || '')
    const { data: nextRuns } = useScheduleNextRuns(app.id, app.node_id || '')
    const { data: nodes = [] } = useNodes()
    const metrics = useFleetMetrics(nodes).forApp(app.node_id ?? '', app.name)
    const restartService = useRestartAppService()
    const { toast } = useToast()
    const [serviceToRestart, setServiceToRestart] = useState<string | null>(null)

    // Parse compose content to extract networks and volumes (services come from backend)
    const composeInfo: ComposeInfo = useMemo(() => {
        const info: ComposeInfo = {
            networks: [],
            volumes: []
        }

        try {
            const lines = app.compose_content.split('\n')
            let inNetworksSection = false
            let inVolumesSection = false
            let currentIndent = 0

            for (let i = 0; i < lines.length; i++) {
                const line = lines[i]
                const trimmedLine = line.trim()

                // Skip empty lines and comments
                if (!trimmedLine || trimmedLine.startsWith('#')) continue

                // Calculate indentation
                const indent = line.search(/\S/)

                // Check for top-level sections
                if (indent === 0) {
                    if (trimmedLine.startsWith('networks:')) {
                        inNetworksSection = true
                        inVolumesSection = false
                        currentIndent = 0
                        continue
                    } else if (trimmedLine.startsWith('volumes:')) {
                        inNetworksSection = false
                        inVolumesSection = true
                        currentIndent = 0
                        continue
                    } else if (trimmedLine.startsWith('version:') || trimmedLine.startsWith('services:')) {
                        continue
                    }
                }

                // Extract networks (first level under 'networks:')
                if (inNetworksSection && indent > 0) {
                    if (currentIndent === 0) {
                        currentIndent = indent
                    }
                    if (indent === currentIndent && trimmedLine.includes(':')) {
                        const networkName = trimmedLine.split(':')[0].trim()
                        if (networkName && !info.networks.includes(networkName)) {
                            info.networks.push(networkName)
                        }
                    }
                }

                // Extract volumes (first level under 'volumes:')
                if (inVolumesSection && indent > 0) {
                    if (currentIndent === 0) {
                        currentIndent = indent
                    }
                    if (indent === currentIndent && trimmedLine.includes(':')) {
                        const volumeName = trimmedLine.split(':')[0].trim()
                        if (volumeName && !info.volumes.includes(volumeName)) {
                            info.volumes.push(volumeName)
                        }
                    }
                }
            }

            // Also count bind mounts in services
            const bindMounts = (app.compose_content.match(/- ['"]*[/~]/g) || []).length
            if (bindMounts > 0 && info.volumes.length === 0) {
                info.volumes = [`${bindMounts} bind mount${bindMounts > 1 ? 's' : ''}`]
            }
        } catch (error) {
            console.error('Failed to parse compose content:', error)
        }

        return info
    }, [app.compose_content])

    const formatDate = (dateString: string) => {
        const date = new Date(dateString)
        const now = new Date()
        const diffMs = now.getTime() - date.getTime()
        const diffDays = Math.floor(diffMs / 86400000)

        if (diffDays === 0) return 'Today'
        if (diffDays === 1) return 'Yesterday'
        if (diffDays < 7) return `${diffDays} days ago`
        return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
    }

    const handleRestartService = () => {
        if (!serviceToRestart || !app.node_id) return

        restartService.mutate(
            {
                appId: app.id,
                nodeId: app.node_id,
                serviceName: serviceToRestart,
            },
            {
                onSuccess: () => {
                    toast.success('Service restarted', `Service "${serviceToRestart}" has been restarted successfully`)
                    setServiceToRestart(null)
                },
                onError: (error) => {
                    toast.error('Failed to restart service', error instanceof Error ? error.message : 'Unknown error')
                },
            }
        )
    }

    const upcoming = upcomingRuns(nextRuns)
    const isRunning = app.status === 'running'
    const tiles = [
        { label: 'CPU', value: isRunning && metrics ? formatPercent(metrics.cpuPercent) : '-' },
        { label: 'Memory', value: isRunning && metrics ? formatBytes(metrics.memoryBytes) : '-' },
        { label: 'Containers', value: isRunning && metrics ? String(metrics.containers) : '-' },
        { label: 'Restarts', value: isRunning && metrics ? String(metrics.restarts) : '-' },
    ]

    return (
        <div className="flex flex-col gap-5">
            {app.status === 'error' && <FailedStartGuide app={app} />}

            <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
                {tiles.map((tile) => (
                    <Card key={tile.label} className="flex flex-col gap-1 p-4">
                        <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">{tile.label}</span>
                        <span className="text-2xl font-semibold tabular-nums">{tile.value}</span>
                    </Card>
                ))}
            </div>

            <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
                <div className="flex flex-col gap-5">
                    <Card>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2 text-base">
                                <Layers className="h-4 w-4 text-muted-foreground" />
                                Containers
                            </CardTitle>
                        </CardHeader>
                        <CardContent>
                            {services.length > 0 ? (
                                <ul className="flex flex-col gap-2">
                                    {services.map((service) => {
                                        const isRestarting = restartService.isPending && serviceToRestart === service
                                        return (
                                            <li key={service} className="flex items-center justify-between gap-3 rounded-lg bg-muted/50 px-3 py-2">
                                                <span className="truncate font-mono text-sm font-medium">{service}</span>
                                                <div className="flex items-center gap-2">
                                                    <StatusPill kind={isRunning ? 'ok' : 'idle'} size="sm">
                                                        {isRunning ? 'Running' : 'Stopped'}
                                                    </StatusPill>
                                                    {isRunning && (
                                                        <Button
                                                            variant="ghost"
                                                            size="icon"
                                                            aria-label={`Restart ${service}`}
                                                            onClick={() => setServiceToRestart(service)}
                                                            disabled={isRestarting}
                                                        >
                                                            {isRestarting ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
                                                        </Button>
                                                    )}
                                                </div>
                                            </li>
                                        )
                                    })}
                                </ul>
                            ) : (
                                <p className="py-4 text-center text-sm text-muted-foreground">No containers found</p>
                            )}
                        </CardContent>
                    </Card>

                    <Card>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2 text-base">
                                <Globe className="h-4 w-4 text-muted-foreground" />
                                Access
                            </CardTitle>
                        </CardHeader>
                        <CardContent className="flex flex-col gap-3">
                            {app.public_url ? (
                                <div className="flex flex-wrap items-center gap-2">
                                    <a href={app.public_url} target="_blank" rel="noopener noreferrer" className="inline-flex min-h-[44px] items-center break-all font-mono text-sm text-status-info-fg hover:underline md:min-h-0">
                                        {app.public_url.replace(/^https?:\/\//, '')}
                                    </a>
                                    {app.tunnel_mode === 'quick' && <StatusPill kind="warn" size="sm">Temporary</StatusPill>}
                                </div>
                            ) : (
                                <p className="text-sm text-muted-foreground">Only reachable on your network. Add a public address when you want to share it.</p>
                            )}
                            <Link to={appHref(app, 'access')} className="inline-flex min-h-[44px] items-center text-sm font-medium hover:underline md:min-h-0">
                                Manage access
                            </Link>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2 text-base">
                                <Calendar className="h-4 w-4 text-muted-foreground" />
                                Schedule
                            </CardTitle>
                        </CardHeader>
                        <CardContent className="flex flex-col gap-3">
                            {app.schedule?.enabled && nextRuns ? (
                                upcoming.length > 0 ? (
                                    <ol className="flex flex-col gap-2.5">
                                        {upcoming.slice(0, OVERVIEW_RUNS).map((run) => (
                                            <li key={`${run.action}-${run.at}`} className="flex items-center justify-between gap-3 text-sm">
                                                <span className="flex items-center gap-2">
                                                    {run.action === 'start' ? <Play aria-hidden="true" className="h-4 w-4 text-status-ok-fg" /> : <Square aria-hidden="true" className="h-4 w-4 text-muted-foreground" />}
                                                    <span>
                                                        <span className="font-medium">{run.action === 'start' ? 'Starts' : 'Stops'}</span>{' '}
                                                        <span className="text-muted-foreground">{formatRunTime(run.at, app.schedule?.timezone)}</span>
                                                    </span>
                                                </span>
                                                <span className="shrink-0 text-[13px] tabular-nums text-muted-foreground">{formatUntil(run.at)}</span>
                                            </li>
                                        ))}
                                    </ol>
                                ) : (
                                    <p className="text-sm text-muted-foreground">No upcoming scheduled actions</p>
                                )
                            ) : (
                                <p className="text-sm text-muted-foreground">Runs all the time. Set a schedule to start and stop it automatically.</p>
                            )}
                            <Link to={appHref(app, 'schedule')} className="inline-flex min-h-[44px] items-center text-sm font-medium hover:underline md:min-h-0">
                                {app.schedule?.enabled ? 'Edit schedule' : 'Set a schedule'}
                            </Link>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2 text-base">
                                <HardDrive className="h-4 w-4 text-muted-foreground" />
                                Resources
                            </CardTitle>
                        </CardHeader>
                        <CardContent className="flex flex-col gap-3 text-sm">
                            <div className="flex items-center justify-between gap-3">
                                <span className="text-muted-foreground">Networks</span>
                                <div className="flex flex-wrap justify-end gap-1">
                                    {(composeInfo.networks.length > 0 ? composeInfo.networks : ['default']).map((network) => (
                                        <span key={network} className="rounded-full bg-muted px-2.5 py-0.5 font-mono text-xs">{network}</span>
                                    ))}
                                </div>
                            </div>
                            <div className="flex items-center justify-between gap-3">
                                <span className="text-muted-foreground">Volumes</span>
                                <span className="text-right font-medium">{composeInfo.volumes.length > 0 ? composeInfo.volumes.join(', ') : 'None'}</span>
                            </div>
                            <div className="flex items-center justify-between gap-3">
                                <span className="text-muted-foreground">Last updated</span>
                                <span className="font-medium">{formatDate(app.updated_at)}</span>
                            </div>
                        </CardContent>
                    </Card>
                </div>

                <Card className="flex max-h-[720px] flex-col">
                    <CardHeader className="flex-shrink-0">
                        <CardTitle className="flex items-center gap-2 text-base">
                            <Activity className="h-4 w-4 text-muted-foreground" />
                            Recent activity
                        </CardTitle>
                    </CardHeader>
                    <CardContent className="min-h-0 flex-1 overflow-y-auto">
                        <ActivityTimeline app={app} />
                    </CardContent>
                </Card>
            </div>

            <ConfirmationDialog
                open={!!serviceToRestart}
                onOpenChange={(open: boolean) => !open && setServiceToRestart(null)}
                title={`Restart ${serviceToRestart ?? 'container'}?`}
                description="It is unavailable for a moment while it restarts."
                confirmText="Restart"
                cancelText="Cancel"
                onConfirm={handleRestartService}
                isLoading={restartService.isPending}
                variant="default"
            />
        </div>
    )
}

export default AppOverview
