import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Dashboard from '@/features/dashboard';
import CreateApp from '@/features/create-app';
import AppDetails from '@/features/app-details';
import Cloudflare from '@/features/cloudflare';
import Settings from '@/features/settings';
import Login from '@/features/login';
import Header from '@/shared/components/layout/Header';
import MainLayout from '@/shared/components/layout/MainLayout';
import { AuthProvider, useAuth } from '@/shared/components/auth/AuthProvider';
import { useToast, ToastContainer } from '@/shared/components/ui/Toast';
import { Agentation } from '@/shared/components/dev/Agentation';
import { ThemeProvider } from '@/shared/components/theme/ThemeProvider';
// import { Agentation, type Annotation } from 'agentation';

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
        <div className="min-h-screen bg-background text-foreground">
            <Header />
            <MainLayout>{children}</MainLayout>
        </div>
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
        return <Navigate to="/dashboard" replace />;
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
                        <Navigate to="/dashboard" replace />
                    </ProtectedRoute>
                }
            />
            <Route
                path="/dashboard"
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

            {/* Catch all - redirect to dashboard */}
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
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
                        <AppRoutes />
                        <ToastContainer toasts={toasts} removeToast={removeToast} />
                    </AuthProvider>
                </ThemeProvider>
            </BrowserRouter>
            <Agentation />
        </>
    );
}

export default App;
