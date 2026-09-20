import { formatAgo } from '@/shared/lib/attention'
import React from 'react'
import { Clock, Play, Pause, RefreshCw, AlertTriangle, CheckCircle, Upload, Globe, Zap, Loader2, ChevronDown, ChevronRight } from 'lucide-react'
import { useAppJobs } from '@/shared/services/api'
import { StatusPill } from '@/shared/components/ui/StatusPill'
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
        case 'app_start': return <Play className="h-3.5 w-3.5" />
        case 'tunnel_create': return <Globe className="h-3.5 w-3.5" />
        case 'quick_tunnel': return <Zap className="h-3.5 w-3.5" />
        default: return <RefreshCw className="h-3.5 w-3.5" />
    }
}

const getJobColor = (job: Job) => {
    if (job.status === 'failed') return 'text-status-err-fg'
    if (job.status === 'running') return 'text-status-info-fg'
    if (job.status === 'pending') return 'text-status-warn-fg'
    return 'text-status-ok-fg' // completed
}

const getJobDotColor = (job: Job) => {
    if (job.status === 'failed') return 'bg-status-err'
    if (job.status === 'running') return 'bg-status-info'
    if (job.status === 'pending') return 'bg-status-warn'
    return 'bg-status-ok'
}

const getJobDescription = (job: Job) => {
    const typeMap: Record<string, string> = {
        'app_create': 'App creation',
        'app_update': 'App update',
        'app_start': 'App start',
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
            return <StatusPill kind="ok" size="sm">Completed</StatusPill>
        case 'failed':
            return <StatusPill kind="err" size="sm">Failed</StatusPill>
        case 'running':
            return <StatusPill kind="info" size="sm">Running</StatusPill>
        case 'pending':
            return <StatusPill kind="warn" size="sm">Pending</StatusPill>
        default:
            return null
    }
}

function ActivityTimeline({ app }: ActivityTimelineProps) {
    // Fetch job history for this app
    const { data: jobs } = useAppJobs(app.id, app.node_id)
    
    // Track expanded activity items
    const [expandedItems, setExpandedItems] = React.useState<Set<string>>(new Set())

    const activities: Activity[] = React.useMemo(() => {
        const items: Activity[] = []

        // Add job history
        if (jobs && jobs.length > 0) {
            jobs.forEach(job => {
                // For failed jobs, add a hint to check deployment logs if the error is very generic
                let details = job.status === 'failed' ? job.error_message : job.progress_message
                if (details && job.status === 'failed') {
                    // If error is just "exit status 1" or similar, add helpful hint
                    if (details.trim() === 'exit status 1' || details.trim().match(/^exit status \d+$/)) {
                        details = 'Build or deployment failed - click to view logs'
                    }
                }
                
                items.push({
                    id: job.id,
                    type: 'job',
                    timestamp: new Date(job.completed_at || job.started_at || job.created_at),
                    description: getJobDescription(job),
                    details,
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
                color: 'text-status-ok-fg',
                dotColor: 'bg-status-ok'
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
                color: 'text-status-ok-fg',
                dotColor: 'bg-status-ok'
            })
        } else if (app.status === 'stopped') {
            items.push({
                id: 'status-stopped',
                type: 'stop',
                timestamp: new Date(app.updated_at),
                description: 'App is currently stopped',
                icon: <Pause className="h-3.5 w-3.5" />,
                color: 'text-status-idle-fg',
                dotColor: 'bg-status-idle'
            })
        }

        return items.sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
    }, [app, jobs])

    const toggleExpanded = (activityId: string) => {
        setExpandedItems(prev => {
            const next = new Set(prev)
            if (next.has(activityId)) {
                next.delete(activityId)
            } else {
                next.add(activityId)
            }
            return next
        })
    }
    
    const isLongText = (text: string | undefined) => {
        return text && text.length > 80
    }

    if (activities.length === 0) {
        return (
            <div className="flex flex-col items-center gap-2 py-8 text-center text-muted-foreground">
                <Clock aria-hidden="true" className="h-6 w-6" />
                <p className="text-sm">Nothing has happened yet</p>
            </div>
        )
    }

    return (
        <ol className="relative flex flex-col">
            <div aria-hidden="true" className="absolute bottom-3 left-[7px] top-3 w-px bg-border" />
            {activities.map((activity) => {
                const isExpanded = expandedItems.has(activity.id)
                return (
                    <li key={activity.id} className="relative flex gap-3 py-2.5 pl-6">
                        <span aria-hidden="true" className={`absolute left-0 top-[15px] h-[15px] w-[15px] rounded-full ring-4 ring-card ${activity.dotColor}`} />
                        <div className="min-w-0 flex-1">
                            <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                                <span aria-hidden="true" className={`shrink-0 ${activity.color}`}>{activity.icon}</span>
                                <p className="text-sm font-medium">{activity.description}</p>
                                {activity.status && getStatusBadge(activity.status)}
                            </div>
                            {activity.details && (
                                <p className={`mt-1 text-[13px] ${isExpanded ? 'break-words' : 'line-clamp-1'} ${activity.status === 'failed' ? 'text-status-err-fg' : 'text-muted-foreground'}`}>
                                    {activity.details}
                                </p>
                            )}
                            {isLongText(activity.details) && (
                                <button
                                    type="button"
                                    aria-expanded={isExpanded}
                                    onClick={() => toggleExpanded(activity.id)}
                                    className="mt-1 inline-flex min-h-[44px] items-center gap-1 text-[13px] text-muted-foreground hover:text-foreground md:min-h-0"
                                >
                                    {isExpanded ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
                                    {isExpanded ? 'Show less' : 'Show more'}
                                </button>
                            )}
                            <p className="mt-0.5 text-[13px] text-muted-foreground">{formatAgo(activity.timestamp.toISOString())}</p>
                        </div>
                    </li>
                )
            })}
        </ol>
    )
}

export default ActivityTimeline
