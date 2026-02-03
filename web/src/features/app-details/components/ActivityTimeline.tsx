import React from 'react'
import { Clock, Play, Pause, RefreshCw, AlertTriangle, CheckCircle, Upload, Globe, Zap, Loader2 } from 'lucide-react'
import { useAppJobs } from '@/shared/services/api'
import type { App, Job } from '@/shared/types/api'

interface ActivityTimelineProps {
    app: App
}

interface Activity {
    id: string
    type: 'start' | 'stop' | 'update' | 'error' | 'create' | 'job'
    timestamp: Date
    description: string
    details?: string
    icon: React.ReactNode
    color: string
    status?: 'completed' | 'failed' | 'running' | 'pending'
}

// Helper functions defined outside component for stability
const getJobIcon = (job: Job) => {
    if (job.status === 'running') return <Loader2 className="h-4 w-4 animate-spin" />
    if (job.status === 'failed') return <AlertTriangle className="h-4 w-4" />

    switch (job.type) {
        case 'app_create': return <CheckCircle className="h-4 w-4" />
        case 'app_update': return <Upload className="h-4 w-4" />
        case 'tunnel_create': return <Globe className="h-4 w-4" />
        case 'quick_tunnel': return <Zap className="h-4 w-4" />
        default: return <RefreshCw className="h-4 w-4" />
    }
}

const getJobColor = (job: Job) => {
    if (job.status === 'failed') return 'text-red-500'
    if (job.status === 'running') return 'text-blue-500'
    if (job.status === 'pending') return 'text-yellow-500'
    return 'text-green-500' // completed
}

const getJobDescription = (job: Job) => {
    const typeMap: Record<string, string> = {
        'app_create': 'App creation',
        'app_update': 'App update',
        'tunnel_create': 'Custom tunnel creation',
        'quick_tunnel': 'Quick Tunnel setup'
    }

    const action = typeMap[job.type] || job.type

    if (job.status === 'completed') return `${action} completed`
    if (job.status === 'failed') return `${action} failed`
    if (job.status === 'running') return `${action} in progress`
    return `${action} started`
}

function ActivityTimeline({ app }: ActivityTimelineProps) {
    // Fetch job history for this app
    const { data: jobs } = useAppJobs(app.id, app.node_id)

    const activities: Activity[] = React.useMemo(() => {
        const items: Activity[] = []

        // Add job history
        if (jobs && jobs.length > 0) {
            jobs.forEach(job => {
                items.push({
                    id: job.id,
                    type: 'job',
                    timestamp: new Date(job.completed_at || job.started_at || job.created_at),
                    description: getJobDescription(job),
                    details: job.status === 'failed' ? job.error_message : job.progress_message,
                    icon: getJobIcon(job),
                    color: getJobColor(job),
                    status: job.status
                })
            })
        }

        // Add create event if no create job exists
        const hasCreateJob = jobs?.some(j => j.type === 'app_create')
        if (!hasCreateJob) {
            items.push({
                id: 'create',
                type: 'create',
                timestamp: new Date(app.created_at),
                description: 'App was created',
                icon: <CheckCircle className="h-4 w-4" />,
                color: 'text-green-500'
            })
        }

        // Add current status indicator if app is running/stopped
        if (app.status === 'running') {
            items.push({
                id: 'status-running',
                type: 'start',
                timestamp: new Date(app.updated_at),
                description: 'App is currently running',
                icon: <Play className="h-4 w-4" />,
                color: 'text-green-500'
            })
        } else if (app.status === 'stopped') {
            items.push({
                id: 'status-stopped',
                type: 'stop',
                timestamp: new Date(app.updated_at),
                description: 'App is currently stopped',
                icon: <Pause className="h-4 w-4" />,
                color: 'text-gray-500'
            })
        }

        return items.sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
    }, [app, jobs])

    const formatRelativeTime = (date: Date) => {
        const now = new Date()
        const diffMs = now.getTime() - date.getTime()
        const diffMins = Math.floor(diffMs / 60000)
        const diffHours = Math.floor(diffMs / 3600000)
        const diffDays = Math.floor(diffMs / 86400000)

        if (diffMins < 1) return 'Just now'
        if (diffMins < 60) return `${diffMins} minute${diffMins !== 1 ? 's' : ''} ago`
        if (diffHours < 24) return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`
        if (diffDays < 7) return `${diffDays} day${diffDays !== 1 ? 's' : ''} ago`
        return date.toLocaleDateString()
    }

    const formatFullTime = (date: Date) => {
        return date.toLocaleString('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        })
    }

    if (activities.length === 0) {
        return (
            <div className="text-center py-8 text-muted-foreground">
                <Clock className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p>No activity recorded yet</p>
            </div>
        )
    }

    return (
        <div className="relative">
            {/* Timeline line */}
            <div className="absolute left-4 top-2 bottom-2 w-0.5 bg-border" />

            <div className="space-y-4">
                {activities.map((activity) => (
                    <div key={activity.id} className="relative flex items-start gap-4 pl-10 fade-in">
                        {/* Timeline dot */}
                        <div className={`absolute left-0 w-8 h-8 rounded-full flex items-center justify-center ${activity.color} bg-background border-2 border-border`}>
                            {activity.icon}
                        </div>

                        {/* Activity content */}
                        <div className="flex-1 pb-4">
                            <div className="flex items-start justify-between gap-4">
                                <div className="flex-1">
                                    <p className="text-sm font-medium">{activity.description}</p>
                                    {activity.details && (
                                        <p className={`text-xs mt-1 ${activity.status === 'failed' ? 'text-red-500' : 'text-muted-foreground'}`}>
                                            {activity.details}
                                        </p>
                                    )}
                                    <p className="text-xs text-muted-foreground mt-1">
                                        {formatRelativeTime(activity.timestamp)}
                                    </p>
                                </div>
                                <span className="text-xs text-muted-foreground whitespace-nowrap">
                                    {formatFullTime(activity.timestamp)}
                                </span>
                            </div>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    )
}

export default ActivityTimeline
