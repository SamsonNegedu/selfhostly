import React from 'react'
import { useApps, useQueryClient } from '@/shared/services/api'
import { useAppStore } from '@/shared/stores/app-store'
import AppList from './components/AppList'
import { Loader2 } from 'lucide-react'
import type { App } from '@/shared/types/api'

function Dashboard() {
    const { data: apps, isLoading, error } = useApps()
    const setApps = useAppStore((state) => state.setApps)
    const queryClient = useQueryClient()

    // Subscribe to query cache updates and sync with Zustand store
    React.useEffect(() => {
        const unsubscribe = queryClient.getQueryCache().subscribe(() => {
            // Only update when apps data changes
            const appsQuery = queryClient.getQueryCache().findAll({ queryKey: ['apps'] })
            if (appsQuery.length > 0) {
                const appsData = appsQuery[0].state.data as App[]
                if (appsData) {
                    setApps(appsData)
                }
            }
        })

        // Initial sync
        if (apps) {
            setApps(apps)
        }

        return () => unsubscribe()
    }, [apps, setApps, queryClient])

    if (isLoading) {
        return (
            <div className="flex items-center justify-center h-64">
                <Loader2 className="h-8 w-8 animate-spin" />
            </div>
        )
    }

    if (error) {
        return (
            <div className="text-center text-destructive">
                Failed to load apps. Please try again.
            </div>
        )
    }

    return (
        <div>
            <div className="mb-6">
                <h1 className="text-3xl font-bold">Dashboard</h1>
                <p className="text-muted-foreground mt-2">
                    Manage your self-hosted applications
                </p>
            </div>
            <AppList />
        </div>
    )
}

export default Dashboard
