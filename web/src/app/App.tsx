import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Dashboard from '@/features/dashboard';
import CreateApp from '@/features/create-app';
import AppDetails from '@/features/app-details';
import Cloudflare from '@/features/cloudflare';
import Monitoring from '@/features/monitoring';
import Nodes from '@/features/nodes';
import RegisterNode from '@/features/nodes/register';
import Settings from '@/features/settings';
import Login from '@/features/login';
import MainLayout from '@/shared/components/layout/MainLayout';
import { AuthProvider, useAuth } from '@/shared/components/auth/AuthProvider';
import { useToast, ToastContainer } from '@/shared/components/ui/Toast';
import { Agentation } from '@/shared/components/dev/Agentation';
import { ThemeProvider } from '@/shared/components/theme/ThemeProvider';
import { NodeContextProvider } from '@/shared/contexts/NodeContext';

// Protected route wrapper
function ProtectedRoute({ children }: { children: React.ReactNode }) {
    const { isAuthenticated, isLoading } = useAuth();

    if (isLoading) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
        );
    }

    if (!isAuthenticated) {
        // Check if we just came from GitHub OAuth callback
        const params = new URLSearchParams(window.location.search);
        const hasOAuthParams = params.has('code') || params.has('state');

        // If we have OAuth params but still not authenticated, it means validation failed
        // (likely whitelist rejection)
        if (hasOAuthParams) {
            return <Navigate to="/login?error=unauthorized&error_description=not+authorized" replace />;
        }

        return <Navigate to="/login" replace />;
    }

    return (
        <MainLayout>{children}</MainLayout>
    );
}

// Public route - redirect to dashboard if already authenticated
function PublicRoute({ children }: { children: React.ReactNode }) {
    const { isAuthenticated, isLoading } = useAuth();

    if (isLoading) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
        );
    }

    if (isAuthenticated) {
        return <Navigate to="/apps" replace />;
    }

    return <>{children}</>;
}

function AppRoutes() {
    return (
        <Routes>
            {/* Public routes */}
            <Route
                path="/login"
                element={
                    <PublicRoute>
                        <Login />
                    </PublicRoute>
                }
            />

            {/* Protected routes */}
            <Route
                path="/"
                element={
                    <ProtectedRoute>
                        <Navigate to="/apps" replace />
                    </ProtectedRoute>
                }
            />
            <Route
                path="/apps"
                element={
                    <ProtectedRoute>
                        <Dashboard />
                    </ProtectedRoute>
                }
            />
            <Route
                path="/apps/new"
                element={
                    <ProtectedRoute>
                        <CreateApp />
                    </ProtectedRoute>
                }
            />
            <Route
                path="/apps/:id"
                element={
                    <ProtectedRoute>
                        <AppDetails />
                    </ProtectedRoute>
                }
            />
            <Route
                path="/settings"
                element={
                    <ProtectedRoute>
                        <Settings />
                    </ProtectedRoute>
                }
            />
            <Route
                path="/cloudflare"
                element={
                    <ProtectedRoute>
                        <Cloudflare />
                    </ProtectedRoute>
                }
            />
            <Route
                path="/monitoring"
                element={
                    <ProtectedRoute>
                        <Monitoring />
                    </ProtectedRoute>
                }
            />
            <Route
                path="/nodes"
                element={
                    <ProtectedRoute>
                        <Nodes />
                    </ProtectedRoute>
                }
            />
            <Route
                path="/nodes/new"
                element={
                    <ProtectedRoute>
                        <RegisterNode />
                    </ProtectedRoute>
                }
            />

            {/* Catch all - redirect to dashboard */}
            <Route path="*" element={<Navigate to="/apps" replace />} />
        </Routes>
    );
}

function App() {
    const { toasts, removeToast } = useToast();

    return (
        <>
            <BrowserRouter>
                <ThemeProvider>
                    <AuthProvider>
                        <NodeContextProvider>
                            <AppRoutes />
                            <ToastContainer toasts={toasts} removeToast={removeToast} />
                        </NodeContextProvider>
                    </AuthProvider>
                </ThemeProvider>
            </BrowserRouter>
            <Agentation />
        </>
    );
}

export default App;
