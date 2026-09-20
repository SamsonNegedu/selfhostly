import { useState } from 'react'
import { Globe } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/shared/components/ui/Tabs'
import { useToast } from '@/shared/components/ui/Toast'
import { useCopyToClipboard } from '@/shared/hooks/useCopyToClipboard'
import { describeError } from '@/shared/lib/errors'
import { useDeleteTunnel, useSyncTunnel, useTunnel } from '@/shared/services/api'
import { IngressConfiguration } from '@/features/cloudflare/IngressConfiguration'
import type { TunnelByAppResponse } from '@/shared/types/api'
import NamedTunnelCard from './NamedTunnelCard'
import QuickTunnelCard from './QuickTunnelCard'
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

// How the app is reached from outside: no tunnel, a temporary Quick Tunnel, or a named tunnel on your own domain.
function CloudflareTab({ appId, nodeId, composeContent }: CloudflareTabProps) {
    const { data: tunnel, isLoading, error, refetch } = useTunnel(appId, nodeId)
    const syncTunnel = useSyncTunnel()
    const deleteTunnel = useDeleteTunnel()
    const { toast } = useToast()
    const copyText = useCopyToClipboard()
    const [tab, setTab] = useState<TunnelTab>('overview')
    const [dialog, setDialog] = useState<Dialog>(null)
    const [confirmDelete, setConfirmDelete] = useState(false)

    const dialogProps = (name: Exclude<Dialog, null>) => ({
        open: dialog === name,
        onOpenChange: (open: boolean) => setDialog(open ? name : null),
        appId,
        nodeId,
        composeContent,
    })
    const dialogs = (
        <>
            <QuickTunnelDialog {...dialogProps('quick')} />
            <QuickTunnelDialog {...dialogProps('quick-again')} recreate />
            <CustomDomainDialog {...dialogProps('custom')} />
            <CustomDomainDialog {...dialogProps('switch')} switching />
        </>
    )

    const copy = (text: string) => copyText(text, 'The address is on your clipboard')

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
        return (
            <EmptyState
                icon={<Globe className="h-5 w-5" />}
                title="Only reachable on your network"
                description="This app has no public address. Tunnels give it one without opening ports on your router."
                className="py-14"
            />
        )
    }

    if (isNoTunnelResponse(tunnel)) {
        // A Quick Tunnel is only shown once it has an address to show.
        const quickUrl = tunnel.tunnel_mode === 'quick' && tunnel.public_url ? tunnel.public_url : undefined
        return (
            <div className="flex flex-col gap-5">
                {quickUrl ? (
                    <QuickTunnelCard
                        publicUrl={quickUrl}
                        onSwitch={() => setDialog('switch')}
                        onNewAddress={() => setDialog('quick-again')}
                    />
                ) : (
                    <EmptyState
                        icon={<Globe className="h-5 w-5" />}
                        title="No public address yet"
                        description="Create a temporary Quick Tunnel to try it out, or set up a stable address on your own domain."
                        action={
                            <div className="flex flex-wrap justify-center gap-2">
                                <Button onClick={() => setDialog('custom')}>Use my own domain</Button>
                                <Button variant="outline" onClick={() => setDialog('quick')}>
                                    Create a Quick Tunnel
                                </Button>
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
            },
        )
    const remove = () =>
        deleteTunnel.mutate(
            { appId, nodeId },
            {
                onSuccess: () => setConfirmDelete(false),
                onError: (failure) => toast.error('Could not delete the tunnel', describeError(failure)),
            },
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
                <NamedTunnelCard
                    data={data}
                    onCopy={copy}
                    onSync={sync}
                    syncing={syncTunnel.isPending}
                    onDelete={() => setConfirmDelete(true)}
                    deleting={deleteTunnel.isPending}
                />
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
