import { Card } from '@/shared/components/ui/Card'
import { useFleetMetrics } from '@/features/dashboard/hooks/useFleetMetrics'
import { formatBytes, formatPercent } from '@/shared/lib/format'
import { useNodes } from '@/shared/services/api'
import type { App } from '@/shared/types/api'

// CPU, memory, containers and restarts for a running app. Anything else shows a dash.
function MetricTiles({ app }: { app: App }) {
    const { data: nodes = [] } = useNodes()
    const metrics = useFleetMetrics(nodes).forApp(app.node_id ?? '', app.name)
    const isRunning = app.status === 'running'
    const tiles = [
        { label: 'CPU', value: isRunning && metrics ? formatPercent(metrics.cpuPercent) : '-' },
        { label: 'Memory', value: isRunning && metrics ? formatBytes(metrics.memoryBytes) : '-' },
        { label: 'Containers', value: isRunning && metrics ? String(metrics.containers) : '-' },
        { label: 'Restarts', value: isRunning && metrics ? String(metrics.restarts) : '-' },
    ]

    return (
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            {tiles.map((tile) => (
                <Card key={tile.label} className="flex flex-col gap-1 p-4">
                    <span className="text-caption font-semibold uppercase tracking-wider text-muted-foreground">
                        {tile.label}
                    </span>
                    <span className="text-2xl font-semibold tabular-nums">{tile.value}</span>
                </Card>
            ))}
        </div>
    )
}

export default MetricTiles
