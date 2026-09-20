import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Container, Download, RefreshCw, TerminalSquare } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent } from '@/shared/components/ui/Card'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/components/ui/Select'
import { SegmentedControl } from '@/shared/components/ui/SegmentedControl'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { Terminal, type TerminalLine } from '@/shared/components/ui/Terminal'
import { isDeployJobType, useDeploymentLogStream } from '@/shared/hooks/useDeploymentLogStream'
import { useAppJobs, useAppServices } from '@/shared/services/api'
import { describeError } from '@/shared/lib/errors'
import { jobStatusMeta } from '@/shared/lib/status'
import type { Job } from '@/shared/types/api'
import { toTerminalLines } from '../lib/log-lines'

type LogsSubTab = 'deployment' | 'containers'

const REFRESH_MS = 5000
const ALL_SERVICES = '__all__'
const TERMINAL_CLASS = 'h-[420px] max-md:h-[60vh]'

const SUB_TAB_OPTIONS = [
    { value: 'deployment', label: 'Deployment', icon: <TerminalSquare className="h-4 w-4" /> },
    { value: 'containers', label: 'Containers', icon: <Container className="h-4 w-4" /> },
]

interface ContainerLogsState {
    logs: string | null
    error: unknown
    isLoading: boolean
    refresh: () => void
}

// The API answers a failed request with a JSON body. That must never be shown as if it were log output.
function useContainerLogs(appId: string, nodeId: string, service: string, enabled: boolean): ContainerLogsState {
    const [logs, setLogs] = useState<string | null>(null)
    const [error, setError] = useState<unknown>(null)
    const [isLoading, setIsLoading] = useState(false)

    const refresh = useCallback(async () => {
        setIsLoading(true)
        try {
            const params = new URLSearchParams({ node_id: nodeId })
            if (service) params.append('service', service)
            const response = await fetch(`/api/apps/${appId}/logs?${params.toString()}`)
            const text = await response.text()
            if (!response.ok) throw new Error(text)
            setLogs(text)
            setError(null)
        } catch (failure) {
            setError(failure)
        } finally {
            setIsLoading(false)
        }
    }, [appId, nodeId, service])

    useEffect(() => {
        if (!enabled) return
        // The first load waits a tick, so the loading flag is not set from inside the effect itself.
        const first = setTimeout(() => void refresh(), 0)
        const interval = setInterval(() => void refresh(), REFRESH_MS)
        return () => {
            clearTimeout(first)
            clearInterval(interval)
        }
    }, [enabled, refresh])

    return { logs, error, isLoading, refresh: () => void refresh() }
}

function download(appId: string, name: string, lines: TerminalLine[]) {
    const blob = new Blob([lines.map((line) => line.text).join('\n')], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${name}-${appId}.log`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
}

export function AppLogsPanel({ appId, nodeId }: { appId: string; nodeId: string }) {
    const [subTab, setSubTab] = useState<LogsSubTab>('containers')
    const [service, setService] = useState('')
    const lastAutoSwitchedJobId = useRef<string | null>(null)
    const containersActive = subTab === 'containers'

    const { data: services = [] } = useAppServices(appId, nodeId, containersActive)
    const container = useContainerLogs(appId, nodeId, service, containersActive)
    const { data: appJobs = [] } = useAppJobs(appId, nodeId)

    const deployJob = useMemo(() => appJobs.find((job) => isDeployJobType(job.type)) ?? null, [appJobs])

    // A deployment that starts while you are looking at the logs takes over the view once.
    useEffect(() => {
        if (!deployJob?.id) return
        if (deployJob.status !== 'pending' && deployJob.status !== 'running') return
        if (lastAutoSwitchedJobId.current === deployJob.id) return
        lastAutoSwitchedJobId.current = deployJob.id
        setSubTab('deployment')
    }, [deployJob?.id, deployJob?.status])

    const containerLines = useMemo(() => (container.logs ? toTerminalLines(container.logs) : []), [container.logs])

    return (
        <Card>
            <CardContent className="flex flex-col gap-4 p-4">
                <div className="flex flex-wrap items-center gap-2">
                    <SegmentedControl
                        aria-label="Log type"
                        options={SUB_TAB_OPTIONS}
                        value={subTab}
                        onValueChange={(value) => setSubTab(value as LogsSubTab)}
                    />

                    {containersActive && (
                        <div className="ml-auto flex flex-wrap items-center gap-2">
                            {services.length > 0 && (
                                <div className="w-[180px] max-md:flex-1">
                                    <Select
                                        value={service || ALL_SERVICES}
                                        onValueChange={(value) => setService(value === ALL_SERVICES ? '' : value)}
                                    >
                                        <SelectTrigger aria-label="Container">
                                            <SelectValue />
                                        </SelectTrigger>
                                        <SelectContent>
                                            <SelectItem value={ALL_SERVICES}>All containers</SelectItem>
                                            {services.map((name) => (
                                                <SelectItem key={name} value={name}>
                                                    {name}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                </div>
                            )}
                            <Button
                                variant="outline"
                                size="icon"
                                aria-label="Refresh logs"
                                onClick={container.refresh}
                                disabled={container.isLoading}
                            >
                                <RefreshCw className={`h-4 w-4 ${container.isLoading ? 'animate-spin' : ''}`} />
                            </Button>
                            <Button
                                variant="outline"
                                size="icon"
                                aria-label="Download logs"
                                onClick={() => download(appId, 'container-logs', containerLines)}
                                disabled={containerLines.length === 0}
                            >
                                <Download className="h-4 w-4" />
                            </Button>
                        </div>
                    )}
                </div>

                {subTab === 'deployment' ? (
                    deployJob ? (
                        <DeploymentLogBody job={deployJob} nodeId={nodeId} appId={appId} />
                    ) : (
                        <EmptyState
                            icon={<TerminalSquare className="h-5 w-5" />}
                            title="No deployment yet"
                            description="Start, update or create the app and the pull and start output shows up here."
                            className="py-12"
                        />
                    )
                ) : container.error && container.logs === null ? (
                    <ErrorState title="Could not load the logs" error={container.error} onRetry={container.refresh} />
                ) : container.logs === null ? (
                    <div role="status" aria-label="Loading logs" className="flex flex-col gap-2">
                        {[70, 90, 55, 80, 65].map((width) => (
                            <Skeleton key={width} className="h-4" style={{ width: `${width}%` }} />
                        ))}
                    </div>
                ) : containerLines.length === 0 ? (
                    <EmptyState
                        icon={<Container className="h-5 w-5" />}
                        title="No output yet"
                        description="The containers have not written anything. Logs refresh every few seconds."
                        className="py-12"
                    />
                ) : (
                    <>
                        {container.error !== null && (
                            <p role="status" className="text-[13px] text-status-warn-fg">
                                Could not refresh. {describeError(container.error)} Showing the last output.
                            </p>
                        )}
                        <Terminal
                            lines={containerLines}
                            aria-label="Container logs"
                            className={TERMINAL_CLASS}
                            follow
                        />
                    </>
                )}
            </CardContent>
        </Card>
    )
}

function DeploymentLogBody({ job, nodeId, appId }: { job: Job; nodeId: string; appId: string }) {
    const active = job.status === 'pending' || job.status === 'running'
    const { lines } = useDeploymentLogStream(job.id, nodeId, true)
    const meta = jobStatusMeta(job.status)
    const terminalLines = useMemo(() => lines.map((line) => toTerminalLines(line.text)[0]), [lines])

    return (
        <div className="flex flex-col gap-3">
            <div className="flex flex-wrap items-center gap-2 text-[13px] text-muted-foreground">
                <StatusPill kind={meta.kind} size="sm">
                    {active ? 'Live' : meta.label}
                </StatusPill>
                <span>
                    <span className="font-mono">[step]</span> milestones · <span className="font-mono">[pull]</span> and{' '}
                    <span className="font-mono">[up]</span> compose output
                </span>
                {terminalLines.length > 0 && (
                    <Button
                        variant="ghost"
                        size="sm"
                        className="ml-auto"
                        onClick={() => download(appId, 'deployment-log', terminalLines)}
                    >
                        <Download className="h-4 w-4" />
                        Download
                    </Button>
                )}
            </div>
            {terminalLines.length > 0 ? (
                <Terminal
                    lines={terminalLines}
                    aria-label="Deployment log"
                    className={TERMINAL_CLASS}
                    follow={active}
                />
            ) : (
                <EmptyState
                    icon={<TerminalSquare className="h-5 w-5" />}
                    title={active ? 'Waiting for output' : 'No output was captured'}
                    description={
                        active
                            ? 'The first lines appear as soon as the deployment writes them.'
                            : 'This run finished without writing any output.'
                    }
                    className="py-12"
                />
            )}
        </div>
    )
}
