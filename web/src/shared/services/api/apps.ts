import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useAppStore } from '../../stores/app-store'
import { apiClient } from '../../lib/api-client'
import { useNodeContext } from '../../contexts/NodeContext'
import type {
    App,
    CreateAppRequest,
    UpdateAppRequest,
    ComposeVersion,
    RollbackRequest,
    Job,
    JobResponse,
} from '../../types/api'

// Apps API
export function useApps(nodeIdsOverride?: string[]) {
    const { selectedNodeIds: globalNodeIds } = useNodeContext()

    // Use override if provided, otherwise use global context
    const nodeIds = nodeIdsOverride ?? globalNodeIds

    // Build query key with node filter
    const queryKey = nodeIds && nodeIds.length > 0 ? ['apps', { nodeIds }] : ['apps']

    return useQuery<App[]>({
        queryKey,
        // an install with no apps used to answer null, which a default value (= []) does not replace
        select: (apps) => apps ?? [],
        queryFn: () => {
            // Build node_ids parameter
            if (nodeIds && nodeIds.length > 0) {
                const nodeIdsParam = nodeIds.join(',')
                return apiClient.get<App[]>('/api/apps', { node_ids: nodeIdsParam })
            }
            // Default: fetch from all nodes
            return apiClient.get<App[]>('/api/apps', { node_ids: 'all' })
        },
    })
}

export function useApp(id: string, nodeId: string) {
    return useQuery<App>({
        queryKey: ['app', id, nodeId],
        queryFn: () => {
            return apiClient.get<App>(`/api/apps/${id}`, { node_id: nodeId })
        },
        enabled: !!id && !!nodeId, // Require nodeId to be present before fetching
    })
}

export function useCreateApp() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (data: CreateAppRequest) => apiClient.post<App, CreateAppRequest>('/api/apps', data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['apps'] })
        },
    })
}

export function useUpdateApp(id: string, nodeId: string) {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (data: UpdateAppRequest) => {
            return apiClient.put<App, UpdateAppRequest>(`/api/apps/${id}?node_id=${nodeId}`, data)
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['apps'] })
            queryClient.invalidateQueries({ queryKey: ['app', id] })
        },
    })
}

export function useDeleteApp() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ id, nodeId }: { id: string; nodeId: string }) => {
            return apiClient.delete<{ message: string; appID: string }>(`/api/apps/${id}?node_id=${nodeId}`)
        },
        // Optimistic update - remove from cache immediately
        onMutate: async ({ id }) => {
            // Cancel any outgoing refetches
            await queryClient.cancelQueries({ queryKey: ['apps'] })

            // Snapshot previous value
            const previousApps = queryClient.getQueryData(['apps'])

            // Remove the deleted app from cache
            if (previousApps) {
                queryClient.setQueryData(['apps'], (previousApps: any[]) =>
                    previousApps.filter((app: any) => app.id !== id),
                )
            }

            return { previousApps }
        },
        // Rollback in case of error
        onError: (_err, _variables, context: any) => {
            if (context?.previousApps) {
                queryClient.setQueryData(['apps'], context.previousApps)
            }
        },
        // Refetch after error or success
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: ['apps'] })
        },
    })
}

export function useStartApp() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ id, nodeId }: { id: string; nodeId: string }) => {
            return apiClient.post<App>(`/api/apps/${id}/start?node_id=${nodeId}`)
        },
        onMutate: async ({ id }) => {
            // Cancel any outgoing refetches
            await queryClient.cancelQueries({ queryKey: ['apps'] })
            await queryClient.cancelQueries({ queryKey: ['app', id] })

            // Snapshot previous values
            const previousApps = queryClient.getQueryData(['apps'])
            const previousApp = queryClient.getQueryData(['app', id])

            // Optimistically update both cache and store
            if (previousApp) {
                const updatedApp = { ...previousApp, status: 'running' as const }
                queryClient.setQueryData(['app', id], updatedApp)
            }

            if (previousApps) {
                queryClient.setQueryData(['apps'], (previousApps: any[]) =>
                    previousApps.map((app: any) => (app.id === id ? { ...app, status: 'running' as const } : app)),
                )
            }

            // Update Zustand store optimistically
            useAppStore.getState().updateApp(id, { status: 'running' })

            return { previousApps, previousApp }
        },
        onError: (_err, { id }, context: any) => {
            // Rollback both cache and store on error
            if (context?.previousApps) {
                queryClient.setQueryData(['apps'], context.previousApps)
            }
            if (context?.previousApp) {
                queryClient.setQueryData(['app', id], context.previousApp)
            }
            useAppStore.getState().updateApp(id, context?.previousApp)
        },
        onSettled: () => {
            // Refetch data after mutation settles
            queryClient.invalidateQueries({ queryKey: ['apps'] })
        },
    })
}

export function useStopApp() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ id, nodeId }: { id: string; nodeId: string }) => {
            return apiClient.post<App>(`/api/apps/${id}/stop?node_id=${nodeId}`)
        },
        onMutate: async ({ id }) => {
            // Cancel any outgoing refetches
            await queryClient.cancelQueries({ queryKey: ['apps'] })
            await queryClient.cancelQueries({ queryKey: ['app', id] })

            // Snapshot previous values
            const previousApps = queryClient.getQueryData(['apps'])
            const previousApp = queryClient.getQueryData(['app', id])

            // Optimistically update both cache and store
            if (previousApp) {
                const updatedApp = { ...previousApp, status: 'stopped' as const }
                queryClient.setQueryData(['app', id], updatedApp)
            }

            if (previousApps) {
                queryClient.setQueryData(['apps'], (previousApps: any[]) =>
                    previousApps.map((app: any) => (app.id === id ? { ...app, status: 'stopped' as const } : app)),
                )
            }

            // Update Zustand store optimistically
            useAppStore.getState().updateApp(id, { status: 'stopped' })

            return { previousApps, previousApp }
        },
        onError: (_err, { id }, context: any) => {
            // Rollback both cache and store on error
            if (context?.previousApps) {
                queryClient.setQueryData(['apps'], context.previousApps)
            }
            if (context?.previousApp) {
                queryClient.setQueryData(['app', id], context.previousApp)
            }
            useAppStore.getState().updateApp(id, context?.previousApp)
        },
        onSuccess: () => {
            // Show success notification
            console.log('App stopped successfully')
        },
        onSettled: () => {
            // Refetch data after mutation settles
            queryClient.invalidateQueries({ queryKey: ['apps'] })
        },
    })
}

export function useUpdateAppContainers() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ id, nodeId }: { id: string; nodeId: string }) => {
            return apiClient.post<JobResponse>(`/api/apps/${id}/update?node_id=${nodeId}`)
        },
        onMutate: async ({ id }) => {
            // Cancel any outgoing refetches
            await queryClient.cancelQueries({ queryKey: ['apps'] })
            await queryClient.cancelQueries({ queryKey: ['app', id] })

            // Snapshot previous values
            const previousApps = queryClient.getQueryData(['apps'])
            const previousApp = queryClient.getQueryData(['app', id])

            // Optimistically update status to 'updating' in both cache and store
            if (previousApp) {
                const updatedApp = { ...previousApp, status: 'updating' as const }
                queryClient.setQueryData(['app', id], updatedApp)
            }

            if (previousApps) {
                queryClient.setQueryData(['apps'], (previousApps: any[]) =>
                    previousApps.map((app: any) => (app.id === id ? { ...app, status: 'updating' as const } : app)),
                )
            }

            // Update Zustand store optimistically
            useAppStore.getState().updateApp(id, { status: 'updating' })

            return { previousApps, previousApp }
        },
        onSuccess: (_, { id, nodeId }) => {
            // Invalidate jobs query so AppActions picks up the new job and shows progress
            queryClient.invalidateQueries({ queryKey: ['jobs', 'app', id, nodeId] })
        },
        onError: (_err, { id }, context: any) => {
            // Rollback both cache and store on error
            if (context?.previousApps) {
                queryClient.setQueryData(['apps'], context.previousApps)
            }
            if (context?.previousApp) {
                queryClient.setQueryData(['app', id], context.previousApp)
            }
            useAppStore.getState().updateApp(id, context?.previousApp)
        },
    })
}

// Restart a specific service within an app
export function useRestartAppService() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ appId, nodeId, serviceName }: { appId: string; nodeId: string; serviceName: string }) => {
            return apiClient.post<{ message: string; service: string }>(
                `/api/apps/${appId}/services/${serviceName}/restart?node_id=${nodeId}`,
            )
        },
        onSuccess: (_, { appId }) => {
            // Invalidate app stats and services to refresh the UI
            queryClient.invalidateQueries({ queryKey: ['app-stats', appId] })
            queryClient.invalidateQueries({ queryKey: ['app-services', appId] })
        },
    })
}

// Compose Versions API
export function useComposeVersions(appId: string, nodeId: string) {
    return useQuery<ComposeVersion[]>({
        queryKey: ['compose-versions', appId, nodeId],
        queryFn: () => {
            return apiClient.get<ComposeVersion[]>(`/api/apps/${appId}/compose/versions?node_id=${nodeId}`)
        },
        enabled: !!appId && !!nodeId,
    })
}

export function useRollbackToVersion(appId: string, nodeId: string) {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ version, change_reason }: { version: number; change_reason?: string }) => {
            const body: RollbackRequest = change_reason ? { change_reason } : {}
            return apiClient.post<{ message: string; app: App; new_version: ComposeVersion }>(
                `/api/apps/${appId}/compose/rollback/${version}?node_id=${nodeId}`,
                body,
            )
        },
        onSuccess: () => {
            // Invalidate related queries
            queryClient.invalidateQueries({ queryKey: ['app', appId] })
            queryClient.invalidateQueries({ queryKey: ['apps'] })
            queryClient.invalidateQueries({ queryKey: ['compose-versions', appId] })
        },
    })
}

// ============================================================================
// Job API
// ============================================================================

// Get recent jobs for an app
export function useAppJobs(appId: string, nodeId: string) {
    return useQuery<Job[]>({
        queryKey: ['jobs', 'app', appId, nodeId],
        queryFn: () => apiClient.get<Job[]>(`/api/apps/${appId}/jobs`, { node_id: nodeId }),
        enabled: !!appId && !!nodeId,
        refetchInterval: (query) => {
            // Check if there are any active jobs (pending or running)
            const data = query.state.data
            if (!data) return 5000 // Poll every 5 seconds if no data yet

            const hasActiveJob = data.some((job) => job.status === 'pending' || job.status === 'running')
            if (hasActiveJob) {
                return 5000 // Continue polling if there's an active job
            }
            return false // Stop polling when no active jobs
        },
    })
}

// Get list of services for an app
export function useAppServices(appId: string, nodeId: string, queryEnabled = true) {
    return useQuery<string[]>({
        queryKey: ['app-services', appId, nodeId],
        queryFn: () => apiClient.get<string[]>(`/api/apps/${appId}/services`, { node_id: nodeId }),
        enabled: !!appId && !!nodeId && queryEnabled,
    })
}
