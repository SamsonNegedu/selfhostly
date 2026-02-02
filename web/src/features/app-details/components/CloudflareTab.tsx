import { useState } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Badge } from '@/shared/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/shared/components/ui/dialog'
import { Input } from '@/shared/components/ui/input'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { RefreshCw, AlertCircle, CheckCircle2, Copy, Globe, Shield, ArrowRight, Clock, Trash2, Server, PlusCircle } from 'lucide-react'
import { useTunnel, useSyncTunnel, useDeleteTunnel, useCreateTunnelForApp, useCreateQuickTunnelForApp, useSwitchAppToCustomTunnel } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import { IngressConfiguration } from '@/features/cloudflare/IngressConfiguration'
import type { IngressRule, TunnelByAppResponse } from '@/shared/types/api'

interface CloudflareTabProps {
    appId: string;
    nodeId: string;
}

type TunnelTab = 'overview' | 'ingress'

function isNoTunnelResponse(r: unknown): r is TunnelByAppResponse & { tunnel: null } {
    return !!r && typeof r === 'object' && 'tunnel' in r && (r as { tunnel: unknown }).tunnel === null
}

function CloudflareTab({ appId, nodeId }: CloudflareTabProps) {
    const { data: tunnel, isLoading, error, refetch } = useTunnel(appId, nodeId)
    const syncTunnel = useSyncTunnel()
    const deleteTunnel = useDeleteTunnel()
    const createTunnel = useCreateTunnelForApp()
    const createQuickTunnel = useCreateQuickTunnelForApp()
    const switchToCustom = useSwitchAppToCustomTunnel()
    const { toast } = useToast()

    const [activeTab, setActiveTab] = useState<TunnelTab>('overview')
    const [showDeleteDialog, setShowDeleteDialog] = useState(false)
    const [showSwitchDialog, setShowSwitchDialog] = useState(false)
    const [switchIngressRules, setSwitchIngressRules] = useState<IngressRule[]>([{ service: '', hostname: null, path: null }])
    const [switchFormError, setSwitchFormError] = useState<string | null>(null)
    const [showCreateDialog, setShowCreateDialog] = useState(false)
    const [createIngressRules, setCreateIngressRules] = useState<IngressRule[]>([{ service: '', hostname: null, path: null }])
    const [createFormError, setCreateFormError] = useState<string | null>(null)
    const [showQuickTunnelDialog, setShowQuickTunnelDialog] = useState(false)
    const [quickTunnelService, setQuickTunnelService] = useState('')
    const [quickTunnelPort, setQuickTunnelPort] = useState<number>(80)
    const [quickTunnelFormError, setQuickTunnelFormError] = useState<string | null>(null)

    const handleSync = () => {
        syncTunnel.mutate({ appId, nodeId }, {
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
        deleteTunnel.mutate({ appId, nodeId }, {
            onSuccess: () => {
                // Don't show success toast here - let AppActions show job progress
                // The job completion handler will show success/error toast
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
        // Cloudflare API returns healthy/degraded/inactive/down; we normalize to active/inactive/error
        const isActive = status === 'active' || status === 'healthy' || status === 'degraded'
        if (isActive) {
            return (
                <Badge variant="default" className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                    <CheckCircle2 className="h-3 w-3 mr-1" />
                    Healthy
                </Badge>
            )
        }
        if (status === 'inactive' || status === 'down') {
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

    if (isNoTunnelResponse(tunnel)) {
        const isQuick = tunnel.tunnel_mode === 'quick'
        return (
            <div className="space-y-6 fade-in">
                <Card>
                    <CardContent className="flex flex-col items-center justify-center py-12">
                        <Globe className="h-12 w-12 text-muted-foreground mb-4" />
                        <h2 className="text-xl font-semibold mb-2">
                            {isQuick ? 'Quick Tunnel' : 'No Tunnel Configured'}
                        </h2>
                        {isQuick && tunnel.public_url ? (
                            <>
                                <p className="text-muted-foreground mb-2 max-w-md mx-auto">
                                    This app uses a temporary Quick Tunnel URL.
                                </p>
                                <a
                                    href={tunnel.public_url}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-primary font-medium mb-4 inline-flex items-center gap-1"
                                >
                                    {tunnel.public_url}
                                    <ArrowRight className="h-4 w-4" />
                                </a>
                                <p className="text-sm text-muted-foreground mb-4 max-w-md mx-auto">
                                    If the tunnel stops working, you can recreate it. Or switch to a custom domain for a stable URL.
                                </p>
                                <div className="flex flex-wrap gap-3 justify-center">
                                    <Button
                                        onClick={() => {
                                            setQuickTunnelService('')
                                            setQuickTunnelPort(80)
                                            setQuickTunnelFormError(null)
                                            setShowQuickTunnelDialog(true)
                                        }}
                                        variant="outline"
                                        className="button-press"
                                    >
                                        <RefreshCw className="h-4 w-4 mr-2" />
                                        Recreate Quick Tunnel
                                    </Button>
                                    <Button
                                        onClick={() => {
                                            setSwitchIngressRules([{ service: '', hostname: null, path: null }])
                                            setSwitchFormError(null)
                                            setShowSwitchDialog(true)
                                        }}
                                        className="button-press"
                                    >
                                        Switch to custom domain
                                    </Button>
                                </div>
                                <Dialog open={showSwitchDialog} onOpenChange={(open) => { setShowSwitchDialog(open); if (!open) setSwitchFormError(null) }}>
                                    <DialogContent className="max-w-md">
                                        <DialogHeader>
                                            <DialogTitle>Switch to custom domain</DialogTitle>
                                            <DialogDescription>
                                                Add at least one ingress rule with your custom hostname (e.g. app.example.com) and the service URL (e.g. http://vert:90 or http://web:80). This will replace the temporary Quick Tunnel URL.
                                            </DialogDescription>
                                        </DialogHeader>
                                        <div className="space-y-4 py-2">
                                            {switchFormError && (
                                                <p className="text-sm text-destructive">{switchFormError}</p>
                                            )}
                                            {switchIngressRules.map((rule, index) => (
                                                <div key={index} className="space-y-2 p-3 border rounded-lg">
                                                    <div className="grid grid-cols-1 gap-2">
                                                        <div>
                                                            <label className="text-xs font-medium text-muted-foreground">Hostname (e.g. app.example.com)</label>
                                                            <Input
                                                                value={rule.hostname ?? ''}
                                                                onChange={(e) => {
                                                                    const next = [...switchIngressRules]
                                                                    next[index] = { ...next[index], hostname: e.target.value.trim() || null }
                                                                    setSwitchIngressRules(next)
                                                                }}
                                                                placeholder="app.example.com"
                                                                className="mt-1"
                                                            />
                                                        </div>
                                                        <div>
                                                            <label className="text-xs font-medium text-muted-foreground">Service URL</label>
                                                            <Input
                                                                value={rule.service}
                                                                onChange={(e) => {
                                                                    const next = [...switchIngressRules]
                                                                    next[index] = { ...next[index], service: e.target.value.trim() }
                                                                    setSwitchIngressRules(next)
                                                                }}
                                                                placeholder="http://vert:90"
                                                                className="mt-1"
                                                            />
                                                            <p className="text-xs text-muted-foreground mt-1">Full URL to your service, e.g. http://vert:90 or http://web:80</p>
                                                        </div>
                                                        <div>
                                                            <label className="text-xs font-medium text-muted-foreground">Path (optional)</label>
                                                            <Input
                                                                value={rule.path ?? ''}
                                                                onChange={(e) => {
                                                                    const next = [...switchIngressRules]
                                                                    next[index] = { ...next[index], path: e.target.value.trim() || null }
                                                                    setSwitchIngressRules(next)
                                                                }}
                                                                placeholder="/"
                                                                className="mt-1"
                                                            />
                                                        </div>
                                                    </div>
                                                    {switchIngressRules.length > 1 && (
                                                        <Button type="button" variant="ghost" size="sm" className="text-destructive" onClick={() => setSwitchIngressRules(switchIngressRules.filter((_, i) => i !== index))}>
                                                            Remove rule
                                                        </Button>
                                                    )}
                                                </div>
                                            ))}
                                            <Button type="button" variant="outline" size="sm" onClick={() => setSwitchIngressRules([...switchIngressRules, { service: '', hostname: null, path: null }])}>
                                                <PlusCircle className="h-3 w-3 mr-1" />
                                                Add rule
                                            </Button>
                                        </div>
                                        <DialogFooter>
                                            <Button variant="outline" onClick={() => setShowSwitchDialog(false)}>Cancel</Button>
                                            <Button
                                                disabled={switchToCustom.isPending}
                                                onClick={() => {
                                                    const valid = switchIngressRules.filter(r => r.service.trim() !== '')
                                                    if (valid.length === 0) {
                                                        setSwitchFormError('At least one rule with a service URL is required.')
                                                        return
                                                    }
                                                    const withHostname = valid.filter(r => r.hostname && r.hostname.trim() !== '')
                                                    if (withHostname.length === 0) {
                                                        setSwitchFormError('At least one rule needs a hostname (your custom domain) so the public URL is not a placeholder.')
                                                        return
                                                    }
                                                    setSwitchFormError(null)
                                                    switchToCustom.mutate(
                                                        { appId, nodeId, ingressRules: valid.map(r => ({ hostname: r.hostname || undefined, service: r.service.trim(), path: r.path || undefined })) },
                                                        {
                                                            onSuccess: () => {
                                                                setShowSwitchDialog(false)
                                                                const hasIngressRules = valid.length > 0
                                                                if (hasIngressRules) {
                                                                    toast.info('Switching to custom tunnel...', 'Creating tunnel and applying your custom domain rules.')
                                                                } else {
                                                                    toast.info('Switching to custom tunnel...', 'Creating custom tunnel in the background.')
                                                                }
                                                                // App will refresh automatically when job completes via AppActions
                                                            },
                                                            onError: (e) => setSwitchFormError(e.message),
                                                        }
                                                    )
                                                }}
                                            >
                                                {switchToCustom.isPending ? 'Switching...' : 'Switch to custom domain'}
                                            </Button>
                                        </DialogFooter>
                                    </DialogContent>
                                </Dialog>
                                <Dialog open={showQuickTunnelDialog} onOpenChange={(open) => { setShowQuickTunnelDialog(open); if (!open) setQuickTunnelFormError(null) }}>
                                    <DialogContent className="max-w-md">
                                        <DialogHeader>
                                            <DialogTitle>Recreate Quick Tunnel</DialogTitle>
                                            <DialogDescription>
                                                Recreate the Quick Tunnel with a new temporary trycloudflare.com URL. Enter the compose service name and port that serves the app (e.g. web and 80).
                                            </DialogDescription>
                                        </DialogHeader>
                                        <div className="space-y-4 py-2">
                                            {quickTunnelFormError && (
                                                <p className="text-sm text-destructive">{quickTunnelFormError}</p>
                                            )}
                                            <div>
                                                <label className="text-xs font-medium text-muted-foreground">Service name (compose service)</label>
                                                <Input
                                                    value={quickTunnelService}
                                                    onChange={(e) => setQuickTunnelService(e.target.value.trim())}
                                                    placeholder="web"
                                                    className="mt-1"
                                                />
                                            </div>
                                            <div>
                                                <label className="text-xs font-medium text-muted-foreground">Port</label>
                                                <Input
                                                    type="number"
                                                    min={1}
                                                    max={65535}
                                                    value={quickTunnelPort}
                                                    onChange={(e) => setQuickTunnelPort(parseInt(e.target.value, 10) || 80)}
                                                    className="mt-1"
                                                />
                                            </div>
                                        </div>
                                        <DialogFooter>
                                            <Button variant="outline" onClick={() => setShowQuickTunnelDialog(false)}>Cancel</Button>
                                            <Button
                                                disabled={createQuickTunnel.isPending}
                                                onClick={() => {
                                                    if (!quickTunnelService.trim()) {
                                                        setQuickTunnelFormError('Service name is required.')
                                                        return
                                                    }
                                                    const port = Number(quickTunnelPort)
                                                    if (port < 1 || port > 65535) {
                                                        setQuickTunnelFormError('Port must be between 1 and 65535.')
                                                        return
                                                    }
                                                    setQuickTunnelFormError(null)
                                                    createQuickTunnel.mutate(
                                                        { appId: appId, nodeId: nodeId, service: quickTunnelService.trim(), port },
                                                        {
                                                            onSuccess: () => {
                                                                setShowQuickTunnelDialog(false)
                                                                toast.info('Quick Tunnel creation started', 'Setting up your temporary URL in the background...')
                                                                // App will refresh automatically when job completes via AppActions
                                                            },
                                                            onError: (e: Error) => setQuickTunnelFormError(e.message),
                                                        }
                                                    )
                                                }}
                                            >
                                                {createQuickTunnel.isPending ? 'Recreating...' : 'Recreate Quick Tunnel'}
                                            </Button>
                                        </DialogFooter>
                                    </DialogContent>
                                </Dialog>
                            </>
                        ) : (
                            <>
                                <p className="text-muted-foreground mb-4 max-w-md mx-auto text-center">
                                    This app doesn't have a Cloudflare tunnel yet. Create a temporary Quick Tunnel URL or a named tunnel with your custom domain.
                                </p>
                                <div className="flex flex-wrap gap-3 justify-center">
                                    <Button
                                        onClick={() => {
                                            setQuickTunnelService('')
                                            setQuickTunnelPort(80)
                                            setQuickTunnelFormError(null)
                                            setShowQuickTunnelDialog(true)
                                        }}
                                        variant="outline"
                                        className="button-press"
                                    >
                                        <Globe className="h-4 w-4 mr-2" />
                                        Create Quick Tunnel
                                    </Button>
                                    <Button
                                        onClick={() => {
                                            setCreateIngressRules([{ service: '', hostname: null, path: null }])
                                            setCreateFormError(null)
                                            setShowCreateDialog(true)
                                        }}
                                        className="button-press"
                                    >
                                        <PlusCircle className="h-4 w-4 mr-2" />
                                        Create custom domain tunnel
                                    </Button>
                                </div>
                                <Dialog open={showQuickTunnelDialog} onOpenChange={(open) => { setShowQuickTunnelDialog(open); if (!open) setQuickTunnelFormError(null) }}>
                                    <DialogContent className="max-w-md">
                                        <DialogHeader>
                                            <DialogTitle>Create Quick Tunnel</DialogTitle>
                                            <DialogDescription>
                                                Expose this app with a temporary trycloudflare.com URL. Enter the compose service name and port that serves the app (e.g. web and 80).
                                            </DialogDescription>
                                        </DialogHeader>
                                        <div className="space-y-4 py-2">
                                            {quickTunnelFormError && (
                                                <p className="text-sm text-destructive">{quickTunnelFormError}</p>
                                            )}
                                            <div>
                                                <label className="text-xs font-medium text-muted-foreground">Service name (compose service)</label>
                                                <Input
                                                    value={quickTunnelService}
                                                    onChange={(e) => setQuickTunnelService(e.target.value.trim())}
                                                    placeholder="web"
                                                    className="mt-1"
                                                />
                                            </div>
                                            <div>
                                                <label className="text-xs font-medium text-muted-foreground">Port</label>
                                                <Input
                                                    type="number"
                                                    min={1}
                                                    max={65535}
                                                    value={quickTunnelPort}
                                                    onChange={(e) => setQuickTunnelPort(parseInt(e.target.value, 10) || 80)}
                                                    className="mt-1"
                                                />
                                            </div>
                                        </div>
                                        <DialogFooter>
                                            <Button variant="outline" onClick={() => setShowQuickTunnelDialog(false)}>Cancel</Button>
                                            <Button
                                                disabled={createQuickTunnel.isPending}
                                                onClick={() => {
                                                    if (!quickTunnelService.trim()) {
                                                        setQuickTunnelFormError('Service name is required.')
                                                        return
                                                    }
                                                    const port = Number(quickTunnelPort)
                                                    if (port < 1 || port > 65535) {
                                                        setQuickTunnelFormError('Port must be between 1 and 65535.')
                                                        return
                                                    }
                                                    setQuickTunnelFormError(null)
                                                    createQuickTunnel.mutate(
                                                        { appId: appId, nodeId: nodeId, service: quickTunnelService.trim(), port },
                                                        {
                                                            onSuccess: () => {
                                                                setShowQuickTunnelDialog(false)
                                                                toast.info('Quick Tunnel creation started', 'Setting up your temporary URL in the background...')
                                                                // App will refresh automatically when job completes via AppActions
                                                            },
                                                            onError: (e: Error) => setQuickTunnelFormError(e.message),
                                                        }
                                                    )
                                                }}
                                            >
                                                {createQuickTunnel.isPending ? 'Creating...' : 'Create Quick Tunnel'}
                                            </Button>
                                        </DialogFooter>
                                    </DialogContent>
                                </Dialog>
                                <Dialog open={showCreateDialog} onOpenChange={(open) => { setShowCreateDialog(open); if (!open) setCreateFormError(null) }}>
                                    <DialogContent className="max-w-md">
                                        <DialogHeader>
                                            <DialogTitle>Create custom domain tunnel</DialogTitle>
                                            <DialogDescription>
                                                Add at least one ingress rule with your custom hostname (e.g. app.example.com) and the service URL (e.g. http://vert:90 or http://web:80). The tunnel will be created with these rules so the public URL is your domain, not a placeholder.
                                            </DialogDescription>
                                        </DialogHeader>
                                        <div className="space-y-4 py-2">
                                            {createFormError && (
                                                <p className="text-sm text-destructive">{createFormError}</p>
                                            )}
                                            {createIngressRules.map((rule, index) => (
                                                <div key={index} className="space-y-2 p-3 border rounded-lg">
                                                    <div className="grid grid-cols-1 gap-2">
                                                        <div>
                                                            <label className="text-xs font-medium text-muted-foreground">Hostname (e.g. app.example.com)</label>
                                                            <Input
                                                                value={rule.hostname ?? ''}
                                                                onChange={(e) => {
                                                                    const next = [...createIngressRules]
                                                                    next[index] = { ...next[index], hostname: e.target.value.trim() || null }
                                                                    setCreateIngressRules(next)
                                                                }}
                                                                placeholder="app.example.com"
                                                                className="mt-1"
                                                            />
                                                        </div>
                                                        <div>
                                                            <label className="text-xs font-medium text-muted-foreground">Service URL</label>
                                                            <Input
                                                                value={rule.service}
                                                                onChange={(e) => {
                                                                    const next = [...createIngressRules]
                                                                    next[index] = { ...next[index], service: e.target.value.trim() }
                                                                    setCreateIngressRules(next)
                                                                }}
                                                                placeholder="http://vert:90"
                                                                className="mt-1"
                                                            />
                                                            <p className="text-xs text-muted-foreground mt-1">Full URL to your service, e.g. http://vert:90 or http://web:80</p>
                                                        </div>
                                                        <div>
                                                            <label className="text-xs font-medium text-muted-foreground">Path (optional)</label>
                                                            <Input
                                                                value={rule.path ?? ''}
                                                                onChange={(e) => {
                                                                    const next = [...createIngressRules]
                                                                    next[index] = { ...next[index], path: e.target.value.trim() || null }
                                                                    setCreateIngressRules(next)
                                                                }}
                                                                placeholder="/"
                                                                className="mt-1"
                                                            />
                                                        </div>
                                                    </div>
                                                    {createIngressRules.length > 1 && (
                                                        <Button type="button" variant="ghost" size="sm" className="text-destructive" onClick={() => setCreateIngressRules(createIngressRules.filter((_, i) => i !== index))}>
                                                            Remove rule
                                                        </Button>
                                                    )}
                                                </div>
                                            ))}
                                            <Button type="button" variant="outline" size="sm" onClick={() => setCreateIngressRules([...createIngressRules, { service: '', hostname: null, path: null }])}>
                                                <PlusCircle className="h-3 w-3 mr-1" />
                                                Add rule
                                            </Button>
                                        </div>
                                        <DialogFooter>
                                            <Button variant="outline" onClick={() => setShowCreateDialog(false)}>Cancel</Button>
                                            <Button
                                                disabled={createTunnel.isPending}
                                                onClick={() => {
                                                    const valid = createIngressRules.filter(r => r.service.trim() !== '')
                                                    if (valid.length === 0) {
                                                        setCreateFormError('At least one rule with a service URL is required.')
                                                        return
                                                    }
                                                    const withHostname = valid.filter(r => r.hostname && r.hostname.trim() !== '')
                                                    if (withHostname.length === 0) {
                                                        setCreateFormError('At least one rule needs a hostname (your custom domain) so the public URL is not a placeholder.')
                                                        return
                                                    }
                                                    setCreateFormError(null)
                                                    createTunnel.mutate(
                                                        { appId, nodeId, ingressRules: valid.map(r => ({ hostname: r.hostname || undefined, service: r.service.trim(), path: r.path || undefined })) },
                                                        {
                                                            onSuccess: () => {
                                                                setShowCreateDialog(false)
                                                                toast.info('Tunnel creation started', 'Setting up your custom domain tunnel in the background...')
                                                                // App will refresh automatically when job completes via AppActions
                                                            },
                                                            onError: (e) => setCreateFormError(e.message),
                                                        }
                                                    )
                                                }}
                                            >
                                                {createTunnel.isPending ? 'Creating...' : 'Create custom domain tunnel'}
                                            </Button>
                                        </DialogFooter>
                                    </DialogContent>
                                </Dialog>
                            </>
                        )}
                    </CardContent>
                </Card>
            </div>
        )
    }

    const tunnelData = tunnel.tunnel!
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
                                    {getHealthBadge(tunnelData.status)}
                                </div>
                            </CardTitle>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            {/* Public URL with inline copy */}
                            <div>
                                <h4 className="text-sm font-medium text-muted-foreground mb-3">Public URL</h4>
                                {tunnelData.public_url ? (
                                    <div className="inline-flex items-center gap-2">
                                        <a
                                            href={tunnelData.public_url}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            className="text-primary hover:underline font-medium text-sm"
                                        >
                                            {tunnelData.public_url}
                                        </a>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => copyToClipboard(tunnelData.public_url, 'URL')}
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
                                        <p className="font-medium mt-1 truncate">{tunnelData.tunnel_name}</p>
                                    </div>
                                    <div>
                                        <p className="text-muted-foreground">Tunnel ID</p>
                                        <p className="font-mono font-medium mt-1 truncate text-xs">{tunnelData.tunnel_id}</p>
                                    </div>
                                    <div>
                                        <p className="text-muted-foreground">Created</p>
                                        <p className="font-medium mt-1">{new Date(tunnelData.created_at).toLocaleDateString()}</p>
                                    </div>
                                    <div>
                                        <p className="text-muted-foreground">Last Synced</p>
                                        <p className="font-medium mt-1">
                                            {tunnelData.last_synced_at
                                                ? new Date(tunnelData.last_synced_at).toLocaleDateString()
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
                    nodeId={nodeId}
                    existingIngress={tunnelData.ingress_rules || []}
                    existingHostname={tunnelData.public_url?.replace(/^https?:\/\//, '').split('/')[0] || ''}
                    tunnelID={tunnelData.tunnel_id}
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
