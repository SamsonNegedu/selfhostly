import React, { useState } from 'react' // React and useState both used
import { useAppStore } from '@/shared/stores/app-store'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { Play, Pause, RefreshCw, Trash2 } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useDeleteApp, useStartApp, useStopApp, useUpdateAppContainers } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'

interface AppToDelete {
    id: number
    name: string
}

function AppList() {
    /* @ts-ignore - React is used for JSX */
    const ReactUsedForJSX = React
    const apps = useAppStore((state) => state.apps)
    const navigate = useNavigate()
    const deleteApp = useDeleteApp()
    const startApp = useStartApp()
    const stopApp = useStopApp()
    const updateApp = useUpdateAppContainers()
    const { toast } = useToast()

    // State for confirmation dialog
    const [appToDelete, setAppToDelete] = useState<AppToDelete | null>(null)

    // Handle delete with confirmation dialog
    const handleDelete = (appId: number, appName: string) => {
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

    if (apps.length === 0) {
        return (
            <div className="text-center py-12">
                <p className="text-muted-foreground mb-4">No apps found</p>
                <Button onClick={() => navigate('/apps/new')}>
                    Create your first app
                </Button>
            </div>
        )
    }

    return (
        <>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {apps.map((app) => (
                    <Card
                        key={app.id}
                        className="cursor-pointer hover:shadow-md transition-shadow"
                        onClick={() => navigate(`/apps/${app.id}`)}
                    >
                        <CardHeader>
                            <div className="flex items-center justify-between">
                                <CardTitle className="text-xl">{app.name}</CardTitle>
                                <div
                                    className={`px-2 py-1 rounded-full text-xs font-medium flex items-center gap-1 ${app.status === 'running'
                                        ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                                        : app.status === 'stopped'
                                            ? 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
                                            : app.status === 'updating'
                                                ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
                                                : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                                        }`}
                                >
                                    {app.status === 'running' && (
                                        <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
                                    )}
                                    {app.status === 'updating' && (
                                        <div className="w-2 h-2 bg-blue-500 rounded-full animate-spin"></div>
                                    )}
                                    {app.status === 'error' && (
                                        <div className="w-2 h-2 bg-red-500 rounded-full"></div>
                                    )}
                                    {app.status}
                                </div>
                            </div>
                        </CardHeader>
                        <CardContent>
                            <p className="text-sm text-muted-foreground mb-4">
                                {app.description || 'No description'}
                            </p>
                            {app.public_url && (
                                <div className="text-sm mb-4">
                                    <a
                                        href={app.public_url}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="text-primary hover:underline"
                                    >
                                        {app.public_url}
                                    </a>
                                </div>
                            )}
                            {app.status === 'error' && app.error_message && (
                                <div className="text-sm text-red-600 dark:text-red-400 mb-4 p-2 bg-red-50 dark:bg-red-900/20 rounded">
                                    <span className="font-medium">Error:</span> {app.error_message}
                                </div>
                            )}
                            <div className="flex gap-2" onClick={(e) => e.stopPropagation()}>
                                {(stopApp.isPending && app.status === 'running') || (startApp.isPending && app.status === 'stopped') ? (
                                    <Button variant="outline" size="icon" title="Processing" disabled>
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
                                    className="text-destructive hover:text-destructive"
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
