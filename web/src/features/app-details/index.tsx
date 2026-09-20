import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Button } from '@/shared/components/ui/Button'
import { AppDetailsSkeleton } from '@/shared/components/ui/Skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/shared/components/ui/Tabs'
import { useIsPhone } from '@/shared/hooks/useMediaQuery'
import { APP_TAB_LABELS, AVAILABLE_APP_TABS, resolveAppTab, type AppTab } from '@/shared/lib/routes'
import { useNodes } from '@/shared/services/api'
import { useFleetActions } from '@/features/dashboard/hooks/useFleetActions'
import AppHeader from './components/AppHeader'
import AppTabContent from './components/AppTabContent'
import { useAppRecord } from './hooks/useAppRecord'

type TabType = Extract<AppTab, 'overview' | 'config' | 'environment' | 'logs' | 'access' | 'schedule' | 'history'>

function AppDetails() {
    const { id } = useParams<{ id: string }>()
    const [searchParams, setSearchParams] = useSearchParams()
    const appId = id ?? undefined
    const navigate = useNavigate()

    const { app, isLoading, refetch, isFetching } = useAppRecord(appId, searchParams.get('node_id'))
    const { data: nodes = [] } = useNodes()

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

    const nodeName = nodes.find((node) => node.id === app.node_id)?.name ?? app.node_id
    const tabs = AVAILABLE_APP_TABS.map((tab) => ({
        id: tab as TabType,
        label: tab === 'overview' ? 'Overview' : APP_TAB_LABELS[tab],
    }))
    return (
        <div className={`flex flex-col gap-5 ${phone && activeTab === 'overview' ? 'pb-28' : ''}`}>
            <AppHeader
                app={app}
                nodeName={nodeName}
                actions={actions}
                isRefreshing={isFetching}
                sticky={phone && activeTab === 'overview'}
                onRefresh={() => refetch()}
            />

            <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as TabType)}>
                <TabsList aria-label="App sections">
                    {tabs.map((tab) => (
                        <TabsTrigger key={tab.id} value={tab.id}>
                            {tab.label}
                        </TabsTrigger>
                    ))}
                </TabsList>
            </Tabs>

            <AppTabContent app={app} tab={activeTab} />

            {actions.dialog}
        </div>
    )
}

export default AppDetails
