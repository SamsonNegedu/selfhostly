import React from 'react'
import { Clock, Play, Pause, RefreshCw, AlertTriangle, CheckCircle, Upload, Globe, Zap, Loader2 } from 'lucide-react'
import { useAppJobs } from '@/shared/services/api'
import { Badge } from '@/shared/components/ui/Badge'
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
    dotColor: string
    status?: 'completed' | 'failed' | 'running' | 'pending'
}

// Helper functions defined outside component for stability
const getJobIcon = (job: Job) => {
    if (job.status === 'running') return <Loader2 className="h-3.5 w-3.5 animate-spin" />
    if (job.status === 'failed') return <AlertTriangle className="h-3.5 w-3.5" />

    switch (job.type) {
        case 'app_create': return <CheckCircle className="h-3.5 w-3.5" />
        case 'app_update': return <Upload className="h-3.5 w-3.5" />
        case 'tunnel_create': return <Globe className="h-3.5 w-3.5" />
        case 'quick_tunnel': return <Zap className="h-3.5 w-3.5" />
        default: return <RefreshCw className="h-3.5 w-3.5" />
    }
}

const getJobColor = (job: Job) => {
    if (job.status === 'failed') return 'text-red-500'
    if (job.status === 'running') return 'text-blue-500'
    if (job.status === 'pending') return 'text-yellow-500'
    return 'text-green-500' // completed
}

const getJobDotColor = (job: Job) => {
    if (job.status === 'failed') return 'bg-red-500'
    if (job.status === 'running') return 'bg-blue-500'
    if (job.status === 'pending') return 'bg-yellow-500'
    return 'bg-green-500'
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

const getStatusBadge = (status?: string) => {
    switch (status) {
        case 'completed':
            return <Badge variant="secondary" className="text-[10px] px-1.5 py-0 bg-green-500/10 text-green-600 border-green-500/20 dark:text-green-400">Completed</Badge>
        case 'failed':
            return <Badge variant="secondary" className="text-[10px] px-1.5 py-0 bg-red-500/10 text-red-600 border-red-500/20 dark:text-red-400">Failed</Badge>
        case 'running':
            return <Badge variant="secondary" className="text-[10px] px-1.5 py-0 bg-blue-500/10 text-blue-600 border-blue-500/20 dark:text-blue-400">Running</Badge>
        case 'pending':
            return <Badge variant="secondary" className="text-[10px] px-1.5 py-0 bg-yellow-500/10 text-yellow-600 border-yellow-500/20 dark:text-yellow-400">Pending</Badge>
        default:
            return null
    }
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
                    dotColor: getJobDotColor(job),
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
                icon: <CheckCircle className="h-3.5 w-3.5" />,
                color: 'text-green-500',
                dotColor: 'bg-green-500'
            })
        }

        // Add current status indicator if app is running/stopped
        if (app.status === 'running') {
            items.push({
                id: 'status-running',
                type: 'start',
                timestamp: new Date(app.updated_at),
                description: 'App is currently running',
                icon: <Play className="h-3.5 w-3.5" />,
                color: 'text-green-500',
                dotColor: 'bg-green-500'
            })
        } else if (app.status === 'stopped') {
            items.push({
                id: 'status-stopped',
                type: 'stop',
                timestamp: new Date(app.updated_at),
                description: 'App is currently stopped',
                icon: <Pause className="h-3.5 w-3.5" />,
                color: 'text-gray-500',
                dotColor: 'bg-gray-500'
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
        if (diffMins < 60) return `${diffMins}m ago`
        if (diffHours < 24) return `${diffHours}h ago`
        if (diffDays < 7) return `${diffDays}d ago`
        return date.toLocaleDateString()
    }

    if (activities.length === 0) {
        return (
            <div className="text-center py-8 text-muted-foreground">
                <Clock className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p className="text-sm">No activity recorded yet</p>
            </div>
        )
    }

    return (
        <div className="relative">
            {/* Timeline line */}
            <div className="absolute left-[11px] top-1 bottom-1 w-px bg-border" />

            <div className="space-y-1">
                {activities.map((activity, index) => (
                    <div
                        key={activity.id}
                        className={`relative flex items-start gap-3 pl-8 py-2 rounded-lg transition-colors hover:bg-muted/50 ${index === 0 ? 'bg-muted/30' : ''
                            }`}
                    >
                        {/* Timeline dot */}
                        <div className={`absolute left-1 top-[14px] w-[14px] h-[14px] rounded-full flex items-center justify-center ring-2 ring-background ${activity.dotColor}`}>
                            <div className="w-1.5 h-1.5 rounded-full bg-white/80" />
                        </div>

                        {/* Activity content */}
                        <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                                <span className={`flex-shrink-0 ${activity.color}`}>
                                    {activity.icon}
                                </span>
                                <p className="text-sm font-medium truncate">{activity.description}</p>
                                {activity.status && getStatusBadge(activity.status)}
                            </div>
                            {activity.details && (
                                <p className={`text-xs mt-0.5 ml-[22px] truncate ${activity.status === 'failed' ? 'text-red-500' : 'text-muted-foreground'}`}>
                                    {activity.details}
                                </p>
                            )}
                            <p className="text-[11px] text-muted-foreground mt-0.5 ml-[22px]">
                                {formatRelativeTime(activity.timestamp)}
                            </p>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    )
}

export default ActivityTimeline
