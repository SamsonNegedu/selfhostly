import { useMemo, useState } from 'react'
import { Clock } from 'lucide-react'
import { useAppJobs } from '@/shared/services/api'
import type { App } from '@/shared/types/api'
import { buildActivities } from '../lib/activity'
import ActivityItem from './ActivityItem'

interface ActivityTimelineProps {
    app: App
}

// What happened to an app lately, newest first.
function ActivityTimeline({ app }: ActivityTimelineProps) {
    const { data: jobs } = useAppJobs(app.id, app.node_id)
    const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set())
    const activities = useMemo(() => buildActivities(app, jobs), [app, jobs])

    const toggleExpanded = (activityId: string) => {
        setExpandedItems((prev) => {
            const next = new Set(prev)
            if (next.has(activityId)) next.delete(activityId)
            else next.add(activityId)
            return next
        })
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
            {activities.map((activity) => (
                <ActivityItem
                    key={activity.id}
                    activity={activity}
                    expanded={expandedItems.has(activity.id)}
                    onToggle={() => toggleExpanded(activity.id)}
                />
            ))}
        </ol>
    )
}

export default ActivityTimeline
