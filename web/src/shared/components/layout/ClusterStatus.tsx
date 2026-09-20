import { StatusDot } from '../ui/StatusPill'
import { formatAgo } from '@/shared/lib/attention'
import { useNodes } from '@/shared/services/api'
import type { StatusKind } from '@/shared/lib/status'

interface ClusterStatusProps {
    collapsed: boolean
}

// How many of the cluster's machines are reachable, with the first one that is not.
function ClusterStatus({ collapsed }: ClusterStatusProps) {
    const { data: nodes = [] } = useNodes()
    if (nodes.length === 0) return null

    const online = nodes.filter((node) => node.status === 'online').length
    const firstDown = nodes.find((node) => node.status !== 'online')
    const kind: StatusKind = online === nodes.length ? 'ok' : 'warn'
    const headline = online === nodes.length ? `All ${nodes.length} nodes online` : `${online} of ${nodes.length} nodes online`
    const detail = firstDown
        ? `${firstDown.name} ${firstDown.status}${firstDown.last_seen ? ` since ${formatAgo(firstDown.last_seen).replace(' ago', '')} ago` : ''}`
        : undefined

    if (collapsed) {
        return (
            <div className="hidden justify-center py-2 md:flex" title={headline} role="status" aria-label={headline}>
                <StatusDot kind={kind} className="h-2.5 w-2.5" />
            </div>
        )
    }

    return (
        <div className="rounded-lg border border-border bg-card p-3" role="status">
            <div className="flex items-center gap-2 text-[13px] font-semibold">
                <StatusDot kind={kind} />
                {headline}
            </div>
            {detail && <p className="mt-1 text-xs text-muted-foreground">{detail}</p>}
        </div>
    )
}

export default ClusterStatus
