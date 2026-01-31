import { createContext, useContext, ReactNode } from 'react';
import { useCurrentUser, User } from '@/shared/services/api';

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType>({
  user: null,
  isLoading: true,
  isAuthenticated: false,
});

export function AuthProvider({ children }: { children: ReactNode }) {
  const { data: user, isLoading, error } = useCurrentUser();

  // If /api/me returns 404, backend auth is disabled (e.g., using Cloudflare Zero Trust)
  // Allow access with a mock user
  const authDisabled = error instanceof Error && error.message === 'NOT_FOUND';

  const mockUser: User = {
    id: 'system',
    name: 'User',
    picture: '',
  };

  return (
    <AuthContext.Provider
      value={{
        user: authDisabled ? mockUser : (user ?? null),
        isLoading,
        isAuthenticated: authDisabled || !!user,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
