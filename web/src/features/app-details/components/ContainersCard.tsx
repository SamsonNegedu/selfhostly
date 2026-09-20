import { useState } from 'react'
import { Layers, Loader2, RefreshCw } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { useToast } from '@/shared/components/ui/Toast'
import { useAppServices, useRestartAppService } from '@/shared/services/api'
import type { App } from '@/shared/types/api'
import OverviewCard from './OverviewCard'

// The app's containers, with a button to restart one of them after a confirmation.
function ContainersCard({ app }: { app: App }) {
    // Services come from the server, the same list the log viewer uses.
    const { data: services = [] } = useAppServices(app.id, app.node_id || '')
    const restartService = useRestartAppService()
    const { toast } = useToast()
    const [serviceToRestart, setServiceToRestart] = useState<string | null>(null)
    const isRunning = app.status === 'running'

    const handleRestartService = () => {
        if (!serviceToRestart || !app.node_id) return

        restartService.mutate(
            {
                appId: app.id,
                nodeId: app.node_id,
                serviceName: serviceToRestart,
            },
            {
                onSuccess: () => {
                    toast.success('Service restarted', `Service "${serviceToRestart}" has been restarted successfully`)
                    setServiceToRestart(null)
                },
                onError: (error) => {
                    toast.error('Failed to restart service', error instanceof Error ? error.message : 'Unknown error')
                },
            },
        )
    }

    return (
        <>
            <OverviewCard
                icon={<Layers className="h-4 w-4 text-muted-foreground" />}
                title="Containers"
                contentClassName=""
            >
                {services.length > 0 ? (
                    <ul className="flex flex-col gap-2">
                        {services.map((service) => {
                            const isRestarting = restartService.isPending && serviceToRestart === service
                            return (
                                <li
                                    key={service}
                                    className="flex items-center justify-between gap-3 rounded-lg bg-muted/50 px-3 py-2"
                                >
                                    <span className="truncate font-mono text-sm font-medium" title={service}>
                                        {service}
                                    </span>
                                    <div className="flex items-center gap-2">
                                        <StatusPill kind={isRunning ? 'ok' : 'idle'} size="sm">
                                            {isRunning ? 'Running' : 'Stopped'}
                                        </StatusPill>
                                        {isRunning && (
                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                aria-label={`Restart ${service}`}
                                                onClick={() => setServiceToRestart(service)}
                                                disabled={isRestarting}
                                            >
                                                {isRestarting ? (
                                                    <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                                                ) : (
                                                    <RefreshCw className="h-4 w-4" />
                                                )}
                                            </Button>
                                        )}
                                    </div>
                                </li>
                            )
                        })}
                    </ul>
                ) : (
                    <p className="py-4 text-center text-sm text-muted-foreground">No containers found</p>
                )}
            </OverviewCard>

            <ConfirmationDialog
                open={!!serviceToRestart}
                onOpenChange={(open: boolean) => !open && setServiceToRestart(null)}
                title={`Restart ${serviceToRestart ?? 'container'}?`}
                description="It is unavailable for a moment while it restarts."
                confirmText="Restart"
                cancelText="Cancel"
                onConfirm={handleRestartService}
                isLoading={restartService.isPending}
                variant="default"
            />
        </>
    )
}

export default ContainersCard
