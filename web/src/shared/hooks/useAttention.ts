import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/shared/lib/api-client'
import { getAttentionItems, type AttentionItem } from '@/shared/lib/attention'
import { useNodes } from '@/shared/services/api'
import type { App } from '@/shared/types/api'

const ATTENTION_REFRESH_MS = 30_000
const ALL_NODES = 'all'

interface Attention {
    items: AttentionItem[]
    failedApps: number
    isLoading: boolean
}

// Watches every node, whichever ones the scope switcher is showing, so a problem elsewhere is never hidden.
export function useAttention(): Attention {
    const { data: nodes = [], isLoading: nodesLoading } = useNodes()
    const { data: apps = [], isLoading: appsLoading } = useQuery<App[]>({
        queryKey: ['apps', { nodeIds: [ALL_NODES] }],
        queryFn: () => apiClient.get<App[]>('/api/apps', { node_ids: ALL_NODES }),
        refetchInterval: ATTENTION_REFRESH_MS,
    })

    return useMemo(() => {
        const items = getAttentionItems(apps ?? [], nodes)
        return {
            items,
            failedApps: items.filter((item) => item.kind === 'app-error').length,
            isLoading: nodesLoading || appsLoading,
        }
    }, [apps, nodes, nodesLoading, appsLoading])
}
