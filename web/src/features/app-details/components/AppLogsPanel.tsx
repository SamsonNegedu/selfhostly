import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Card, CardContent, CardHeader } from '@/shared/components/ui/Card'
import { Button } from '@/shared/components/ui/Button'
import { SimpleDropdown, SimpleDropdownItem } from '@/shared/components/ui/SimpleDropdown'
import { Download, RefreshCw, ChevronDown, Terminal, Container } from 'lucide-react'
import { useDeploymentLogStream, isDeployJobType } from '@/shared/hooks/useDeploymentLogStream'
import { useAppServices, useAppJobs } from '@/shared/services/api'
import type { Job } from '@/shared/types/api'

type LogsSubTab = 'deployment' | 'containers'

type AppLogsPanelProps = {
  appId: string
  nodeId: string
}

function useContainerLogs(appId: string, nodeId: string, enabled: boolean) {
  const [logs, setLogs] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [selectedService, setSelectedService] = useState<string>('')
  const { data: services = [] } = useAppServices(appId, nodeId, enabled)

  const fetchLogs = useCallback(async () => {
    setIsLoading(true)
    try {
      const params = new URLSearchParams({ node_id: nodeId })
      if (selectedService) params.append('service', selectedService)
      const response = await fetch(`/api/apps/${appId}/logs?${params.toString()}`)
      const text = await response.text()
      setLogs(text)
    } catch (e) {
      console.error('Failed to fetch logs:', e)
    } finally {
      setIsLoading(false)
    }
  }, [appId, nodeId, selectedService])

  useEffect(() => {
    if (!enabled) return
    void fetchLogs()
    const interval = setInterval(() => void fetchLogs(), 5000)
    return () => clearInterval(interval)
  }, [enabled, fetchLogs])

  const downloadLogs = useCallback(() => {
    const blob = new Blob([logs], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `app-${appId}-logs.txt`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }, [appId, logs])

  return {
    logs,
    services,
    selectedService,
    setSelectedService,
    fetchLogs,
    downloadLogs,
    isLoading,
  }
}

function ContainerLogsToolbar({
  services,
  selectedService,
  onSelectService,
  onRefresh,
  onDownload,
  isLoading,
}: {
  services: string[]
  selectedService: string
  onSelectService: (s: string) => void
  onRefresh: () => void
  onDownload: () => void
  isLoading: boolean
}) {
  return (
    <div className="flex items-center gap-2 flex-wrap justify-end sm:ml-auto">
      {services.length > 0 && (
        <SimpleDropdown
          trigger={
            <Button variant="outline" size="sm" className="gap-2">
              <span>{selectedService || 'All Services'}</span>
              <ChevronDown className="h-4 w-4" />
            </Button>
          }
        >
          <div className="py-1 min-w-[160px]">
            <SimpleDropdownItem onClick={() => onSelectService('')}>All Services</SimpleDropdownItem>
            {services.map((service) => (
              <SimpleDropdownItem key={service} onClick={() => onSelectService(service)}>
                {service}
              </SimpleDropdownItem>
            ))}
          </div>
        </SimpleDropdown>
      )}
      <Button variant="outline" size="icon" onClick={onRefresh} disabled={isLoading}>
        <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
      </Button>
      <Button variant="outline" size="icon" onClick={onDownload}>
        <Download className="h-4 w-4" />
      </Button>
    </div>
  )
}

function ContainerLogsBody({ logs }: { logs: string }) {
  return (
    <div className="bg-slate-950 dark:bg-black text-green-400 dark:text-green-300 p-4 rounded-md font-mono text-sm overflow-auto max-h-[600px] border border-slate-800 dark:border-slate-900">
      {logs ? (
        <pre className="whitespace-pre-wrap">{logs}</pre>
      ) : (
        <p className="text-muted-foreground">No logs available</p>
      )}
    </div>
  )
}

export function AppLogsPanel({ appId, nodeId }: AppLogsPanelProps) {
  const [subTab, setSubTab] = useState<LogsSubTab>('containers')
  const lastAutoSwitchedJobId = useRef<string | null>(null)
  const containersActive = subTab === 'containers'
  const containerLogs = useContainerLogs(appId, nodeId, containersActive)

  const { data: appJobs = [] } = useAppJobs(appId, nodeId)

  const jobForDeploy = useMemo(() => {
    for (const j of appJobs) {
      if (isDeployJobType(j.type)) return j
    }
    return null
  }, [appJobs])

  const hasDeploy = !!(jobForDeploy && isDeployJobType(jobForDeploy.type))

  useEffect(() => {
    if (!jobForDeploy?.id) return
    if (jobForDeploy.status !== 'pending' && jobForDeploy.status !== 'running') return
    if (lastAutoSwitchedJobId.current === jobForDeploy.id) return
    lastAutoSwitchedJobId.current = jobForDeploy.id
    setSubTab('deployment')
  }, [jobForDeploy?.id, jobForDeploy?.status])

  return (
    <Card>
      <CardHeader className="pb-0 pt-3 px-4 sm:px-6">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-3">
          <div
            className="flex rounded-lg border border-border p-0.5 bg-muted/30 w-fit max-w-full overflow-x-auto"
            role="tablist"
            aria-label="Log type"
          >
            <button
              type="button"
              role="tab"
              aria-selected={subTab === 'deployment'}
              disabled={!hasDeploy}
              onClick={() => setSubTab('deployment')}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md whitespace-nowrap transition-colors ${
                subTab === 'deployment'
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              } ${!hasDeploy ? 'opacity-50 cursor-not-allowed' : ''}`}
            >
              <Terminal className="h-3.5 w-3.5 shrink-0" />
              Deployment
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={subTab === 'containers'}
              onClick={() => setSubTab('containers')}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md whitespace-nowrap transition-colors ${
                subTab === 'containers'
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <Container className="h-3.5 w-3.5 shrink-0" />
              Containers
            </button>
          </div>

          {containersActive && (
            <ContainerLogsToolbar
              services={containerLogs.services}
              selectedService={containerLogs.selectedService}
              onSelectService={containerLogs.setSelectedService}
              onRefresh={() => void containerLogs.fetchLogs()}
              onDownload={containerLogs.downloadLogs}
              isLoading={containerLogs.isLoading}
            />
          )}
        </div>
      </CardHeader>
      <CardContent className="pt-2 px-4 pb-4 sm:px-6 sm:pt-3">
        {subTab === 'deployment' && (
          <>
            {hasDeploy && jobForDeploy ? (
              <DeploymentLogBody job={jobForDeploy} nodeId={nodeId} />
            ) : (
              <p className="text-sm text-muted-foreground py-8 text-center">
                No deployment run in context. Start, update, or create an app to see compose pull / build / up output here.
              </p>
            )}
          </>
        )}
        {subTab === 'containers' && <ContainerLogsBody logs={containerLogs.logs} />}
      </CardContent>
    </Card>
  )
}

function DeploymentLogBody({ job, nodeId }: { job: Job; nodeId: string }) {
  const endRef = useRef<HTMLDivElement>(null)
  const active = job.status === 'pending' || job.status === 'running'
  const { lines } = useDeploymentLogStream(job.id, nodeId, true)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [lines])

  const lineClass = (text: string) => {
    if (text.startsWith('[step]')) return 'text-sky-400 dark:text-sky-300 font-medium'
    if (text.startsWith('[pull]')) return 'text-green-400/90 dark:text-green-300/90'
    if (text.startsWith('[up]')) return 'text-emerald-400/90 dark:text-emerald-300/90'
    return 'text-green-400 dark:text-green-300'
  }

  return (
    <div>
      <p className="text-xs text-muted-foreground mb-2">
        <span className="font-medium text-sky-600 dark:text-sky-400">[step]</span> app milestones ·{' '}
        <span className="font-medium text-green-600 dark:text-green-400">[pull]</span> /{' '}
        <span className="font-medium text-emerald-600 dark:text-emerald-400">[up]</span> compose CLI
        {active ? ' · live' : ''}
      </p>
      <div className="bg-slate-950 dark:bg-black p-3 rounded-md font-mono text-xs sm:text-sm overflow-auto max-h-[min(55vh,480px)] border border-slate-800 dark:border-slate-900">
        {lines.length > 0 ? (
          <pre className="whitespace-pre-wrap break-words">
            {lines.map((l) => (
              <span key={l.seq} className={lineClass(l.text)}>
                {l.text}
                {'\n'}
              </span>
            ))}
            <div ref={endRef} />
          </pre>
        ) : (
          <p className="text-muted-foreground">
            {active ? 'Waiting for output…' : 'No deployment output captured for this run.'}
          </p>
        )}
      </div>
    </div>
  )
}
