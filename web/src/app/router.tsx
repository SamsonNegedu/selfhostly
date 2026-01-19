import { createBrowserRouter, Navigate } from 'react-router-dom'
import Dashboard from '@/features/dashboard'
import CreateApp from '@/features/create-app'
import AppDetails from '@/features/app-details'
import Settings from '@/features/settings'
import App from './App'

const router = createBrowserRouter([
    {
        path: '/',
        element: <App />,
        children: [
            { index: true, element: <Navigate to="/dashboard" replace /> },
            { path: 'dashboard', element: <Dashboard /> },
            { path: 'apps/new', element: <CreateApp /> },
            { path: 'apps/:id', element: <AppDetails /> },
            { path: 'settings', element: <Settings /> },
        ],
    },
])

export default router
