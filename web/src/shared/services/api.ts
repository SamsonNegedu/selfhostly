import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type {
  App,
  CreateAppRequest,
  UpdateAppRequest,
  Settings,
  UpdateSettingsRequest,
  CloudflareTunnel,
  CloudflareTunnelResponse,
} from '../types/api';

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
    queryFn: async () => {
      const response = await fetch(`/api/apps`, {
        credentials: 'include',
      });
      if (!response.ok) {
        if (response.status === 401) {
          throw new Error('UNAUTHORIZED');
        }
        throw new Error('Failed to fetch apps');
      }
      return response.json();
    },
  });
}

export function useApp(id: number) {
  return useQuery<App>({
    queryKey: ['app', id],
    queryFn: async () => {
      const response = await fetch(`/api/apps/${id}`, {
        credentials: 'include',
      });
      if (!response.ok) {
        if (response.status === 401) {
          throw new Error('UNAUTHORIZED');
        }
        throw new Error('Failed to fetch app');
      }
      return response.json();
    },
    enabled: !!id,
  });
}

export function useCreateApp() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (data: CreateAppRequest) => {
      const response = await fetch(`/api/apps`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
        credentials: 'include',
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to create app');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useUpdateApp(id: number) {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (data: UpdateAppRequest) => {
      const response = await fetch(`/api/apps/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
        credentials: 'include',
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to update app');
      }
      return response.json();
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
    mutationFn: async (id: number) => {
      const response = await fetch(`/api/apps/${id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to delete app');
      }
      return response.json();
    },
    // Optimistic update - remove from cache immediately
    onMutate: async (id: number) => {
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
    mutationFn: async (id: number) => {
      const response = await fetch(`/api/apps/${id}/start`, {
        method: 'POST',
        credentials: 'include',
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to start app');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useStopApp() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (id: number) => {
      const response = await fetch(`/api/apps/${id}/stop`, {
        method: 'POST',
        credentials: 'include',
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to stop app');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useUpdateAppContainers() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (id: number) => {
      const response = await fetch(`/api/apps/${id}/update`, {
        method: 'POST',
        credentials: 'include',
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to update app containers');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

// Settings API
export function useSettings() {
  return useQuery<Settings>({
    queryKey: ['settings'],
    queryFn: async () => {
      const response = await fetch(`/api/settings`, {
        credentials: 'include',
      });
      if (!response.ok) {
        if (response.status === 401) {
          throw new Error('UNAUTHORIZED');
        }
        throw new Error('Failed to fetch settings');
      }
      return response.json();
    },
  });
}

export function useUpdateSettings() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (data: UpdateSettingsRequest) => {
      const response = await fetch(`/api/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
        credentials: 'include',
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to update settings');
      }
      return response.json();
    },
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
      const response = await fetch(`/api/me`, {
        credentials: 'include',
      });
      if (!response.ok) {
        if (response.status === 401) {
          return null;
        }
        throw new Error('Failed to fetch user');
      }
      return response.json();
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: false,
  });
}

// Cloudflare tunnel management hooks
export function useCloudflareTunnels() {
  return useQuery<CloudflareTunnelResponse>({
    queryKey: ['cloudflare', 'tunnels'],
    queryFn: async () => {
      const response = await fetch('/api/cloudflare/tunnels', {
        credentials: 'include',
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to fetch tunnels');
      }
      return response.json();
    },
  });
}

export function useCloudflareTunnel(appId: number) {
  return useQuery<CloudflareTunnel>({
    queryKey: ['cloudflare', 'tunnel', appId],
    queryFn: async () => {
      const response = await fetch(`/api/cloudflare/apps/${appId}/tunnel`, {
        credentials: 'include',
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to fetch tunnel');
      }
      return response.json();
    },
    enabled: !!appId,
  });
}

export function useSyncCloudflareTunnel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (appId: number) => {
      const response = await fetch(`/api/cloudflare/apps/${appId}/tunnel/sync`, {
        method: 'POST',
        credentials: 'include',
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to sync tunnel');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnels'] });
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });
}

export function useDeleteCloudflareTunnel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (appId: number) => {
      const response = await fetch(`/api/cloudflare/apps/${appId}/tunnel`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to delete tunnel');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnels'] });
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
  const redirectTo = encodeURIComponent(`${FRONTEND_URL}/dashboard`);
  window.location.href = `${AUTH_URL}/auth/github/login?from=${redirectTo}`;
}
