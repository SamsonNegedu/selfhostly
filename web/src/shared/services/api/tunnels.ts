import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../../lib/api-client'
import type {
    CloudflareTunnelResponse,
    TunnelByAppResponse,
    ProviderFeatures,
    TunnelProvidersResponse,
    JobResponse,
} from '../../types/api'

interface IngressRule {
    hostname?: string | null
    service: string
    path?: string | null
    originRequest?: Record<string, any>
}

// ============================================================================
// Provider-Agnostic Tunnel Hooks
// ============================================================================

// List all available tunnel providers
export function useProviders() {
    return useQuery({
        queryKey: ['tunnels', 'providers'],
        queryFn: () => apiClient.get<TunnelProvidersResponse>('/api/tunnels/providers'),
    })
}

// Get features supported by a specific provider
export function useProviderFeatures(provider: string) {
    return useQuery({
        queryKey: ['tunnels', 'providers', provider, 'features'],
        queryFn: () => apiClient.get<ProviderFeatures>(`/api/tunnels/providers/${provider}/features`),
        enabled: !!provider,
    })
}

// List all tunnels (provider-agnostic)
export function useTunnels(nodeIds?: string[]) {
    return useQuery({
        queryKey: ['tunnels', 'list', nodeIds],
        queryFn: () => {
            if (nodeIds && nodeIds.length > 0) {
                const nodeIdsParam = nodeIds.join(',')
                return apiClient.get<CloudflareTunnelResponse>('/api/tunnels', { node_ids: nodeIdsParam })
            }
            return apiClient.get<CloudflareTunnelResponse>('/api/tunnels', { node_ids: 'all' })
        },
    })
}

// Get tunnel for specific app (provider-agnostic). When no tunnel, returns 200 with tunnel: null and app_id, tunnel_mode, node_id.
export function useTunnel(appId: string, nodeId: string) {
    return useQuery({
        queryKey: ['tunnels', 'app', appId, nodeId],
        queryFn: () => {
            return apiClient.get<TunnelByAppResponse>(`/api/tunnels/apps/${appId}?node_id=${nodeId}`)
        },
        enabled: !!appId && !!nodeId,
    })
}

// Create named (custom domain) tunnel for an app that has none
export function useCreateTunnelForApp() {
    const queryClient = useQueryClient()
    return useMutation({
        mutationFn: ({
            appId,
            nodeId,
            ingressRules,
        }: {
            appId: string
            nodeId: string
            ingressRules?: IngressRule[]
        }) => {
            const body = ingressRules && ingressRules.length > 0 ? { ingress_rules: ingressRules } : {}
            return apiClient.post<JobResponse>(`/api/tunnels/apps/${appId}?node_id=${nodeId}`, body)
        },
        onSuccess: (_, variables) => {
            // Invalidate jobs query so AppActions picks up the new job and shows progress
            queryClient.invalidateQueries({ queryKey: ['jobs', 'app', variables.appId, variables.nodeId] })
            // Note: Don't invalidate app/tunnel queries here - let the job completion handler do it
        },
    })
}

// Create Quick Tunnel (temporary trycloudflare.com URL) for an app that has no tunnel
export function useCreateQuickTunnelForApp() {
    const queryClient = useQueryClient()
    return useMutation({
        mutationFn: ({
            appId,
            nodeId,
            service,
            port,
        }: {
            appId: string
            nodeId: string
            service: string
            port: number
        }) => {
            return apiClient.post<JobResponse>(`/api/apps/${appId}/quick-tunnel?node_id=${nodeId}`, { service, port })
        },
        onSuccess: (_, variables) => {
            // Invalidate jobs query so AppActions picks up the new job and shows progress
            queryClient.invalidateQueries({ queryKey: ['jobs', 'app', variables.appId, variables.nodeId] })
            // Note: Don't invalidate app/tunnel queries here - let the job completion handler do it
        },
    })
}

// Switch app from Quick Tunnel to custom (named) tunnel. Now async - ingress rules must be applied after.
export function useSwitchAppToCustomTunnel() {
    const queryClient = useQueryClient()
    return useMutation({
        mutationFn: ({
            appId,
            nodeId,
            ingressRules,
        }: {
            appId: string
            nodeId: string
            ingressRules?: IngressRule[]
        }) => {
            const body = ingressRules && ingressRules.length > 0 ? { ingress_rules: ingressRules } : {}
            return apiClient.post<JobResponse>(`/api/tunnels/apps/${appId}/switch-to-custom?node_id=${nodeId}`, body)
        },
        onSuccess: (_, variables) => {
            // Invalidate jobs query so AppActions picks up the new job and shows progress
            queryClient.invalidateQueries({ queryKey: ['jobs', 'app', variables.appId, variables.nodeId] })
            // Note: Don't invalidate app/tunnel queries here - let the job completion handler do it
        },
    })
}

// Sync tunnel status (provider-agnostic, may return 501 if not supported)
export function useSyncTunnel() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ appId, nodeId }: { appId: string; nodeId: string }) => {
            return apiClient.post<{ message: string }>(`/api/tunnels/apps/${appId}/sync?node_id=${nodeId}`, {})
        },
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({ queryKey: ['tunnels', 'app', variables.appId] })
            queryClient.invalidateQueries({ queryKey: ['tunnels', 'list'] })
        },
    })
}

// Update tunnel ingress (provider-agnostic, may return 501 if not supported)
export function useUpdateTunnelIngress() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({
            appId,
            nodeId,
            ingressRules,
            hostname,
            targetDomain,
        }: {
            appId: string
            nodeId: string
            ingressRules: IngressRule[]
            hostname?: string
            targetDomain?: string
        }) => {
            return apiClient.put<{ message: string }>(`/api/tunnels/apps/${appId}/ingress?node_id=${nodeId}`, {
                ingress_rules: ingressRules,
                hostname,
                target_domain: targetDomain,
            })
        },
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({ queryKey: ['tunnels', 'app', variables.appId] })
            queryClient.invalidateQueries({ queryKey: ['tunnels', 'list'] })
            queryClient.invalidateQueries({ queryKey: ['app', variables.appId] })
        },
    })
}

// Create DNS record (provider-agnostic, may return 501 if not supported)
export function useCreateTunnelDNSRecord() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ appId, nodeId, hostname }: { appId: string; nodeId: string; hostname: string }) => {
            return apiClient.post<{ message: string; hostname: string }>(
                `/api/tunnels/apps/${appId}/dns?node_id=${nodeId}`,
                { hostname },
            )
        },
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({ queryKey: ['tunnels', 'app', variables.appId] })
            queryClient.invalidateQueries({ queryKey: ['tunnels', 'list'] })
            queryClient.invalidateQueries({ queryKey: ['app', variables.appId] })
        },
    })
}

// Delete tunnel (provider-agnostic)
export function useDeleteTunnel() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ appId, nodeId }: { appId: string; nodeId: string }) => {
            return apiClient.delete<JobResponse>(`/api/tunnels/apps/${appId}?node_id=${nodeId}`)
        },
        onSuccess: (_, variables) => {
            // Invalidate jobs query so AppActions picks up the new job and shows progress
            queryClient.invalidateQueries({ queryKey: ['jobs', 'app', variables.appId, variables.nodeId] })
            // Invalidate tunnels list to immediately update UI
            queryClient.invalidateQueries({ queryKey: ['tunnels', 'app', variables.appId] })
            queryClient.invalidateQueries({ queryKey: ['tunnels', 'list'] })
            // Also invalidate app query to refresh app details
            queryClient.invalidateQueries({ queryKey: ['app', variables.appId] })
        },
    })
}
