import { Copy, Globe, Loader2, RefreshCw, Trash2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import type { TunnelByAppResponse } from '@/shared/types/api'

// Cloudflare reports healthy, degraded, inactive or down. Degraded still serves traffic.
function HealthPill({ status }: { status: string }) {
    if (status === 'active' || status === 'healthy') return <StatusPill kind="ok">Healthy</StatusPill>
    if (status === 'degraded') return <StatusPill kind="warn">Degraded</StatusPill>
    if (status === 'inactive' || status === 'down') return <StatusPill kind="idle">Inactive</StatusPill>
    if (status === 'error') return <StatusPill kind="err">Error</StatusPill>
    return <StatusPill kind="idle">Unknown</StatusPill>
}

const formatDate = (iso?: string) => (iso ? new Date(iso).toLocaleDateString() : 'Never')

interface NamedTunnelCardProps {
    data: NonNullable<TunnelByAppResponse['tunnel']>
    onCopy: (text: string) => void | Promise<void>
    onSync: () => void
    syncing: boolean
    // Asks to delete. The confirmation is the caller's.
    onDelete: () => void
    deleting: boolean
}

// A named tunnel on your own domain: its address, its details, and the actions on it.
function NamedTunnelCard({ data, onCopy, onSync, syncing, onDelete, deleting }: NamedTunnelCardProps) {
    return (
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
                        <a
                            href={data.public_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex min-h-[44px] items-center break-all font-mono text-sm font-medium text-status-info-fg hover:underline md:min-h-0"
                        >
                            {data.public_url}
                        </a>
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => void onCopy(data.public_url)}
                            aria-label="Copy URL"
                        >
                            <Copy className="h-4 w-4" />
                        </Button>
                    </div>
                ) : (
                    <p className="text-sm text-muted-foreground">No public URL configured</p>
                )}

                <dl className="grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
                    <div>
                        <dt className="text-muted-foreground">Name</dt>
                        <dd className="mt-1 truncate font-medium" title={data.tunnel_name}>
                            {data.tunnel_name}
                        </dd>
                    </div>
                    <div>
                        <dt className="text-muted-foreground">ID</dt>
                        <dd className="mt-1 truncate font-mono text-xs font-medium" title={data.tunnel_id}>
                            {data.tunnel_id}
                        </dd>
                    </div>
                    <div>
                        <dt className="text-muted-foreground">Created</dt>
                        <dd className="mt-1 font-medium">{formatDate(data.created_at)}</dd>
                    </div>
                    <div>
                        <dt className="text-muted-foreground">Last synced</dt>
                        <dd className="mt-1 font-medium">{formatDate(data.last_synced_at)}</dd>
                    </div>
                </dl>

                <div className="flex flex-wrap gap-2 border-t border-border pt-4">
                    <Button variant="outline" onClick={onSync} disabled={syncing}>
                        {syncing ? (
                            <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                        ) : (
                            <RefreshCw className="h-4 w-4" />
                        )}
                        Sync from Cloudflare
                    </Button>
                    <Button variant="danger" onClick={() => onDelete()} disabled={deleting}>
                        <Trash2 className="h-4 w-4" />
                        Delete tunnel
                    </Button>
                </div>
            </CardContent>
        </Card>
    )
}

export default NamedTunnelCard
