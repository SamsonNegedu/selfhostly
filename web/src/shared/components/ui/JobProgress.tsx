import { Clock, Loader2, CheckCircle, XCircle, AlertCircle } from 'lucide-react'
import type { Job } from '@/shared/types/api'

interface JobProgressProps {
  job: Job
  compact?: boolean
}

export function JobProgress({ job, compact = false }: JobProgressProps) {
  const statusConfig = {
    pending: {
      icon: <Clock className="h-4 w-4 text-muted-foreground" />,
      color: 'text-muted-foreground',
    },
    running: {
      icon: <Loader2 className="h-4 w-4 animate-spin text-primary" />,
      color: 'text-primary',
    },
    completed: {
      icon: <CheckCircle className="h-4 w-4 text-green-600" />,
      color: 'text-green-600',
    },
    failed: {
      icon: <XCircle className="h-4 w-4 text-destructive" />,
      color: 'text-destructive',
    },
  }

  const config = statusConfig[job.status]
  const message = job.progress_message || getDefaultMessage(job.status)

  if (compact) {
    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {config.icon}
            <span className={`text-sm ${config.color}`}>{message}</span>
          </div>
          {(job.status === 'running' || job.status === 'pending') && (
            <span className="text-sm text-muted-foreground">{job.progress}%</span>
          )}
        </div>
        {(job.status === 'pending' || job.status === 'running') && (
          <div className="w-full bg-secondary rounded-full h-2">
            <div
              className="bg-primary h-2 rounded-full transition-all duration-300"
              style={{ width: `${job.progress}%` }}
            />
          </div>
        )}
        {job.status === 'failed' && job.error_message && (
          <div className="flex items-start gap-2 p-2 bg-destructive/10 rounded text-sm text-destructive">
            <AlertCircle className="h-4 w-4 flex-shrink-0 mt-0.5" />
            <span>{job.error_message}</span>
          </div>
        )}
      </div>
    )
  }

  // Full layout for cards
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {config.icon}
          <div>
            <p className={`text-sm font-medium ${config.color}`}>{message}</p>
            <p className="text-xs text-muted-foreground">
              {job.status === 'completed' && job.completed_at
                ? `Completed ${formatRelativeTime(job.completed_at)}`
                : job.status === 'failed' && job.completed_at
                  ? `Failed ${formatRelativeTime(job.completed_at)}`
                  : job.status === 'running' && job.started_at
                    ? `Started ${formatRelativeTime(job.started_at)}`
                    : ''}
            </p>
          </div>
        </div>
        {(job.status === 'running' || job.status === 'pending') && (
          <span className="text-lg font-semibold text-muted-foreground">{job.progress}%</span>
        )}
      </div>

      {(job.status === 'pending' || job.status === 'running') && (
        <div className="w-full bg-secondary rounded-full h-2.5">
          <div
            className="bg-primary h-2.5 rounded-full transition-all duration-300"
            style={{ width: `${job.progress}%` }}
          />
        </div>
      )}

      {job.status === 'failed' && job.error_message && (
        <div className="flex items-start gap-2 p-3 bg-destructive/10 rounded border border-destructive/20">
          <AlertCircle className="h-5 w-5 text-destructive flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-medium text-destructive">Operation Failed</p>
            <p className="text-sm text-destructive/80 mt-1">{job.error_message}</p>
          </div>
        </div>
      )}
    </div>
  )
}

function getDefaultMessage(status: Job['status']): string {
  switch (status) {
    case 'pending':
      return 'Queued...'
    case 'running':
      return 'Processing...'
    case 'completed':
      return 'Completed'
    case 'failed':
      return 'Failed'
  }
}

function formatRelativeTime(timestamp: string): string {
  const now = new Date()
  const then = new Date(timestamp)
  const diffMs = now.getTime() - then.getTime()
  const diffSecs = Math.floor(diffMs / 1000)

  if (diffSecs < 60) return `${diffSecs}s ago`
  const diffMins = Math.floor(diffSecs / 60)
  if (diffMins < 60) return `${diffMins}m ago`
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  const diffDays = Math.floor(diffHours / 24)
  return `${diffDays}d ago`
}
