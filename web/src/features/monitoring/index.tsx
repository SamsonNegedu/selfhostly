import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { ServerCrash } from 'lucide-react'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { buttonClasses } from '@/shared/components/ui/Button'
import { useNodeContext } from '@/shared/contexts/NodeContext'
import { formatAgo } from '@/shared/lib/attention'
import { ROUTES } from '@/shared/lib/routes'
import { useApps, useNodes, useSystemStats } from '@/shared/services/api'
import ContainersSection from './components/ContainersSection'
import InsightAlerts from './components/InsightAlerts'
import NodeResources from './components/NodeResources'
import SilentNodesCard from './components/SilentNodesCard'
import { useStatsHistory } from './hooks/useStatsHistory'
import { getInsightAlerts } from './lib/alerts'

const REFRESH_MS = 10_000

function Insights() {
    const { selectedNodeIds } = useNodeContext()
    const { data: nodes } = useNodes()
    const { data: apps = [] } = useApps(selectedNodeIds)

    // Only nodes that answer are asked for readings, because one that does not would hold up all of them.
    const selected = useMemo(
        () => (nodes ?? []).filter((node) => selectedNodeIds.length === 0 || selectedNodeIds.includes(node.id)),
        [nodes, selectedNodeIds],
    )
    const onlineIds = useMemo(
        () => selected.filter((node) => node.status === 'online').map((node) => node.id),
        [selected],
    )
    const silent = selected.filter((node) => node.status !== 'online')

    const { data: stats, error, dataUpdatedAt, refetch } = useSystemStats(REFRESH_MS, onlineIds)
    const history = useStatsHistory(stats, dataUpdatedAt)

    const online = useMemo(() => (stats ?? []).filter((node) => node.status === 'online' && !node.error), [stats])
    const alerts = useMemo(() => getInsightAlerts(online), [online])
    const containers = useMemo(() => online.flatMap((node) => node.containers ?? []), [online])
    const nodeName = (id: string) => nodes?.find((node) => node.id === id)?.name ?? id

    if (nodes === undefined || (onlineIds.length > 0 && stats === undefined && !error)) {
        return (
            <div role="status" aria-label="Loading insights" className="flex flex-col gap-5">
                <Skeleton className="h-12 w-56" />
                <Skeleton className="h-16 rounded-xl" />
                <Skeleton className="h-48 rounded-xl" />
                <Skeleton className="h-64 rounded-xl" />
            </div>
        )
    }

    if (error && stats === undefined) {
        return <ErrorState title="Could not load insights" error={error} onRetry={() => refetch()} />
    }

    const updated = dataUpdatedAt ? formatAgo(new Date(dataUpdatedAt).toISOString()) : 'just now'

    return (
        <div className="flex flex-col gap-5">
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">Insights</h1>
                <p className="text-muted-foreground">
                    {online.length === 0
                        ? 'No node is answering'
                        : `${online.length} ${online.length === 1 ? 'node' : 'nodes'} answering`}{' '}
                    · Updated {updated}
                </p>
            </div>

            {silent.length > 0 && <SilentNodesCard nodes={silent} />}

            {online.length === 0 ? (
                <EmptyState
                    icon={<ServerCrash className="h-5 w-5" />}
                    title="Nothing to show yet"
                    description="No selected node is answering, so there are no readings. Check the nodes, or choose another scope."
                    action={
                        <Link to={ROUTES.nodes} className={buttonClasses()}>
                            Open Nodes
                        </Link>
                    }
                    className="py-14"
                />
            ) : (
                <>
                    <InsightAlerts alerts={alerts} />

                    <section aria-label="Resources" className="flex flex-col gap-4">
                        {online.map((node) => (
                            <NodeResources key={node.node_id} stats={node} history={history[node.node_id]} />
                        ))}
                        <p className="text-compact text-muted-foreground">
                            Charts show the readings taken while this page has been open. Older history is not kept.
                        </p>
                    </section>

                    <ContainersSection containers={containers} apps={apps} nodeName={nodeName} />
                </>
            )}
        </div>
    )
}

export default Insights
