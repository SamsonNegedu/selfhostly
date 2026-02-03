import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAppStore } from '../stores/app-store';
import { apiClient } from '../lib/api-client';
import { useNodeContext } from '../contexts/NodeContext';

// Export query client hooks for components that need them
export { useQueryClient };
import type {
  App,
  CreateAppRequest,
  UpdateAppRequest,
  Settings,
  UpdateSettingsRequest,
  CloudflareTunnelResponse,
  TunnelByAppResponse,
  ComposeVersion,
  RollbackRequest,
  SystemStats,
  Node,
  RegisterNodeRequest,
  UpdateNodeRequest,
  ProviderFeatures,
  TunnelProvidersResponse,
  Job,
  JobResponse,
} from '../types/api';

interface IngressRule {
  hostname?: string | null
  service: string
  path?: string | null
  originRequest?: Record<string, any>
}


// User type from go-pkgz/auth
export interface User {
  id: string;
  name: string;
  picture?: string;
}

// Apps API
export function useApps(nodeIdsOverride?: string[]) {
  const { selectedNodeIds: globalNodeIds } = useNodeContext();
  
  // Use override if provided, otherwise use global context
  const nodeIds = nodeIdsOverride ?? globalNodeIds;
  
  // Build query key with node filter
  const queryKey = nodeIds && nodeIds.length > 0 
    ? ['apps', { nodeIds }] 
    : ['apps'];
  
  return useQuery<App[]>({
    queryKey,
    queryFn: () => {
      // Build node_ids parameter
      if (nodeIds && nodeIds.length > 0) {
        const nodeIdsParam = nodeIds.join(',');
        return apiClient.get<App[]>('/api/apps', { node_ids: nodeIdsParam });
      }
      // Default: fetch from all nodes
      return apiClient.get<App[]>('/api/apps', { node_ids: 'all' });
    },
  });
}

export function useApp(id: string, nodeId: string) {
  return useQuery<App>({
    queryKey: ['app', id, nodeId],
    queryFn: () => {
      return apiClient.get<App>(`/api/apps/${id}`, { node_id: nodeId });
    },
    enabled: !!id && !!nodeId, // Require nodeId to be present before fetching
  });
}

export function useCreateApp() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: CreateAppRequest) => apiClient.post<App, CreateAppRequest>('/api/apps', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useUpdateApp(id: string, nodeId: string) {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: UpdateAppRequest) => {
      return apiClient.put<App, UpdateAppRequest>(`/api/apps/${id}?node_id=${nodeId}`, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apps'] });
      queryClient.invalidateQueries({ queryKey: ['app', id] });
    },
  });
}

export function useDeleteApp() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, nodeId }: { id: string; nodeId: string }) => {
      return apiClient.delete<{ message: string; appID: string }>(`/api/apps/${id}?node_id=${nodeId}`);
    },
    // Optimistic update - remove from cache immediately
    onMutate: async ({ id }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['apps'] });
      
      // Snapshot previous value
      const previousApps = queryClient.getQueryData(['apps']);
      
      // Remove the deleted app from cache
      if (previousApps) {
        queryClient.setQueryData(['apps'], (previousApps: any[]) => previousApps.filter((app: any) => app.id !== id));
      }
      
      return { previousApps };
    },
    // Rollback in case of error
    onError: (_err, _variables, context: any) => {
      if (context?.previousApps) {
        queryClient.setQueryData(['apps'], context.previousApps);
      }
    },
    // Refetch after error or success
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useStartApp() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, nodeId }: { id: string; nodeId: string }) => {
      return apiClient.post<App>(`/api/apps/${id}/start?node_id=${nodeId}`);
    },
    onMutate: async ({ id }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['apps'] });
      await queryClient.cancelQueries({ queryKey: ['app', id] });
      
      // Snapshot previous values
      const previousApps = queryClient.getQueryData(['apps']);
      const previousApp = queryClient.getQueryData(['app', id]);
      
      // Optimistically update both cache and store
      if (previousApp) {
        const updatedApp = { ...previousApp, status: 'running' as const };
        queryClient.setQueryData(['app', id], updatedApp);
      }
      
      if (previousApps) {
        queryClient.setQueryData(['apps'], (previousApps: any[]) =>
          previousApps.map((app: any) => (app.id === id ? { ...app, status: 'running' as const } : app))
        );
      }
      
      // Update Zustand store optimistically
      useAppStore.getState().updateApp(id, { status: 'running' });
      
      return { previousApps, previousApp };
    },
    onError: (_err, { id }, context: any) => {
      // Rollback both cache and store on error
      if (context?.previousApps) {
        queryClient.setQueryData(['apps'], context.previousApps);
      }
      if (context?.previousApp) {
        queryClient.setQueryData(['app', id], context.previousApp);
      }
      useAppStore.getState().updateApp(id, context?.previousApp);
    },
    onSettled: () => {
      // Refetch data after mutation settles
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useStopApp() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, nodeId }: { id: string; nodeId: string }) => {
      return apiClient.post<App>(`/api/apps/${id}/stop?node_id=${nodeId}`);
    },
    onMutate: async ({ id }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['apps'] });
      await queryClient.cancelQueries({ queryKey: ['app', id] });
      
      // Snapshot previous values
      const previousApps = queryClient.getQueryData(['apps']);
      const previousApp = queryClient.getQueryData(['app', id]);
      
      // Optimistically update both cache and store
      if (previousApp) {
        const updatedApp = { ...previousApp, status: 'stopped' as const };
        queryClient.setQueryData(['app', id], updatedApp);
      }
      
      if (previousApps) {
        queryClient.setQueryData(['apps'], (previousApps: any[]) =>
          previousApps.map((app: any) => (app.id === id ? { ...app, status: 'stopped' as const } : app))
        );
      }
      
      // Update Zustand store optimistically
      useAppStore.getState().updateApp(id, { status: 'stopped' });
      
      return { previousApps, previousApp };
    },
    onError: (_err, { id }, context: any) => {
      // Rollback both cache and store on error
      if (context?.previousApps) {
        queryClient.setQueryData(['apps'], context.previousApps);
      }
      if (context?.previousApp) {
        queryClient.setQueryData(['app', id], context.previousApp);
      }
      useAppStore.getState().updateApp(id, context?.previousApp);
    },
    onSuccess: () => {
      // Show success notification
      console.log('App stopped successfully');
    },
    onSettled: () => {
      // Refetch data after mutation settles
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useUpdateAppContainers() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, nodeId }: { id: string; nodeId: string }) => {
      return apiClient.post<JobResponse>(`/api/apps/${id}/update?node_id=${nodeId}`);
    },
    onMutate: async ({ id }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['apps'] });
      await queryClient.cancelQueries({ queryKey: ['app', id] });
      
      // Snapshot previous values
      const previousApps = queryClient.getQueryData(['apps']);
      const previousApp = queryClient.getQueryData(['app', id]);
      
      // Optimistically update status to 'updating' in both cache and store
      if (previousApp) {
        const updatedApp = { ...previousApp, status: 'updating' as const };
        queryClient.setQueryData(['app', id], updatedApp);
      }
      
      if (previousApps) {
        queryClient.setQueryData(['apps'], (previousApps: any[]) =>
          previousApps.map((app: any) => (app.id === id ? { ...app, status: 'updating' as const } : app))
        );
      }
      
      // Update Zustand store optimistically
      useAppStore.getState().updateApp(id, { status: 'updating' });
      
      return { previousApps, previousApp };
    },
    onSuccess: (_, { id, nodeId }) => {
      // Invalidate jobs query so AppActions picks up the new job and shows progress
      queryClient.invalidateQueries({ queryKey: ['jobs', 'app', id, nodeId] });
    },
    onError: (_err, { id }, context: any) => {
      // Rollback both cache and store on error
      if (context?.previousApps) {
        queryClient.setQueryData(['apps'], context.previousApps);
      }
      if (context?.previousApp) {
        queryClient.setQueryData(['app', id], context.previousApp);
      }
      useAppStore.getState().updateApp(id, context?.previousApp);
    },
  });
}

// Compose Versions API
export function useComposeVersions(appId: string, nodeId: string) {
  return useQuery<ComposeVersion[]>({
    queryKey: ['compose-versions', appId, nodeId],
    queryFn: () => {
      return apiClient.get<ComposeVersion[]>(`/api/apps/${appId}/compose/versions?node_id=${nodeId}`);
    },
    enabled: !!appId && !!nodeId,
  });
}

export function useComposeVersion(appId: string, version: number, nodeId: string) {
  return useQuery<ComposeVersion>({
    queryKey: ['compose-version', appId, version, nodeId],
    queryFn: () => {
      return apiClient.get<ComposeVersion>(`/api/apps/${appId}/compose/versions/${version}?node_id=${nodeId}`);
    },
    enabled: !!appId && version > 0 && !!nodeId,
  });
}

export function useRollbackToVersion(appId: string, nodeId: string) {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ version, change_reason }: { version: number; change_reason?: string }) => {
      const body: RollbackRequest = change_reason ? { change_reason } : {};
      return apiClient.post<{ message: string; app: App; new_version: ComposeVersion }>(`/api/apps/${appId}/compose/rollback/${version}?node_id=${nodeId}`, body);
    },
    onSuccess: () => {
      // Invalidate related queries
      queryClient.invalidateQueries({ queryKey: ['app', appId] });
      queryClient.invalidateQueries({ queryKey: ['apps'] });
      queryClient.invalidateQueries({ queryKey: ['compose-versions', appId] });
    },
  });
}

// Settings API
export function useSettings() {
  return useQuery<Settings>({
    queryKey: ['settings'],
    queryFn: () => apiClient.get<Settings>('/api/settings'),
  });
}

export function useUpdateSettings() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: UpdateSettingsRequest) => apiClient.put<Settings, UpdateSettingsRequest>('/api/settings', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
  });
}

// Auth API - GitHub OAuth via go-pkgz/auth
// Auth endpoints:
//   - GET /auth/github/login - Redirects to GitHub for OAuth
//   - GET /auth/logout - Clears session and logs out
//   - GET /api/me - Get current user info

// Backend URL for auth redirects (browser navigation bypasses Vite proxy)
const AUTH_URL = import.meta.env?.DEV ? 'http://localhost:8080' : '';

// Frontend URL for redirects after auth
const FRONTEND_URL = import.meta.env?.DEV ? 'http://localhost:5173' : window.location.origin;

// Get current authenticated user
export function useCurrentUser() {
  return useQuery<User | null>({
    queryKey: ['currentUser'],
    queryFn: async () => {
      try {
        return await apiClient.get<User>('/api/me');
      } catch (error) {
        // Return null for 401, but let 404 bubble up (auth disabled)
        if (error instanceof Error && error.message === 'UNAUTHORIZED') {
          return null;
        }
        throw error;
      }
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: false,
  });
}

// ============================================================================
// Provider-Agnostic Tunnel Hooks
// ============================================================================

// List all available tunnel providers
export function useProviders() {
  return useQuery({
    queryKey: ['tunnels', 'providers'],
    queryFn: () => apiClient.get<TunnelProvidersResponse>('/api/tunnels/providers'),
  });
}

// Get features supported by a specific provider
export function useProviderFeatures(provider: string) {
  return useQuery({
    queryKey: ['tunnels', 'providers', provider, 'features'],
    queryFn: () => apiClient.get<ProviderFeatures>(`/api/tunnels/providers/${provider}/features`),
    enabled: !!provider,
  });
}

// List all tunnels (provider-agnostic)
export function useTunnels(nodeIds?: string[]) {
  return useQuery({
    queryKey: ['tunnels', 'list', nodeIds],
    queryFn: () => {
      if (nodeIds && nodeIds.length > 0) {
        const nodeIdsParam = nodeIds.join(',');
        return apiClient.get<CloudflareTunnelResponse>('/api/tunnels', { node_ids: nodeIdsParam });
      }
      return apiClient.get<CloudflareTunnelResponse>('/api/tunnels', { node_ids: 'all' });
    },
  });
}

// Get tunnel for specific app (provider-agnostic). When no tunnel, returns 200 with tunnel: null and app_id, tunnel_mode, node_id.
export function useTunnel(appId: string, nodeId: string) {
  return useQuery({
    queryKey: ['tunnels', 'app', appId, nodeId],
    queryFn: () => {
      return apiClient.get<TunnelByAppResponse>(`/api/tunnels/apps/${appId}?node_id=${nodeId}`);
    },
    enabled: !!appId && !!nodeId,
  });
}

// Create named (custom domain) tunnel for an app that has none
export function useCreateTunnelForApp() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ appId, nodeId, ingressRules }: { appId: string; nodeId: string; ingressRules?: IngressRule[] }) => {
      const body = ingressRules && ingressRules.length > 0 ? { ingress_rules: ingressRules } : {};
      return apiClient.post<JobResponse>(`/api/tunnels/apps/${appId}?node_id=${nodeId}`, body);
    },
    onSuccess: (_, variables) => {
      // Invalidate jobs query so AppActions picks up the new job and shows progress
      queryClient.invalidateQueries({ queryKey: ['jobs', 'app', variables.appId, variables.nodeId] });
      // Note: Don't invalidate app/tunnel queries here - let the job completion handler do it
    },
  });
}

// Create Quick Tunnel (temporary trycloudflare.com URL) for an app that has no tunnel
export function useCreateQuickTunnelForApp() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ appId, nodeId, service, port }: { appId: string; nodeId: string; service: string; port: number }) => {
      return apiClient.post<JobResponse>(`/api/apps/${appId}/quick-tunnel?node_id=${nodeId}`, { service, port });
    },
    onSuccess: (_, variables) => {
      // Invalidate jobs query so AppActions picks up the new job and shows progress
      queryClient.invalidateQueries({ queryKey: ['jobs', 'app', variables.appId, variables.nodeId] });
      // Note: Don't invalidate app/tunnel queries here - let the job completion handler do it
    },
  });
}

// Switch app from Quick Tunnel to custom (named) tunnel. Now async - ingress rules must be applied after.
export function useSwitchAppToCustomTunnel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ appId, nodeId, ingressRules }: { appId: string; nodeId: string; ingressRules?: IngressRule[] }) => {
      const body = ingressRules && ingressRules.length > 0 ? { ingress_rules: ingressRules } : {};
      return apiClient.post<JobResponse>(`/api/tunnels/apps/${appId}/switch-to-custom?node_id=${nodeId}`, body);
    },
    onSuccess: (_, variables) => {
      // Invalidate jobs query so AppActions picks up the new job and shows progress
      queryClient.invalidateQueries({ queryKey: ['jobs', 'app', variables.appId, variables.nodeId] });
      // Note: Don't invalidate app/tunnel queries here - let the job completion handler do it
    },
  });
}

// Sync tunnel status (provider-agnostic, may return 501 if not supported)
export function useSyncTunnel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ appId, nodeId }: { appId: string; nodeId: string }) => {
      return apiClient.post<{ message: string }>(`/api/tunnels/apps/${appId}/sync?node_id=${nodeId}`, {});
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['tunnels', 'app', variables.appId] });
      queryClient.invalidateQueries({ queryKey: ['tunnels', 'list'] });
    },
  });
}

// Update tunnel ingress (provider-agnostic, may return 501 if not supported)
export function useUpdateTunnelIngress() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ 
      appId, 
      nodeId, 
      ingressRules, 
      hostname, 
      targetDomain 
    }: { 
      appId: string; 
      nodeId: string; 
      ingressRules: IngressRule[]; 
      hostname?: string; 
      targetDomain?: string;
    }) => {
      return apiClient.put<{ message: string }>(`/api/tunnels/apps/${appId}/ingress?node_id=${nodeId}`, {
        ingress_rules: ingressRules,
        hostname,
        target_domain: targetDomain,
      });
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['tunnels', 'app', variables.appId] });
      queryClient.invalidateQueries({ queryKey: ['tunnels', 'list'] });
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId] });
    },
  });
}

// Create DNS record (provider-agnostic, may return 501 if not supported)
export function useCreateTunnelDNSRecord() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ appId, nodeId, hostname }: { appId: string; nodeId: string; hostname: string }) => {
      return apiClient.post<{ message: string; hostname: string }>(`/api/tunnels/apps/${appId}/dns?node_id=${nodeId}`, { hostname });
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['tunnels', 'app', variables.appId] });
      queryClient.invalidateQueries({ queryKey: ['tunnels', 'list'] });
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId] });
    },
  });
}

// Delete tunnel (provider-agnostic)
export function useDeleteTunnel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ appId, nodeId }: { appId: string; nodeId: string }) => {
      return apiClient.delete<JobResponse>(`/api/tunnels/apps/${appId}?node_id=${nodeId}`);
    },
    onSuccess: (_, variables) => {
      // Invalidate jobs query so AppActions picks up the new job and shows progress
      queryClient.invalidateQueries({ queryKey: ['jobs', 'app', variables.appId, variables.nodeId] });
      // Invalidate tunnels list to immediately update UI
      queryClient.invalidateQueries({ queryKey: ['tunnels', 'app', variables.appId] });
      queryClient.invalidateQueries({ queryKey: ['tunnels', 'list'] });
      // Also invalidate app query to refresh app details
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId] });
    },
  });
}

// Logout - call logout endpoint then redirect to login page
export async function logout() {
  try {
    // Call logout endpoint to clear the session cookie
    await fetch(`${AUTH_URL}/auth/logout`, {
      method: 'GET',
      credentials: 'include',
    });
  } catch {
    // Ignore errors - we'll redirect anyway
  }
  // Redirect to login page
  window.location.href = `${FRONTEND_URL}/login`;
}

// Login with GitHub - redirect to OAuth endpoint
// Pass 'from' parameter so go-pkgz/auth redirects back to frontend after login
export function loginWithGitHub() {
  const redirectTo = encodeURIComponent(`${FRONTEND_URL}/apps`);
  window.location.href = `${AUTH_URL}/auth/github/login?from=${redirectTo}`;
}

// System monitoring API
export function useSystemStats(refreshInterval: number = 10000, nodeIdsOverride?: string[]) {
  const { selectedNodeIds: globalNodeIds } = useNodeContext();
  
  // Use override if provided, otherwise use global context
  const nodeIds = nodeIdsOverride ?? globalNodeIds;
  
  // Build query key with node filter
  const queryKey = nodeIds && nodeIds.length > 0 
    ? ['system', 'stats', { nodeIds }] 
    : ['system', 'stats'];
  
  return useQuery<SystemStats[]>({
    queryKey,
    queryFn: () => {
      // Build node_ids parameter
      if (nodeIds && nodeIds.length > 0) {
        const nodeIdsParam = nodeIds.join(',');
        return apiClient.get<SystemStats[]>('/api/system/stats', { node_ids: nodeIdsParam });
      }
      // If no nodes selected (empty array during initialization), don't make the request yet
      // Return empty array to avoid fetching with node_ids=all
      return Promise.resolve([]);
    },
    // Don't run the query until we have selected nodes
    enabled: nodeIds && nodeIds.length > 0,
    refetchInterval: refreshInterval,
    refetchIntervalInBackground: false, // Only poll when tab is visible
  });
}

export function useRestartContainer() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ containerId, nodeId }: { containerId: string; nodeId: string }) => 
      apiClient.post<{ message: string; container_id: string }>(`/api/system/containers/${containerId}/restart`, undefined, { node_id: nodeId }),
    onSuccess: () => {
      // Refresh system stats after container action
      queryClient.invalidateQueries({ queryKey: ['system', 'stats'] });
    },
  });
}

export function useStopContainer() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ containerId, nodeId }: { containerId: string; nodeId: string }) => 
      apiClient.post<{ message: string; container_id: string }>(`/api/system/containers/${containerId}/stop`, undefined, { node_id: nodeId }),
    onSuccess: () => {
      // Refresh system stats after container action
      queryClient.invalidateQueries({ queryKey: ['system', 'stats'] });
    },
  });
}

export function useDeleteContainer() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ containerId, nodeId }: { containerId: string; nodeId: string }) => 
      apiClient.delete<{ message: string; container_id: string }>(`/api/system/containers/${containerId}`, { node_id: nodeId }),
    onSuccess: () => {
      // Refresh system stats after container deletion
      queryClient.invalidateQueries({ queryKey: ['system', 'stats'] });
    },
  });
}

// Node management API
export function useNodes() {
  return useQuery<Node[]>({
    queryKey: ['nodes'],
    queryFn: () => apiClient.get<Node[]>('/api/nodes'),
  });
}

export function useNode(id: string) {
  return useQuery<Node>({
    queryKey: ['node', id],
    queryFn: () => apiClient.get<Node>(`/api/nodes/${id}`),
    enabled: !!id,
  });
}

export function useRegisterNode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: RegisterNodeRequest) => 
      apiClient.post<Node, RegisterNodeRequest>('/api/nodes', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
    },
  });
}

export function useUpdateNode(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateNodeRequest) => 
      apiClient.put<Node, UpdateNodeRequest>(`/api/nodes/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
      queryClient.invalidateQueries({ queryKey: ['node', id] });
    },
  });
}

export function useDeleteNode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => 
      apiClient.delete<{ message: string; nodeID: string }>(`/api/nodes/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
    },
  });
}

export function useNodeHealth(id: string) {
  return useMutation({
    mutationFn: () => 
      apiClient.get<{ message: string; nodeID: string }>(`/api/nodes/${id}/health`),
  });
}

// Get current node info
export function useCurrentNode() {
  return useQuery<Node>({
    queryKey: ['current-node'],
    queryFn: () => apiClient.get<Node>('/api/node/info'),
    staleTime: 60000, // Cache for 1 minute
  });
}

// ============================================================================
// Job API
// ============================================================================

// Get a specific job by ID
export function useJob(jobId: string | null, nodeId: string | null) {
  return useQuery<Job>({
    queryKey: ['job', jobId, nodeId],
    queryFn: () => apiClient.get<Job>(`/api/jobs/${jobId}`, { node_id: nodeId! }),
    enabled: !!jobId && !!nodeId,
  });
}

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
      
      const hasActiveJob = data.some(job => job.status === 'pending' || job.status === 'running')
      if (hasActiveJob) {
        return 5000 // Continue polling if there's an active job
      }
      return false // Stop polling when no active jobs
    },
  });
}

// Get list of services for an app
export function useAppServices(appId: string, nodeId: string) {
  return useQuery<string[]>({
    queryKey: ['app-services', appId, nodeId],
    queryFn: () => apiClient.get<string[]>(`/api/apps/${appId}/services`, { node_id: nodeId }),
    enabled: !!appId && !!nodeId,
  });
}
