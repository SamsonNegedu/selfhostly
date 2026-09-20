import { useQuery } from '@tanstack/react-query'
import { apiClient } from '../../lib/api-client'

// User type from go-pkgz/auth
export interface User {
    id: string
    name: string
    picture?: string
}

// Auth API - GitHub OAuth via go-pkgz/auth
// Auth endpoints:
//   - GET /auth/github/login - Redirects to GitHub for OAuth
//   - GET /auth/logout - Clears session and logs out
//   - GET /api/me - Get current user info

// Backend URL for auth redirects (browser navigation bypasses Vite proxy)
const AUTH_URL = import.meta.env?.DEV ? 'http://localhost:8080' : ''

// Frontend URL for redirects after auth
const FRONTEND_URL = import.meta.env?.DEV ? 'http://localhost:5173' : window.location.origin

// Get current authenticated user
export function useCurrentUser() {
    return useQuery<User | null>({
        queryKey: ['currentUser'],
        queryFn: async () => {
            try {
                return await apiClient.get<User>('/api/me')
            } catch (error) {
                // Return null for 401, but let 404 bubble up (auth disabled)
                if (error instanceof Error && error.message === 'UNAUTHORIZED') {
                    return null
                }
                throw error
            }
        },
        staleTime: 5 * 60 * 1000, // 5 minutes
        retry: false,
    })
}

// Logout - call logout endpoint then redirect to login page
export async function logout() {
    try {
        // Call logout endpoint to clear the session cookie
        await fetch(`${AUTH_URL}/auth/logout`, {
            method: 'GET',
            credentials: 'include',
        })
    } catch {
        // Ignore errors - we'll redirect anyway
    }
    // Redirect to login page
    window.location.href = `${FRONTEND_URL}/login`
}

// Login with GitHub - redirect to OAuth endpoint
// Pass 'from' parameter so go-pkgz/auth redirects back to frontend after login
export function loginWithGitHub() {
    const redirectTo = encodeURIComponent(`${FRONTEND_URL}/apps`)
    window.location.href = `${AUTH_URL}/auth/github/login?from=${redirectTo}`
}
