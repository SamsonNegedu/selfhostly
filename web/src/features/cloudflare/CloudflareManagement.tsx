import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Badge } from '@/shared/components/ui/badge'
import { RefreshCw, Trash2, ExternalLink, AlertCircle, CheckCircle, Clock } from 'lucide-react'
import { useCloudflareTunnels, useSyncCloudflareTunnel, useDeleteCloudflareTunnel } from '@/shared/services/api'

function CloudflareManagement() {
    const { data: tunnelsData, isLoading, error } = useCloudflareTunnels()
    const syncTunnel = useSyncCloudflareTunnel()
    const deleteTunnel = useDeleteCloudflareTunnel()

    const handleSync = (appId: string) => {
        syncTunnel.mutate(appId)
    }

    const handleDelete = (appId: string) => {
        if (window.confirm('Are you sure you want to delete this Cloudflare tunnel? This will remove the tunnel configuration and cannot be undone.')) {
            deleteTunnel.mutate(appId)
        }
    }

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'active':
                return <CheckCircle className="h-4 w-4 text-green-500" />
            case 'inactive':
                return <Clock className="h-4 w-4 text-yellow-500" />
            case 'error':
                return <AlertCircle className="h-4 w-4 text-red-500" />
            case 'deleted':
                return <Trash2 className="h-4 w-4 text-gray-500" />
            default:
                return <Clock className="h-4 w-4 text-gray-500" />
        }
    }

    const getStatusBadge = (isActive: boolean) => {
        if (isActive) {
            return (
                <Badge variant="default" className="bg-green-100 text-green-800">
                    <CheckCircle className="h-3 w-3 mr-1" />
                    Active
                </Badge>
            )
        }
        return (
            <Badge variant="secondary" className="bg-yellow-100 text-yellow-800">
                <Clock className="h-3 w-3 mr-1" />
                Inactive
            </Badge>
        )
    }

    if (isLoading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="h-8 w-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
            </div>
        )
    }

    if (error) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle className="text-red-600">Error Loading Cloudflare Tunnels</CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-muted-foreground">{error.message}</p>
                </CardContent>
            </Card>
        )
    }

    const tunnels = tunnelsData?.tunnels || []

    return (
        <div className="space-y-6">
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <RefreshCw className="h-5 w-5" />
                        Cloudflare Tunnel Management
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="text-sm text-muted-foreground mb-4">
                        Manage your Cloudflare tunnels for self-hosted applications. Active tunnels provide secure access to your applications via Cloudflare.
                    </div>

                    {tunnels.length === 0 ? (
                        <div className="text-center py-8">
                            <AlertCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                            <h3 className="text-lg font-medium mb-2">No Cloudflare Tunnels Found</h3>
                            <p className="text-muted-foreground mb-4">
                                You don't have any Cloudflare tunnels configured yet. Create an app with Cloudflare tunnel enabled to get started.
                            </p>
                        </div>
                    ) : (
                        <div className="space-y-4">
                            <div className="text-sm text-muted-foreground">
                                Showing {tunnels.length} tunnel{tunnels.length !== 1 ? 's' : ''}
                            </div>

                            {tunnels.map((tunnel) => (
                                <Card key={tunnel.id}>
                                    <CardContent className="p-4">
                                        <div className="flex items-center justify-between">
                                            <div className="flex items-center gap-3">
                                                {getStatusIcon(tunnel.status)}
                                                <div>
                                                    <h3 className="font-medium">{tunnel.tunnel_name}</h3>
                                                    <div className="flex items-center gap-2 mt-1">
                                                        {getStatusBadge(tunnel.is_active)}
                                                        {tunnel.public_url && (
                                                            <Badge variant="outline" className="bg-blue-50 text-blue-700">
                                                                <ExternalLink className="h-3 w-3 mr-1" />
                                                                {new URL(tunnel.public_url).hostname}
                                                            </Badge>
                                                        )}
                                                    </div>
                                                    {tunnel.error_details && (
                                                        <p className="text-sm text-red-600 mt-2">{tunnel.error_details}</p>
                                                    )}
                                                </div>
                                            </div>

                                            <div className="flex items-center gap-2">
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => handleSync(tunnel.app_id)}
                                                    disabled={syncTunnel.isPending}
                                                >
                                                    <RefreshCw className={`h-4 w-4 ${syncTunnel.isPending ? 'animate-spin' : ''}`} />
                                                    Sync
                                                </Button>
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => handleDelete(tunnel.app_id)}
                                                    disabled={deleteTunnel.isPending}
                                                    className="text-destructive hover:text-destructive"
                                                >
                                                    <Trash2 className="h-4 w-4" />
                                                    Delete
                                                </Button>
                                            </div>
                                        </div>

                                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-4 pt-4 border-t">
                                            <div>
                                                <p className="text-xs text-muted-foreground">Tunnel ID</p>
                                                <p className="text-sm font-medium truncate">{tunnel.tunnel_id}</p>
                                            </div>
                                            <div>
                                                <p className="text-xs text-muted-foreground">Status</p>
                                                <p className="text-sm font-medium">{tunnel.status}</p>
                                            </div>
                                            <div>
                                                <p className="text-xs text-muted-foreground">Created</p>
                                                <p className="text-sm font-medium">
                                                    {new Date(tunnel.created_at).toLocaleDateString()}
                                                </p>
                                            </div>
                                            <div>
                                                <p className="text-xs text-muted-foreground">Last Synced</p>
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
        </div>
    )
}

export default CloudflareManagement
