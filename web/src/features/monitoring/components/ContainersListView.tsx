import { useState } from 'react'
import { Badge } from '@/shared/components/ui/Badge'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { Network, HardDrive, RotateCw, Square, Trash2, Server } from 'lucide-react'
import { useRestartContainer, useStopContainer, useDeleteContainer } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import { DataTable, ColumnDef, RowAction } from '@/shared/components/ui/DataTable'
import type { ContainerInfo } from '@/shared/types/api'

interface ContainersListViewProps {
    containers: ContainerInfo[]
}

// Format bytes helper
function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
}

// Status badge component
const getStatusBadge = (state: string) => {
    switch (state) {
        case 'running':
            return (
                <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md text-xs font-semibold text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20">
                    <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                    running
                </div>
            )
        case 'paused':
            return (
                <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md text-xs font-semibold text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20">
                    <div className="w-2 h-2 bg-amber-500 rounded-full" />
                    paused
                </div>
            )
        case 'stopped':
            return (
                <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md text-xs font-semibold text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-800">
                    <div className="w-2 h-2 bg-gray-400 rounded-full" />
                    stopped
                </div>
            )
        default:
            return (
                <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md text-xs font-semibold text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-800">
                    {state}
                </div>
            )
    }
}

function ContainersListView({ containers }: ContainersListViewProps) {
    const restartMutation = useRestartContainer()
    const stopMutation = useStopContainer()
    const deleteMutation = useDeleteContainer()
    const { toast } = useToast()

    const [containerToRestart, setContainerToRestart] = useState<ContainerInfo | null>(null)
    const [containerToStop, setContainerToStop] = useState<ContainerInfo | null>(null)
    const [containerToDelete, setContainerToDelete] = useState<ContainerInfo | null>(null)
    const [loadingContainerIds, setLoadingContainerIds] = useState<Set<string>>(new Set())

    // Handle restart
    const handleRestart = async () => {
        if (!containerToRestart) return
        setLoadingContainerIds(prev => new Set(prev).add(containerToRestart.id))
        try {
            await restartMutation.mutateAsync({ containerId: containerToRestart.id, nodeId: containerToRestart.node_id })
            const action = containerToRestart.state === 'stopped' ? 'started' : 'restarted'
            toast.success('Container Updated', `Container "${containerToRestart.name}" ${action} successfully`)
            setContainerToRestart(null)
        } catch (error) {
            toast.error('Failed to restart container', error instanceof Error ? error.message : 'Unknown error')
            setContainerToRestart(null)
        } finally {
            setLoadingContainerIds(prev => {
                const next = new Set(prev)
                next.delete(containerToRestart.id)
                return next
            })
        }
    }

    // Handle stop
    const handleStop = async () => {
        if (!containerToStop) return
        setLoadingContainerIds(prev => new Set(prev).add(containerToStop.id))
        try {
            await stopMutation.mutateAsync({ containerId: containerToStop.id, nodeId: containerToStop.node_id })
            toast.success('Container Stopped', `Container "${containerToStop.name}" stopped successfully`)
            setContainerToStop(null)
        } catch (error) {
            toast.error('Failed to stop container', error instanceof Error ? error.message : 'Unknown error')
            setContainerToStop(null)
        } finally {
            setLoadingContainerIds(prev => {
                const next = new Set(prev)
                next.delete(containerToStop.id)
                return next
            })
        }
    }

    // Handle delete
    const handleDelete = async () => {
        if (!containerToDelete) return
        setLoadingContainerIds(prev => new Set(prev).add(containerToDelete.id))
        try {
            await deleteMutation.mutateAsync({ containerId: containerToDelete.id, nodeId: containerToDelete.node_id })
            toast.success('Container Deleted', `Container "${containerToDelete.name}" deleted successfully`)
            setContainerToDelete(null)
        } catch (error) {
            toast.error('Failed to delete container', error instanceof Error ? error.message : 'Unknown error')
            setContainerToDelete(null)
        } finally {
            setLoadingContainerIds(prev => {
                const next = new Set(prev)
                next.delete(containerToDelete.id)
                return next
            })
        }
    }

    // Define columns
    const columns: ColumnDef<ContainerInfo>[] = [
        {
            key: 'status',
            label: 'Status',
            width: 'w-28',
            render: (container) => getStatusBadge(container.state)
        },
        {
            key: 'name',
            label: 'Container',
            width: 'flex-1 min-w-0',
            render: (container) => {
                const memPercent = container.memory_limit_bytes > 0
                    ? (container.memory_usage_bytes / container.memory_limit_bytes) * 100
                    : 0
                const isHighUsage = container.state === 'running' && (container.cpu_percent > 80 || memPercent > 80)
                const isMediumUsage = container.state === 'running' && (container.cpu_percent > 50 || memPercent > 50)

                return (
                    <div className="flex flex-col min-w-0">
                        <div className="flex items-center gap-2">
                            <span className="font-mono text-sm font-semibold truncate">{container.name}</span>
                            {isHighUsage && (
                                <Badge variant="outline" className="text-xs px-1.5 py-0 h-5 bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-400 border-red-200 dark:border-red-800">
                                    High Usage
                                </Badge>
                            )}
                            {isMediumUsage && !isHighUsage && (
                                <Badge variant="outline" className="text-xs px-1.5 py-0 h-5 bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-400 border-yellow-200 dark:border-yellow-800">
                                    Medium Usage
                                </Badge>
                            )}
                            {container.restart_count > 0 && (
                                <Badge variant="outline" className="text-xs px-1.5 py-0 h-5 bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-400 border-yellow-200 dark:border-yellow-800">
                                    {container.restart_count} restart{container.restart_count !== 1 ? 's' : ''}
                                </Badge>
                            )}
                        </div>
                        <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
                            <span className="font-mono">{container.id.substring(0, 12)}</span>
                        </div>
                    </div>
                )
            }
        },
        {
            key: 'app',
            label: 'App',
            width: 'w-48',
            render: (container) => (
                <div className="flex items-center gap-2">
                    <span className="font-medium text-sm truncate">{container.app_name}</span>
                    {container.is_managed ? (
                        <Badge variant="outline" className="text-xs px-1.5 py-0 h-5 bg-blue-50 dark:bg-blue-950 text-blue-700 dark:text-blue-400 border-blue-200 dark:border-blue-800">
                            Managed
                        </Badge>
                    ) : container.app_name !== 'unmanaged' ? (
                        <Badge variant="outline" className="text-xs px-1.5 py-0 h-5 bg-amber-50 dark:bg-amber-950 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800">
                            External
                        </Badge>
                    ) : null}
                </div>
            )
        },
        {
            key: 'cpu',
            label: 'CPU',
            width: 'w-32',
            render: (container) => {
                if (container.state !== 'running') {
                    return <span className="text-muted-foreground text-sm">—</span>
                }
                const cpuPercent = container.cpu_percent
                const colorClass = cpuPercent > 80 ? 'bg-red-500' : cpuPercent > 50 ? 'bg-yellow-500' : 'bg-blue-500'
                return (
                    <div className="space-y-1">
                        <div className="flex items-baseline gap-1">
                            <span className="text-sm font-medium">{cpuPercent.toFixed(1)}%</span>
                        </div>
                        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                            <div
                                className={`h-1.5 rounded-full transition-all ${colorClass}`}
                                style={{ width: `${Math.min(cpuPercent, 100)}%` }}
                            />
                        </div>
                    </div>
                )
            }
        },
        {
            key: 'memory',
            label: 'Memory',
            width: 'w-40',
            render: (container) => {
                if (container.state !== 'running') {
                    return <span className="text-muted-foreground text-sm">—</span>
                }
                const memPercent = container.memory_limit_bytes > 0
                    ? (container.memory_usage_bytes / container.memory_limit_bytes) * 100
                    : 0
                const colorClass = memPercent > 80 ? 'bg-red-500' : memPercent > 50 ? 'bg-yellow-500' : 'bg-green-500'
                return (
                    <div className="space-y-1">
                        <div className="flex items-baseline gap-1">
                            <span className="text-sm font-medium">{formatBytes(container.memory_usage_bytes)}</span>
                            {container.memory_limit_bytes > 0 && (
                                <span className="text-xs text-muted-foreground">
                                    / {formatBytes(container.memory_limit_bytes)}
                                </span>
                            )}
                        </div>
                        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                            <div
                                className={`h-1.5 rounded-full transition-all ${colorClass}`}
                                style={{ width: `${Math.min(memPercent, 100)}%` }}
                            />
                        </div>
                    </div>
                )
            }
        }
    ]

    // Define actions
    const actions: RowAction<ContainerInfo>[] = [
        {
            label: container => container.state === 'stopped' ? 'Start Container' : 'Restart Container',
            icon: <RotateCw className="h-4 w-4" />,
            onClick: (container) => setContainerToRestart(container),
            loading: (container) => loadingContainerIds.has(container.id) && restartMutation.isPending
        },
        {
            label: 'Stop Container',
            icon: <Square className="h-4 w-4" />,
            onClick: (container) => setContainerToStop(container),
            show: (container) => container.state === 'running',
            loading: (container) => loadingContainerIds.has(container.id) && stopMutation.isPending
        },
        {
            label: 'Delete Container',
            icon: <Trash2 className="h-4 w-4" />,
            onClick: (container) => setContainerToDelete(container),
            variant: 'destructive',
            show: (container) => !container.is_managed, // Only show for non-managed containers
            loading: (container) => loadingContainerIds.has(container.id) && deleteMutation.isPending
        }
    ]

    // Expandable content for detailed stats
    const expandableContent = (container: ContainerInfo) => {
        if (container.state !== 'running') {
            return (
                <div className="text-sm text-muted-foreground">
                    Container is {container.state}. Resource metrics are only available for running containers.
                </div>
            )
        }

        const memPercent = container.memory_limit_bytes > 0
            ? (container.memory_usage_bytes / container.memory_limit_bytes) * 100
            : 0

        return (
            <div className="space-y-4">
                {/* Resource Usage Summary */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    {/* CPU */}
                    <div>
                        <p className="text-xs text-muted-foreground mb-2">CPU Usage</p>
                        <div className="flex items-baseline gap-1 mb-2">
                            <p className="text-lg font-semibold">{container.cpu_percent.toFixed(1)}%</p>
                        </div>
                        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                            <div
                                className={`h-2 rounded-full transition-all ${container.cpu_percent > 80
                                    ? 'bg-red-500'
                                    : container.cpu_percent > 50
                                        ? 'bg-yellow-500'
                                        : 'bg-blue-500'
                                    }`}
                                style={{ width: `${Math.min(container.cpu_percent, 100)}%` }}
                            />
                        </div>
                    </div>

                    {/* Memory */}
                    <div>
                        <p className="text-xs text-muted-foreground mb-2">Memory Usage</p>
                        <div className="flex items-baseline gap-1 mb-2">
                            <p className="text-lg font-semibold">{formatBytes(container.memory_usage_bytes)}</p>
                            {container.memory_limit_bytes > 0 && (
                                <p className="text-xs text-muted-foreground">
                                    / {formatBytes(container.memory_limit_bytes)} ({memPercent.toFixed(1)}%)
                                </p>
                            )}
                        </div>
                        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                            <div
                                className={`h-2 rounded-full transition-all ${memPercent > 80
                                    ? 'bg-red-500'
                                    : memPercent > 50
                                        ? 'bg-yellow-500'
                                        : 'bg-green-500'
                                    }`}
                                style={{ width: `${Math.min(memPercent, 100)}%` }}
                            />
                        </div>
                    </div>

                    {/* Network I/O */}
                    <div>
                        <p className="text-xs text-muted-foreground mb-2 flex items-center gap-1">
                            <Network className="h-3 w-3" />
                            Network I/O
                        </p>
                        <div className="space-y-1.5">
                            <div className="flex items-center gap-2 text-sm">
                                <span className="text-blue-600 dark:text-blue-400">↓</span>
                                <span className="font-medium">{formatBytes(container.network_rx_bytes)}</span>
                                <span className="text-xs text-muted-foreground">received</span>
                            </div>
                            <div className="flex items-center gap-2 text-sm">
                                <span className="text-green-600 dark:text-green-400">↑</span>
                                <span className="font-medium">{formatBytes(container.network_tx_bytes)}</span>
                                <span className="text-xs text-muted-foreground">sent</span>
                            </div>
                        </div>
                    </div>

                    {/* Disk I/O */}
                    <div>
                        <p className="text-xs text-muted-foreground mb-2 flex items-center gap-1">
                            <HardDrive className="h-3 w-3" />
                            Disk I/O
                        </p>
                        <div className="space-y-1.5">
                            <div className="flex items-center gap-2 text-sm">
                                <span className="text-blue-600 dark:text-blue-400">R</span>
                                <span className="font-medium">{formatBytes(container.block_read_bytes)}</span>
                                <span className="text-xs text-muted-foreground">read</span>
                            </div>
                            <div className="flex items-center gap-2 text-sm">
                                <span className="text-orange-600 dark:text-orange-400">W</span>
                                <span className="font-medium">{formatBytes(container.block_write_bytes)}</span>
                                <span className="text-xs text-muted-foreground">written</span>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Additional Info */}
                <div className="pt-3 border-t flex items-center gap-4 text-xs text-muted-foreground">
                    <div>
                        <span className="font-medium">Container ID:</span>{' '}
                        <span className="font-mono">{container.id}</span>
                    </div>
                    {container.restart_count > 0 && (
                        <div>
                            <span className="font-medium">Restarts:</span>{' '}
                            <span>{container.restart_count}</span>
                        </div>
                    )}
                    <div>
                        <span className="font-medium">Created:</span>{' '}
                        <span>{new Date(container.created_at).toLocaleString()}</span>
                    </div>
                </div>
            </div>
        )
    }

    // Empty state
    const emptyState = (
        <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-muted mb-4">
                <Server className="w-8 h-8 text-muted-foreground" />
            </div>
            <h3 className="text-lg font-semibold mb-2">No containers found</h3>
            <p className="text-muted-foreground max-w-sm mx-auto">
                No containers match the selected filters
            </p>
        </div>
    )

    return (
        <>
            {/* Data Table */}
            <DataTable
                data={containers}
                columns={columns}
                getRowKey={(container) => container.id}
                actions={actions}
                expandableContent={expandableContent}
                emptyState={emptyState}
            />

            {/* Restart/Start Confirmation Dialog */}
            <ConfirmationDialog
                open={!!containerToRestart}
                onOpenChange={(open: boolean) => !open && setContainerToRestart(null)}
                title={containerToRestart?.state === 'stopped' ? 'Start Container' : 'Restart Container'}
                description={
                    containerToRestart?.state === 'stopped'
                        ? `Are you sure you want to start "${containerToRestart?.name}"?`
                        : `Are you sure you want to restart "${containerToRestart?.name}"? This will cause a brief service interruption.`
                }
                confirmText={containerToRestart?.state === 'stopped' ? 'Start' : 'Restart'}
                cancelText="Cancel"
                onConfirm={handleRestart}
                isLoading={restartMutation.isPending}
                variant="default"
            />

            {/* Stop Confirmation Dialog */}
            <ConfirmationDialog
                open={!!containerToStop}
                onOpenChange={(open: boolean) => !open && setContainerToStop(null)}
                title="Stop Container"
                description={`Are you sure you want to stop "${containerToStop?.name}"? The container will remain stopped until manually started again.`}
                confirmText="Stop"
                cancelText="Cancel"
                onConfirm={handleStop}
                isLoading={stopMutation.isPending}
                variant="destructive"
            />

            {/* Delete Confirmation Dialog */}
            <ConfirmationDialog
                open={!!containerToDelete}
                onOpenChange={(open: boolean) => !open && setContainerToDelete(null)}
                title="Delete Container"
                description={`Are you sure you want to permanently delete "${containerToDelete?.name}"? This action cannot be undone and will remove the container and any data stored in it (volumes may persist depending on configuration).${containerToDelete?.app_name !== 'unmanaged' ? `\n\nNote: This is an external container (${containerToDelete?.app_name}) not managed by this system.` : ''}`}
                confirmText="Delete Container"
                cancelText="Cancel"
                onConfirm={handleDelete}
                isLoading={deleteMutation.isPending}
                variant="destructive"
            />
        </>
    )
}

export default ContainersListView
