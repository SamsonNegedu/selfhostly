import {
    AlertTriangle,
    CheckCircle,
    ChevronDown,
    ChevronRight,
    Globe,
    Loader2,
    Pause,
    Play,
    RefreshCw,
    Upload,
    Zap,
} from 'lucide-react'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { formatAgo } from '@/shared/lib/attention'
import type { Activity, ActivityIcon, ActivityTone } from '../lib/activity'

// The long text of an entry is cut to one line until it is opened.
const LONG_TEXT = 80

const ICONS: Record<ActivityIcon, React.ReactNode> = {
    loading: <Loader2 aria-hidden="true" className="h-3.5 w-3.5 animate-spin" />,
    alert: <AlertTriangle className="h-3.5 w-3.5" />,
    created: <CheckCircle className="h-3.5 w-3.5" />,
    upload: <Upload className="h-3.5 w-3.5" />,
    play: <Play className="h-3.5 w-3.5" />,
    globe: <Globe className="h-3.5 w-3.5" />,
    zap: <Zap className="h-3.5 w-3.5" />,
    refresh: <RefreshCw className="h-3.5 w-3.5" />,
    pause: <Pause className="h-3.5 w-3.5" />,
}

const TEXT_TONE: Record<ActivityTone, string> = {
    ok: 'text-status-ok-fg',
    err: 'text-status-err-fg',
    info: 'text-status-info-fg',
    warn: 'text-status-warn-fg',
    idle: 'text-status-idle-fg',
}

const DOT_TONE: Record<ActivityTone, string> = {
    ok: 'bg-status-ok',
    err: 'bg-status-err',
    info: 'bg-status-info',
    warn: 'bg-status-warn',
    idle: 'bg-status-idle',
}

const STATUS_PILLS = {
    completed: { kind: 'ok', label: 'Completed' },
    failed: { kind: 'err', label: 'Failed' },
    running: { kind: 'info', label: 'Running' },
    pending: { kind: 'warn', label: 'Pending' },
} as const

interface ActivityItemProps {
    activity: Activity
    expanded: boolean
    onToggle: () => void
}

// One entry on the timeline: what happened, how it went, and when.
function ActivityItem({ activity, expanded, onToggle }: ActivityItemProps) {
    const pill = activity.status ? STATUS_PILLS[activity.status] : null
    const isLong = !!activity.details && activity.details.length > LONG_TEXT

    return (
        <li className="relative flex gap-3 py-2.5 pl-6">
            <span
                aria-hidden="true"
                className={`absolute left-0 top-[15px] h-[15px] w-[15px] rounded-full ring-4 ring-card ${DOT_TONE[activity.tone]}`}
            />
            <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                    <span aria-hidden="true" className={`shrink-0 ${TEXT_TONE[activity.tone]}`}>
                        {ICONS[activity.icon]}
                    </span>
                    <p className="text-sm font-medium">{activity.description}</p>
                    {pill && (
                        <StatusPill kind={pill.kind} size="sm">
                            {pill.label}
                        </StatusPill>
                    )}
                </div>
                {activity.details && (
                    <p
                        className={`mt-1 text-compact ${expanded ? 'break-words' : 'line-clamp-1'} ${activity.status === 'failed' ? 'text-status-err-fg' : 'text-muted-foreground'}`}
                    >
                        {activity.details}
                    </p>
                )}
                {isLong && (
                    <button
                        type="button"
                        aria-expanded={expanded}
                        onClick={onToggle}
                        className="mt-1 inline-flex min-h-[44px] items-center gap-1 text-compact text-muted-foreground hover:text-foreground md:min-h-0"
                    >
                        {expanded ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
                        {expanded ? 'Show less' : 'Show more'}
                    </button>
                )}
                <p className="mt-0.5 text-compact text-muted-foreground">
                    {formatAgo(activity.timestamp.toISOString())}
                </p>
            </div>
        </li>
    )
}

export default ActivityItem
