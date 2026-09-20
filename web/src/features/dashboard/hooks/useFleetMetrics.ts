import { useMemo } from 'react'
import { useSystemStats } from '@/shared/services/api'
import type { Node } from '@/shared/types/api'

const REFRESH_MS = 10_000
const PERCENT = 100

export interface AppMetrics {
    cpuPercent: number
    memoryBytes: number
    containers: number
    restarts: number
}

export interface NodeMetrics {
    cpuPercent: number
    memoryPercent: number
}

interface FleetMetrics {
    forApp: (nodeId: string, appName: string) => AppMetrics | undefined
    forNode: (nodeId: string) => NodeMetrics | undefined
}

// Live numbers for the cards and node headers. Only nodes that are online are asked, because asking an
// unreachable node makes the whole request wait or fail. An app with no running container has no metrics.
export function useFleetMetrics(nodes: Node[]): FleetMetrics {
    const onlineIds = useMemo(() => nodes.filter((node) => node.status === 'online').map((node) => node.id), [nodes])
    const { data: stats = [] } = useSystemStats(REFRESH_MS, onlineIds)

    return useMemo(() => {
        const apps = new Map<string, AppMetrics>()
        const nodeMetrics = new Map<string, NodeMetrics>()

        for (const node of stats) {
            if (node.error || node.status !== 'online') continue
            const total = node.memory.total_bytes
            nodeMetrics.set(node.node_id, {
                cpuPercent: node.cpu.usage_percent,
                memoryPercent: total > 0 ? (node.memory.used_bytes / total) * PERCENT : 0,
            })

            for (const container of node.containers) {
                if (!container.is_managed || container.state !== 'running') continue
                const key = `${node.node_id}:${container.app_name}`
                const current = apps.get(key) ?? { cpuPercent: 0, memoryBytes: 0, containers: 0, restarts: 0 }
                apps.set(key, {
                    cpuPercent: current.cpuPercent + container.cpu_percent,
                    memoryBytes: current.memoryBytes + container.memory_usage_bytes,
                    containers: current.containers + 1,
                    restarts: current.restarts + container.restart_count,
                })
            }
        }

        return {
            forApp: (nodeId, appName) => apps.get(`${nodeId}:${appName}`),
            forNode: (nodeId) => nodeMetrics.get(nodeId),
        }
    }, [stats])
}
