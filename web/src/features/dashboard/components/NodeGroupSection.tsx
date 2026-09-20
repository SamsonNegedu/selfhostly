import { StatusDot, StatusPill } from '@/shared/components/ui/StatusPill'
import { formatPercent } from '@/shared/lib/format'
import { nodeStatusMeta } from '@/shared/lib/status'
import type { NodeGroup } from '../lib/fleet'
import type { NodeMetrics } from '../hooks/useFleetMetrics'
import { useIsPhone } from '@/shared/hooks/useMediaQuery'
import FleetAppCard from './FleetAppCard'
import FleetAppRow from './FleetAppRow'
import type { AppMetrics } from '../hooks/useFleetMetrics'
import type { useFleetActions } from '../hooks/useFleetActions'

interface NodeGroupSectionProps {
    group: NodeGroup
    showHeader: boolean
    nodeMetrics?: NodeMetrics
    metricsFor: (nodeId: string, appName: string) => AppMetrics | undefined
    actions: ReturnType<typeof useFleetActions>
}

// The apps on one node, under a header that says how that node is doing.
function NodeGroupSection({ group, showHeader, nodeMetrics, metricsFor, actions }: NodeGroupSectionProps) {
    const phone = useIsPhone()
    const status = group.node ? nodeStatusMeta(group.node.status) : undefined
    const count = group.apps.length

    return (
        <section aria-label={showHeader ? `Apps on ${group.name}` : 'Apps'} className="flex flex-col gap-3">
            {showHeader && (
                <div className="flex items-center gap-3">
                    <StatusDot kind={status?.kind ?? 'idle'} className="h-2.5 w-2.5" />
                    <h2 className="text-[15px] font-semibold">{group.name}</h2>
                    <span className="text-[13px] text-muted-foreground">
                        {count} {count === 1 ? 'app' : 'apps'}
                        {nodeMetrics && ` · CPU ${formatPercent(nodeMetrics.cpuPercent)} · Memory ${formatPercent(nodeMetrics.memoryPercent)}`}
                    </span>
                    {status && status.kind !== 'ok' && <StatusPill kind={status.kind} size="sm">{status.label}</StatusPill>}
                    <div className="h-px flex-1 bg-border" />
                </div>
            )}
            <div className={phone ? 'flex flex-col gap-2' : 'grid gap-4 md:grid-cols-2 xl:grid-cols-3'}>
                {group.apps.map((fleetApp) => {
                    const Item = phone ? FleetAppRow : FleetAppCard
                    return <Item key={fleetApp.app.id} fleetApp={fleetApp} metrics={metricsFor(fleetApp.app.node_id, fleetApp.app.name)} actions={actions} />
                })}
            </div>
        </section>
    )
}

export default NodeGroupSection
