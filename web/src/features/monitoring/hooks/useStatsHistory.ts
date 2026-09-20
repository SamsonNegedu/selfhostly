import { useState } from 'react'
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
    const [history, setHistory] = useState<Record<string, NodeHistory>>({})
    const [seen, setSeen] = useState<{ stats: SystemStats[] | undefined; updatedAt: number }>({
        stats: undefined,
        updatedAt: 0,
    })

    // A new reading is added while rendering, so the trend is never a render behind.
    if (seen.stats !== stats || seen.updatedAt !== updatedAt) {
        setSeen({ stats, updatedAt })
        if (stats && updatedAt !== 0) {
            const next = { ...history }
            for (const node of stats) {
                if (node.status !== 'online') continue
                const entry = next[node.node_id] ?? { cpu: [], memory: [], disk: [] }
                next[node.node_id] = {
                    cpu: [...entry.cpu, node.cpu.usage_percent].slice(-MAX_SAMPLES),
                    memory: [...entry.memory, node.memory.usage_percent].slice(-MAX_SAMPLES),
                    disk: [...entry.disk, node.disk.usage_percent].slice(-MAX_SAMPLES),
                }
            }
            setHistory(next)
        }
    }

    return history
}
