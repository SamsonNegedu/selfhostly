import React from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Badge } from '@/shared/components/ui/badge'
import { RefreshCw, Trash2, ExternalLink, AlertCircle, CheckCircle, Clock, Cloud, Settings, Eye, Plus } from 'lucide-react'
import { useCloudflareTunnel, useSyncCloudflareTunnel, useDeleteCloudflareTunnel } from '@/shared/services/api'
import { IngressConfiguration } from '@/features/cloudflare/IngressConfiguration'

interface CloudflareTabProps {
    appId: number
}

function CloudflareTab({ appId }: CloudflareTabProps) {
    const { data: tunnel, isLoading, error } = useCloudflareTunnel(appId)
    const syncTunnel = useSyncCloudflareTunnel()
    const deleteTunnel = useDeleteCloudflareTunnel()
    const [showIngressConfig, setShowIngressConfig] = React.useState(false)

    const handleSync = () => {
        syncTunnel.mutate(appId)
    }

    const handleDelete = () => {
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

    if (isLoading) {
        return (
            <Card>
                <CardContent className="flex items-center justify-center h-32">
                    <div className="h-8 w-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
                </CardContent>
            </Card>
        )
    }

    if (error) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle className="text-red-600">Error Loading Cloudflare Tunnel</CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-muted-foreground">{error.message}</p>
                </CardContent>
            </Card>
        )
    }

    if (!tunnel) {
        return (
            <Card>
                <CardContent className="text-center py-8">
                    <div className="text-muted-foreground">
                        No Cloudflare tunnel configured for this app.
                    </div>
                </CardContent>
            </Card>
        )
    }

    return (
        <div className="space-y-6">
            {showIngressConfig ? (
                <IngressConfiguration
                    appId={appId}
                    existingIngress={(tunnel.ingress_rules || [])}
                    existingHostname={tunnel?.public_url ? new URL(tunnel.public_url).hostname : ''}
                    tunnelID={tunnel?.tunnel_id || ''}
                    onSave={() => {
                        // Handle successful ingress configuration save
                        setShowIngressConfig(false)
                    }}
                />
            ) : (
                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                            <Cloud className="h-5 w-5" />
                            Cloudflare Tunnel Configuration
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="grid gap-6">
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div className="space-y-3">
                                    <div>
                                        <label className="text-sm font-medium text-muted-foreground">Tunnel Name</label>
                                        <p className="text-sm">{tunnel.tunnel_name}</p>
                                    </div>
                                    <div>
                                        <label className="text-sm font-medium text-muted-foreground">Status</label>
                                        <div className="flex items-center gap-2 mt-1">
                                            {getStatusIcon(tunnel.status)}
                                            <Badge variant={tunnel.is_active ? "default" : "secondary"}>
                                                {tunnel.status}
                                            </Badge>
                                        </div>
                                    </div>
                                </div>

                                <div className="space-y-3">
                                    <div>
                                        <label className="text-sm font-medium text-muted-foreground">Tunnel ID</label>
                                        <p className="text-sm font-mono text-xs">{tunnel.tunnel_id}</p>
                                    </div>
                                    <div>
                                        <label className="text-sm font-medium text-muted-foreground">Public URL</label>
                                        {tunnel.public_url && (
                                            <div className="flex items-center gap-2 mt-1">
                                                <a
                                                    href={tunnel.public_url}
                                                    target="_blank"
                                                    rel="noopener noreferrer"
                                                    className="text-primary hover:underline text-sm flex items-center gap-1"
                                                >
                                                    {tunnel.public_url}
                                                    <ExternalLink className="h-3 w-3" />
                                                </a>
                                            </div>
                                        )}
                                    </div>
                                </div>
                            </div>

                            {tunnel.error_details && (
                                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3">
                                    <div className="flex items-center gap-2 text-red-700 dark:text-red-400">
                                        <AlertCircle className="h-4 w-4" />
                                        <span className="font-medium">Error Details</span>
                                    </div>
                                    <p className="text-sm text-red-600 dark:text-red-400 mt-1">{tunnel.error_details}</p>
                                </div>
                            )}

                            {/* Ingress Rules Section */}
                            {(tunnel.ingress_rules?.length || 0) > 0 ? (
                                <div className="pt-4 border-t">
                                    <div className="flex items-center justify-between mb-3">
                                        <h3 className="text-sm font-medium">Ingress Rules</h3>
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => setShowIngressConfig(true)}
                                            className="text-xs"
                                        >
                                            <Settings className="h-3 w-3 mr-1" />
                                            Configure
                                        </Button>
                                    </div>
                                    <div className="space-y-2">
                                        {(tunnel.ingress_rules || []).map((rule, index) => (
                                            <div key={index} className="bg-muted/50 rounded-lg p-3 text-sm">
                                                <div className="flex items-center justify-between mb-1">
                                                    <span className="font-medium">Rule {index + 1}</span>
                                                    {index === (tunnel.ingress_rules || []).length - 1 && rule.service === 'http_status:404' && (
                                                        <Badge variant="secondary" className="text-xs">
                                                            Catch-all
                                                        </Badge>
                                                    )}
                                                </div>
                                                <div className="grid grid-cols-1 md:grid-cols-3 gap-2 text-muted-foreground">
                                                    <div>
                                                        <span className="text-xs">Hostname:</span>
                                                        <p className="font-medium">
                                                            {rule.hostname
                                                                ? <span className="text-blue-600">{rule.hostname}</span>
                                                                : <span className="italic">(tunnel URL)</span>
                                                            }
                                                        </p>
                                                    </div>
                                                    <div>
                                                        <span className="text-xs">Service:</span>
                                                        <p className="font-medium font-mono text-xs break-all">
                                                            {rule.service}
                                                        </p>
                                                    </div>
                                                    <div>
                                                        <span className="text-xs">Path:</span>
                                                        <p className="font-medium">
                                                            {rule.path
                                                                ? <span className="text-green-600">{rule.path}</span>
                                                                : <span className="italic">(any path)</span>
                                                            }
                                                        </p>
                                                    </div>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            ) : (
                                <div className="pt-4 border-t">
                                    <div className="flex items-center justify-between mb-3">
                                        <h3 className="text-sm font-medium">Ingress Rules</h3>
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => setShowIngressConfig(true)}
                                            className="flex items-center gap-1"
                                        >
                                            <Plus className="h-3 w-3" />
                                            Configure
                                        </Button>
                                    </div>
                                    <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 text-center">
                                        <Cloud className="h-8 w-8 text-blue-500 mx-auto mb-2" />
                                        <p className="text-sm text-blue-700 dark:text-blue-300">
                                            No ingress rules configured yet
                                        </p>
                                        <p className="text-xs text-blue-600 dark:text-blue-400 mt-1">
                                            Configure ingress rules to control how your app is accessed through the tunnel
                                        </p>
                                    </div>
                                </div>
                            )}

                            <div className="flex items-center gap-2 pt-4 border-t">
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={handleSync}
                                    disabled={syncTunnel.isPending}
                                >
                                    <RefreshCw className={`h-4 w-4 ${syncTunnel.isPending ? 'animate-spin' : ''}`} />
                                    {syncTunnel.isPending ? 'Syncing...' : 'Sync Tunnel'}
                                </Button>

                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => setShowIngressConfig(true)}
                                    className="flex items-center gap-1"
                                >
                                    <Settings className="h-4 w-4" />
                                    Configure Ingress
                                </Button>

                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={handleDelete}
                                    disabled={deleteTunnel.isPending}
                                    className="text-destructive hover:text-destructive"
                                >
                                    <Trash2 className="h-4 w-4" />
                                    {deleteTunnel.isPending ? 'Deleting...' : 'Delete Tunnel'}
                                </Button>
                            </div>

                            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-4 border-t">
                                <div>
                                    <label className="text-xs text-muted-foreground">Created</label>
                                    <p className="text-sm font-medium">
                                        {new Date(tunnel.created_at).toLocaleString()}
                                    </p>
                                </div>
                                <div>
                                    <label className="text-xs text-muted-foreground">Last Updated</label>
                                    <p className="text-sm font-medium">
                                        {new Date(tunnel.updated_at).toLocaleString()}
                                    </p>
                                </div>
                                <div>
                                    <label className="text-xs text-muted-foreground">Last Synced</label>
                                    <p className="text-sm font-medium">
                                        {tunnel.last_synced_at
                                            ? new Date(tunnel.last_synced_at).toLocaleString()
                                            : 'Never'
                                        }
                                    </p>
                                </div>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            )}
        </div>
    )
}

export default CloudflareTab
