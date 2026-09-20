import { createContext, useContext, ReactNode } from 'react'
import { useCurrentUser, User } from '@/shared/services/api'

interface AuthContextType {
    user: User | null
    isLoading: boolean
    isAuthenticated: boolean
    /** False when the server has sign-in turned off, so there are no sessions to end. */
    authEnabled: boolean
    /** True when the server could not be reached, which is different from being signed out. */
    serverUnreachable: boolean
    retry: () => void
}

const AuthContext = createContext<AuthContextType>({
    user: null,
    isLoading: true,
    isAuthenticated: false,
    authEnabled: true,
    serverUnreachable: false,
    retry: () => {},
})

export function AuthProvider({ children }: { children: ReactNode }) {
    const { data: user, isLoading, error, refetch } = useCurrentUser()

    // If /api/me returns 404, backend auth is disabled (e.g., using Cloudflare Zero Trust)
    // Allow access with a mock user
    const authDisabled = error instanceof Error && error.message === 'NOT_FOUND'

    // Any other failure means the server did not answer properly. A 401 is turned into a null user upstream.
    const serverUnreachable = !!error && !authDisabled

    const mockUser: User = {
        id: 'system',
        name: 'User',
        picture: '',
    }

    return (
        <AuthContext.Provider
            value={{
                user: authDisabled ? mockUser : (user ?? null),
                isLoading,
                isAuthenticated: authDisabled || !!user,
                authEnabled: !authDisabled,
                serverUnreachable,
                retry: () => {
                    void refetch()
                },
            }}
        >
            {children}
        </AuthContext.Provider>
    )
}

export function useAuth() {
    return useContext(AuthContext)
}
