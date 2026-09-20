import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../../lib/api-client'
import type { AppSchedule, UpdateScheduleRequest, ScheduleNextRuns } from '../../types/api'

// Schedule API
export function useAppSchedule(appId: string, nodeId: string) {
    return useQuery<AppSchedule | null>({
        queryKey: ['app-schedule', appId, nodeId],
        queryFn: () => apiClient.get<AppSchedule | null>(`/api/apps/${appId}/schedule`, { node_id: nodeId }),
        enabled: !!appId && !!nodeId,
    })
}

export function useUpdateAppSchedule(appId: string, nodeId: string) {
    const queryClient = useQueryClient()
    return useMutation<AppSchedule, Error, UpdateScheduleRequest>({
        mutationFn: (data) =>
            apiClient.post<AppSchedule, UpdateScheduleRequest>(`/api/apps/${appId}/schedule`, data, {
                node_id: nodeId,
            }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['apps'] })
            queryClient.invalidateQueries({ queryKey: ['app', appId] })
            queryClient.invalidateQueries({ queryKey: ['app-schedule', appId] })
        },
    })
}

export function useDeleteAppSchedule(appId: string, nodeId: string) {
    const queryClient = useQueryClient()
    return useMutation<null, Error, void>({
        mutationFn: () => apiClient.delete<null>(`/api/apps/${appId}/schedule`, { node_id: nodeId }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['apps'] })
            queryClient.invalidateQueries({ queryKey: ['app', appId] })
            queryClient.invalidateQueries({ queryKey: ['app-schedule', appId] })
        },
    })
}

export function useTestSchedule() {
    return useMutation<ScheduleNextRuns, Error, UpdateScheduleRequest & { app_id?: string; node_id?: string }>({
        mutationFn: (data) => {
            const appId = data.app_id || ''
            const nodeId = data.node_id || ''
            const { app_id, node_id, ...scheduleData } = data
            return apiClient.post<ScheduleNextRuns, UpdateScheduleRequest>(
                `/api/apps/${appId}/schedule/test`,
                scheduleData,
                { node_id: nodeId },
            )
        },
    })
}

export function useScheduleNextRuns(appId: string, nodeId: string) {
    return useQuery<ScheduleNextRuns>({
        queryKey: ['schedule-next-runs', appId, nodeId],
        queryFn: () => apiClient.get<ScheduleNextRuns>(`/api/apps/${appId}/schedule/next-runs`, { node_id: nodeId }),
        enabled: !!appId && !!nodeId,
        refetchInterval: 60000, // Refresh every minute
    })
}
