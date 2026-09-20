import { lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import MainLayout from '@/shared/components/layout/MainLayout'
import { AuthProvider, useAuth } from '@/shared/components/auth/AuthProvider'
import ServerUnavailable from '@/shared/components/auth/ServerUnavailable'
import { useToast, ToastContainer } from '@/shared/components/ui/Toast'
import { ErrorBoundary } from '@/shared/components/ErrorBoundary'
import { NotFound } from '@/shared/components/NotFound'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { Agentation } from '@/shared/components/dev/Agentation'
import { LEGACY_REDIRECTS, ROUTES } from '@/shared/lib/routes'
import { ThemeProvider } from '@/shared/components/theme/ThemeProvider'
import { NodeContextProvider } from '@/shared/contexts/NodeContext'

// Each page loads when it is first visited, so the first paint does not wait for all of them.
const Dashboard = lazy(() => import('@/features/dashboard'))
const CreateApp = lazy(() => import('@/features/create-app'))
const AppDetails = lazy(() => import('@/features/app-details'))
const Cloudflare = lazy(() => import('@/features/cloudflare'))
const Monitoring = lazy(() => import('@/features/monitoring'))
const Nodes = lazy(() => import('@/features/nodes'))
const RegisterNode = lazy(() => import('@/features/nodes/register'))
const Settings = lazy(() => import('@/features/settings'))
const Login = lazy(() => import('@/features/login'))

// Component gallery for verifying shared UI. The conditional lets the production build drop it.
const UiGallery = import.meta.env.DEV ? lazy(() => import('@/features/dev/ui-gallery')) : null

function PageFallback() {
    return (
        <div className="space-y-4 p-4 md:p-6" aria-busy="true">
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-40 w-full" />
        </div>
    )
}

// Protected route wrapper
function ProtectedRoute({ children }: { children: React.ReactNode }) {
    const { isAuthenticated, isLoading, serverUnreachable, retry } = useAuth()

    if (isLoading) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
        )
    }

    if (serverUnreachable) {
        return <ServerUnavailable onRetry={retry} />
    }

    if (!isAuthenticated) {
        // Check if we just came from GitHub OAuth callback
        const params = new URLSearchParams(window.location.search)
        const hasOAuthParams = params.has('code') || params.has('state')

        // If we have OAuth params but still not authenticated, it means validation failed
        // (likely whitelist rejection)
        if (hasOAuthParams) {
            return <Navigate to="/login?error=unauthorized&error_description=not+authorized" replace />
        }

        return <Navigate to="/login" replace />
    }

    return (
        <MainLayout>
            <Suspense fallback={<PageFallback />}>{children}</Suspense>
        </MainLayout>
    )
}

// Public route - redirect to dashboard if already authenticated
function PublicRoute({ children }: { children: React.ReactNode }) {
    const { isAuthenticated, isLoading, serverUnreachable, retry } = useAuth()

    if (isLoading) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
        )
    }

    if (serverUnreachable) {
        return <ServerUnavailable onRetry={retry} />
    }

    if (isAuthenticated) {
        return <Navigate to="/apps" replace />
    }

    return <Suspense fallback={null}>{children}</Suspense>
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
                path={ROUTES.access}
                element={
                    <ProtectedRoute>
                        <Cloudflare />
                    </ProtectedRoute>
                }
            />
            <Route
                path={ROUTES.insights}
                element={
                    <ProtectedRoute>
                        <Monitoring />
                    </ProtectedRoute>
                }
            />
            {/* Old addresses forward to the renamed pages */}
            {LEGACY_REDIRECTS.map(({ from, to }) => (
                <Route key={from} path={from} element={<Navigate to={to} replace />} />
            ))}
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

            {UiGallery && (
                <Route
                    path="/dev/ui"
                    element={
                        <ProtectedRoute>
                            <Suspense fallback={null}>
                                <UiGallery />
                            </Suspense>
                        </ProtectedRoute>
                    }
                />
            )}

            {/* The login page while signed in, so it can be checked. It is left out of production builds. */}
            {import.meta.env.DEV && (
                <Route
                    path="/dev/login"
                    element={
                        <Suspense fallback={null}>
                            <Login />
                        </Suspense>
                    }
                />
            )}

            {/* Anything else is a page that does not exist */}
            <Route
                path="*"
                element={
                    <ProtectedRoute>
                        <NotFound />
                    </ProtectedRoute>
                }
            />
        </Routes>
    )
}

function App() {
    const { toasts, removeToast } = useToast()

    return (
        <>
            <BrowserRouter>
                <ThemeProvider>
                    <AuthProvider>
                        <NodeContextProvider>
                            <ErrorBoundary>
                                <AppRoutes />
                            </ErrorBoundary>
                            <ToastContainer toasts={toasts} removeToast={removeToast} />
                        </NodeContextProvider>
                    </AuthProvider>
                </ThemeProvider>
            </BrowserRouter>
            <Agentation />
        </>
    )
}

export default App
