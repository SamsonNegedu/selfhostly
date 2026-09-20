import { Link } from 'react-router-dom'
import { AlertTriangle, Loader2, Play, Server } from 'lucide-react'
import { Button, buttonClasses } from '@/shared/components/ui/Button'
import { Card } from '@/shared/components/ui/Card'
import { useAttention } from '@/shared/hooks/useAttention'
import { appHref } from '@/shared/lib/routes'
import type { AttentionItem } from '@/shared/lib/attention'
import type { App } from '@/shared/types/api'
import type { useFleetActions } from '../hooks/useFleetActions'

const MAX_VISIBLE = 3

const TILE_CLASSES: Record<AttentionItem['kind'], string> = {
    'app-error': 'bg-status-err-bg text-status-err-fg',
    'node-offline': 'bg-status-warn-bg text-status-warn-fg',
}

// The few things that need a person now, at the top of Fleet. It is empty when everything is fine.
function AttentionStrip({ actions }: { actions: ReturnType<typeof useFleetActions> }) {
    const { items } = useAttention()
    if (items.length === 0) return null

    const visible = items.slice(0, MAX_VISIBLE)
    const hidden = items.length - visible.length

    return (
        <Card className="divide-y divide-border" aria-label="Needs attention" role="region">
            {visible.map((item) => {
                const Icon = item.kind === 'app-error' ? AlertTriangle : Server
                return (
                    <div key={item.id} className="flex flex-wrap items-center gap-3 p-4">
                        <div
                            aria-hidden="true"
                            className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${TILE_CLASSES[item.kind]}`}
                        >
                            <Icon className="h-[17px] w-[17px]" />
                        </div>
                        <div className="flex min-w-0 flex-1 flex-col">
                            <span className="font-semibold">{item.title}</span>
                            <span className="text-compact text-muted-foreground">{item.detail}</span>
                        </div>
                        <div className="flex flex-wrap items-center gap-2">
                            {item.app ? (
                                <>
                                    <Button
                                        onClick={() => actions.start(item.app as App)}
                                        disabled={actions.isBusy(item.app.id)}
                                    >
                                        {actions.isBusy(item.app.id) ? (
                                            <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                                        ) : (
                                            <Play className="h-4 w-4" />
                                        )}
                                        Retry start
                                    </Button>
                                    <Link
                                        to={appHref(item.app, 'logs')}
                                        className={buttonClasses({ variant: 'outline' })}
                                    >
                                        View logs
                                    </Link>
                                </>
                            ) : (
                                <Link to={item.href} className={buttonClasses({ variant: 'outline' })}>
                                    View node
                                </Link>
                            )}
                        </div>
                    </div>
                )
            })}
            {hidden > 0 && (
                <p className="px-4 py-2.5 text-compact text-muted-foreground">
                    And {hidden} more. Open the bell in the top bar to see them all.
                </p>
            )}
        </Card>
    )
}

export default AttentionStrip
