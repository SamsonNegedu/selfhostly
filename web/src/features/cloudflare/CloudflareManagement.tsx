import React from 'react'
import { Card, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Badge } from '@/shared/components/ui/badge'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import {
    RefreshCw,
    ExternalLink,
    AlertCircle,
    CheckCircle2,
    Clock,
    Plus,
    Search,
    Filter,
    Copy,
    Activity,
    Link2,
    Trash2,
    Eye,
    ArrowUpDown
} from 'lucide-react'
import { useCloudflareTunnels, useSyncCloudflareTunnel, useDeleteCloudflareTunnel } from '@/shared/services/api'
import { useNodeContext } from '@/shared/contexts/NodeContext'
import { useAppStore } from '@/shared/stores/app-store'
import { useToast } from '@/shared/components/ui/Toast'
import { useState } from 'react'
import type { CloudflareTunnel } from '@/shared/types/api'

type SortField = 'name' | 'status' | 'created' | 'updated'
type SortOrder = 'asc' | 'desc'
type StatusFilter = 'all' | 'active' | 'inactive' | 'error'

interface TunnelTableProps {
    tunnels: CloudflareTunnel[]
    apps: Array<{ id: string; node_id?: string }>
    onSync: (appId: string, appName: string) => void
    onDelete: (appId: string, appName: string) => void
    onCopy: (text: string, label: string) => void
    isSyncing: boolean
    isDeleting: boolean
}

function TunnelTable({ tunnels, apps, onSync, onDelete, onCopy, isSyncing, isDeleting }: TunnelTableProps) {
    return (
        <Card className="border-2">
            <div className="overflow-x-auto">
                <table className="w-full">
                    <thead className="border-b-2 bg-muted/50">
                        <tr>
                            <th className="text-left p-4 text-sm font-semibold">Status</th>
                            <th className="text-left p-4 text-sm font-semibold">Tunnel Name</th>
                            <th className="text-left p-4 text-sm font-semibold">Public URL</th>
                            <th className="text-left p-4 text-sm font-semibold">Tunnel ID</th>
                            <th className="text-left p-4 text-sm font-semibold">Created</th>
                            <th className="text-right p-4 text-sm font-semibold">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {tunnels.map((tunnel) => (
                            <tr key={tunnel.id} className="border-b hover:bg-muted/30 transition-colors group">
                                <td className="p-4">
                                    <div className="flex items-center gap-2">
                                        {tunnel.is_active ? (
                                            <Badge variant="default" className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 border-green-200">
                                                <CheckCircle2 className="h-3 w-3 mr-1" />
                                                Active
                                            </Badge>
                                        ) : (
                                            <Badge variant="secondary" className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 border-yellow-200">
                                                <Clock className="h-3 w-3 mr-1" />
                                                Inactive
                                            </Badge>
                                        )}
                                    </div>
                                </td>
                                <td className="p-4">
                                    <div className="font-medium">{tunnel.tunnel_name}</div>
                                    {tunnel.error_details && (
                                        <div className="text-xs text-red-600 dark:text-red-400 mt-1 flex items-center gap-1">
                                            <AlertCircle className="h-3 w-3" />
                                            {tunnel.error_details}
                                        </div>
                                    )}
                                </td>
                                <td className="p-4">
                                    {tunnel.public_url ? (
                                        <div className="flex items-center gap-2">
                                            <a
                                                href={tunnel.public_url}
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                className="text-blue-600 dark:text-blue-400 hover:underline flex items-center gap-1 text-sm"
                                            >
                                                <ExternalLink className="h-3.5 w-3.5" />
                                                {new URL(tunnel.public_url).hostname}
                                            </a>
                                            <button
                                                onClick={() => onCopy(tunnel.public_url, 'Public URL')}
                                                className="opacity-0 group-hover:opacity-100 transition-opacity"
                                            >
                                                <Copy className="h-3.5 w-3.5 text-muted-foreground hover:text-foreground" />
                                            </button>
                                        </div>
                                    ) : (
                                        <span className="text-muted-foreground text-sm">—</span>
                                    )}
                                </td>
                                <td className="p-4">
                                    <div className="flex items-center gap-2">
                                        <code className="text-xs font-mono text-muted-foreground">{tunnel.tunnel_id.substring(0, 8)}...</code>
                                        <button
                                            onClick={() => onCopy(tunnel.tunnel_id, 'Tunnel ID')}
                                            className="opacity-0 group-hover:opacity-100 transition-opacity"
                                        >
                                            <Copy className="h-3.5 w-3.5 text-muted-foreground hover:text-foreground" />
                                        </button>
                                    </div>
                                </td>
                                <td className="p-4 text-sm text-muted-foreground">
                                    {new Date(tunnel.created_at).toLocaleDateString()}
                                </td>
                                <td className="p-4">
                                    <div className="flex items-center justify-end gap-2">
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => onSync(tunnel.app_id, tunnel.tunnel_name)}
                                            disabled={isSyncing}
                                            className="button-press"
                                            title="Sync tunnel"
                                        >
                                            <RefreshCw className={`h-3.5 w-3.5 ${isSyncing ? 'animate-spin' : ''}`} />
                                        </Button>
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => {
                                                const app = apps.find(a => a.id === tunnel.app_id)
                                                const nodeIdParam = app?.node_id ? `?node_id=${app.node_id}` : ''
                                                window.location.href = `/apps/${tunnel.app_id}${nodeIdParam}`
                                            }}
                                            className="button-press"
                                            title="View app"
                                        >
                                            <Eye className="h-3.5 w-3.5" />
                                        </Button>
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => onDelete(tunnel.app_id, tunnel.tunnel_name)}
                                            disabled={isDeleting}
                                            className="text-red-600 hover:text-red-700 button-press"
                                            title="Delete tunnel"
                                        >
                                            <Trash2 className="h-3.5 w-3.5" />
                                        </Button>
                                    </div>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </Card>
    )
}

function CloudflareManagement() {
    // Get global node context for filtering tunnels by selected nodes
    const { selectedNodeIds } = useNodeContext()

    const { data: tunnelsData, isLoading, error, refetch } = useCloudflareTunnels(selectedNodeIds)
    const syncTunnel = useSyncCloudflareTunnel()
    const deleteTunnel = useDeleteCloudflareTunnel()
    const { toast } = useToast()

    const [searchQuery, setSearchQuery] = useState('')
    const [tunnelToDelete, setTunnelToDelete] = useState<{ id: string; name: string } | null>(null)
    const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
    const [sortField, setSortField] = useState<SortField>('name')
    const [sortOrder, setSortOrder] = useState<SortOrder>('asc')

    const tunnels = tunnelsData?.tunnels || []

    // Copy to clipboard helper
    const copyToClipboard = async (text: string, label: string) => {
        try {
            await navigator.clipboard.writeText(text)
            toast.success('Copied!', `${label} copied to clipboard`)
        } catch (err) {
            toast.error('Copy Failed', 'Failed to copy to clipboard')
        }
    }

    // Filter, sort, and search tunnels
    const processedTunnels = React.useMemo(() => {
        let result = [...tunnels]

        // Apply status filter
        if (statusFilter !== 'all') {
            result = result.filter(tunnel => {
                switch (statusFilter) {
                    case 'active':
                        return tunnel.is_active
                    case 'inactive':
                        return !tunnel.is_active
                    case 'error':
                        return tunnel.status === 'error'
                    default:
                        return true
                }
            })
        }

        // Apply search filter
        if (searchQuery) {
            const query = searchQuery.toLowerCase()
            result = result.filter(tunnel =>
                tunnel.tunnel_name.toLowerCase().includes(query) ||
                tunnel.tunnel_id.toLowerCase().includes(query) ||
                (tunnel.public_url && tunnel.public_url.toLowerCase().includes(query))
            )
        }

        // Apply sorting
        result.sort((a, b) => {
            let aValue: any
            let bValue: any

            switch (sortField) {
                case 'name':
                    aValue = a.tunnel_name.toLowerCase()
                    bValue = b.tunnel_name.toLowerCase()
                    break
                case 'status':
                    aValue = a.is_active ? 1 : 0
                    bValue = b.is_active ? 1 : 0
                    break
                case 'created':
                    aValue = new Date(a.created_at).getTime()
                    bValue = new Date(b.created_at).getTime()
                    break
                case 'updated':
                    aValue = new Date(a.updated_at).getTime()
                    bValue = new Date(b.updated_at).getTime()
                    break
                default:
                    return 0
            }

            if (aValue < bValue) return sortOrder === 'asc' ? -1 : 1
            if (aValue > bValue) return sortOrder === 'asc' ? 1 : -1
            return 0
        })

        return result
    }, [tunnels, searchQuery, statusFilter, sortField, sortOrder])

    const toggleSort = (field: SortField) => {
        if (sortField === field) {
            setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
        } else {
            setSortField(field)
            setSortOrder('asc')
        }
    }

    // Get apps from store to find node_id for tunnels
    const apps = useAppStore((state) => state.apps)

    const handleSync = (appId: string, appName: string) => {
        const app = apps.find(a => a.id === appId)
        syncTunnel.mutate({ appId, nodeId: app?.node_id }, {
            onSuccess: () => {
                toast.success('Tunnel Synced', `${appName} tunnel configuration synced successfully`)
            },
            onError: (error) => {
                toast.error('Sync Failed', error.message)
            }
        })
    }

    const handleDelete = (appId: string, appName: string) => {
        setTunnelToDelete({ id: appId, name: appName })
    }

    const confirmDelete = () => {
        if (!tunnelToDelete) return

        const app = apps.find(a => a.id === tunnelToDelete.id)
        deleteTunnel.mutate({ appId: tunnelToDelete.id, nodeId: app?.node_id }, {
            onSuccess: () => {
                toast.success('Tunnel Deleted', `${tunnelToDelete.name} tunnel has been removed`)
                setTunnelToDelete(null)
            },
            onError: (error) => {
                toast.error('Delete Failed', error.message)
            }
        })
    }

    if (isLoading) {
        return (
            <div className="space-y-6 fade-in">
                <div className="flex items-center justify-between">
                    <Skeleton className="h-10 w-64" />
                    <Skeleton className="h-9 w-24" />
                </div>
                <Skeleton className="h-12 w-full" />
                <div className="space-y-4">
                    {[1, 2, 3].map((i) => (
                        <Card key={i} className="border-2">
                            <CardContent className="p-6">
                                <div className="space-y-4">
                                    <div className="flex items-start justify-between">
                                        <div className="space-y-2 flex-1">
                                            <Skeleton className="h-6 w-48" />
                                            <Skeleton className="h-5 w-32" />
                                        </div>
                                        <Skeleton className="h-9 w-24" />
                                    </div>
                                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                                        {[1, 2, 3, 4].map((j) => (
                                            <div key={j}>
                                                <Skeleton className="h-4 w-20 mb-1" />
                                                <Skeleton className="h-5 w-32" />
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </div>
        )
    }

    if (error) {
        return (
            <div className="space-y-6 fade-in">
                <div>
                    <h1 className="flex items-center gap-3 text-3xl font-bold">
                        <div className="p-2 rounded-lg bg-primary/10">
                            <Activity className="h-6 w-6 text-primary" />
                        </div>
                        Cloudflare Tunnels
                    </h1>
                </div>
                <Card className="border-2">
                    <CardContent className="py-12">
                        <div className="flex flex-col items-center gap-4">
                            <AlertCircle className="h-12 w-12 text-red-500" />
                            <div className="text-center">
                                <h2 className="text-xl font-semibold mb-2">Failed to load tunnels</h2>
                                <p className="text-muted-foreground mb-4">{error.message}</p>
                                <Button onClick={() => refetch()} className="button-press">
                                    <RefreshCw className="h-4 w-4 mr-2" />
                                    Retry
                                </Button>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </div>
        )
    }

    const activeCount = tunnels.filter(t => t.is_active).length

    return (
        <div className="space-y-6 fade-in">
            {/* Header */}
            <div className="flex items-center justify-between flex-wrap gap-4">
                <div className="space-y-1">
                    <h1 className="flex items-center gap-3 text-3xl font-bold">
                        <div className="p-2 rounded-lg bg-primary/10">
                            <Activity className="h-6 w-6 text-primary" />
                        </div>
                        Cloudflare Tunnels
                    </h1>
                </div>
                <Button
                    onClick={() => refetch()}
                    variant="outline"
                    size="sm"
                    className="button-press"
                >
                    <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                    Refresh
                </Button>
            </div>

            <div className="space-y-6">
                {/* Info Banner */}
                {tunnels.length > 0 && (
                    <div className="flex items-start gap-3 p-4 rounded-lg bg-muted/50 border-2">
                        <Activity className="h-5 w-5 text-primary flex-shrink-0 mt-0.5" />
                        <div className="flex-1">
                            <p className="text-sm font-medium mb-1">
                                Cloudflare Tunnel Status
                            </p>
                            <p className="text-sm text-muted-foreground">
                                {activeCount > 0
                                    ? `${activeCount} tunnel${activeCount !== 1 ? 's' : ''} actively routing traffic to your applications.`
                                    : 'No active tunnels. Start your applications to establish secure connections.'
                                }
                            </p>
                        </div>
                    </div>
                )}

                {/* Search and Filters */}
                <div className="flex flex-col sm:flex-row gap-3">
                    <div className="relative flex-1">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
                        <input
                            type="text"
                            placeholder="Search tunnels by name, ID, or URL..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="flex h-11 w-full rounded-lg border-2 border-input bg-background px-3 py-2 pl-10 pr-10 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:border-primary transition-colors"
                        />
                        {searchQuery && (
                            <button
                                onClick={() => setSearchQuery('')}
                                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                                aria-label="Clear search"
                            >
                                <span className="text-xl font-light">×</span>
                            </button>
                        )}
                    </div>

                    <div className="flex items-center gap-2">
                        {/* Status Filter */}
                        <div className="flex items-center gap-1 border-2 rounded-lg p-1 bg-muted/50">
                            <Button
                                variant={statusFilter === 'all' ? 'default' : 'ghost'}
                                size="sm"
                                onClick={() => setStatusFilter('all')}
                                className="h-8 px-3"
                            >
                                All
                            </Button>
                            <Button
                                variant={statusFilter === 'active' ? 'default' : 'ghost'}
                                size="sm"
                                onClick={() => setStatusFilter('active')}
                                className="h-8 px-3"
                            >
                                <CheckCircle2 className="h-3.5 w-3.5 mr-1" />
                                Active
                            </Button>
                            <Button
                                variant={statusFilter === 'inactive' ? 'default' : 'ghost'}
                                size="sm"
                                onClick={() => setStatusFilter('inactive')}
                                className="h-8 px-3"
                            >
                                <Clock className="h-3.5 w-3.5 mr-1" />
                                Inactive
                            </Button>
                            <Button
                                variant={statusFilter === 'error' ? 'default' : 'ghost'}
                                size="sm"
                                onClick={() => setStatusFilter('error')}
                                className="h-8 px-3"
                            >
                                <AlertCircle className="h-3.5 w-3.5 mr-1" />
                                Error
                            </Button>
                        </div>

                        {/* Sort */}
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={() => toggleSort('name')}
                            className="h-9 border-2"
                        >
                            <ArrowUpDown className="h-4 w-4 mr-2" />
                            Sort
                        </Button>
                    </div>
                </div>

                {/* Empty State */}
                {tunnels.length === 0 && (
                    <Card className="border-2 border-dashed">
                        <CardContent className="py-16 text-center">
                            <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-primary/10 mb-6">
                                <Link2 className="h-10 w-10 text-primary" />
                            </div>
                            <h3 className="text-2xl font-semibold mb-3">No Tunnels Yet</h3>
                            <p className="text-muted-foreground mb-6 max-w-md mx-auto text-base">
                                You don't have any Cloudflare tunnels configured. Create your first app with Cloudflare tunnel enabled to establish secure public access.
                            </p>
                            <Button onClick={() => window.location.href = '/apps/new'} size="lg" className="button-press">
                                <Plus className="h-5 w-5 mr-2" />
                                Create Your First App
                            </Button>
                        </CardContent>
                    </Card>
                )}

                {/* No Search/Filter Results */}
                {tunnels.length > 0 && processedTunnels.length === 0 && (
                    <Card className="border-2 border-dashed">
                        <CardContent className="py-16 text-center">
                            <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-muted mb-4">
                                <Search className="h-8 w-8 text-muted-foreground" />
                            </div>
                            <h3 className="text-xl font-semibold mb-2">No Matching Tunnels</h3>
                            <p className="text-muted-foreground mb-4">
                                {searchQuery
                                    ? `No tunnels found matching "${searchQuery}"`
                                    : 'No tunnels match the selected filters'
                                }
                            </p>
                            <Button
                                variant="outline"
                                onClick={() => {
                                    setSearchQuery('')
                                    setStatusFilter('all')
                                }}
                                className="button-press"
                            >
                                <Filter className="h-4 w-4 mr-2" />
                                Clear Filters
                            </Button>
                        </CardContent>
                    </Card>
                )}

                {/* Tunnels List */}
                {processedTunnels.length > 0 && (
                    <div className="space-y-4">
                        <div className="flex items-center justify-between">
                            <p className="text-sm font-medium text-muted-foreground">
                                Showing <span className="text-foreground font-semibold">{processedTunnels.length}</span> of <span className="text-foreground font-semibold">{tunnels.length}</span> tunnel{tunnels.length !== 1 ? 's' : ''}
                            </p>
                        </div>

                        <TunnelTable
                            tunnels={processedTunnels}
                            apps={apps}
                            onSync={handleSync}
                            onDelete={handleDelete}
                            onCopy={copyToClipboard}
                            isSyncing={syncTunnel.isPending}
                            isDeleting={deleteTunnel.isPending}
                        />
                    </div>
                )}
            </div>

            {/* Delete Confirmation Dialog */}
            <ConfirmationDialog
                open={!!tunnelToDelete}
                onOpenChange={(open) => !open && setTunnelToDelete(null)}
                title="Delete Cloudflare Tunnel"
                description={`Are you sure you want to delete "${tunnelToDelete?.name}"? This will remove the tunnel configuration and your app will no longer be accessible via the public URL. This action cannot be undone.`}
                confirmText="Delete Tunnel"
                cancelText="Cancel"
                onConfirm={confirmDelete}
                isLoading={deleteTunnel.isPending}
                variant="destructive"
            />
        </div>
    )
}

export default CloudflareManagement
