import { useEffect, useRef, useState } from 'react'
import type { SystemStats } from '@/shared/types/api'

const MAX_SAMPLES = 60

export interface NodeHistory {
    cpu: number[]
    memory: number[]
    disk: number[]
}

// The API reports the present only, so the trend is what this page has seen since it was opened. Each new
// reading adds a point, and the oldest points fall off.
export function useStatsHistory(stats: SystemStats[] | undefined, updatedAt: number): Record<string, NodeHistory> {
    const history = useRef<Record<string, NodeHistory>>({})
    const [, setVersion] = useState(0)

    useEffect(() => {
        if (!stats || updatedAt === 0) return
        for (const node of stats) {
            if (node.status !== 'online') continue
            const entry = history.current[node.node_id] ?? { cpu: [], memory: [], disk: [] }
            entry.cpu = [...entry.cpu, node.cpu.usage_percent].slice(-MAX_SAMPLES)
            entry.memory = [...entry.memory, node.memory.usage_percent].slice(-MAX_SAMPLES)
            entry.disk = [...entry.disk, node.disk.usage_percent].slice(-MAX_SAMPLES)
            history.current[node.node_id] = entry
        }
        setVersion((value) => value + 1)
    }, [stats, updatedAt])

    return history.current
}
