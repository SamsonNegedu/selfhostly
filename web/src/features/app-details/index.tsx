import React, { useMemo, useEffect } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { useApp, useApps, useNodes } from '@/shared/services/api'
import { useAppStore } from '@/shared/stores/app-store'
import { useNavigate } from 'react-router-dom'
import { AlertTriangle, Globe } from 'lucide-react'
import { AppTile } from '@/shared/components/ui/AppTile'
import { Button } from '@/shared/components/ui/Button'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { Tabs, TabsList, TabsTrigger } from '@/shared/components/ui/Tabs'
import { appStatusMeta } from '@/shared/lib/status'
import { APP_TAB_LABELS, AVAILABLE_APP_TABS } from '@/shared/lib/routes'
import { useFleetActions } from '@/features/dashboard/hooks/useFleetActions'
import { useIsPhone } from '@/shared/hooks/useMediaQuery'
import { AppLogsPanel } from './components/AppLogsPanel'
import ComposeEditor from './components/ComposeEditor'
import CloudflareTab from './components/CloudflareTab'
import { AppActions } from './components/AppActions'
import { AppDetailsSkeleton } from '@/shared/components/ui/Skeleton'
import { resolveAppTab, type AppTab } from '@/shared/lib/routes'
import AppOverview from './components/AppOverview'
import EnvironmentTab from './components/EnvironmentTab'
import HistoryTab from './components/HistoryTab'
import { ScheduleEditor } from './components/ScheduleEditor'

type TabType = Extract<AppTab, 'overview' | 'config' | 'environment' | 'logs' | 'access' | 'schedule' | 'history'>

function AppDetails() {
    const { id } = useParams<{ id: string }>()
    const [searchParams, setSearchParams] = useSearchParams()
    const appId = id ?? undefined
    const navigate = useNavigate()

    // Try to get node_id from URL query param first
    const nodeIdFromUrl = searchParams.get('node_id')

    // Get node_id from app store if available (for initial load)
    const apps = useAppStore((state) => state.apps)
    const cachedApp = apps.find((a) => a.id === appId)

    // Fetch apps list if nodeId is not available from cache or URL
    const shouldFetchApps = !nodeIdFromUrl && !cachedApp?.node_id
    const { data: appsList, isLoading: isLoadingApps } = useApps(undefined) // Fetch from all nodes

    // Determine nodeId: URL param > cache > fetched apps list
    const nodeId = useMemo(() => {
        if (nodeIdFromUrl) return nodeIdFromUrl
        if (cachedApp?.node_id) return cachedApp.node_id
        if (appsList) {
            const foundApp = appsList.find((a) => a.id === appId)
            return foundApp?.node_id
        }
        return undefined
    }, [nodeIdFromUrl, cachedApp?.node_id, appsList, appId])

    const { data: app, isLoading: isLoadingApp, refetch, isFetching } = useApp(appId!, nodeId || '')
    const { data: nodes = [] } = useNodes()

    // Track if nodeId was previously undefined (query was disabled)
    const prevNodeIdRef = React.useRef<string | undefined>(undefined)

    // Refetch app data when nodeId becomes available for the first time
    // This ensures we get fresh data from the backend, not stale cached data
    useEffect(() => {
        const wasDisabled = prevNodeIdRef.current === undefined || prevNodeIdRef.current === ''
        const isNowEnabled = !!nodeId && nodeId !== ''

        if (wasDisabled && isNowEnabled && appId) {
            // Query was just enabled - ensure we refetch to get deterministic backend state
            // This handles the case where app was created and we navigated here before nodeId was resolved
            refetch()
        }

        prevNodeIdRef.current = nodeId
    }, [appId, nodeId, refetch])

    // Combined loading state: wait for apps list if we need it to find nodeId
    const isLoading = isLoadingApp || (shouldFetchApps && isLoadingApps)
    const actions = useFleetActions({ onDeleted: () => navigate('/apps') })
    const phone = useIsPhone()

    // Get active tab from URL, default to 'overview'
    const activeTab = resolveAppTab(searchParams.get('tab')) as TabType

    // Update URL when tab changes
    const setActiveTab = (tab: TabType) => {
        setSearchParams(
            (prev) => {
                const newParams = new URLSearchParams(prev)
                newParams.set('tab', tab)
                return newParams
            },
            { replace: true },
        ) // Use replace to avoid cluttering browser history
    }

    if (isLoading) {
        return <AppDetailsSkeleton />
    }

    if (!app) {
        return (
            <ErrorState title="App not found" description="This app does not exist or has been deleted.">
                <Button onClick={() => navigate('/apps')}>Back to Fleet</Button>
            </ErrorState>
        )
    }

    const meta = appStatusMeta(app.status)
    const nodeName = nodes.find((node) => node.id === app.node_id)?.name ?? app.node_id
    const tabs = AVAILABLE_APP_TABS.map((tab) => ({
        id: tab as TabType,
        label: tab === 'overview' ? 'Overview' : APP_TAB_LABELS[tab],
    }))
    const missingNode = (what: string) => (
        <div className="flex min-h-[200px] items-center justify-center text-muted-foreground">
            <AlertTriangle className="mr-2 h-5 w-5" />
            Unable to load {what}: node_id is missing
        </div>
    )

    return (
        <div className={`flex flex-col gap-5 ${phone && activeTab === 'overview' ? 'pb-28' : ''}`}>
            <header className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
                <div className="flex min-w-0 items-start gap-3.5">
                    <AppTile name={app.name} size="lg" />
                    <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2.5">
                            <h1 className="truncate text-2xl font-semibold tracking-tight">{app.name}</h1>
                            <StatusPill kind={meta.kind}>{meta.label}</StatusPill>
                        </div>
                        <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-[13px] text-muted-foreground">
                            {app.public_url && (
                                <a
                                    href={app.public_url}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="inline-flex min-h-[44px] items-center gap-1.5 font-mono text-status-info-fg hover:underline md:min-h-0"
                                >
                                    <Globe className="h-3.5 w-3.5 shrink-0" />
                                    <span className="truncate">{app.public_url.replace(/^https?:\/\//, '')}</span>
                                </a>
                            )}
                            {nodeName && <span>on {nodeName}</span>}
                        </div>
                        {app.description && <p className="mt-1 text-sm text-muted-foreground">{app.description}</p>}
                    </div>
                </div>
                <AppActions
                    appId={app.id}
                    nodeId={app.node_id}
                    appStatus={app.status}
                    publicUrl={app.public_url}
                    isBusy={actions.isBusy(app.id)}
                    isRefreshing={isFetching}
                    sticky={phone && activeTab === 'overview'}
                    onRefresh={() => refetch()}
                    onStart={() => actions.start(app)}
                    onStop={() => actions.requestStop(app)}
                    onUpdate={() => actions.requestUpdate(app)}
                    onDelete={() => actions.requestDelete(app)}
                />
            </header>

            <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as TabType)}>
                <TabsList aria-label="App sections">
                    {tabs.map((tab) => (
                        <TabsTrigger key={tab.id} value={tab.id}>
                            {tab.label}
                        </TabsTrigger>
                    ))}
                </TabsList>
            </Tabs>

            <div>
                {activeTab === 'overview' && <AppOverview app={app} />}
                {activeTab === 'config' &&
                    (app.node_id ? (
                        <ComposeEditor
                            appId={app.id}
                            nodeId={app.node_id}
                            initialComposeContent={app.compose_content}
                        />
                    ) : (
                        missingNode('the compose editor')
                    ))}
                {activeTab === 'environment' &&
                    (app.node_id ? <EnvironmentTab app={app} /> : missingNode('the environment'))}
                {activeTab === 'logs' &&
                    (app.node_id ? <AppLogsPanel appId={app.id} nodeId={app.node_id} /> : missingNode('logs'))}
                {activeTab === 'access' &&
                    (app.node_id ? (
                        <CloudflareTab appId={app.id} nodeId={app.node_id} composeContent={app.compose_content} />
                    ) : (
                        missingNode('tunnel info')
                    ))}
                {activeTab === 'history' && (app.node_id ? <HistoryTab app={app} /> : missingNode('the history'))}
                {activeTab === 'schedule' &&
                    (app.node_id ? (
                        <ScheduleEditor appId={app.id} nodeId={app.node_id} />
                    ) : (
                        missingNode('the schedule')
                    ))}
            </div>

            {actions.dialog}
        </div>
    )
}

export default AppDetails
