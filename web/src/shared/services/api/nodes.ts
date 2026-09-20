import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../../lib/api-client'
import type { Node, RegisterNodeRequest } from '../../types/api'

// Node management API
export function useNodes(options?: { refetchInterval?: number | false }) {
    return useQuery<Node[]>({
        queryKey: ['nodes'],
        queryFn: () => apiClient.get<Node[]>('/api/nodes'),
        refetchInterval: options?.refetchInterval,
        // Waiting for a machine to join means leaving this tab to work on the other machine, so keep checking.
        refetchIntervalInBackground: !!options?.refetchInterval,
    })
}

export interface JoinToken {
    token: string
    expires_at: string
    usage: string
}

// A single-use token that lets a new machine add itself to the cluster.
export function useCreateJoinToken() {
    return useMutation({
        mutationFn: () => apiClient.post<JoinToken, Record<string, never>>('/api/nodes/join-tokens', {}),
    })
}

export function useRegisterNode() {
    const queryClient = useQueryClient()
    return useMutation({
        mutationFn: (data: RegisterNodeRequest) => apiClient.post<Node, RegisterNodeRequest>('/api/nodes', data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['nodes'] })
        },
    })
}

export function useDeleteNode() {
    const queryClient = useQueryClient()
    return useMutation({
        // Force also drops the records of apps on a node that is not answering, which cannot be cleaned up there.
        mutationFn: ({ id, force = false }: { id: string; force?: boolean }) =>
            apiClient.delete<{ message: string; nodeID: string }>(`/api/nodes/${id}${force ? '?force=true' : ''}`),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['nodes'] })
        },
    })
}

export function useNodeHealth(id: string) {
    return useMutation({
        mutationFn: () => apiClient.get<{ message: string; nodeID: string }>(`/api/nodes/${id}/health`),
    })
}

// Get current node info
export function useCurrentNode() {
    return useQuery<Node>({
        queryKey: ['current-node'],
        queryFn: () => apiClient.get<Node>('/api/node/info'),
        staleTime: 60000, // Cache for 1 minute
    })
}
