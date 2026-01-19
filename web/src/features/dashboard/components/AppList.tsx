import { useState } from 'react'
import { useAppStore } from '@/shared/stores/app-store'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { Play, Pause, RefreshCw, Trash2, ExternalLink, Clock, Search } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useDeleteApp, useStartApp, useStopApp, useUpdateAppContainers } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
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

    // State for confirmation dialog
    const [appToDelete, setAppToDelete] = useState<AppToDelete | null>(null)

    // Handle delete with confirmation dialog
    const handleDelete = (appId: string, appName: string) => {
        setAppToDelete({ id: appId, name: appName })
    }

    // Confirm deletion
    const confirmDelete = () => {
        if (appToDelete) {
            // Optimistically remove from local store
            useAppStore.getState().removeApp(appToDelete.id)

            // Then trigger the actual deletion
            deleteApp.mutate(appToDelete.id)

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
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {apps.map((app, index) => (
                    <Card
                        key={app.id}
                        className={`cursor-pointer card-hover border-2 hover:border-primary/20 ${app.status === 'running' ? 'border-green-200 dark:border-green-900/30' : ''} fade-in stagger-${(index % 6) + 1}`}
                        onClick={() => navigate(`/apps/${app.id}`)}
                    >
                        <CardHeader>
                            <div className="flex items-center justify-between">
                                <CardTitle className="text-xl truncate">{app.name}</CardTitle>
                                <div
                                    className={`px-2 py-1 rounded-full text-xs font-medium flex items-center gap-1 ${getStatusColor(app.status)}`}
                                >
                                    {getStatusIcon(app.status)}
                                    {app.status}
                                </div>
                            </div>
                        </CardHeader>
                        <CardContent>
                            <p className="text-sm text-muted-foreground mb-4 line-clamp-2 min-h-[2.5rem]">
                                {app.description || 'No description'}
                            </p>

                            {app.public_url && (
                                <div className="text-sm mb-4">
                                    <a
                                        href={app.public_url}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="text-primary hover:underline flex items-center gap-1 interactive-element"
                                        onClick={(e) => e.stopPropagation()}
                                    >
                                        <ExternalLink className="h-3 w-3" />
                                        <span className="truncate">{app.public_url}</span>
                                    </a>
                                </div>
                            )}

                            {app.status === 'error' && app.error_message && (
                                <div className="text-sm text-red-600 dark:text-red-400 mb-4 p-2 bg-red-50 dark:bg-red-900/20 rounded">
                                    <span className="font-medium">Error:</span> {app.error_message}
                                </div>
                            )}

                            <div className="flex items-center justify-between text-xs text-muted-foreground mb-4">
                                <span className="flex items-center gap-1">
                                    <Clock className="h-3 w-3" />
                                    {new Date(app.updated_at).toLocaleDateString()}
                                </span>
                            </div>

                            <div className="flex gap-2" onClick={(e) => e.stopPropagation()}>
                                {(stopApp.isPending && app.status === 'running') || (startApp.isPending && app.status === 'stopped') ? (
                                    <Button variant="outline" size="icon" title="Processing" disabled className="button-press">
                                        <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                                    </Button>
                                ) : (
                                    <>
                                        {app.status === 'running' && (
                                            <Button
                                                variant="outline"
                                                size="icon"
                                                onClick={() => stopApp.mutate(app.id, {
                                                    onSuccess: () => {
                                                        toast.success('App stopped', `${app.name} has been stopped successfully`)
                                                    },
                                                    onError: (error) => {
                                                        toast.error('Failed to stop app', error.message)
                                                    }
                                                })}
                                                title="Stop app"
                                                disabled={stopApp.isPending}
                                                className="button-press"
                                            >
                                                {stopApp.isPending ? (
                                                    <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                                                ) : (
                                                    <Pause className="h-4 w-4" />
                                                )}
                                            </Button>
                                        )}
                                        {app.status === 'stopped' && (
                                            <Button
                                                variant="outline"
                                                size="icon"
                                                onClick={() => startApp.mutate(app.id, {
                                                    onSuccess: () => {
                                                        toast.success('App started', `${app.name} has been started successfully`)
                                                    },
                                                    onError: (error) => {
                                                        toast.error('Failed to start app', error.message)
                                                    }
                                                })}
                                                title="Start app"
                                                disabled={startApp.isPending}
                                                className="button-press"
                                            >
                                                {startApp.isPending ? (
                                                    <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                                                ) : (
                                                    <Play className="h-4 w-4" />
                                                )}
                                            </Button>
                                        )}
                                        <Button
                                            variant="outline"
                                            size="icon"
                                            onClick={() => updateApp.mutate(app.id, {
                                                onSuccess: () => {
                                                    toast.success('Update started', `${app.name} update process has begun`)
                                                },
                                                onError: (error) => {
                                                    toast.error('Failed to start update', error.message)
                                                }
                                            })}
                                            title="Update containers"
                                            disabled={updateApp.isPending}
                                            className="button-press"
                                        >
                                            {updateApp.isPending ? (
                                                <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                                            ) : (
                                                <RefreshCw className="h-4 w-4" />
                                            )}
                                        </Button>
                                    </>
                                )}
                                <Button
                                    variant="outline"
                                    size="icon"
                                    onClick={() => handleDelete(app.id, app.name)}
                                    title="Delete app"
                                    className="text-destructive hover:text-destructive button-press"
                                    disabled={deleteApp.isPending}
                                >
                                    {deleteApp.isPending ? (
                                        <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                                    ) : (
                                        <Trash2 className="h-4 w-4" />
                                    )}
                                </Button>
                            </div>
                        </CardContent>
                    </Card>
                ))}
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
