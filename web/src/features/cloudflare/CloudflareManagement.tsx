import React from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Badge } from '@/shared/components/ui/badge'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { RefreshCw, ExternalLink, AlertCircle, CheckCircle2, Clock, Plus, Search } from 'lucide-react'
import { useCloudflareTunnels, useSyncCloudflareTunnel, useDeleteCloudflareTunnel } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import { useState } from 'react'

function CloudflareManagement() {
    const { data: tunnelsData, isLoading, error, refetch } = useCloudflareTunnels()
    const syncTunnel = useSyncCloudflareTunnel()
    const deleteTunnel = useDeleteCloudflareTunnel()
    const { toast } = useToast()

    const [searchQuery, setSearchQuery] = useState('')
    const [tunnelToDelete, setTunnelToDelete] = useState<{ id: string; name: string } | null>(null)

    const tunnels = tunnelsData?.tunnels || []

    // Filter tunnels by search
    const filteredTunnels = React.useMemo(() => {
        if (!searchQuery) return tunnels
        const query = searchQuery.toLowerCase()
        return tunnels.filter(tunnel =>
            tunnel.tunnel_name.toLowerCase().includes(query) ||
            tunnel.tunnel_id.toLowerCase().includes(query) ||
            (tunnel.public_url && tunnel.public_url.toLowerCase().includes(query))
        )
    }, [tunnels, searchQuery])

    const handleSync = (appId: string, appName: string) => {
        syncTunnel.mutate(appId, {
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

        deleteTunnel.mutate(tunnelToDelete.id, {
            onSuccess: () => {
                toast.success('Tunnel Deleted', `${tunnelToDelete.name} tunnel has been removed`)
                setTunnelToDelete(null)
            },
            onError: (error) => {
                toast.error('Delete Failed', error.message)
            }
        })
    }

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'active':
                return <CheckCircle2 className="h-4 w-4 text-green-500" />
            case 'inactive':
                return <Clock className="h-4 w-4 text-yellow-500" />
            case 'error':
                return <AlertCircle className="h-4 w-4 text-red-500" />
            case 'deleted':
                return <Clock className="h-4 w-4 text-gray-500" />
            default:
                return <Clock className="h-4 w-4 text-gray-500" />
        }
    }

    const getStatusBadge = (isActive: boolean) => {
        if (isActive) {
            return (
                <Badge variant="default" className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                    <CheckCircle2 className="h-3 w-3 mr-1" />
                    Active
                </Badge>
            )
        }
        return (
            <Badge variant="secondary" className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200">
                <Clock className="h-3 w-3 mr-1" />
                Inactive
            </Badge>
        )
    }

    if (isLoading) {
        return (
            <div className="flex items-center justify-center min-h-[400px] fade-in">
                <Card className="w-full max-w-md">
                    <CardContent className="py-12">
                        <div className="flex flex-col items-center gap-4">
                            <div className="h-12 w-12 border-4 border-primary border-t-transparent rounded-full animate-spin" />
                            <p className="text-muted-foreground">Loading Cloudflare tunnels...</p>
                        </div>
                    </CardContent>
                </Card>
            </div>
        )
    }

    if (error) {
        return (
            <div className="flex items-center justify-center min-h-[400px] fade-in">
                <Card className="w-full max-w-md">
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
    const inactiveCount = tunnels.filter(t => !t.is_active).length

    return (
        <div className="space-y-6 fade-in">
            <Card>
                <CardHeader>
                    <div className="flex items-center justify-between flex-wrap gap-4">
                        <CardTitle className="flex items-center gap-2">
                            <RefreshCw className="h-5 w-5" />
                            Cloudflare Tunnel Management
                        </CardTitle>
                        <Button
                            onClick={() => refetch()}
                            variant="outline"
                            size="sm"
                            className="button-press"
                        >
                            <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
                            Refresh
                        </Button>
                    </div>
                </CardHeader>
                <CardContent className="space-y-6">
                    <p className="text-sm text-muted-foreground">
                        Manage your Cloudflare tunnels for self-hosted applications. Active tunnels provide secure access to your applications via Cloudflare.
                    </p>

                    {/* Stats */}
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <Card className="border-primary/20">
                            <CardContent className="p-4 text-center">
                                <RefreshCw className="h-8 w-8 text-primary mx-auto mb-2" />
                                <div className="text-2xl font-bold">{tunnels.length}</div>
                                <div className="text-sm text-muted-foreground">Total Tunnels</div>
                            </CardContent>
                        </Card>
                        <Card className="border-green-200 dark:border-green-900/30">
                            <CardContent className="p-4 text-center">
                                <CheckCircle2 className="h-8 w-8 text-green-500 mx-auto mb-2" />
                                <div className="text-2xl font-bold text-green-600 dark:text-green-400">{activeCount}</div>
                                <div className="text-sm text-muted-foreground">Active</div>
                            </CardContent>
                        </Card>
                        <Card className="border-yellow-200 dark:border-yellow-900/30">
                            <CardContent className="p-4 text-center">
                                <Clock className="h-8 w-8 text-yellow-500 mx-auto mb-2" />
                                <div className="text-2xl font-bold text-yellow-600 dark:text-yellow-400">{inactiveCount}</div>
                                <div className="text-sm text-muted-foreground">Inactive</div>
                            </CardContent>
                        </Card>
                    </div>

                    {/* Search */}
                    <div className="relative">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                        <input
                            type="text"
                            placeholder="Search tunnels by name or ID..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 pl-10 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                        />
                        {searchQuery && (
                            <button
                                onClick={() => setSearchQuery('')}
                                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                            >
                                ×
                            </button>
                        )}
                    </div>

                    {/* Empty State */}
                    {tunnels.length === 0 && (
                        <div className="text-center py-12">
                            <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-muted mb-4">
                                <AlertCircle className="h-8 w-8 text-muted-foreground" />
                            </div>
                            <h3 className="text-lg font-semibold mb-2">No Cloudflare Tunnels Found</h3>
                            <p className="text-muted-foreground mb-4 max-w-md mx-auto">
                                You don't have any Cloudflare tunnels configured yet. Create an app with Cloudflare tunnel enabled to get started.
                            </p>
                            <Button onClick={() => window.location.href = '/apps/new'} className="button-press">
                                <Plus className="h-4 w-4 mr-2" />
                                Create New App
                            </Button>
                        </div>
                    )}

                    {/* No Search Results */}
                    {tunnels.length > 0 && filteredTunnels.length === 0 && (
                        <div className="text-center py-12">
                            <Search className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                            <h3 className="text-lg font-semibold mb-2">No Matching Tunnels</h3>
                            <p className="text-muted-foreground mb-4">
                                No tunnels found matching "{searchQuery}". Try adjusting your search query.
                            </p>
                        </div>
                    )}

                    {/* Tunnels List */}
                    {filteredTunnels.length > 0 && (
                        <div className="space-y-4">
                            <div className="text-sm text-muted-foreground mb-2">
                                Showing {filteredTunnels.length} of {tunnels.length} tunnel{tunnels.length !== 1 ? 's' : ''}
                            </div>

                            {filteredTunnels.map((tunnel) => (
                                <Card key={tunnel.id} className="card-hover border-2">
                                    <CardContent className="p-4">
                                        <div className="flex items-start justify-between mb-4">
                                            <div className="flex items-center gap-3">
                                                {getStatusIcon(tunnel.status)}
                                                <div>
                                                    <h3 className="font-semibold text-base">{tunnel.tunnel_name}</h3>
                                                    <div className="flex items-center gap-2 mt-1">
                                                        {getStatusBadge(tunnel.is_active)}
                                                        {tunnel.public_url && (
                                                            <Badge variant="outline" className="bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-300 ml-2">
                                                                <ExternalLink className="h-3 w-3 mr-1" />
                                                                {new URL(tunnel.public_url).hostname}
                                                            </Badge>
                                                        )}
                                                    </div>
                                                    {tunnel.error_details && (
                                                        <p className="text-sm text-red-600 dark:text-red-400 mt-1">
                                                            <AlertCircle className="h-3 w-3 mr-1" />
                                                            {tunnel.error_details}
                                                        </p>
                                                    )}
                                                </div>
                                            </div>

                                            <div className="flex items-center gap-2">
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => handleSync(tunnel.app_id, tunnel.tunnel_name)}
                                                    disabled={syncTunnel.isPending}
                                                    className="button-press"
                                                >
                                                    <RefreshCw className={`h-4 w-4 ${syncTunnel.isPending ? 'animate-spin' : ''}`} />
                                                    Sync
                                                </Button>
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => handleDelete(tunnel.app_id, tunnel.tunnel_name)}
                                                    disabled={deleteTunnel.isPending}
                                                    className="text-destructive hover:text-destructive button-press"
                                                >
                                                    <AlertCircle className="h-4 w-4" />
                                                    Delete
                                                </Button>
                                            </div>
                                        </div>

                                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 pt-4 border-t">
                                            <div>
                                                <p className="text-xs text-muted-foreground mb-1">Tunnel ID</p>
                                                <p className="text-sm font-medium font-mono truncate" title={tunnel.tunnel_id}>{tunnel.tunnel_id}</p>
                                            </div>
                                            <div>
                                                <p className="text-xs text-muted-foreground mb-1">Status</p>
                                                <p className="text-sm font-medium">{tunnel.status}</p>
                                            </div>
                                            <div>
                                                <p className="text-xs text-muted-foreground mb-1">Created</p>
                                                <p className="text-sm font-medium">
                                                    {new Date(tunnel.created_at).toLocaleDateString()}
                                                </p>
                                            </div>
                                            <div>
                                                <p className="text-xs text-muted-foreground mb-1">Last Synced</p>
                                                <p className="text-sm font-medium">
                                                    {tunnel.last_synced_at
                                                        ? new Date(tunnel.last_synced_at).toLocaleString()
                                                        : 'Never'
                                                    }
                                                </p>
                                            </div>
                                        </div>
                                    </CardContent>
                                </Card>
                            ))}
                        </div>
                    )}
                </CardContent>
            </Card>

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
