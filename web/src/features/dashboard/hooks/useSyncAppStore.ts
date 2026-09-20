import { useEffect } from 'react'
import { useQueryClient } from '@/shared/services/api'
import { useAppStore } from '@/shared/stores/app-store'
import type { App } from '@/shared/types/api'

// Keeps the shared apps store in step with the query cache, as the other views expect.
export function useSyncAppStore(apps: App[] | undefined) {
    const setApps = useAppStore((state) => state.setApps)
    const queryClient = useQueryClient()

    // Keep the shared store in step with the query cache, as the other views expect.
    useEffect(() => {
        const unsubscribe = queryClient.getQueryCache().subscribe(() => {
            const appsQuery = queryClient.getQueryCache().findAll({ queryKey: ['apps'] })
            if (appsQuery.length > 0) {
                const appsData = appsQuery[0].state.data as App[]
                if (appsData) setApps(appsData)
            }
        })
        if (apps) setApps(apps)
        return () => unsubscribe()
    }, [apps, setApps, queryClient])
}
