import React, { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useApp, useStartApp, useStopApp, useUpdateAppContainers, useDeleteApp } from '@/shared/services/api'
import { useAppStore } from '@/shared/stores/app-store'
import { useNavigate } from 'react-router-dom'
import { useToast } from '@/shared/components/ui/Toast'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { Play, Pause, RefreshCw, Trash2, Terminal, Settings, Cloud } from 'lucide-react'
import UpdateProgress from './components/UpdateProgress'
import LogViewer from './components/LogViewer'
import ComposeEditor from './components/ComposeEditor'
import CloudflareTab from './components/CloudflareTab'

type TabType = 'overview' | 'compose' | 'logs' | 'update' | 'cloudflare'

function AppDetails() {
    const { id } = useParams<{ id: string }>()
    const appId = id ? parseInt(id) : undefined
    const navigate = useNavigate()
    const { data: app, isLoading } = useApp(appId!)
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
            // Remove from local store if exists
            useAppStore.getState().removeApp(app.id)

            // Redirect to dashboard after deletion
            deleteApp.mutate(app.id, {
                onSuccess: () => {
                    navigate('/dashboard')
                }
            })

            // Close dialog
            setShowDeleteDialog(false)
        }
    }

    const [activeTab, setActiveTab] = React.useState<TabType>('overview')

    if (isLoading) {
        return (
            <div className="flex items-center justify-center h-64">
                Loading...
            </div>
        )
    }

    if (!app) {
        return (
            <div className="text-center text-destructive">
                App not found
            </div>
        )
    }

    return (
        <div className="space-y-6">
            <Card>
                <CardHeader>
                    <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                            <CardTitle className="text-2xl">{app.name}</CardTitle>
                            <div
                                className={`px-3 py-1 rounded-full text-sm font-medium flex items-center gap-1 ${app.status === 'running'
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
                        <div className="flex gap-2">
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
                                disabled={updateApp.isPending}
                            >
                                {updateApp.isPending ? (
                                    <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                                ) : (
                                    <RefreshCw className="h-4 w-4" />
                                )}
                            </Button>
                            <Button
                                variant="outline"
                                size="icon"
                                onClick={handleDelete}
                                className="text-destructive hover:text-destructive"
                                title="Delete app"
                                disabled={deleteApp.isPending}
                            >
                                {deleteApp.isPending ? (
                                    <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                                ) : (
                                    <Trash2 className="h-4 w-4" />
                                )}
                            </Button>
                        </div>
                    </div>

                    <div className="flex border-b">
                        <button
                            className={`px-4 py-2 text-sm font-medium ${activeTab === 'overview'
                                ? 'border-b-2 border-primary text-primary'
                                : 'text-muted-foreground hover:text-foreground'
                                }`}
                            onClick={() => setActiveTab('overview')}
                        >
                            Overview
                        </button>
                        <button
                            className={`px-4 py-2 text-sm font-medium ${activeTab === 'compose'
                                ? 'border-b-2 border-primary text-primary'
                                : 'text-muted-foreground hover:text-foreground'
                                }`}
                            onClick={() => setActiveTab('compose')}
                        >
                            <Settings className="h-4 w-4 inline mr-2" />
                            Compose Editor
                        </button>
                        <button
                            className={`px-4 py-2 text-sm font-medium ${activeTab === 'update'
                                ? 'border-b-2 border-primary text-primary'
                                : 'text-muted-foreground hover:text-foreground'
                                }`}
                            onClick={() => setActiveTab('update')}
                        >
                            <RefreshCw className="h-4 w-4 inline mr-2" />
                            Update
                        </button>
                        <button
                            className={`px-4 py-2 text-sm font-medium ${activeTab === 'logs'
                                ? 'border-b-2 border-primary text-primary'
                                : 'text-muted-foreground hover:text-foreground'
                                }`}
                            onClick={() => setActiveTab('logs')}
                        >
                            <Terminal className="h-4 w-4 inline mr-2" />
                            Logs
                        </button>
                        <button
                            className={`px-4 py-2 text-sm font-medium ${activeTab === 'cloudflare'
                                ? 'border-b-2 border-primary text-primary'
                                : 'text-muted-foreground hover:text-foreground'
                                }`}
                            onClick={() => setActiveTab('cloudflare')}
                        >
                            <Cloud className="h-4 w-4 inline mr-2" />
                            Cloudflare
                        </button>
                    </div>
                </CardHeader>
                <CardContent>
                    <p className="text-sm text-muted-foreground mb-4">
                        {app.description || 'No description'}
                    </p>
                    {app.public_url && (
                        <div className="mb-4">
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
                        <div className="text-sm text-red-600 dark:text-red-400 mb-4 p-3 bg-red-50 dark:bg-red-900/20 rounded border border-red-200 dark:border-red-800">
                            <span className="font-medium">Error:</span> {app.error_message}
                        </div>
                    )}
                    {activeTab === 'overview' && (
                        <div className="space-y-4">
                            <div className="flex items-center gap-3 p-3 bg-muted/50 rounded-lg">
                                <span className="text-sm font-medium">Status:</span>
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
                            <div>
                                <span className="text-sm font-medium">Created:</span>
                                <span className="ml-2 text-sm text-muted-foreground">
                                    {new Date(app.created_at).toLocaleString()}
                                </span>
                            </div>
                            <div>
                                <span className="text-sm font-medium">Updated:</span>
                                <span className="ml-2 text-sm text-muted-foreground">
                                    {new Date(app.updated_at).toLocaleString()}
                                </span>
                            </div>
                        </div>
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
