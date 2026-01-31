import { useState } from 'react'
import { useAppStore } from '@/shared/stores/app-store'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { Play, Pause, RefreshCw, Trash2, ExternalLink, Clock, Search, Loader2, MoreVertical, TrendingUp } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useDeleteApp, useStartApp, useStopApp, useUpdateAppContainers } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import { SimpleDropdown, SimpleDropdownItem } from '@/shared/components/ui/simple-dropdown'
import type { App } from '@/shared/types/api'

interface AppToDelete {
    id: string
    name: string
}

interface AppListProps {
    filteredApps?: App[]
}

function AppList({ filteredApps }: AppListProps) {
    const apps = filteredApps || useAppStore((state) => state.apps)
    const navigate = useNavigate()
    const deleteApp = useDeleteApp()
    const startApp = useStartApp()
    const stopApp = useStopApp()
    const updateApp = useUpdateAppContainers()
    const { toast } = useToast()

    // State for confirmation dialog and deletion tracking
    const [appToDelete, setAppToDelete] = useState<AppToDelete | null>(null)
    const [deletingAppId, setDeletingAppId] = useState<string | null>(null)

    // Handle delete with confirmation dialog
    const handleDelete = (appId: string, appName: string) => {
        setAppToDelete({ id: appId, name: appName })
    }

    // Confirm deletion
    const confirmDelete = () => {
        if (appToDelete) {
            const appName = appToDelete.name
            const appId = appToDelete.id
            // Get the node_id from the app
            const app = apps.find(a => a.id === appId)

            if (!app?.node_id) {
                toast.error('Delete Failed', 'Unable to determine app node')
                setAppToDelete(null)
                return
            }

            // Mark app as being deleted
            setDeletingAppId(appId)

            // Show immediate feedback that deletion started
            toast.info('Deleting app', `Deleting "${appName}"...`)

            // Then trigger the actual deletion
            deleteApp.mutate({ id: appId, nodeId: app.node_id }, {
                onSuccess: () => {
                    // Optimistically remove from local store on success
                    useAppStore.getState().removeApp(appId)
                    toast.success('App deleted', `"${appName}" has been deleted successfully`)
                    setDeletingAppId(null)
                },
                onError: (error) => {
                    toast.error('Failed to delete app', error.message)
                    setDeletingAppId(null)
                }
            })

            // Reset dialog state
            setAppToDelete(null)
        }
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

    // Check if we have any apps in the store (for empty state logic)
    const hasAnyApps = useAppStore((state) => state.apps.length > 0)

    if (apps.length === 0) {
        if (hasAnyApps && filteredApps) {
            // Apps exist but none match filters
            return (
                <div className="text-center py-16 scale-in">
                    <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-muted mb-4">
                        <Search className="w-8 h-8 text-muted-foreground" />
                    </div>
                    <h3 className="text-lg font-semibold mb-2">No matching apps</h3>
                    <p className="text-muted-foreground mb-4 max-w-sm mx-auto">
                        Try adjusting your search or filter criteria
                    </p>
                </div>
            )
        }
        return (
            <div className="text-center py-16 scale-in">
                <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-muted mb-4">
                    <svg
                        className="w-8 h-8 text-muted-foreground"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                    >
                        <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z"
                        />
                    </svg>
                </div>
                <h3 className="text-lg font-semibold mb-2">No apps yet</h3>
                <p className="text-muted-foreground mb-4 max-w-sm mx-auto">
                    Get started by creating your first self-hosted application
                </p>
                <Button
                    onClick={() => navigate('/apps/new')}
                    className="button-press"
                >
                    Create your first app
                </Button>
            </div>
        )
    }

    return (
        <>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3 sm:gap-4 lg:gap-5">
                {apps.map((app, index) => {
                    const isDeleting = deletingAppId === app.id
                    const statusColorMap = {
                        running: 'border-l-green-500',
                        stopped: 'border-l-gray-400',
                        updating: 'border-l-blue-500',
                        error: 'border-l-red-500'
                    }
                    const statusColor = statusColorMap[app.status as keyof typeof statusColorMap] || 'border-l-gray-400'

                    return (
                        <Card
                            key={app.id}
                            className={`group relative overflow-hidden border-l-4 ${statusColor} ${isDeleting ? 'opacity-60 pointer-events-none' : 'card-hover cursor-pointer'} fade-in stagger-${(index % 6) + 1} flex flex-col`}
                        >
                            {/* Deletion Overlay */}
                            {isDeleting && (
                                <div className="absolute inset-0 bg-background/80 backdrop-blur-sm rounded-lg z-10 flex items-center justify-center">
                                    <div className="flex flex-col items-center gap-3 text-center p-4">
                                        <Loader2 className="h-8 w-8 animate-spin text-destructive" />
                                        <div>
                                            <p className="font-semibold text-sm">Deleting...</p>
                                            <p className="text-xs text-muted-foreground">Removing app and resources</p>
                                        </div>
                                    </div>
                                </div>
                            )}

                            {/* Clickable Card Area */}
                            <div
                                className="flex-1 flex flex-col"
                                onClick={() => !isDeleting && navigate(`/apps/${app.id}${app.node_id ? `?node_id=${app.node_id}` : ''}`)}
                            >
                                {/* Card Header */}
                                <CardHeader className="pb-3 p-4 sm:p-5 space-y-2">
                                    {/* Title Row */}
                                    <div className="flex items-start justify-between gap-2">
                                        <CardTitle className="text-base sm:text-lg lg:text-xl font-bold truncate flex-1 pr-2">{app.name}</CardTitle>

                                        {/* Desktop Actions Menu */}
                                        <div onClick={(e) => e.stopPropagation()} className="hidden sm:block">
                                            <SimpleDropdown
                                                trigger={
                                                    <Button variant="ghost" size="icon" className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity">
                                                        <MoreVertical className="h-4 w-4" />
                                                    </Button>
                                                }
                                            >
                                                <div className="py-1 min-w-[160px]">
                                                    {app.status === 'running' && (
                                                        <SimpleDropdownItem
                                                            onClick={() => stopApp.mutate({ id: app.id, nodeId: app.node_id }, {
                                                                onSuccess: () => toast.success('App stopped', `${app.name} has been stopped successfully`),
                                                                onError: (error) => toast.error('Failed to stop app', error.message)
                                                            })}
                                                        >
                                                            <div className="flex items-center">
                                                                <Pause className="h-4 w-4 mr-2" />
                                                                <span>Stop App</span>
                                                            </div>
                                                        </SimpleDropdownItem>
                                                    )}
                                                    {app.status === 'stopped' && (
                                                        <SimpleDropdownItem
                                                            onClick={() => startApp.mutate({ id: app.id, nodeId: app.node_id }, {
                                                                onSuccess: () => toast.success('App started', `${app.name} has been started successfully`),
                                                                onError: (error) => toast.error('Failed to start app', error.message)
                                                            })}
                                                        >
                                                            <div className="flex items-center">
                                                                <Play className="h-4 w-4 mr-2" />
                                                                <span>Start App</span>
                                                            </div>
                                                        </SimpleDropdownItem>
                                                    )}
                                                    <SimpleDropdownItem
                                                        onClick={() => updateApp.mutate({ id: app.id, nodeId: app.node_id }, {
                                                            onSuccess: () => toast.success('Update started', `${app.name} update process has begun`),
                                                            onError: (error) => toast.error('Failed to start update', error.message)
                                                        })}
                                                    >
                                                        <div className="flex items-center">
                                                            <RefreshCw className="h-4 w-4 mr-2" />
                                                            <span>Update</span>
                                                        </div>
                                                    </SimpleDropdownItem>
                                                    <div className="border-t my-1"></div>
                                                    <SimpleDropdownItem
                                                        onClick={() => handleDelete(app.id, app.name)}
                                                    >
                                                        <div className="flex items-center text-destructive">
                                                            <Trash2 className="h-4 w-4 mr-2" />
                                                            <span>Delete</span>
                                                        </div>
                                                    </SimpleDropdownItem>
                                                </div>
                                            </SimpleDropdown>
                                        </div>
                                    </div>

                                    {/* Status and Quick Actions Row */}
                                    <div className="flex items-center gap-2 flex-wrap">
                                        <div
                                            className={`px-2.5 py-1 rounded-md text-xs font-semibold flex items-center gap-1.5 ${getStatusColor(app.status)}`}
                                        >
                                            {getStatusIcon(app.status)}
                                            {app.status}
                                        </div>
                                        {app.public_url && (
                                            <a
                                                href={app.public_url}
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
                                                onClick={(e) => e.stopPropagation()}
                                                title="Open app"
                                            >
                                                <ExternalLink className="h-3.5 w-3.5" />
                                                <span className="hidden sm:inline">Open</span>
                                            </a>
                                        )}
                                    </div>
                                </CardHeader>

                                {/* Card Content */}
                                <CardContent className="pt-0 pb-3 p-4 sm:p-5 sm:pt-0 flex-1">
                                    {app.description && (
                                        <p className="text-xs sm:text-sm text-muted-foreground line-clamp-2 mb-3">
                                            {app.description}
                                        </p>
                                    )}

                                    {app.status === 'error' && app.error_message && (
                                        <div className="text-xs text-red-600 dark:text-red-400 p-2.5 bg-red-50 dark:bg-red-900/20 rounded border-l-2 border-red-500">
                                            <span className="font-semibold">Error:</span> {app.error_message}
                                        </div>
                                    )}
                                </CardContent>

                                {/* Card Footer */}
                                <div className="px-4 sm:px-5 py-2.5 bg-muted/30 border-t">
                                    <div className="flex items-center justify-between text-xs text-muted-foreground">
                                        <span className="flex items-center gap-1.5">
                                            <Clock className="h-3.5 w-3.5 flex-shrink-0" />
                                            <span className="truncate">{new Date(app.updated_at).toLocaleDateString()}</span>
                                        </span>
                                        <span className="flex items-center gap-1 text-primary font-medium">
                                            <span className="hidden sm:inline">Details</span>
                                            <TrendingUp className="h-3.5 w-3.5" />
                                        </span>
                                    </div>
                                </div>
                            </div>

                            {/* Mobile Action Bar - Outside clickable area */}
                            <div className="sm:hidden border-t bg-muted/50 px-3 py-2 flex items-center justify-between gap-2" onClick={(e) => e.stopPropagation()}>
                                <div className="flex items-center gap-1">
                                    {app.status === 'running' && (
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => stopApp.mutate({ id: app.id, nodeId: app.node_id }, {
                                                onSuccess: () => toast.success('App stopped', `${app.name} has been stopped successfully`),
                                                onError: (error) => toast.error('Failed to stop app', error.message)
                                            })}
                                            className="h-8 px-3 text-xs"
                                        >
                                            <Pause className="h-3.5 w-3.5 mr-1.5" />
                                            Stop
                                        </Button>
                                    )}
                                    {app.status === 'stopped' && (
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => startApp.mutate({ id: app.id, nodeId: app.node_id }, {
                                                onSuccess: () => toast.success('App started', `${app.name} has been started successfully`),
                                                onError: (error) => toast.error('Failed to start app', error.message)
                                            })}
                                            className="h-8 px-3 text-xs"
                                        >
                                            <Play className="h-3.5 w-3.5 mr-1.5" />
                                            Start
                                        </Button>
                                    )}
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => updateApp.mutate({ id: app.id, nodeId: app.node_id }, {
                                            onSuccess: () => toast.success('Update started', `${app.name} update process has begun`),
                                            onError: (error) => toast.error('Failed to start update', error.message)
                                        })}
                                        className="h-8 px-3 text-xs"
                                    >
                                        <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
                                        Update
                                    </Button>
                                </div>
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => handleDelete(app.id, app.name)}
                                    className="h-8 w-8 p-0 text-destructive hover:text-destructive hover:bg-destructive/10"
                                >
                                    <Trash2 className="h-4 w-4" />
                                </Button>
                            </div>
                        </Card>
                    )
                })}
            </div>

            {/* Confirmation Dialog */}
            <ConfirmationDialog
                open={!!appToDelete}
                onOpenChange={(open: boolean) => !open && setAppToDelete(null)}
                title="Delete App"
                description={`Are you sure you want to delete "${appToDelete?.name}"? This action cannot be undone.`}
                confirmText="Delete"
                cancelText="Cancel"
                onConfirm={confirmDelete}
                isLoading={deleteApp.isPending}
                variant="destructive"
            />
        </>
    )
}

export default AppList
