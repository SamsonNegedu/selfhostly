import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../../lib/api-client'
import type { Settings, UpdateSettingsRequest } from '../../types/api'

// Settings API
export function useSettings() {
    return useQuery<Settings>({
        queryKey: ['settings'],
        queryFn: () => apiClient.get<Settings>('/api/settings'),
    })
}

export function useUpdateSettings() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (data: UpdateSettingsRequest) =>
            apiClient.put<Settings, UpdateSettingsRequest>('/api/settings', data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['settings'] })
            // Whether a provider counts as connected depends on the saved credentials.
            queryClient.invalidateQueries({ queryKey: ['tunnels', 'providers'] })
        },
    })
}

export interface AuditEntry {
    id: number
    time: string
    actor: string
    method: string
    path: string
    status: number
    remote_addr: string
    /** What was done and to what, when the server knows. Empty for requests it does not describe. */
    action: string
    target_type: string
    target_id: string
    target_name: string
}

// The most recent state-changing requests, newest first.
export function useAuditLog(limit = 50) {
    return useQuery<AuditEntry[]>({
        queryKey: ['security', 'audit', limit],
        queryFn: async () => (await apiClient.get<AuditEntry[] | null>('/api/security/audit', { limit })) ?? [],
    })
}

// Signs everyone out, on every device, including this one.
export function useRevokeSessions() {
    return useMutation({
        mutationFn: () => apiClient.post<{ message: string; revoked_before: string }>('/api/security/revoke-sessions'),
    })
}
