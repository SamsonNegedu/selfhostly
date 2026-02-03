import { useState } from 'react'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import {
    RefreshCw,
    ExternalLink,
    AlertCircle,
    CheckCircle2,
    Clock,
    Copy,
    Trash2,
    Eye,
    Link2
} from 'lucide-react'
import { useSyncTunnel, useDeleteTunnel, useApps } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import { DataTable, ColumnDef, RowAction } from '@/shared/components/ui/DataTable'
import type { CloudflareTunnel } from '@/shared/types/api'

interface TunnelsListViewProps {
    tunnels: CloudflareTunnel[]
    apps: Array<{ id: string; node_id?: string }>
}

// Status badge component
const getStatusBadge = (tunnel: CloudflareTunnel) => {
    if (tunnel.is_active) {
        return (
            <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md text-xs font-semibold text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20">
                <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                active
            </div>
        )
    }

    if (tunnel.status === 'error') {
        return (
            <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md text-xs font-semibold text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20">
                <AlertCircle className="h-3.5 w-3.5" />
                error
            </div>
        )
    }

    return (
        <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md text-xs font-semibold text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-800">
            <div className="w-2 h-2 bg-gray-400 rounded-full" />
            inactive
        </div>
    )
}

function TunnelsListView({ tunnels, apps: appsProp }: TunnelsListViewProps) {
    const syncTunnel = useSyncTunnel()
    const deleteTunnel = useDeleteTunnel()
    const { toast } = useToast()
    const { data: appsData } = useApps() // Fetch apps to ensure we have latest data

    // Use apps from hook if available, otherwise fall back to prop
    const apps = appsData || appsProp

    const [tunnelToDelete, setTunnelToDelete] = useState<{ id: string; name: string; nodeId?: string } | null>(null)
    const [loadingTunnelIds, setLoadingTunnelIds] = useState<Set<string>>(new Set())

    // Copy to clipboard helper
    const copyToClipboard = async (text: string, label: string) => {
        try {
            await navigator.clipboard.writeText(text)
            toast.success('Copied!', `${label} copied to clipboard`)
        } catch (err) {
            toast.error('Copy Failed', 'Failed to copy to clipboard')
        }
    }

    // Define columns
    const columns: ColumnDef<CloudflareTunnel>[] = [
        {
            key: 'status',
            label: 'Status',
            width: 'w-28',
            render: (tunnel) => getStatusBadge(tunnel),
            sortable: true,
            sortValue: (tunnel) => tunnel.is_active ? 'active' : tunnel.status || 'inactive'
        },
        {
            key: 'name',
            label: 'Tunnel Name',
            width: 'flex-1 min-w-0',
            sortable: true,
            sortValue: (tunnel) => tunnel.tunnel_name.toLowerCase(),
            render: (tunnel) => (
                <div>
                    <div className="font-medium">{tunnel.tunnel_name}</div>
                    {tunnel.error_details && (
                        <div className="text-xs text-red-600 dark:text-red-400 mt-1 flex items-center gap-1">
                            <AlertCircle className="h-3 w-3" />
                            {tunnel.error_details}
                        </div>
                    )}
                </div>
            )
        },
        {
            key: 'public_url',
            label: 'Public URL',
            width: 'w-64',
            sortable: true,
            sortValue: (tunnel) => tunnel.public_url ? tunnel.public_url.toLowerCase() : '',
            render: (tunnel) => {
                if (!tunnel.public_url) {
                    return <span className="text-muted-foreground text-sm">—</span>
                }
                return (
                    <div className="flex items-center gap-2">
                        <a
                            href={tunnel.public_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-blue-600 dark:text-blue-400 hover:underline flex items-center gap-1 text-sm truncate"
                            onClick={(e) => e.stopPropagation()}
                        >
                            <ExternalLink className="h-3.5 w-3.5 flex-shrink-0" />
                            <span className="truncate">{new URL(tunnel.public_url).hostname}</span>
                        </a>
                        <button
                            onClick={(e) => {
                                e.stopPropagation()
                                copyToClipboard(tunnel.public_url, 'Public URL')
                            }}
                            className="text-muted-foreground hover:text-foreground transition-colors"
                            title="Copy URL"
                        >
                            <Copy className="h-3.5 w-3.5" />
                        </button>
                    </div>
                )
            }
        },
        {
            key: 'tunnel_id',
            label: 'Tunnel ID',
            width: 'w-40',
            render: (tunnel) => (
                <div className="flex items-center gap-2">
                    <code className="text-xs font-mono text-muted-foreground">{tunnel.tunnel_id.substring(0, 8)}...</code>
                    <button
                        onClick={(e) => {
                            e.stopPropagation()
                            copyToClipboard(tunnel.tunnel_id, 'Tunnel ID')
                        }}
                        className="text-muted-foreground hover:text-foreground transition-colors"
                        title="Copy ID"
                    >
                        <Copy className="h-3.5 w-3.5" />
                    </button>
                </div>
            )
        },
        {
            key: 'created',
            label: 'Created',
            width: 'w-36',
            sortable: true,
            sortValue: (tunnel) => new Date(tunnel.created_at),
            render: (tunnel) => (
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Clock className="h-3 w-3 flex-shrink-0" />
                    <span>{new Date(tunnel.created_at).toLocaleDateString()}</span>
                </div>
            )
        }
    ]

    // Handle sync
    const handleSync = (tunnel: CloudflareTunnel) => {
        const app = apps.find(a => a.id === tunnel.app_id)
        if (!app?.node_id) {
            toast.error('Sync Failed', 'Unable to determine app node')
            return
        }
        setLoadingTunnelIds(prev => new Set(prev).add(tunnel.app_id))
        syncTunnel.mutate({ appId: tunnel.app_id, nodeId: app.node_id }, {
            onSuccess: () => {
                toast.success('Tunnel Synced', `${tunnel.tunnel_name} tunnel configuration synced successfully`)
                setLoadingTunnelIds(prev => {
                    const next = new Set(prev)
                    next.delete(tunnel.app_id)
                    return next
                })
            },
            onError: (error) => {
                toast.error('Sync Failed', error.message)
                setLoadingTunnelIds(prev => {
                    const next = new Set(prev)
                    next.delete(tunnel.app_id)
                    return next
                })
            }
        })
    }

    // Handle view app
    const handleViewApp = (tunnel: CloudflareTunnel) => {
        const app = apps.find(a => a.id === tunnel.app_id)
        const nodeIdParam = app?.node_id ? `?node_id=${app.node_id}` : ''
        window.location.href = `/apps/${tunnel.app_id}${nodeIdParam}`
    }

    // Define actions
    const actions: RowAction<CloudflareTunnel>[] = [
        {
            label: 'Sync Tunnel',
            icon: <RefreshCw className="h-4 w-4" />,
            onClick: (tunnel) => handleSync(tunnel),
            loading: (tunnel) => loadingTunnelIds.has(tunnel.app_id) && syncTunnel.isPending
        },
        {
            label: 'View App',
            icon: <Eye className="h-4 w-4" />,
            onClick: (tunnel) => handleViewApp(tunnel)
        },
        {
            label: 'Delete Tunnel',
            icon: <Trash2 className="h-4 w-4" />,
            onClick: (tunnel) => {
                const app = apps.find(a => a.id === tunnel.app_id)
                setTunnelToDelete({ 
                    id: tunnel.app_id, 
                    name: tunnel.tunnel_name,
                    nodeId: app?.node_id 
                })
            },
            variant: 'destructive',
            loading: (tunnel) => loadingTunnelIds.has(tunnel.app_id) && deleteTunnel.isPending
        }
    ]

    // Handle tunnel deletion confirmation
    const confirmDelete = () => {
        if (!tunnelToDelete) {
            console.error('confirmDelete called but tunnelToDelete is null')
            return
        }

        // Try to get node_id from stored value first, then from apps array
        let nodeId = tunnelToDelete.nodeId
        if (!nodeId) {
            const app = apps.find(a => a.id === tunnelToDelete.id)
            nodeId = app?.node_id
        }

        if (!nodeId) {
            console.error('Unable to determine node_id for tunnel:', tunnelToDelete)
            toast.error('Delete Failed', `Unable to determine app node for "${tunnelToDelete.name}". Please try refreshing the page.`)
            setTunnelToDelete(null)
            return
        }

        console.log('Deleting tunnel:', { appId: tunnelToDelete.id, nodeId, name: tunnelToDelete.name })
        setLoadingTunnelIds(prev => new Set(prev).add(tunnelToDelete.id))
        
        deleteTunnel.mutate({ appId: tunnelToDelete.id, nodeId }, {
            onSuccess: () => {
                toast.success('Tunnel Deleted', `${tunnelToDelete.name} tunnel has been removed`)
                setTunnelToDelete(null)
                setLoadingTunnelIds(prev => {
                    const next = new Set(prev)
                    next.delete(tunnelToDelete.id)
                    return next
                })
            },
            onError: (error) => {
                console.error('Delete tunnel error:', error)
                toast.error('Delete Failed', error.message || 'Failed to delete tunnel')
                setTunnelToDelete(null)
                setLoadingTunnelIds(prev => {
                    const next = new Set(prev)
                    next.delete(tunnelToDelete.id)
                    return next
                })
            }
        })
    }

    // Expandable content for error details and more info
    const expandableContent = (tunnel: CloudflareTunnel) => (
        <div className="space-y-3">
            {tunnel.error_details && (
                <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                    <div className="flex items-start gap-2">
                        <AlertCircle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
                        <div>
                            <p className="text-sm font-semibold text-red-800 dark:text-red-300">Error Details</p>
                            <p className="text-xs text-red-700 dark:text-red-400 mt-1">{tunnel.error_details}</p>
                        </div>
                    </div>
                </div>
            )}
            <div className="flex items-center gap-4 text-xs text-muted-foreground">
                <div className="flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    <span>Created: {new Date(tunnel.created_at).toLocaleString()}</span>
                </div>
                <div className="flex items-center gap-1">
                    <RefreshCw className="h-3 w-3" />
                    <span>Updated: {new Date(tunnel.updated_at).toLocaleString()}</span>
                </div>
                {tunnel.last_synced_at && (
                    <div className="flex items-center gap-1">
                        <CheckCircle2 className="h-3 w-3" />
                        <span>Last Synced: {new Date(tunnel.last_synced_at).toLocaleString()}</span>
                    </div>
                )}
            </div>
            {tunnel.public_url && (
                <div className="flex items-center gap-2 text-sm">
                    <ExternalLink className="h-4 w-4 text-muted-foreground" />
                    <span className="text-muted-foreground">Full URL:</span>
                    <a
                        href={tunnel.public_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-primary hover:underline break-all"
                    >
                        {tunnel.public_url}
                    </a>
                </div>
            )}
        </div>
    )

    // Empty state
    const emptyState = (
        <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-muted mb-4">
                <Link2 className="w-8 h-8 text-muted-foreground" />
            </div>
            <h3 className="text-lg font-semibold mb-2">No tunnels found</h3>
            <p className="text-muted-foreground max-w-sm mx-auto">
                No tunnels match the selected filters
            </p>
        </div>
    )

    return (
        <>
            {/* Data Table */}
            <DataTable
                data={tunnels}
                columns={columns}
                getRowKey={(tunnel) => tunnel.id}
                actions={actions}
                expandableContent={expandableContent}
                emptyState={emptyState}
            />

            {/* Confirmation Dialog */}
            <ConfirmationDialog
                open={!!tunnelToDelete}
                onOpenChange={(open: boolean) => !open && setTunnelToDelete(null)}
                title="Delete Cloudflare Tunnel"
                description={`Are you sure you want to delete "${tunnelToDelete?.name}"? This will remove the tunnel configuration and your app will no longer be accessible via the public URL. This action cannot be undone.`}
                confirmText="Delete Tunnel"
                cancelText="Cancel"
                onConfirm={confirmDelete}
                isLoading={deleteTunnel.isPending}
                variant="destructive"
            />
        </>
    )
}

export default TunnelsListView
