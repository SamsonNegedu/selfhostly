import { useState } from 'react'
import { AlertTriangle, Copy, Globe, Loader2, RefreshCw, Trash2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { Tabs, TabsList, TabsTrigger } from '@/shared/components/ui/Tabs'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { useDeleteTunnel, useSyncTunnel, useTunnel } from '@/shared/services/api'
import { IngressConfiguration } from '@/features/cloudflare/IngressConfiguration'
import type { TunnelByAppResponse } from '@/shared/types/api'
import { CustomDomainDialog, QuickTunnelDialog } from './TunnelDialogs'

interface CloudflareTabProps {
    appId: string
    nodeId: string
    composeContent?: string
}

type TunnelTab = 'overview' | 'ingress'
type Dialog = 'quick' | 'quick-again' | 'custom' | 'switch' | null

function isNoTunnelResponse(r: unknown): r is TunnelByAppResponse & { tunnel: null } {
    return !!r && typeof r === 'object' && 'tunnel' in r && (r as { tunnel: unknown }).tunnel === null
}

// Cloudflare reports healthy, degraded, inactive or down. Degraded still serves traffic.
function HealthPill({ status }: { status: string }) {
    if (status === 'active' || status === 'healthy') return <StatusPill kind="ok">Healthy</StatusPill>
    if (status === 'degraded') return <StatusPill kind="warn">Degraded</StatusPill>
    if (status === 'inactive' || status === 'down') return <StatusPill kind="idle">Inactive</StatusPill>
    if (status === 'error') return <StatusPill kind="err">Error</StatusPill>
    return <StatusPill kind="idle">Unknown</StatusPill>
}

const formatDate = (iso?: string) => (iso ? new Date(iso).toLocaleDateString() : 'Never')

// How the app is reached from outside: no tunnel, a temporary Quick Tunnel, or a named tunnel on your own domain.
function CloudflareTab({ appId, nodeId, composeContent }: CloudflareTabProps) {
    const { data: tunnel, isLoading, error, refetch } = useTunnel(appId, nodeId)
    const syncTunnel = useSyncTunnel()
    const deleteTunnel = useDeleteTunnel()
    const { toast } = useToast()
    const [tab, setTab] = useState<TunnelTab>('overview')
    const [dialog, setDialog] = useState<Dialog>(null)
    const [confirmDelete, setConfirmDelete] = useState(false)

    const dialogProps = (name: Exclude<Dialog, null>) => ({ open: dialog === name, onOpenChange: (open: boolean) => setDialog(open ? name : null), appId, nodeId, composeContent })
    const dialogs = (
        <>
            <QuickTunnelDialog {...dialogProps('quick')} />
            <QuickTunnelDialog {...dialogProps('quick-again')} recreate />
            <CustomDomainDialog {...dialogProps('custom')} />
            <CustomDomainDialog {...dialogProps('switch')} switching />
        </>
    )

    const copy = async (text: string) => {
        try {
            await navigator.clipboard.writeText(text)
            toast.success('Copied', 'The address is on your clipboard')
        } catch {
            toast.error('Could not copy', 'Your browser blocked access to the clipboard')
        }
    }

    if (isLoading) {
        return (
            <div role="status" aria-label="Loading access" className="flex flex-col gap-4">
                <Skeleton className="h-40 w-full rounded-xl" />
                <Skeleton className="h-20 w-full rounded-xl" />
            </div>
        )
    }

    if (error) return <ErrorState title="Could not load access" error={error} onRetry={() => refetch()} />

    if (!tunnel) {
        return <EmptyState icon={<Globe className="h-5 w-5" />} title="Only reachable on your network" description="This app has no public address. Tunnels give it one without opening ports on your router." className="py-14" />
    }

    if (isNoTunnelResponse(tunnel)) {
        const quick = tunnel.tunnel_mode === 'quick' && !!tunnel.public_url
        return (
            <div className="flex flex-col gap-5">
                {quick ? (
                    <>
                        <Card>
                            <CardHeader>
                                <CardTitle className="flex flex-wrap items-center gap-2.5 text-base">
                                    <Globe className="h-4 w-4 text-muted-foreground" />
                                    Quick Tunnel
                                    <StatusPill kind="warn">Temporary</StatusPill>
                                </CardTitle>
                            </CardHeader>
                            <CardContent className="flex flex-col gap-3">
                                <a href={tunnel.public_url} target="_blank" rel="noopener noreferrer" className="inline-flex min-h-[44px] items-center break-all font-mono text-sm font-medium text-status-info-fg hover:underline md:min-h-0">
                                    {tunnel.public_url}
                                </a>
                                <div className="flex items-start gap-2 rounded-lg bg-status-warn-bg px-3 py-2.5 text-[13px] text-status-warn-fg">
                                    <AlertTriangle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0" />
                                    <p>The address can change when the app restarts, it is limited to 200 requests at a time, and it does not support server-sent events. Use your own domain for anything you rely on.</p>
                                </div>
                                <div className="flex flex-wrap gap-2">
                                    <Button onClick={() => setDialog('switch')}>Switch to my own domain</Button>
                                    <Button variant="outline" onClick={() => setDialog('quick-again')}>
                                        <RefreshCw className="h-4 w-4" />
                                        Get a new address
                                    </Button>
                                </div>
                            </CardContent>
                        </Card>
                    </>
                ) : (
                    <EmptyState
                        icon={<Globe className="h-5 w-5" />}
                        title="No public address yet"
                        description="Create a temporary Quick Tunnel to try it out, or set up a stable address on your own domain."
                        action={
                            <div className="flex flex-wrap justify-center gap-2">
                                <Button onClick={() => setDialog('custom')}>Use my own domain</Button>
                                <Button variant="outline" onClick={() => setDialog('quick')}>Create a Quick Tunnel</Button>
                            </div>
                        }
                        className="py-14"
                    />
                )}
                {dialogs}
            </div>
        )
    }

    const data = tunnel.tunnel!
    const sync = () =>
        syncTunnel.mutate(
            { appId, nodeId },
            {
                onSuccess: () => {
                    toast.success('Tunnel synced', 'Refreshed from Cloudflare')
                    refetch()
                },
                onError: (failure) => toast.error('Could not sync', describeError(failure)),
            }
        )
    const remove = () =>
        deleteTunnel.mutate(
            { appId, nodeId },
            {
                onSuccess: () => setConfirmDelete(false),
                onError: (failure) => toast.error('Could not delete the tunnel', describeError(failure)),
            }
        )

    return (
        <div className="flex flex-col gap-5">
            <Tabs value={tab} onValueChange={(value) => setTab(value as TunnelTab)}>
                <TabsList aria-label="Access sections">
                    <TabsTrigger value="overview">Tunnel</TabsTrigger>
                    <TabsTrigger value="ingress">Routes</TabsTrigger>
                </TabsList>
            </Tabs>

            {tab === 'overview' && (
                <Card>
                    <CardHeader>
                        <CardTitle className="flex flex-wrap items-center gap-2.5 text-base">
                            <Globe className="h-4 w-4 text-muted-foreground" />
                            Tunnel
                            <HealthPill status={data.status} />
                        </CardTitle>
                    </CardHeader>
                    <CardContent className="flex flex-col gap-5">
                        {data.public_url ? (
                            <div className="flex items-center gap-1">
                                <a href={data.public_url} target="_blank" rel="noopener noreferrer" className="inline-flex min-h-[44px] items-center break-all font-mono text-sm font-medium text-status-info-fg hover:underline md:min-h-0">
                                    {data.public_url}
                                </a>
                                <Button variant="ghost" size="icon" onClick={() => void copy(data.public_url)} aria-label="Copy URL">
                                    <Copy className="h-4 w-4" />
                                </Button>
                            </div>
                        ) : (
                            <p className="text-sm text-muted-foreground">No public URL configured</p>
                        )}

                        <dl className="grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
                            <div><dt className="text-muted-foreground">Name</dt><dd className="mt-1 truncate font-medium">{data.tunnel_name}</dd></div>
                            <div><dt className="text-muted-foreground">ID</dt><dd className="mt-1 truncate font-mono text-xs font-medium">{data.tunnel_id}</dd></div>
                            <div><dt className="text-muted-foreground">Created</dt><dd className="mt-1 font-medium">{formatDate(data.created_at)}</dd></div>
                            <div><dt className="text-muted-foreground">Last synced</dt><dd className="mt-1 font-medium">{formatDate(data.last_synced_at)}</dd></div>
                        </dl>

                        <div className="flex flex-wrap gap-2 border-t border-border pt-4">
                            <Button variant="outline" onClick={sync} disabled={syncTunnel.isPending}>
                                {syncTunnel.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
                                Sync from Cloudflare
                            </Button>
                            <Button variant="danger" onClick={() => setConfirmDelete(true)} disabled={deleteTunnel.isPending}>
                                <Trash2 className="h-4 w-4" />
                                Delete tunnel
                            </Button>
                        </div>
                    </CardContent>
                </Card>
            )}

            {tab === 'ingress' && (
                <IngressConfiguration
                    appId={appId}
                    nodeId={nodeId}
                    existingIngress={data.ingress_rules || []}
                    onSave={() => {
                        refetch()
                        toast.success('Routes saved', 'The new routes are live')
                    }}
                />
            )}

            <ConfirmationDialog
                open={confirmDelete}
                onOpenChange={setConfirmDelete}
                title="Delete this tunnel?"
                description="The app stops being reachable on its public address. The app itself is not touched."
                confirmText="Delete tunnel"
                variant="destructive"
                isLoading={deleteTunnel.isPending}
                onConfirm={remove}
            />
        </div>
    )
}

export default CloudflareTab
