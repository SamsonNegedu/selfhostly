import type { App, Job } from '@/shared/types/api'

// How an entry is coloured, and which icon it wears. The screen maps these to classes and icons.
export type ActivityTone = 'ok' | 'err' | 'info' | 'warn' | 'idle'
export type ActivityIcon = 'loading' | 'alert' | 'created' | 'upload' | 'play' | 'globe' | 'zap' | 'refresh' | 'pause'

export interface Activity {
    id: string
    type: 'start' | 'stop' | 'update' | 'error' | 'create' | 'job'
    timestamp: Date
    description: string
    details?: string
    icon: ActivityIcon
    tone: ActivityTone
    status?: Job['status']
}

const JOB_NAMES: Record<string, string> = {
    app_create: 'App creation',
    app_update: 'App update',
    app_start: 'App start',
    tunnel_create: 'Custom tunnel creation',
    quick_tunnel: 'Quick Tunnel setup',
}

const JOB_ICONS: Record<string, ActivityIcon> = {
    app_create: 'created',
    app_update: 'upload',
    app_start: 'play',
    tunnel_create: 'globe',
    quick_tunnel: 'zap',
}

const JOB_TONES: Record<Job['status'], ActivityTone> = {
    failed: 'err',
    running: 'info',
    pending: 'warn',
    completed: 'ok',
}

// A bare "exit status 1" says nothing, so point at the logs instead.
const GENERIC_EXIT = /^exit status \d+$/
const LOGS_HINT = 'Build or deployment failed - click to view logs'

function jobIcon(job: Job): ActivityIcon {
    if (job.status === 'running') return 'loading'
    if (job.status === 'failed') return 'alert'
    return JOB_ICONS[job.type] ?? 'refresh'
}

function jobDescription(job: Job): string {
    const action = JOB_NAMES[job.type] || job.type

    if (job.status === 'completed') return `${action} completed`
    if (job.status === 'failed') return `${action} failed`
    if (job.status === 'running') return `${action} in progress`
    return `${action} started`
}

function jobDetails(job: Job): string | undefined {
    const details = job.status === 'failed' ? job.error_message : job.progress_message
    if (details && job.status === 'failed' && GENERIC_EXIT.test(details.trim())) return LOGS_HINT
    return details
}

// What happened to an app, newest first: its jobs, its creation when no job says so, and how it is right now.
export function buildActivities(app: App, jobs: Job[] | undefined): Activity[] {
    const items: Activity[] = []

    for (const job of jobs ?? []) {
        items.push({
            id: job.id,
            type: 'job',
            timestamp: new Date(job.completed_at || job.started_at || job.created_at),
            description: jobDescription(job),
            details: jobDetails(job),
            icon: jobIcon(job),
            tone: JOB_TONES[job.status],
            status: job.status,
        })
    }

    if (!jobs?.some((job) => job.type === 'app_create')) {
        items.push({
            id: 'create',
            type: 'create',
            timestamp: new Date(app.created_at),
            description: 'App was created',
            icon: 'created',
            tone: 'ok',
        })
    }

    if (app.status === 'running') {
        items.push({
            id: 'status-running',
            type: 'start',
            timestamp: new Date(app.updated_at),
            description: 'App is currently running',
            icon: 'play',
            tone: 'ok',
        })
    } else if (app.status === 'stopped') {
        items.push({
            id: 'status-stopped',
            type: 'stop',
            timestamp: new Date(app.updated_at),
            description: 'App is currently stopped',
            icon: 'pause',
            tone: 'idle',
        })
    }

    return items.sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
}
