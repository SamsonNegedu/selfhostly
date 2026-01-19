import React from 'react'
import { Clock, Play, Pause, RefreshCw, AlertTriangle, CheckCircle } from 'lucide-react'
import type { App } from '@/shared/types/api'

interface ActivityTimelineProps {
    app: App
}

interface Activity {
    id: string
    type: 'start' | 'stop' | 'update' | 'error' | 'create'
    timestamp: Date
    description: string
    icon: React.ReactNode
    color: string
}

function ActivityTimeline({ app }: ActivityTimelineProps) {
    const activities: Activity[] = React.useMemo(() => {
        const items: Activity[] = []

        // Add create event
        items.push({
            id: 'create',
            type: 'create',
            timestamp: new Date(app.created_at),
            description: 'App was created',
            icon: <CheckCircle className="h-4 w-4" />,
            color: 'text-green-500'
        })

        // Add update event
        if (app.updated_at && app.updated_at !== app.created_at) {
            items.push({
                id: 'update',
                type: 'update',
                timestamp: new Date(app.updated_at),
                description: 'App was last updated',
                icon: <RefreshCw className="h-4 w-4" />,
                color: 'text-blue-500'
            })
        }

        // Add error event if present
        if (app.status === 'error' && app.error_message) {
            items.push({
                id: 'error',
                type: 'error',
                timestamp: new Date(app.updated_at),
                description: `Error: ${app.error_message}`,
                icon: <AlertTriangle className="h-4 w-4" />,
                color: 'text-red-500'
            })
        }

        // Add start/stop events based on status
        if (app.status === 'running') {
            items.push({
                id: 'start',
                type: 'start',
                timestamp: new Date(app.updated_at),
                description: 'App is currently running',
                icon: <Play className="h-4 w-4" />,
                color: 'text-green-500'
            })
        } else if (app.status === 'stopped') {
            items.push({
                id: 'stop',
                type: 'stop',
                timestamp: new Date(app.updated_at),
                description: 'App is currently stopped',
                icon: <Pause className="h-4 w-4" />,
                color: 'text-gray-500'
            })
        }

        return items.sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
    }, [app])

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
                                <div>
                                    <p className="text-sm font-medium">{activity.description}</p>
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
