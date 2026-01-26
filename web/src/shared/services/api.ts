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
  CloudflareTunnel,
  CloudflareTunnelResponse,
  ComposeVersion,
  RollbackRequest,
  SystemStats,
  Node,
  RegisterNodeRequest,
  UpdateNodeRequest,
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

export function useApp(id: string, nodeId?: string) {
  return useQuery<App>({
    queryKey: ['app', id, nodeId],
    queryFn: () => {
      const params = nodeId ? { node_id: nodeId } : undefined;
      return apiClient.get<App>(`/api/apps/${id}`, params);
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

export function useUpdateApp(id: string, nodeId?: string) {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: UpdateAppRequest) => {
      const url = nodeId ? `/api/apps/${id}?node_id=${nodeId}` : `/api/apps/${id}`;
      return apiClient.put<App, UpdateAppRequest>(url, data);
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
    mutationFn: ({ id, nodeId }: { id: string; nodeId?: string }) => {
      const url = nodeId ? `/api/apps/${id}?node_id=${nodeId}` : `/api/apps/${id}`;
      return apiClient.delete<{ message: string; appID: string }>(url);
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
    mutationFn: ({ id, nodeId }: { id: string; nodeId?: string }) => {
      const url = nodeId ? `/api/apps/${id}/start?node_id=${nodeId}` : `/api/apps/${id}/start`;
      return apiClient.post<App>(url);
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
    mutationFn: ({ id, nodeId }: { id: string; nodeId?: string }) => {
      const url = nodeId ? `/api/apps/${id}/stop?node_id=${nodeId}` : `/api/apps/${id}/stop`;
      return apiClient.post<App>(url);
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
    mutationFn: ({ id, nodeId }: { id: string; nodeId?: string }) => {
      const url = nodeId ? `/api/apps/${id}/update?node_id=${nodeId}` : `/api/apps/${id}/update`;
      return apiClient.post<App>(url);
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
      // After successful update, refetch to get the actual status
      queryClient.invalidateQueries({ queryKey: ['apps'] });
      queryClient.invalidateQueries({ queryKey: ['app'] });
    },
  });
}

// Compose Versions API
export function useComposeVersions(appId: string, nodeId?: string) {
  return useQuery<ComposeVersion[]>({
    queryKey: ['compose-versions', appId, nodeId],
    queryFn: () => {
      const url = nodeId ? `/api/apps/${appId}/compose/versions?node_id=${nodeId}` : `/api/apps/${appId}/compose/versions`;
      return apiClient.get<ComposeVersion[]>(url);
    },
    enabled: !!appId,
  });
}

export function useComposeVersion(appId: string, version: number, nodeId?: string) {
  return useQuery<ComposeVersion>({
    queryKey: ['compose-version', appId, version, nodeId],
    queryFn: () => {
      const url = nodeId ? `/api/apps/${appId}/compose/versions/${version}?node_id=${nodeId}` : `/api/apps/${appId}/compose/versions/${version}`;
      return apiClient.get<ComposeVersion>(url);
    },
    enabled: !!appId && version > 0,
  });
}

export function useRollbackToVersion(appId: string, nodeId?: string) {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ version, change_reason }: { version: number; change_reason?: string }) => {
      const body: RollbackRequest = change_reason ? { change_reason } : {};
      const url = nodeId ? `/api/apps/${appId}/compose/rollback/${version}?node_id=${nodeId}` : `/api/apps/${appId}/compose/rollback/${version}`;
      return apiClient.post<{ message: string; app: App; new_version: ComposeVersion }>(url, body);
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
        // Return null for 401 instead of throwing
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

// Cloudflare tunnel management hooks
export function useCloudflareTunnels(nodeIdsOverride?: string[]) {
  const { selectedNodeIds: globalNodeIds } = useNodeContext();
  
  // Use override if provided, otherwise use global context
  const nodeIds = nodeIdsOverride ?? globalNodeIds;
  
  // Build query key with node filter
  const queryKey = nodeIds && nodeIds.length > 0 
    ? ['cloudflare', 'tunnels', { nodeIds }] 
    : ['cloudflare', 'tunnels'];
  
  return useQuery<CloudflareTunnelResponse>({
    queryKey,
    queryFn: () => {
      // Build node_ids parameter
      if (nodeIds && nodeIds.length > 0) {
        const nodeIdsParam = nodeIds.join(',');
        return apiClient.get<CloudflareTunnelResponse>('/api/cloudflare/tunnels', { node_ids: nodeIdsParam });
      }
      // Default: fetch from all nodes
      return apiClient.get<CloudflareTunnelResponse>('/api/cloudflare/tunnels', { node_ids: 'all' });
    },
  });
}

export function useCloudflareTunnel(appId: string, nodeId?: string) {
  return useQuery<CloudflareTunnel>({
    queryKey: ['cloudflare', 'tunnel', appId, nodeId],
    queryFn: () => {
      const url = nodeId ? `/api/cloudflare/apps/${appId}/tunnel?node_id=${nodeId}` : `/api/cloudflare/apps/${appId}/tunnel`;
      return apiClient.get<CloudflareTunnel>(url);
    },
    enabled: !!appId,
  });
}

export function useSyncCloudflareTunnel() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ appId, nodeId }: { appId: string; nodeId?: string }) => {
      const url = nodeId ? `/api/cloudflare/apps/${appId}/tunnel/sync?node_id=${nodeId}` : `/api/cloudflare/apps/${appId}/tunnel/sync`;
      return apiClient.post<CloudflareTunnel>(url);
    },
    onSuccess: (_, variables) => {
      // Invalidate queries with nodeId to match the query key structure
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', variables.appId, variables.nodeId] });
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', variables.appId] }); // Also invalidate without nodeId for backward compatibility
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnels'] });
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId, variables.nodeId] });
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId] }); // Also invalidate without nodeId
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useDeleteCloudflareTunnel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ appId, nodeId }: { appId: string; nodeId?: string }) => {
      const url = nodeId ? `/api/cloudflare/apps/${appId}/tunnel?node_id=${nodeId}` : `/api/cloudflare/apps/${appId}/tunnel`;
      return apiClient.delete<{ message: string }>(url);
    },
    onSuccess: (_, variables) => {
      // Invalidate queries with nodeId to match the query key structure
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', variables.appId, variables.nodeId] });
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', variables.appId] }); // Also invalidate without nodeId for backward compatibility
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnels'] });
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId, variables.nodeId] });
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId] }); // Also invalidate without nodeId
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useUpdateTunnelIngress() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ appId, nodeId, ingressRules, hostname, targetDomain }: { appId: string; nodeId?: string; ingressRules: IngressRule[]; hostname?: string; targetDomain?: string }) => {
      const body: { ingress_rules: IngressRule[]; hostname?: string; target_domain?: string } = {
        ingress_rules: ingressRules,
      };
      
      // Ensure there's a catch-all rule at the end if not provided
      if (ingressRules.length === 0 || ingressRules[ingressRules.length - 1].service !== 'http_status:404') {
        body.ingress_rules = [...ingressRules, { service: 'http_status:404' }];
      }
      
      if (hostname) {
        body.hostname = hostname;
      }

      if (targetDomain) {
        body.target_domain = targetDomain;
      }

      const url = nodeId ? `/api/cloudflare/apps/${appId}/tunnel/ingress?node_id=${nodeId}` : `/api/cloudflare/apps/${appId}/tunnel/ingress`;
      return apiClient.put<CloudflareTunnel>(url, body);
    },
    onSuccess: (_, variables) => {
      // Invalidate queries with nodeId to match the query key structure
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', variables.appId, variables.nodeId] });
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', variables.appId] }); // Also invalidate without nodeId for backward compatibility
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnels'] });
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId, variables.nodeId] });
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId] }); // Also invalidate without nodeId
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useCreateDNSRecord() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ appId, nodeId, hostname, targetDomain }: { appId: string; nodeId?: string; hostname: string; targetDomain?: string }) => {
      const body: { hostname: string; target_domain?: string } = { hostname };
      
      if (targetDomain) {
        body.target_domain = targetDomain;
      }

      const url = nodeId ? `/api/cloudflare/apps/${appId}/tunnel/dns?node_id=${nodeId}` : `/api/cloudflare/apps/${appId}/tunnel/dns`;
      return apiClient.post<{ message: string; tunnel: CloudflareTunnel }>(url, body);
    },
    onSuccess: (_, variables) => {
      // Invalidate queries with nodeId to match the query key structure
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', variables.appId, variables.nodeId] });
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', variables.appId] }); // Also invalidate without nodeId for backward compatibility
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnels'] });
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId, variables.nodeId] });
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId] }); // Also invalidate without nodeId
      queryClient.invalidateQueries({ queryKey: ['apps'] });
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
      // Default: fetch from all nodes
      return apiClient.get<SystemStats[]>('/api/system/stats', { node_ids: 'all' });
    },
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
