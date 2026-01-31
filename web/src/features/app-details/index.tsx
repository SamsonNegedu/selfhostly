import React, { useState, useMemo } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { useApp, useStartApp, useStopApp, useUpdateAppContainers, useDeleteApp, useApps } from '@/shared/services/api'
import { useAppStore } from '@/shared/stores/app-store'
import { useNavigate } from 'react-router-dom'
import { useToast } from '@/shared/components/ui/Toast'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { Terminal, Settings, Cloud, Info, AlertTriangle } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'
import LogViewer from './components/LogViewer'
import ComposeEditor from './components/ComposeEditor'
import CloudflareTab from './components/CloudflareTab'
import { AppActions } from './components/AppActions'
import AppBreadcrumb from '@/shared/components/layout/Breadcrumb'
import { AppDetailsSkeleton } from '@/shared/components/ui/Skeleton'
import AppOverview from './components/AppOverview'

type TabType = 'overview' | 'compose' | 'logs' | 'cloudflare'

function AppDetails() {
    const { id } = useParams<{ id: string }>()
    const [searchParams] = useSearchParams()
    const appId = id ?? undefined
    const navigate = useNavigate()

    // Try to get node_id from URL query param first
    const nodeIdFromUrl = searchParams.get('node_id')

    // Get node_id from app store if available (for initial load)
    const apps = useAppStore((state) => state.apps)
    const cachedApp = apps.find(a => a.id === appId)

    // Fetch apps list if nodeId is not available from cache or URL
    const shouldFetchApps = !nodeIdFromUrl && !cachedApp?.node_id
    const { data: appsList, isLoading: isLoadingApps } = useApps(undefined) // Fetch from all nodes

    // Determine nodeId: URL param > cache > fetched apps list
    const nodeId = useMemo(() => {
        if (nodeIdFromUrl) return nodeIdFromUrl
        if (cachedApp?.node_id) return cachedApp.node_id
        if (appsList) {
            const foundApp = appsList.find(a => a.id === appId)
            return foundApp?.node_id
        }
        return undefined
    }, [nodeIdFromUrl, cachedApp?.node_id, appsList, appId])

    const { data: app, isLoading: isLoadingApp, refetch, isFetching } = useApp(appId!, nodeId || '')

    // Combined loading state: wait for apps list if we need it to find nodeId
    const isLoading = isLoadingApp || (shouldFetchApps && isLoadingApps)
    const startApp = useStartApp()
    const stopApp = useStopApp()
    const updateApp = useUpdateAppContainers()
    const deleteApp = useDeleteApp()
    const { toast } = useToast()

    // State for confirmation dialog
    const [showDeleteDialog, setShowDeleteDialog] = useState(false)

    const handleDelete = () => {
        if (app) {
            setShowDeleteDialog(true)
        }
    }

    const confirmDelete = () => {
        if (app) {
            const appName = app.name

            // Show immediate feedback that deletion started
            toast.info('Deleting app', `Deleting "${appName}"...`)

            // Trigger deletion
            deleteApp.mutate({ id: app.id, nodeId: app.node_id }, {
                onSuccess: () => {
                    // Remove from local store on success
                    useAppStore.getState().removeApp(app.id)
                    toast.success('App deleted', `"${appName}" has been deleted successfully`)
                    // Redirect to dashboard after deletion
                    navigate('/apps')
                },
                onError: (error) => {
                    toast.error('Failed to delete app', error.message)
                }
            })

            // Close dialog
            setShowDeleteDialog(false)
        }
    }

    const [activeTab, setActiveTab] = React.useState<TabType>('overview')

    if (isLoading) {
        return <AppDetailsSkeleton />
    }

    if (!app) {
        return (
            <div className="flex items-center justify-center min-h-[400px]">
                <div className="text-center max-w-md fade-in">
                    <AlertTriangle className="h-12 w-12 text-destructive mx-auto mb-4" />
                    <h2 className="text-xl font-semibold mb-2">App not found</h2>
                    <p className="text-muted-foreground mb-4">
                        The application you're looking for doesn't exist or has been deleted.
                    </p>
                    <Button
                        onClick={() => navigate('/apps')}
                        className="button-press"
                    >
                        Return to Dashboard
                    </Button>
                </div>
            </div>
        )
    }

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'running':
                return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
            case 'stopped':
                return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
            case 'updating':
                return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
            case 'error':
                return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
            default:
                return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
        }
    }

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'running':
                return <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
            case 'updating':
                return <div className="w-2 h-2 bg-blue-500 rounded-full animate-spin"></div>
            case 'error':
                return <div className="w-2 h-2 bg-red-500 rounded-full"></div>
            default:
                return null
        }
    }

    const tabs = [
        { id: 'overview' as TabType, label: 'Overview', icon: Info },
        { id: 'compose' as TabType, label: 'Compose Editor', icon: Settings },
        { id: 'logs' as TabType, label: 'Logs', icon: Terminal },
        { id: 'cloudflare' as TabType, label: 'Cloudflare', icon: Cloud },
    ]

    return (
        <div className="space-y-4 sm:space-y-6 fade-in relative">
            {/* Deletion Overlay */}
            {deleteApp.isPending && (
                <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center">
                    <div className="bg-card border rounded-lg p-8 shadow-lg max-w-md w-full mx-4 fade-in">
                        <div className="flex flex-col items-center text-center gap-4">
                            <div className="h-16 w-16 border-4 border-destructive border-t-transparent rounded-full animate-spin"></div>
                            <div>
                                <h3 className="text-lg font-semibold mb-2">Deleting {app.name}</h3>
                                <p className="text-sm text-muted-foreground">
                                    Please wait while we remove the application and clean up resources...
                                </p>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {/* Breadcrumb Navigation - Desktop only */}
            <AppBreadcrumb
                items={[
                    { label: 'Home', path: '/apps' },
                    { label: 'Apps', path: '/apps' },
                    { label: app.name, isCurrentPage: true }
                ]}
            />

            <Card className="overflow-hidden">
                <CardHeader className="pb-3 sm:pb-4 p-4 sm:p-6">
                    <div className="flex flex-col gap-3 sm:gap-4">
                        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-4">
                            <div className="flex items-center gap-2 sm:gap-3 flex-wrap">
                                <CardTitle className="text-xl sm:text-2xl">{app.name}</CardTitle>
                                <div
                                    className={`px-2 sm:px-3 py-0.5 sm:py-1 rounded-full text-xs sm:text-sm font-medium flex items-center gap-1.5 sm:gap-2 ${getStatusColor(app.status)}`}
                                >
                                    {getStatusIcon(app.status)}
                                    {app.status}
                                </div>
                            </div>
                            <AppActions
                                appStatus={app.status}
                                isStartPending={startApp.isPending}
                                isStopPending={stopApp.isPending}
                                isUpdatePending={updateApp.isPending}
                                isDeletePending={deleteApp.isPending}
                                isRefreshing={isFetching}
                                onRefresh={() => refetch()}
                                onStart={() => startApp.mutate({ id: app.id, nodeId: app.node_id }, {
                                    onSuccess: () => {
                                        toast.success('App started', `${app.name} has been started successfully`)
                                        refetch()
                                    },
                                    onError: (error) => {
                                        toast.error('Failed to start app', error.message)
                                    }
                                })}
                                onStop={() => stopApp.mutate({ id: app.id, nodeId: app.node_id }, {
                                    onSuccess: () => {
                                        toast.success('App stopped', `${app.name} has been stopped successfully`)
                                        refetch()
                                    },
                                    onError: (error) => {
                                        toast.error('Failed to stop app', error.message)
                                    }
                                })}
                                onUpdate={() => updateApp.mutate({ id: app.id, nodeId: app.node_id }, {
                                    onSuccess: () => {
                                        toast.success('Update started', `${app.name} update process has begun`)
                                        refetch()
                                    },
                                    onError: (error) => {
                                        toast.error('Failed to start update', error.message)
                                    }
                                })}
                                onDelete={handleDelete}
                            />
                        </div>

                        {/* Enhanced Tab Navigation */}
                        <div className="flex overflow-x-auto border-b -mx-4 sm:-mx-6 px-4 sm:px-6 scrollbar-hide">
                            {tabs.map((tab) => {
                                const Icon = tab.icon
                                return (
                                    <button
                                        key={tab.id}
                                        className={`flex items-center gap-1.5 sm:gap-2 px-3 sm:px-4 py-2.5 sm:py-3 text-xs sm:text-sm font-medium whitespace-nowrap border-b-2 transition-colors interactive-element ${activeTab === tab.id
                                            ? 'border-primary text-primary'
                                            : 'border-transparent text-muted-foreground hover:text-foreground hover:border-muted'
                                            }`}
                                        onClick={() => setActiveTab(tab.id)}
                                    >
                                        <Icon className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
                                        <span className="hidden xs:inline">{tab.label}</span>
                                    </button>
                                )
                            })}
                        </div>
                    </div>
                </CardHeader>
                <CardContent className="pt-4 sm:pt-6 p-4 sm:p-6">
                    {app.description && (
                        <p className="text-xs sm:text-sm text-muted-foreground mb-3 sm:mb-4">
                            {app.description}
                        </p>
                    )}
                    {app.tunnel_mode === 'quick' && (
                        <div className="mb-4 p-3 rounded-lg border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-900/20 text-sm text-amber-800 dark:text-amber-200">
                            <span className="font-medium">Quick Tunnel:</span> This app uses a temporary trycloudflare.com URL. The URL may change if the container restarts. Limits: 200 concurrent requests, no Server-Sent Events. Switch to a custom domain from the Cloudflare tab for a stable URL.
                        </div>
                    )}
                    {app.public_url && (
                        <div className="mb-4">
                            <a
                                href={app.public_url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-primary hover:underline inline-flex items-center gap-1 interactive-element"
                            >
                                <Cloud className="h-3 w-3" />
                                {app.public_url}
                            </a>
                        </div>
                    )}
                    {app.status === 'error' && app.error_message && (
                        <div className="text-sm text-red-600 dark:text-red-400 mb-4 p-3 bg-red-50 dark:bg-red-900/20 rounded border border-red-200 dark:border-red-800">
                            <span className="font-medium">Error:</span> {app.error_message}
                        </div>
                    )}
                    {activeTab === 'overview' && (
                        <AppOverview app={app} />
                    )}
                    {activeTab === 'compose' && (
                        !app.node_id ? (
                            <div className="flex items-center justify-center min-h-[200px] text-muted-foreground">
                                <AlertTriangle className="h-5 w-5 mr-2" />
                                Unable to load compose editor: node_id is missing
                            </div>
                        ) : (
                            <ComposeEditor
                                appId={app.id}
                                nodeId={app.node_id}
                                initialComposeContent={app.compose_content}
                            />
                        )
                    )}
                    {activeTab === 'logs' && (
                        !app.node_id ? (
                            <div className="flex items-center justify-center min-h-[200px] text-muted-foreground">
                                <AlertTriangle className="h-5 w-5 mr-2" />
                                Unable to load logs: node_id is missing
                            </div>
                        ) : (
                            <LogViewer appId={app.id} nodeId={app.node_id} />
                        )
                    )}
                    {activeTab === 'cloudflare' && (
                        !app.node_id ? (
                            <div className="flex items-center justify-center min-h-[200px] text-muted-foreground">
                                <AlertTriangle className="h-5 w-5 mr-2" />
                                Unable to load tunnel info: node_id is missing
                            </div>
                        ) : (
                            <CloudflareTab appId={app.id} nodeId={app.node_id} />
                        )
                    )}
                </CardContent>
            </Card>

            {/* Confirmation Dialog */}
            {app && (
                <ConfirmationDialog
                    open={showDeleteDialog}
                    onOpenChange={(open: boolean) => !open && setShowDeleteDialog(false)}
                    title="Delete App"
                    description={`Are you sure you want to delete "${app.name}"? This action cannot be undone.`}
                    confirmText="Delete"
                    cancelText="Cancel"
                    onConfirm={confirmDelete}
                    isLoading={deleteApp.isPending}
                    variant="destructive"
                />
            )}
        </div>
    )
}

export default AppDetails
