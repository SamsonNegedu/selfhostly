import { resourceTone, type Resource } from '@/shared/lib/thresholds'
import type { StatusKind } from '@/shared/lib/status'
import type { SystemStats } from '@/shared/types/api'

export interface InsightAlert {
    id: string
    kind: StatusKind
    title: string
    detail: string
}

const CONTAINER_MEMORY_WARN = 85

// What needs attention across the nodes that answered, worst first. An empty list means everything is fine.
export function getInsightAlerts(stats: SystemStats[]): InsightAlert[] {
    const alerts: InsightAlert[] = []

    for (const node of stats) {
        const resources: [string, Resource, number][] = [
            ['CPU', 'cpu', node.cpu.usage_percent],
            ['Memory', 'memory', node.memory.usage_percent],
            ['Disk', 'disk', node.disk.usage_percent],
        ]
        for (const [name, resource, value] of resources) {
            const kind = resourceTone(resource, value)
            if (kind !== 'ok') {
                alerts.push({
                    id: `${node.node_id}-${name}`,
                    kind,
                    title: `${name} is ${kind === 'err' ? 'critical' : 'high'} on ${node.node_name}`,
                    detail: `${Math.round(value)}% in use`,
                })
            }
        }

        const containers = node.containers ?? []
        const memoryHeavy = containers.filter((c) => c.state === 'running' && c.memory_limit_bytes > 0 && (c.memory_usage_bytes / c.memory_limit_bytes) * 100 > CONTAINER_MEMORY_WARN)
        if (memoryHeavy.length > 0) {
            alerts.push({ id: `${node.node_id}-container-memory`, kind: 'warn', title: `${memoryHeavy.length} ${memoryHeavy.length === 1 ? 'container is close to its' : 'containers are close to their'} memory limit`, detail: `${memoryHeavy.map((c) => c.name).join(', ')} on ${node.node_name}` })
        }
        const stopped = containers.filter((c) => c.state === 'stopped' && c.is_managed)
        if (stopped.length > 0) {
            alerts.push({ id: `${node.node_id}-stopped`, kind: 'warn', title: `${stopped.length} ${stopped.length === 1 ? 'container is' : 'containers are'} stopped`, detail: `${stopped.map((c) => c.name).join(', ')} on ${node.node_name}` })
        }
    }

    return alerts.sort((a, b) => (a.kind === b.kind ? 0 : a.kind === 'err' ? -1 : 1))
}
