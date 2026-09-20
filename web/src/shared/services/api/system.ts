import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../../lib/api-client'
import { useNodeContext } from '../../contexts/NodeContext'
import type { SystemStats } from '../../types/api'

// System monitoring API
export function useSystemStats(refreshInterval: number = 10000, nodeIdsOverride?: string[]) {
    const { selectedNodeIds: globalNodeIds } = useNodeContext()

    // Use override if provided, otherwise use global context
    const nodeIds = nodeIdsOverride ?? globalNodeIds

    // Build query key with node filter
    const queryKey = nodeIds && nodeIds.length > 0 ? ['system', 'stats', { nodeIds }] : ['system', 'stats']

    return useQuery<SystemStats[]>({
        queryKey,
        queryFn: () => {
            // Build node_ids parameter
            if (nodeIds && nodeIds.length > 0) {
                const nodeIdsParam = nodeIds.join(',')
                return apiClient.get<SystemStats[]>('/api/system/stats', { node_ids: nodeIdsParam })
            }
            // If no nodes selected (empty array during initialization), don't make the request yet
            // Return empty array to avoid fetching with node_ids=all
            return Promise.resolve([])
        },
        // Don't run the query until we have selected nodes
        enabled: nodeIds && nodeIds.length > 0,
        refetchInterval: refreshInterval,
        refetchIntervalInBackground: false, // Only poll when tab is visible
    })
}

export function useRestartContainer() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ containerId, nodeId }: { containerId: string; nodeId: string }) =>
            apiClient.post<{ message: string; container_id: string }>(
                `/api/system/containers/${containerId}/restart`,
                undefined,
                { node_id: nodeId },
            ),
        onSuccess: () => {
            // Refresh system stats after container action
            queryClient.invalidateQueries({ queryKey: ['system', 'stats'] })
        },
    })
}

export function useStopContainer() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ containerId, nodeId }: { containerId: string; nodeId: string }) =>
            apiClient.post<{ message: string; container_id: string }>(
                `/api/system/containers/${containerId}/stop`,
                undefined,
                { node_id: nodeId },
            ),
        onSuccess: () => {
            // Refresh system stats after container action
            queryClient.invalidateQueries({ queryKey: ['system', 'stats'] })
        },
    })
}

export function useDeleteContainer() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ containerId, nodeId }: { containerId: string; nodeId: string }) =>
            apiClient.delete<{ message: string; container_id: string }>(`/api/system/containers/${containerId}`, {
                node_id: nodeId,
            }),
        onSuccess: () => {
            // Refresh system stats after container deletion
            queryClient.invalidateQueries({ queryKey: ['system', 'stats'] })
        },
    })
}
