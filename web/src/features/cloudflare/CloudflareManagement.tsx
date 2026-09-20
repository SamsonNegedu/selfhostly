import { useMemo, useState } from 'react'
import { Globe } from 'lucide-react'
import { Link } from 'react-router-dom'
import { buttonClasses } from '@/shared/components/ui/Button'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { useNodeContext } from '@/shared/contexts/NodeContext'
import { ROUTES } from '@/shared/lib/routes'
import { useApps, useNodes, useProviders, useTunnels } from '@/shared/services/api'
import AddressTable from './AddressTable'
import { buildAddressRows, type AddressRow } from './lib/address-rows'
import ProviderCard from './ProviderCard'
import RoutesSheet from './RoutesSheet'

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

    const rows = useMemo(() => buildAddressRows(apps ?? [], tunnelData?.tunnels ?? []), [apps, tunnelData])

    if (apps === undefined) {
        if (appsError)
            return <ErrorState title="Could not load your addresses" error={appsError} onRetry={() => refetch()} />
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
                    {rows.length === 0
                        ? 'Nothing is reachable from outside yet'
                        : `${rows.length} ${rows.length === 1 ? 'app is' : 'apps are'} reachable from outside`}
                </p>
            </div>

            <ProviderCard connected={connected} />

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
                <AddressTable rows={rows} nodeName={nodeName} onRoutes={setRoutesFor} />
            )}

            {private_ > 0 && rows.length > 0 && (
                <p className="text-sm text-muted-foreground">
                    {private_} {private_ === 1 ? 'app is' : 'apps are'} only reachable on your network.
                </p>
            )}

            <RoutesSheet row={routesFor} onClose={() => setRoutesFor(null)} onSaved={() => refetch()} />
        </div>
    )
}

export default CloudflareManagement
