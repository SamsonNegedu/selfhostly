import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { ExternalLink, Globe, Route } from 'lucide-react'
import { AppTile } from '@/shared/components/ui/AppTile'
import { Button, buttonClasses } from '@/shared/components/ui/Button'
import { Card } from '@/shared/components/ui/Card'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Sheet, SheetContent, SheetDescription, SheetTitle } from '@/shared/components/ui/Sheet'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/components/ui/Table'
import { useNodeContext } from '@/shared/contexts/NodeContext'
import { appHref, ROUTES } from '@/shared/lib/routes'
import type { StatusKind } from '@/shared/lib/status'
import { useApps, useNodes, useProviders, useTunnels } from '@/shared/services/api'
import type { App, CloudflareTunnel } from '@/shared/types/api'
import { IngressConfiguration } from './IngressConfiguration'

interface AddressRow {
    app: App
    tunnel?: CloudflareTunnel
    kind: 'custom' | 'quick'
    status: { kind: StatusKind; label: string }
}

const hostOf = (url: string) => {
    try {
        return new URL(url).host
    } catch {
        return url
    }
}

function statusFor(app: App, tunnel?: CloudflareTunnel): AddressRow['status'] {
    if (app.tunnel_mode === 'quick') return { kind: 'warn', label: 'Temporary' }
    if (!tunnel) return { kind: 'idle', label: 'Unknown' }
    if (tunnel.status === 'error') return { kind: 'err', label: 'Error' }
    return tunnel.is_active && tunnel.status === 'active' ? { kind: 'ok', label: 'Active' } : { kind: 'idle', label: 'Inactive' }
}

// Every address people can reach your apps on from outside, and how each one is served.
function CloudflareManagement() {
    const { selectedNodeIds } = useNodeContext()
    const { data: apps, error: appsError, refetch } = useApps(selectedNodeIds)
    const { data: tunnelData } = useTunnels(selectedNodeIds)
    const { data: providers } = useProviders()
    const { data: nodes = [] } = useNodes()
    const [routesFor, setRoutesFor] = useState<AddressRow | null>(null)

    const cloudflare = providers?.providers.find((provider) => provider.name === 'cloudflare')
    const connected = cloudflare?.is_configured ?? false

    const rows = useMemo<AddressRow[]>(() => {
        if (!apps) return []
        const tunnels = tunnelData?.tunnels ?? []
        return apps
            .filter((app) => app.public_url)
            .map((app) => {
                const tunnel = tunnels.find((candidate) => candidate.app_id === app.id)
                return { app, tunnel, kind: app.tunnel_mode === 'quick' ? ('quick' as const) : ('custom' as const), status: statusFor(app, tunnel) }
            })
            .sort((a, b) => a.app.name.localeCompare(b.app.name))
    }, [apps, tunnelData])

    if (apps === undefined) {
        if (appsError) return <ErrorState title="Could not load your addresses" error={appsError} onRetry={() => refetch()} />
        return (
            <div role="status" aria-label="Loading addresses" className="flex flex-col gap-5">
                <Skeleton className="h-12 w-56" />
                <Skeleton className="h-20 rounded-xl" />
                <Skeleton className="h-56 rounded-xl" />
            </div>
        )
    }

    const private_ = apps.length - rows.length
    const nodeName = (id: string) => nodes.find((node) => node.id === id)?.name ?? id

    return (
        <div className="flex flex-col gap-5">
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">Access</h1>
                <p className="text-muted-foreground">
                    {rows.length === 0 ? 'Nothing is reachable from outside yet' : `${rows.length} ${rows.length === 1 ? 'app is' : 'apps are'} reachable from outside`}
                </p>
            </div>

            <Card className="flex flex-wrap items-center gap-4 p-4" aria-label="Tunnel provider" role="region">
                <div aria-hidden="true" className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <Globe className="h-5 w-5" />
                </div>
                <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                        <p className="font-semibold">Cloudflare</p>
                        <StatusPill kind={connected ? 'ok' : 'idle'}>{connected ? 'Connected' : 'Not connected'}</StatusPill>
                    </div>
                    <p className="text-[13px] text-muted-foreground">
                        {connected ? 'Custom domains are available. Quick Tunnels also work without it.' : 'Connect it to publish apps on your own domain. Quick Tunnels work without it.'}
                    </p>
                </div>
                <Link to={ROUTES.settings} className={buttonClasses({ variant: 'outline' })}>
                    {connected ? 'Manage' : 'Connect'}
                </Link>
            </Card>

            {rows.length === 0 ? (
                <EmptyState
                    icon={<Globe className="h-5 w-5" />}
                    title="Nothing is exposed yet"
                    description="Give an app a public address from its Access tab, or when you create it."
                    action={
                        <Link to={ROUTES.fleet} className={buttonClasses()}>
                            Go to Fleet
                        </Link>
                    }
                    className="py-14"
                />
            ) : (
                <Card className="overflow-hidden p-0">
                    <Table aria-label="Addresses" className="max-md:[&_td]:px-3 max-md:[&_th]:px-3">
                        <TableHeader>
                            <TableRow className="hover:bg-transparent">
                                <TableHead>App</TableHead>
                                <TableHead>Address</TableHead>
                                <TableHead className="max-md:hidden">Status</TableHead>
                                <TableHead className="max-lg:hidden">Type</TableHead>
                                <TableHead className="max-lg:hidden">Node</TableHead>
                                <TableHead className="w-[1%]">
                                    <span className="sr-only">Actions</span>
                                </TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {rows.map((row) => (
                                <TableRow key={row.app.id} data-address={row.app.name}>
                                    <TableCell>
                                        <div className="flex items-center gap-3">
                                            <AppTile name={row.app.name} size="sm" className="max-md:hidden" />
                                            <div className="flex flex-col items-start">
                                                <Link to={appHref(row.app, 'access')} className="inline-flex min-h-[44px] items-center font-semibold hover:underline md:min-h-0">
                                                    {row.app.name}
                                                </Link>
                                                <StatusPill kind={row.status.kind} size="sm" className="md:hidden">
                                                    {row.status.label}
                                                </StatusPill>
                                                <span className="mb-1 mt-1 text-xs text-muted-foreground lg:hidden">
                                                    {row.kind === 'quick' ? 'Quick Tunnel' : 'Custom domain'} · {nodeName(row.app.node_id)}
                                                </span>
                                            </div>
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        <a href={row.app.public_url} target="_blank" rel="noopener noreferrer" className="inline-flex min-h-[44px] max-w-[140px] items-center gap-1.5 md:max-w-[220px] font-mono text-[12.5px] text-status-info-fg hover:underline md:min-h-0">
                                            <span className="truncate">{hostOf(row.app.public_url ?? '')}</span>
                                            <ExternalLink className="h-3 w-3 shrink-0" aria-hidden="true" />
                                            <span className="sr-only">(opens in a new tab)</span>
                                        </a>
                                    </TableCell>
                                    <TableCell className="max-md:hidden">
                                        <StatusPill kind={row.status.kind}>{row.status.label}</StatusPill>
                                    </TableCell>
                                    <TableCell className="text-muted-foreground max-lg:hidden">{row.kind === 'quick' ? 'Quick Tunnel' : 'Custom domain'}</TableCell>
                                    <TableCell className="text-muted-foreground max-lg:hidden">{nodeName(row.app.node_id)}</TableCell>
                                    <TableCell>
                                        <Button variant="outline" size="sm" onClick={() => setRoutesFor(row)} aria-label={`Routes for ${row.app.name}`}>
                                            <Route className="h-4 w-4" />
                                            <span className="max-md:hidden">Routes</span>
                                        </Button>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </Card>
            )}

            {private_ > 0 && rows.length > 0 && (
                <p className="text-sm text-muted-foreground">
                    {private_} {private_ === 1 ? 'app is' : 'apps are'} only reachable on your network.
                </p>
            )}

            <Sheet open={routesFor !== null} onOpenChange={(open) => !open && setRoutesFor(null)}>
                <SheetContent side="right" className="max-md:inset-x-0 max-md:bottom-0 max-md:top-auto max-md:h-auto max-md:max-h-[88vh] max-md:max-w-none max-md:rounded-t-[20px] max-md:border-l-0 max-md:border-t md:max-w-xl overflow-y-auto">
                    {routesFor && (
                        <>
                            <SheetTitle>{routesFor.app.name} routes</SheetTitle>
                            <SheetDescription>{routesFor.kind === 'quick' ? 'This address is temporary and changes if the app restarts.' : 'One hostname per service.'}</SheetDescription>
                            {routesFor.kind === 'custom' && routesFor.tunnel ? (
                                <IngressConfiguration
                                    appId={routesFor.app.id}
                                    nodeId={routesFor.app.node_id}
                                    existingIngress={routesFor.tunnel.ingress_rules ?? []}
                                    onSave={() => refetch()}
                                    flat
                                />
                            ) : (
                                <div className="flex flex-col gap-3 text-sm">
                                    <p className="font-mono">{routesFor.app.public_url}</p>
                                    <p className="text-muted-foreground">Quick Tunnels have one fixed route to the app and cannot be edited. For a stable address, switch the app to your own domain.</p>
                                    <Link to={appHref(routesFor.app, 'access')} className={buttonClasses({ className: 'self-start' })}>
                                        Open the app's Access tab
                                    </Link>
                                </div>
                            )}
                        </>
                    )}
                </SheetContent>
            </Sheet>
        </div>
    )
}

export default CloudflareManagement
