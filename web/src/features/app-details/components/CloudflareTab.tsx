import { useState } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Badge } from '@/shared/components/ui/badge'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { RefreshCw, AlertCircle, CheckCircle2, Copy, Globe, Shield, ArrowRight, Clock, Trash2, Server } from 'lucide-react'
import { useCloudflareTunnel, useSyncCloudflareTunnel, useDeleteCloudflareTunnel } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import { IngressConfiguration } from '@/features/cloudflare/IngressConfiguration'

interface CloudflareTabProps {
    appId: string;
}

type TunnelTab = 'overview' | 'ingress'

function CloudflareTab({ appId }: CloudflareTabProps) {
    const { data: tunnel, isLoading, error, refetch } = useCloudflareTunnel(appId)
    const syncTunnel = useSyncCloudflareTunnel()
    const deleteTunnel = useDeleteCloudflareTunnel()
    const { toast } = useToast()

    const [activeTab, setActiveTab] = useState<TunnelTab>('overview')
    const [showDeleteDialog, setShowDeleteDialog] = useState(false)

    const handleSync = () => {
        syncTunnel.mutate(appId, {
            onSuccess: () => {
                toast.success('Tunnel Synced', 'Cloudflare tunnel configuration synced successfully')
                refetch()
            },
            onError: (error) => {
                toast.error('Sync Failed', error.message)
            }
        })
    }

    const handleDelete = () => {
        setShowDeleteDialog(true)
    }

    const confirmDelete = () => {
        deleteTunnel.mutate(appId, {
            onSuccess: () => {
                toast.success('Tunnel Deleted', 'Cloudflare tunnel has been removed')
                setShowDeleteDialog(false)
            },
            onError: (error) => {
                toast.error('Delete Failed', error.message)
            }
        })
    }

    const copyToClipboard = (text: string, label: string) => {
        navigator.clipboard.writeText(text)
        toast.success('Copied', `${label} copied to clipboard`)
    }

    const getHealthBadge = (status: string) => {
        if (status === 'active') {
            return (
                <Badge variant="default" className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                    <CheckCircle2 className="h-3 w-3 mr-1" />
                    Healthy
                </Badge>
            )
        }
        if (status === 'inactive') {
            return (
                <Badge variant="secondary" className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200">
                    <Clock className="h-3 w-3 mr-1" />
                    Inactive
                </Badge>
            )
        }
        if (status === 'error') {
            return (
                <Badge variant="destructive" className="bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200">
                    <AlertCircle className="h-3 w-3 mr-1" />
                    Error
                </Badge>
            )
        }
        return (
            <Badge variant="secondary">
                <Server className="h-3 w-3 mr-1" />
                Unknown
            </Badge>
        )
    }

    if (isLoading) {
        return (
            <div className="space-y-6 fade-in">
                <Card className="min-h-[400px] flex items-center justify-center">
                    <CardContent>
                        <div className="flex flex-col items-center gap-4">
                            <div className="h-12 w-12 border-4 border-primary border-t-transparent rounded-full animate-spin" />
                            <p className="text-muted-foreground">Loading tunnel information...</p>
                        </div>
                    </CardContent>
                </Card>
            </div>
        )
    }

    if (error) {
        return (
            <div className="space-y-6 fade-in">
                <Card>
                    <CardContent className="flex flex-col items-center justify-center py-12">
                        <AlertCircle className="h-12 w-12 text-red-500 mb-4" />
                        <h2 className="text-xl font-semibold mb-2">Failed to load tunnel</h2>
                        <p className="text-muted-foreground mb-4">{error.message}</p>
                        <Button onClick={() => refetch()} className="button-press">
                            <RefreshCw className="h-4 w-4 mr-2" />
                            Retry
                        </Button>
                    </CardContent>
                </Card>
            </div>
        )
    }

    if (!tunnel) {
        return (
            <div className="space-y-6 fade-in">
                <Card>
                    <CardContent className="flex flex-col items-center justify-center py-12">
                        <Globe className="h-12 w-12 text-muted-foreground mb-4" />
                        <h2 className="text-xl font-semibold mb-2">No Tunnel Configured</h2>
                        <p className="text-muted-foreground mb-4 max-w-md mx-auto">
                            This app doesn't have a Cloudflare tunnel configured yet.
                        </p>
                        <div className="bg-blue-50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-900/30 rounded-lg p-4 max-w-md mx-auto">
                            <h4 className="font-medium text-sm mb-2 flex items-center gap-2">
                                <Shield className="h-4 w-4 text-blue-500" />
                                How to enable Cloudflare tunnel
                            </h4>
                            <ul className="text-sm text-muted-foreground space-y-1 list-disc list-inside">
                                <li>Cloudflare tunnels provide secure, encrypted access to your apps</li>
                                <li>Configure tunnel settings in app deployment</li>
                                <li>Your app will be accessible via a public Cloudflare URL</li>
                                <li>No need to open ports on your firewall</li>
                            </ul>
                        </div>
                    </CardContent>
                </Card>
            </div>
        )
    }

    const tabs = [
        { id: 'overview' as TunnelTab, label: 'Overview', icon: Globe },
        { id: 'ingress' as TunnelTab, label: 'Ingress Rules', icon: Shield },
    ]

    return (
        <div className="space-y-6 fade-in">
            {/* Tabs */}
            <div className="flex border-b mb-6">
                {tabs.map((tab) => {
                    const Icon = tab.icon
                    return (
                        <button
                            key={tab.id}
                            className={`flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors interactive-element ${activeTab === tab.id
                                ? 'border-primary text-primary'
                                : 'border-transparent text-muted-foreground hover:text-foreground'
                                }`}
                            onClick={() => setActiveTab(tab.id)}
                        >
                            <Icon className="h-4 w-4" />
                            {tab.label}
                        </button>
                    )
                })}
            </div>

            {/* Overview Tab */}
            {activeTab === 'overview' && (
                <div className="space-y-6">
                    {/* Status Card */}
                    <Card>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2">
                                <Globe className="h-5 w-5 text-primary" />
                                <div className="flex items-center gap-2">
                                    Tunnel Status
                                    {getHealthBadge(tunnel.status)}
                                </div>
                            </CardTitle>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            {/* Public URL with inline copy */}
                            <div>
                                <h4 className="text-sm font-medium text-muted-foreground mb-3">Public URL</h4>
                                {tunnel.public_url ? (
                                    <div className="inline-flex items-center gap-2">
                                        <a
                                            href={tunnel.public_url}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            className="text-primary hover:underline font-medium text-sm"
                                        >
                                            {tunnel.public_url}
                                        </a>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => copyToClipboard(tunnel.public_url, 'URL')}
                                            className="h-6 w-6 p-0 hover:bg-muted transition-colors flex items-center justify-center"
                                            title="Copy URL"
                                        >
                                            <Copy className="h-3 w-3 text-muted-foreground" />
                                        </Button>
                                    </div>
                                ) : (
                                    <p className="text-sm text-muted-foreground">No public URL configured</p>
                                )}
                            </div>

                            {/* Tunnel Details */}
                            <div className="pt-4 border-t">
                                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                                    <div>
                                        <p className="text-muted-foreground">Tunnel Name</p>
                                        <p className="font-medium mt-1 truncate">{tunnel.tunnel_name}</p>
                                    </div>
                                    <div>
                                        <p className="text-muted-foreground">Tunnel ID</p>
                                        <p className="font-mono font-medium mt-1 truncate text-xs">{tunnel.tunnel_id}</p>
                                    </div>
                                    <div>
                                        <p className="text-muted-foreground">Created</p>
                                        <p className="font-medium mt-1">{new Date(tunnel.created_at).toLocaleDateString()}</p>
                                    </div>
                                    <div>
                                        <p className="text-muted-foreground">Last Synced</p>
                                        <p className="font-medium mt-1">
                                            {tunnel.last_synced_at
                                                ? new Date(tunnel.last_synced_at).toLocaleDateString()
                                                : 'Never'
                                            }
                                        </p>
                                    </div>
                                </div>
                            </div>
                        </CardContent>
                    </Card>

                    {/* Action Cards */}
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        <Card className="card-hover">
                            <CardContent className="p-6 text-center">
                                <RefreshCw className="h-8 w-8 text-primary mx-auto mb-3" />
                                <h4 className="font-medium mb-2">Sync Tunnel</h4>
                                <p className="text-sm text-muted-foreground mb-4">
                                    Refresh tunnel configuration from Cloudflare
                                </p>
                                <Button
                                    onClick={handleSync}
                                    disabled={syncTunnel.isPending}
                                    className="w-full button-press"
                                >
                                    {syncTunnel.isPending ? (
                                        <>
                                            <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin mr-2" />
                                            Syncing...
                                        </>
                                    ) : (
                                        'Sync Now'
                                    )}
                                </Button>
                            </CardContent>
                        </Card>

                        <Card className="card-hover">
                            <CardContent className="p-6 text-center">
                                <Shield className="h-8 w-8 text-primary mx-auto mb-3" />
                                <h4 className="font-medium mb-2">Configure Ingress</h4>
                                <p className="text-sm text-muted-foreground mb-4">
                                    Manage ingress rules for traffic routing
                                </p>
                                <Button
                                    variant="outline"
                                    onClick={() => setActiveTab('ingress')}
                                    className="w-full button-press"
                                >
                                    Configure
                                    <ArrowRight className="h-4 w-4 ml-2" />
                                </Button>
                            </CardContent>
                        </Card>

                        <Card className="card-hover border-destructive/50 hover:border-destructive">
                            <CardContent className="p-6 text-center">
                                <Trash2 className="h-8 w-8 text-destructive mx-auto mb-3" />
                                <h4 className="font-medium mb-2 text-destructive">Delete Tunnel</h4>
                                <p className="text-sm text-muted-foreground mb-4">
                                    Remove tunnel and its configuration
                                </p>
                                <Button
                                    variant="outline"
                                    onClick={handleDelete}
                                    disabled={deleteTunnel.isPending}
                                    className="w-full text-destructive hover:text-destructive button-press"
                                >
                                    {deleteTunnel.isPending ? (
                                        <>
                                            <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin mr-2" />
                                            Deleting...
                                        </>
                                    ) : (
                                        'Delete'
                                    )}
                                </Button>
                            </CardContent>
                        </Card>
                    </div>
                </div>
            )}

            {/* Ingress Rules Tab */}
            {activeTab === 'ingress' && (
                <IngressConfiguration
                    appId={appId}
                    existingIngress={tunnel.ingress_rules || []}
                    existingHostname={tunnel.public_url?.replace(/^https?:\/\//, '').split('/')[0] || ''}
                    tunnelID={tunnel.tunnel_id}
                    onSave={() => {
                        refetch()
                        toast.success('Ingress Updated', 'Ingress configuration saved successfully')
                    }}
                />
            )}

            {/* Delete Confirmation Dialog */}
            <ConfirmationDialog
                open={showDeleteDialog}
                onOpenChange={(open) => !open && setShowDeleteDialog(false)}
                title="Delete Cloudflare Tunnel"
                description="Are you sure you want to delete this Cloudflare tunnel? This will remove the tunnel configuration and your app will no longer be accessible via its public URL. This action cannot be undone."
                confirmText="Delete Tunnel"
                cancelText="Cancel"
                onConfirm={confirmDelete}
                isLoading={deleteTunnel.isPending}
                variant="destructive"
            />
        </div>
    )
}

export default CloudflareTab
