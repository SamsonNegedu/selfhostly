import React, { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useApp, useStartApp, useStopApp, useUpdateAppContainers, useDeleteApp } from '@/shared/services/api'
import { useAppStore } from '@/shared/stores/app-store'
import { useNavigate } from 'react-router-dom'
import { useToast } from '@/shared/components/ui/Toast'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { RefreshCw, Terminal, Settings, Cloud, Info, AlertTriangle } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'
import UpdateProgress from './components/UpdateProgress'
import LogViewer from './components/LogViewer'
import ComposeEditor from './components/ComposeEditor'
import CloudflareTab from './components/CloudflareTab'
import { AppActions } from './components/AppActions'
import AppBreadcrumb from '@/shared/components/layout/Breadcrumb'
import { AppDetailsSkeleton } from '@/shared/components/ui/Skeleton'
import AppOverview from './components/AppOverview'

type TabType = 'overview' | 'compose' | 'logs' | 'update' | 'cloudflare'

function AppDetails() {
    const { id } = useParams<{ id: string }>()
    const appId = id ?? undefined
    const navigate = useNavigate()
    const { data: app, isLoading, refetch, isFetching } = useApp(appId!)
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
            deleteApp.mutate(app.id, {
                onSuccess: () => {
                    // Remove from local store on success
                    useAppStore.getState().removeApp(app.id)
                    toast.success('App deleted', `"${appName}" has been deleted successfully`)
                    // Redirect to dashboard after deletion
                    navigate('/dashboard')
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
                        onClick={() => navigate('/dashboard')}
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
        { id: 'update' as TabType, label: 'Deploy Updates', icon: RefreshCw },
        { id: 'logs' as TabType, label: 'Logs', icon: Terminal },
        { id: 'cloudflare' as TabType, label: 'Cloudflare', icon: Cloud },
    ]

    return (
        <div className="space-y-6 fade-in relative">
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

            {/* Breadcrumb Navigation */}
            <div>
                <AppBreadcrumb
                    items={[
                        { label: 'Home', path: '/dashboard' },
                        { label: 'Apps', path: '/apps' },
                        { label: app.name, isCurrentPage: true }
                    ]}
                />
            </div>

            <Card className="overflow-hidden">
                <CardHeader className="pb-4">
                    <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
                        <div className="flex items-center gap-3">
                            <CardTitle className="text-2xl">{app.name}</CardTitle>
                            <div
                                className={`px-3 py-1 rounded-full text-sm font-medium flex items-center gap-2 ${getStatusColor(app.status)}`}
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
                            onStart={() => startApp.mutate(app.id, {
                                onSuccess: () => {
                                    toast.success('App started', `${app.name} has been started successfully`)
                                    refetch()
                                },
                                onError: (error) => {
                                    toast.error('Failed to start app', error.message)
                                }
                            })}
                            onStop={() => stopApp.mutate(app.id, {
                                onSuccess: () => {
                                    toast.success('App stopped', `${app.name} has been stopped successfully`)
                                    refetch()
                                },
                                onError: (error) => {
                                    toast.error('Failed to stop app', error.message)
                                }
                            })}
                            onUpdate={() => updateApp.mutate(app.id, {
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
                    <div className="flex overflow-x-auto border-b mt-6 -mx-6 px-6 scrollbar-hide">
                        {tabs.map((tab) => {
                            const Icon = tab.icon
                            return (
                                <button
                                    key={tab.id}
                                    className={`flex items-center gap-2 px-4 py-3 text-sm font-medium whitespace-nowrap border-b-2 transition-colors interactive-element ${activeTab === tab.id
                                        ? 'border-primary text-primary'
                                        : 'border-transparent text-muted-foreground hover:text-foreground hover:border-muted'
                                        }`}
                                    onClick={() => setActiveTab(tab.id)}
                                >
                                    <Icon className="h-4 w-4" />
                                    {tab.label}
                                </button>
                            )
                        })}
                    </div>
                </CardHeader>
                <CardContent className="pt-6">
                    <p className="text-sm text-muted-foreground mb-4">
                        {app.description || 'No description'}
                    </p>
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
                        <ComposeEditor
                            appId={app.id}
                            initialComposeContent={app.compose_content}
                        />
                    )}
                    {activeTab === 'update' && (
                        <UpdateProgress appId={app.id} />
                    )}
                    {activeTab === 'logs' && (
                        <LogViewer appId={app.id} />
                    )}
                    {activeTab === 'cloudflare' && (
                        <CloudflareTab appId={app.id} />
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
