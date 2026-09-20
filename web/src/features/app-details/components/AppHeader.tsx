import { Globe } from 'lucide-react'
import { AppTile } from '@/shared/components/ui/AppTile'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { appStatusMeta } from '@/shared/lib/status'
import type { App } from '@/shared/types/api'
import type { useFleetActions } from '@/features/dashboard/hooks/useFleetActions'
import { AppActions } from './AppActions'

interface AppHeaderProps {
    app: App
    nodeName?: string
    actions: ReturnType<typeof useFleetActions>
    isRefreshing: boolean
    // On phones the actions are pinned to the bottom of the screen.
    sticky: boolean
    onRefresh: () => void
}

// The app's name and status, where it is reachable and where it runs, and the buttons to act on it.
function AppHeader({ app, nodeName, actions, isRefreshing, sticky, onRefresh }: AppHeaderProps) {
    const meta = appStatusMeta(app.status)

    return (
        <header className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
            <div className="flex min-w-0 items-start gap-3.5">
                <AppTile name={app.name} size="lg" />
                <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2.5">
                        <h1 className="truncate text-2xl font-semibold tracking-tight" title={app.name}>
                            {app.name}
                        </h1>
                        <StatusPill kind={meta.kind}>{meta.label}</StatusPill>
                    </div>
                    <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-compact text-muted-foreground">
                        {app.public_url && (
                            <a
                                href={app.public_url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="inline-flex min-h-[44px] items-center gap-1.5 font-mono text-status-info-fg hover:underline md:min-h-0"
                            >
                                <Globe className="h-3.5 w-3.5 shrink-0" />
                                <span className="truncate" title={app.public_url}>
                                    {app.public_url.replace(/^https?:\/\//, '')}
                                </span>
                            </a>
                        )}
                        {nodeName && <span>on {nodeName}</span>}
                    </div>
                    {app.description && <p className="mt-1 text-sm text-muted-foreground">{app.description}</p>}
                </div>
            </div>
            <AppActions
                appId={app.id}
                nodeId={app.node_id}
                appStatus={app.status}
                publicUrl={app.public_url}
                isBusy={actions.isBusy(app.id)}
                isRefreshing={isRefreshing}
                sticky={sticky}
                onRefresh={onRefresh}
                onStart={() => actions.start(app)}
                onStop={() => actions.requestStop(app)}
                onUpdate={() => actions.requestUpdate(app)}
                onDelete={() => actions.requestDelete(app)}
            />
        </header>
    )
}

export default AppHeader
