import { ServerCrash } from 'lucide-react'
import { Card } from '@/shared/components/ui/Card'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { formatAgo } from '@/shared/lib/attention'
import type { Node } from '@/shared/types/api'

// The nodes that are not answering, so their missing readings are not a surprise.
function SilentNodesCard({ nodes }: { nodes: Node[] }) {
    return (
        <Card
            className="divide-y divide-border border-status-warn/50"
            aria-label="Nodes that are not answering"
            role="region"
        >
            {nodes.map((node) => (
                <div key={node.id} className="flex flex-wrap items-center gap-3 p-4">
                    <ServerCrash aria-hidden="true" className="h-5 w-5 shrink-0 text-status-warn-fg" />
                    <div className="min-w-0 flex-1">
                        <p className="font-semibold">{node.name} is not answering</p>
                        <p className="text-compact text-muted-foreground">
                            Its readings are left out until it reconnects.
                            {node.last_seen ? ` Last seen ${formatAgo(node.last_seen)}.` : ''}
                        </p>
                    </div>
                    <StatusPill kind="warn" size="sm">
                        {node.status === 'unreachable' ? 'Unreachable' : 'Offline'}
                    </StatusPill>
                </div>
            ))}
        </Card>
    )
}

export default SilentNodesCard
