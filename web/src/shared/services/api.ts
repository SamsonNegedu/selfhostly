import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAppStore } from '../stores/app-store';
import { apiClient } from '../lib/api-client';

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
export function useApps() {
  return useQuery<App[]>({
    queryKey: ['apps'],
    queryFn: () => apiClient.get<App[]>('/api/apps'),
  });
}

export function useApp(id: string) {
  return useQuery<App>({
    queryKey: ['app', id],
    queryFn: () => apiClient.get<App>(`/api/apps/${id}`),
    enabled: !!id,
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

export function useUpdateApp(id: string) {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: UpdateAppRequest) => apiClient.put<App, UpdateAppRequest>(`/api/apps/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apps'] });
      queryClient.invalidateQueries({ queryKey: ['app', id] });
    },
  });
}

export function useDeleteApp() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => apiClient.delete<{ message: string; appID: string }>(`/api/apps/${id}`),
    // Optimistic update - remove from cache immediately
    onMutate: async (id: string) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['apps'] });
      
      // Snapshot previous value
      const previousApps = queryClient.getQueryData(['apps']);
      
      // Remove the deleted app from cache
      if (previousApps) {
        queryClient.setQueryData(['apps'], (previousApps: any[]) => previousApps.filter((app) => app.id !== id));
      }
      
      return { previousApps };
    },
    // Rollback in case of error
    onError: (_err, _id, context: any) => {
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
    mutationFn: (id: string) => apiClient.post<App>(`/api/apps/${id}/start`),
    onMutate: async (id: string) => {
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
          previousApps.map((app) => (app.id === id ? { ...app, status: 'running' as const } : app))
        );
      }
      
      // Update Zustand store optimistically
      useAppStore.getState().updateApp(id, { status: 'running' });
      
      return { previousApps, previousApp };
    },
    onError: (_err, id, context: any) => {
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
    mutationFn: (id: string) => apiClient.post<App>(`/api/apps/${id}/stop`),
    onMutate: async (id: string) => {
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
          previousApps.map((app) => (app.id === id ? { ...app, status: 'stopped' as const } : app))
        );
      }
      
      // Update Zustand store optimistically
      useAppStore.getState().updateApp(id, { status: 'stopped' });
      
      return { previousApps, previousApp };
    },
    onError: (_err, id, context: any) => {
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
    mutationFn: (id: string) => apiClient.post<App>(`/api/apps/${id}/update`),
    onMutate: async (id: string) => {
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
          previousApps.map((app) => (app.id === id ? { ...app, status: 'updating' as const } : app))
        );
      }
      
      // Update Zustand store optimistically
      useAppStore.getState().updateApp(id, { status: 'updating' });
      
      return { previousApps, previousApp };
    },
    onError: (_err, id, context: any) => {
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
export function useComposeVersions(appId: string) {
  return useQuery<ComposeVersion[]>({
    queryKey: ['compose-versions', appId],
    queryFn: () => apiClient.get<ComposeVersion[]>(`/api/apps/${appId}/compose/versions`),
    enabled: !!appId,
  });
}

export function useComposeVersion(appId: string, version: number) {
  return useQuery<ComposeVersion>({
    queryKey: ['compose-version', appId, version],
    queryFn: () => apiClient.get<ComposeVersion>(`/api/apps/${appId}/compose/versions/${version}`),
    enabled: !!appId && version > 0,
  });
}

export function useRollbackToVersion(appId: string) {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ version, change_reason }: { version: number; change_reason?: string }) => {
      const body: RollbackRequest = change_reason ? { change_reason } : {};
      return apiClient.post<{ message: string; app: App; new_version: ComposeVersion }>(`/api/apps/${appId}/compose/rollback/${version}`, body);
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
export function useCloudflareTunnels() {
  return useQuery<CloudflareTunnelResponse>({
    queryKey: ['cloudflare', 'tunnels'],
    queryFn: () => apiClient.get<CloudflareTunnelResponse>('/api/cloudflare/tunnels'),
  });
}

export function useCloudflareTunnel(appId: string) {
  return useQuery<CloudflareTunnel>({
    queryKey: ['cloudflare', 'tunnel', appId],
    queryFn: () => apiClient.get<CloudflareTunnel>(`/api/cloudflare/apps/${appId}/tunnel`),
    enabled: !!appId,
  });
}

export function useSyncCloudflareTunnel() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (appId: string) => apiClient.post<CloudflareTunnel>(`/api/cloudflare/apps/${appId}/tunnel/sync`),
    onSuccess: (_, appId) => {
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', appId] });
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnels'] });
      queryClient.invalidateQueries({ queryKey: ['app', appId] });
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useDeleteCloudflareTunnel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (appId: string) => apiClient.delete<{ message: string }>(`/api/cloudflare/apps/${appId}/tunnel`),
    onSuccess: (_, appId) => {
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', appId] });
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnels'] });
      queryClient.invalidateQueries({ queryKey: ['app', appId] });
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useUpdateTunnelIngress() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ appId, ingressRules, hostname, targetDomain }: { appId: string; ingressRules: IngressRule[]; hostname?: string; targetDomain?: string }) => {
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

      return apiClient.put<CloudflareTunnel>(`/api/cloudflare/apps/${appId}/tunnel/ingress`, body);
    },
    onSuccess: (_, variables) => {
      // Invalidate the specific tunnel query to refresh the data
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', variables.appId] });
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnels'] });
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId] });
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useCreateDNSRecord() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ appId, hostname, targetDomain }: { appId: string; hostname: string; targetDomain?: string }) => {
      const body: { hostname: string; target_domain?: string } = { hostname };
      
      if (targetDomain) {
        body.target_domain = targetDomain;
      }

      return apiClient.post<{ message: string; tunnel: CloudflareTunnel }>(`/api/cloudflare/apps/${appId}/tunnel/dns`, body);
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', variables.appId] });
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnels'] });
      queryClient.invalidateQueries({ queryKey: ['app', variables.appId] });
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
export function useSystemStats(refreshInterval: number = 10000) {
  return useQuery<SystemStats>({
    queryKey: ['system', 'stats'],
    queryFn: () => apiClient.get<SystemStats>('/api/system/stats'),
    refetchInterval: refreshInterval,
    refetchIntervalInBackground: false, // Only poll when tab is visible
  });
}

export function useRestartContainer() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (containerId: string) => 
      apiClient.post<{ message: string; container_id: string }>(`/api/system/containers/${containerId}/restart`),
    onSuccess: () => {
      // Refresh system stats after container action
      queryClient.invalidateQueries({ queryKey: ['system', 'stats'] });
    },
  });
}

export function useStopContainer() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (containerId: string) => 
      apiClient.post<{ message: string; container_id: string }>(`/api/system/containers/${containerId}/stop`),
    onSuccess: () => {
      // Refresh system stats after container action
      queryClient.invalidateQueries({ queryKey: ['system', 'stats'] });
    },
  });
}

export function useDeleteContainer() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (containerId: string) => 
      apiClient.delete<{ message: string; container_id: string }>(`/api/system/containers/${containerId}`),
    onSuccess: () => {
      // Refresh system stats after container deletion
      queryClient.invalidateQueries({ queryKey: ['system', 'stats'] });
    },
  });
}
