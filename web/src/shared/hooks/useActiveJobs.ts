import { useEffect, useRef } from 'react'
import { useLocation } from 'react-router-dom'
import { useQueries } from '@tanstack/react-query'
import { useToast } from '@/shared/components/ui/Toast'
import { apiClient } from '@/shared/lib/api-client'
import { describeError } from '@/shared/lib/errors'
import { useNodeContext } from '@/shared/contexts/NodeContext'
import { useApps } from '@/shared/services/api'
import type { App, Job } from '@/shared/types/api'

export interface ActiveJob {
    app: App
    job: Job | null
}

const WORKING: App['status'][] = ['updating', 'pending']

// Apps that are being deployed, updated or started right now, with the job doing it. It works from anywhere in
// the app, so progress stays visible after leaving the app's own page. When one finishes, a toast says how it went.
export function useActiveJobs(): ActiveJob[] {
    const { selectedNodeIds } = useNodeContext()
    const { data: apps = [] } = useApps(selectedNodeIds)
    const { toast } = useToast()
    const { pathname } = useLocation()
    const working = apps.filter((app) => WORKING.includes(app.status))

    // The same query keys as `useAppJobs`, so the cache is shared and change events refresh them.
    const jobQueries = useQueries({
        queries: working.map((app) => ({
            queryKey: ['jobs', 'app', app.id, app.node_id],
            queryFn: () => apiClient.get<Job[]>(`/api/apps/${app.id}/jobs`, { node_id: app.node_id }),
            refetchInterval: 5000,
        })),
    })

    // Say how each one ended, once, when it leaves the working set.
    const previous = useRef<Map<string, string>>(new Map())
    useEffect(() => {
        const now = new Map(apps.map((app) => [app.id, app.status]))
        for (const [id, status] of previous.current) {
            if (!WORKING.includes(status as App['status'])) continue
            const app = apps.find((candidate) => candidate.id === id)
            if (!app || WORKING.includes(app.status)) continue
            // The app's own page already reports how it ended.
            if (pathname.startsWith(`/apps/${app.id}`)) continue
            if (app.status === 'error')
                toast.error(`${app.name} failed`, describeError(new Error(app.error_message || 'It did not start.')))
            else toast.success(`${app.name} is ready`, app.status === 'running' ? 'It is running.' : 'It finished.')
        }
        previous.current = now
    }, [apps, toast, pathname])

    return working.map((app, index) => {
        const jobs = jobQueries[index]?.data ?? []
        return { app, job: jobs.find((job) => job.status === 'pending' || job.status === 'running') ?? null }
    })
}
