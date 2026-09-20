import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Server } from 'lucide-react'
import { useFleetMetrics } from '@/features/dashboard/hooks/useFleetMetrics'
import { buttonClasses } from '@/shared/components/ui/Button'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { ROUTES } from '@/shared/lib/routes'
import { useApps, useCurrentNode, useDeleteNode, useNodes } from '@/shared/services/api'
import type { Node } from '@/shared/types/api'
import NodeCard from './components/NodeCard'

function NodesPage() {
    const { data: nodes, error, refetch } = useNodes()
    const { data: currentNode } = useCurrentNode()
    const { data: apps = [] } = useApps(undefined)
    const metrics = useFleetMetrics(nodes ?? [])
    const deleteNode = useDeleteNode()
    const { toast } = useToast()
    const [toRemove, setToRemove] = useState<Node | null>(null)

    // A node that is not answering cannot have its apps cleaned up, so removing it drops their records from here.
    const appsOnRemoved = toRemove ? apps.filter((app) => app.node_id === toRemove.id).length : 0
    const removeDropsApps = toRemove !== null && toRemove.status !== 'online' && appsOnRemoved > 0

    const remove = () => {
        if (!toRemove) return
        const node = toRemove
        setToRemove(null)
        deleteNode.mutate({ id: node.id, force: removeDropsApps }, {
            onSuccess: () => toast.success('Node removed', `${node.name} is no longer part of the cluster`),
            onError: (failure) => toast.error('Could not remove the node', describeError(failure)),
        })
    }

    if (nodes === undefined) {
        if (error) return <ErrorState title="Could not load your nodes" error={error} onRetry={() => refetch()} />
        return (
            <div role="status" aria-label="Loading nodes" className="flex flex-col gap-5">
                <Skeleton className="h-12 w-64" />
                <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                    {[0, 1, 2].map((key) => (
                        <Skeleton key={key} className="h-44 rounded-xl" />
                    ))}
                </div>
            </div>
        )
    }

    const offline = nodes.filter((node) => node.status !== 'online').length

    return (
        <div className="flex flex-col gap-5">
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-semibold tracking-tight">Nodes</h1>
                    <p className="text-muted-foreground">
                        {nodes.length} {nodes.length === 1 ? 'machine' : 'machines'}
                        {offline > 0 ? `, ${offline} not answering` : ', all online'}
                    </p>
                </div>
                <Link to={ROUTES.registerNode} className={buttonClasses()}>
                    <Plus className="h-4 w-4" />
                    Add node
                </Link>
            </div>

            {nodes.length === 0 ? (
                <EmptyState
                    icon={<Server className="h-5 w-5" />}
                    title="No nodes yet"
                    description="Add a second machine, such as a Raspberry Pi or a NAS, and spread your apps across it."
                    action={
                        <Link to={ROUTES.registerNode} className={buttonClasses()}>
                            Add node
                        </Link>
                    }
                    className="py-14"
                />
            ) : (
                <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                    {nodes.map((node) => (
                        <NodeCard
                            key={node.id}
                            node={node}
                            isCurrent={currentNode?.id === node.id}
                            appCount={apps.filter((app) => app.node_id === node.id).length}
                            metrics={metrics.forNode(node.id)}
                            onRemove={setToRemove}
                        />
                    ))}
                </div>
            )}

            {nodes.length === 1 && (
                <p className="text-sm text-muted-foreground">
                    Only one machine so far. <Link to={ROUTES.registerNode} className="font-medium text-foreground underline underline-offset-2">Add another</Link> to run apps in more than one place.
                </p>
            )}

            <ConfirmationDialog
                open={toRemove !== null}
                onOpenChange={(open) => !open && setToRemove(null)}
                title={`Remove ${toRemove?.name ?? 'node'}?`}
                description={
                    removeDropsApps
                        ? `${toRemove?.name} is not answering, so its ${appsOnRemoved} ${appsOnRemoved === 1 ? 'app' : 'apps'} cannot be cleaned up there. Removing it drops ${appsOnRemoved === 1 ? 'that app' : 'those apps'} from Selfhostly. Nothing is deleted on the machine itself.`
                        : toRemove?.status === 'online' && appsOnRemoved > 0
                          ? `${toRemove?.name} still has ${appsOnRemoved} ${appsOnRemoved === 1 ? 'app' : 'apps'}. Delete ${appsOnRemoved === 1 ? 'it' : 'them'} first, then remove the node.`
                          : 'The node leaves the cluster. You can add it again later.'
                }
                confirmText={removeDropsApps ? 'Remove node and drop its apps' : 'Remove node'}
                variant="destructive"
                confirmationText={toRemove?.name}
                onConfirm={remove}
            />
        </div>
    )
}

export default NodesPage
