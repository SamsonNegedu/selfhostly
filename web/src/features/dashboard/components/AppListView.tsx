import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/shared/components/ui/Button'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { Play, Pause, RefreshCw, Trash2, ExternalLink, XCircle, Clock, Loader2 } from 'lucide-react'
import { useDeleteApp, useStartApp, useStopApp, useUpdateAppContainers, useNodes } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import { DataTable, ColumnDef, RowAction } from '@/shared/components/ui/DataTable'
import type { App } from '@/shared/types/api'

interface AppListViewProps {
    filteredApps: App[]
}

// Compute health score based on app status
const computeHealthScore = (app: App): { score: number; label: string; color: string } => {
    if (app.status === 'error') {
        return { score: 0, label: 'Critical', color: 'text-red-600 dark:text-red-400' }
    }
    if (app.status === 'stopped') {
        return { score: 50, label: 'Stopped', color: 'text-gray-600 dark:text-gray-400' }
    }
    if (app.status === 'updating') {
        return { score: 75, label: 'Updating', color: 'text-blue-600 dark:text-blue-400' }
    }
    if (app.status === 'running') {
        if (app.error_message) {
            return { score: 70, label: 'Degraded', color: 'text-amber-600 dark:text-amber-400' }
        }
        return { score: 95, label: 'Healthy', color: 'text-green-600 dark:text-green-400' }
    }
    return { score: 50, label: 'Unknown', color: 'text-gray-600 dark:text-gray-400' }
}

// Status badge component
const getStatusBadge = (status: string) => {
    const colors = {
        running: 'text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20',
        stopped: 'text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-800',
        updating: 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20',
        error: 'text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20',
    }

    const icons = {
        running: <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />,
        updating: <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-500" />,
        error: <XCircle className="h-3.5 w-3.5 text-red-500" />,
        stopped: <div className="w-2 h-2 bg-gray-400 rounded-full" />,
    }

    const colorClass = colors[status as keyof typeof colors] || colors.stopped
    const icon = icons[status as keyof typeof icons] || null

    return (
        <div className={`inline-flex items-center gap-2 px-2.5 py-1 rounded-md text-xs font-semibold ${colorClass}`}>
            {icon}
            {status}
        </div>
    )
}

function AppListView({ filteredApps }: AppListViewProps) {
    const navigate = useNavigate()
    const deleteApp = useDeleteApp()
    const startApp = useStartApp()
    const stopApp = useStopApp()
    const updateApp = useUpdateAppContainers()
    const { toast } = useToast()
    const { data: nodes } = useNodes()

    const [appToDelete, setAppToDelete] = useState<{ id: string; name: string } | null>(null)
    const [selectedApps, setSelectedApps] = useState<Set<string>>(new Set())
    const [loadingAppIds, setLoadingAppIds] = useState<Set<string>>(new Set())

    // Get node name by ID
    const getNodeName = (nodeId: string): string => {
        const node = nodes?.find(n => n.id === nodeId)
        return node?.name || 'Unknown'
    }

    // Define columns
    const columns: ColumnDef<App>[] = [
        {
            key: 'status',
            label: 'Status',
            width: 'w-28',
            render: (app) => getStatusBadge(app.status)
        },
        {
            key: 'app',
            label: 'Application',
            width: 'flex-1 min-w-0',
            render: (app) => (
                <div className="flex flex-col min-w-0">
                    <div className="flex items-center gap-2">
                        <span className="font-semibold text-sm truncate">{app.name}</span>
                        {app.public_url && (
                            <a
                                href={app.public_url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-muted-foreground hover:text-primary transition-colors"
                                onClick={(e) => e.stopPropagation()}
                            >
                                <ExternalLink className="h-3.5 w-3.5" />
                            </a>
                        )}
                    </div>
                    {app.description && (
                        <span className="text-xs text-muted-foreground truncate mt-0.5">
                            {app.description}
                        </span>
                    )}
                </div>
            )
        },
        {
            key: 'node',
            label: 'Node',
            width: 'w-28',
            render: (app) => (
                <div className="flex items-center gap-2">
                    <div className="w-2 h-2 bg-blue-500 rounded-full" />
                    <span className="text-sm text-muted-foreground">
                        {getNodeName(app.node_id)}
                    </span>
                </div>
            )
        },
        {
            key: 'health',
            label: 'Health',
            width: 'w-28',
            render: (app) => {
                const health = computeHealthScore(app)
                return (
                    <div className="flex items-center gap-2">
                        <span className={`text-xs font-semibold ${health.color}`}>
                            {health.label}
                        </span>
                        <div className="w-16 h-1.5 bg-muted rounded-full overflow-hidden">
                            <div
                                className={`h-full transition-all duration-300 ${health.score >= 80 ? 'bg-green-500' :
                                    health.score >= 60 ? 'bg-amber-500' :
                                        health.score >= 40 ? 'bg-blue-500' : 'bg-red-500'
                                    }`}
                                style={{ width: `${health.score}%` }}
                            />
                        </div>
                    </div>
                )
            }
        },
        {
            key: 'tunnel',
            label: 'Tunnel',
            width: 'w-24',
            render: (app) => {
                if (app.tunnel_mode === 'quick') {
                    return <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-200">Quick</span>
                }
                if (app.tunnel_mode === 'custom') {
                    return <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-200">Custom</span>
                }
                return <span className="text-xs text-muted-foreground">—</span>
            }
        },
        {
            key: 'updated',
            label: 'Last Updated',
            width: 'w-36',
            render: (app) => (
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Clock className="h-3 w-3 flex-shrink-0" />
                    <span>{new Date(app.updated_at).toLocaleDateString()}</span>
                </div>
            )
        }
    ]

    // Define actions
    const actions: RowAction<App>[] = [
        {
            label: 'Stop App',
            icon: <Pause className="h-4 w-4" />,
            onClick: (app) => {
                setLoadingAppIds(prev => new Set(prev).add(app.id))
                stopApp.mutate({ id: app.id, nodeId: app.node_id }, {
                    onSuccess: () => {
                        toast.success('App stopped', `${app.name} has been stopped`)
                        setLoadingAppIds(prev => {
                            const next = new Set(prev)
                            next.delete(app.id)
                            return next
                        })
                    },
                    onError: (error) => {
                        toast.error('Failed to stop app', error.message)
                        setLoadingAppIds(prev => {
                            const next = new Set(prev)
                            next.delete(app.id)
                            return next
                        })
                    }
                })
            },
            show: (app) => app.status === 'running',
            loading: (app) => loadingAppIds.has(app.id) && stopApp.isPending
        },
        {
            label: 'Start App',
            icon: <Play className="h-4 w-4" />,
            onClick: (app) => {
                setLoadingAppIds(prev => new Set(prev).add(app.id))
                startApp.mutate({ id: app.id, nodeId: app.node_id }, {
                    onSuccess: () => {
                        toast.success('App started', `${app.name} has been started`)
                        setLoadingAppIds(prev => {
                            const next = new Set(prev)
                            next.delete(app.id)
                            return next
                        })
                    },
                    onError: (error) => {
                        toast.error('Failed to start app', error.message)
                        setLoadingAppIds(prev => {
                            const next = new Set(prev)
                            next.delete(app.id)
                            return next
                        })
                    }
                })
            },
            show: (app) => app.status === 'stopped',
            loading: (app) => loadingAppIds.has(app.id) && startApp.isPending
        },
        {
            label: 'Update',
            icon: <RefreshCw className="h-4 w-4" />,
            onClick: (app) => {
                setLoadingAppIds(prev => new Set(prev).add(app.id))
                updateApp.mutate({ id: app.id, nodeId: app.node_id }, {
                    onSuccess: () => {
                        toast.success('Update started', `${app.name} update process has begun`)
                        setLoadingAppIds(prev => {
                            const next = new Set(prev)
                            next.delete(app.id)
                            return next
                        })
                    },
                    onError: (error) => {
                        toast.error('Failed to start update', error.message)
                        setLoadingAppIds(prev => {
                            const next = new Set(prev)
                            next.delete(app.id)
                            return next
                        })
                    }
                })
            },
            loading: (app) => loadingAppIds.has(app.id) && updateApp.isPending
        },
        {
            label: 'Delete',
            icon: <Trash2 className="h-4 w-4" />,
            onClick: (app) => setAppToDelete({ id: app.id, name: app.name }),
            variant: 'destructive'
        }
    ]

    // Handle app deletion confirmation
    const confirmDelete = () => {
        if (appToDelete) {
            const appName = appToDelete.name
            const appId = appToDelete.id
            const app = filteredApps.find(a => a.id === appId)

            if (!app?.node_id) {
                toast.error('Delete Failed', 'Unable to determine app node')
                setAppToDelete(null)
                return
            }

            toast.info('Deleting app', `Deleting "${appName}"...`)

            deleteApp.mutate({ id: appId, nodeId: app.node_id }, {
                onSuccess: () => {
                    toast.success('App deleted', `"${appName}" has been deleted successfully`)
                },
                onError: (error) => {
                    toast.error('Failed to delete app', error.message)
                }
            })

            setAppToDelete(null)
        }
    }

    // Bulk actions
    const handleBulkStart = () => {
        selectedApps.forEach(appId => {
            const app = filteredApps.find(a => a.id === appId)
            if (app && app.status === 'stopped') {
                startApp.mutate({ id: appId, nodeId: app.node_id })
            }
        })
        setSelectedApps(new Set())
        toast.success('Bulk Start', 'Starting selected apps')
    }

    const handleBulkStop = () => {
        selectedApps.forEach(appId => {
            const app = filteredApps.find(a => a.id === appId)
            if (app && app.status === 'running') {
                stopApp.mutate({ id: appId, nodeId: app.node_id })
            }
        })
        setSelectedApps(new Set())
        toast.success('Bulk Stop', 'Stopping selected apps')
    }

    const handleBulkDelete = () => {
        toast.info('Bulk Delete', 'Deleting selected apps...')
        selectedApps.forEach(appId => {
            const app = filteredApps.find(a => a.id === appId)
            if (app) {
                deleteApp.mutate({ id: appId, nodeId: app.node_id })
            }
        })
        setSelectedApps(new Set())
    }

    // Expandable content
    const expandableContent = (app: App) => (
        <div className="space-y-3">
            {app.status === 'error' && app.error_message && (
                <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                    <div className="flex items-start gap-2">
                        <XCircle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
                        <div>
                            <p className="text-sm font-semibold text-red-800 dark:text-red-300">Error Message</p>
                            <p className="text-xs text-red-700 dark:text-red-400 mt-1">{app.error_message}</p>
                        </div>
                    </div>
                </div>
            )}
            {app.public_url && (
                <div className="flex items-center gap-2 text-sm">
                    <ExternalLink className="h-4 w-4 text-muted-foreground" />
                    <span className="text-muted-foreground">Public URL:</span>
                    <a
                        href={app.public_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-primary hover:underline break-all"
                    >
                        {app.public_url}
                    </a>
                </div>
            )}
            <div className="flex items-center gap-4 text-xs text-muted-foreground">
                <div className="flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    <span>Created: {new Date(app.created_at).toLocaleString()}</span>
                </div>
                <div className="flex items-center gap-1">
                    <RefreshCw className="h-3 w-3" />
                    <span>Updated: {new Date(app.updated_at).toLocaleString()}</span>
                </div>
            </div>
        </div>
    )

    // Empty state
    const emptyState = (
        <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-muted mb-4">
                <XCircle className="w-8 h-8 text-muted-foreground" />
            </div>
            <h3 className="text-lg font-semibold mb-2">No apps found</h3>
            <p className="text-muted-foreground max-w-sm mx-auto">
                Try adjusting your search or filter criteria
            </p>
        </div>
    )

    return (
        <>
            {/* Bulk Actions Bar */}
            {selectedApps.size > 0 && (
                <div className="mb-4 p-3 bg-primary/10 border border-primary/20 rounded-lg flex items-center justify-between">
                    <span className="text-sm font-medium">
                        {selectedApps.size} app{selectedApps.size > 1 ? 's' : ''} selected
                    </span>
                    <div className="flex items-center gap-2">
                        <Button variant="outline" size="sm" onClick={handleBulkStart} className="h-8">
                            <Play className="h-3.5 w-3.5 mr-1.5" />
                            Start
                        </Button>
                        <Button variant="outline" size="sm" onClick={handleBulkStop} className="h-8">
                            <Pause className="h-3.5 w-3.5 mr-1.5" />
                            Stop
                        </Button>
                        <Button variant="destructive" size="sm" onClick={handleBulkDelete} className="h-8">
                            <Trash2 className="h-3.5 w-3.5 mr-1.5" />
                            Delete
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => setSelectedApps(new Set())} className="h-8">
                            Clear
                        </Button>
                    </div>
                </div>
            )}

            {/* Data Table */}
            <DataTable
                data={filteredApps}
                columns={columns}
                getRowKey={(app) => app.id}
                actions={actions}
                expandableContent={expandableContent}
                onRowClick={(app) => navigate(`/apps/${app.id}${app.node_id ? `?node_id=${app.node_id}` : ''}`)}
                emptyState={emptyState}
                selectable={true}
                onSelectionChange={setSelectedApps}
            />

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

export default AppListView
