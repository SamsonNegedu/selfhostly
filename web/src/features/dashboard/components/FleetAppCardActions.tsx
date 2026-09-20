import { Link } from 'react-router-dom'
import { ExternalLink, Loader2, Play, Square } from 'lucide-react'
import { Button, buttonClasses } from '@/shared/components/ui/Button'
import { ROUTES, appHref } from '@/shared/lib/routes'
import type { App } from '@/shared/types/api'
import type { useFleetActions } from '../hooks/useFleetActions'
import FleetAppMenu from './FleetAppMenu'

interface FleetAppCardActionsProps {
    app: App
    actions: ReturnType<typeof useFleetActions>
    unreachable: boolean
    busy: boolean
    isFailed: boolean
    isStopped: boolean
    isRunning: boolean
    inProgress: boolean
}

// The buttons at the foot of an app's card. Which ones show depends on the state the app is in.
function FleetAppCardActions({
    app,
    actions,
    unreachable,
    busy,
    isFailed,
    isStopped,
    isRunning,
    inProgress,
}: FleetAppCardActionsProps) {
    return (
        <div className="relative z-10 mt-auto flex flex-wrap items-center gap-2 pt-1">
            {unreachable && (
                <Link to={ROUTES.nodes} className={buttonClasses({ variant: 'outline' })}>
                    View node
                </Link>
            )}
            {(isStopped || isFailed) && (
                <Button onClick={() => actions.start(app)} disabled={busy}>
                    {busy ? (
                        <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                    ) : (
                        <Play className="h-4 w-4" />
                    )}
                    {isFailed ? 'Retry start' : 'Start'}
                </Button>
            )}
            {isFailed && (
                <Link to={appHref(app, 'logs')} className={buttonClasses({ variant: 'outline' })}>
                    View logs
                </Link>
            )}
            {isStopped && (
                <Link to={appHref(app, 'schedule')} className={buttonClasses({ variant: 'outline' })}>
                    Schedule
                </Link>
            )}
            {inProgress && (
                <Button variant="outline" disabled>
                    <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                    {app.status === 'updating' ? 'Updating' : 'Working'}
                </Button>
            )}
            {isRunning && (
                <Button variant="outline" onClick={() => actions.requestStop(app)} disabled={busy}>
                    {busy ? (
                        <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                    ) : (
                        <Square className="h-4 w-4" />
                    )}
                    Stop
                </Button>
            )}
            {isRunning && app.public_url && (
                <a
                    href={app.public_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className={buttonClasses({ variant: 'outline' })}
                >
                    <ExternalLink className="h-4 w-4" />
                    Open
                </a>
            )}

            <div className="ml-auto">
                <FleetAppMenu
                    app={app}
                    actions={actions}
                    unreachable={unreachable}
                    busy={busy}
                    inProgress={inProgress}
                />
            </div>
        </div>
    )
}

export default FleetAppCardActions
