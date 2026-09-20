import { useState } from 'react'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { useDeleteContainer, useRestartContainer, useStopContainer } from '@/shared/services/api'
import type { ContainerInfo } from '@/shared/types/api'
import { actionCopy, type ContainerAction } from '../lib/container-groups'

interface PendingAction {
    action: ContainerAction
    container: ContainerInfo
}

// Restarting, stopping and deleting containers. An action is first requested, then confirmed in a dialog, and only
// then sent. `copy` is what that dialog says.
export function useContainerActions() {
    const { toast } = useToast()
    const restart = useRestartContainer()
    const stop = useStopContainer()
    const remove = useDeleteContainer()
    const [pending, setPending] = useState<PendingAction | null>(null)

    const request = (action: ContainerAction, container: ContainerInfo) => setPending({ action, container })
    const cancel = () => setPending(null)

    const confirm = async () => {
        if (!pending) return
        const { action, container } = pending
        setPending(null)
        const args = { containerId: container.id, nodeId: container.node_id }
        try {
            if (action === 'restart') await restart.mutateAsync(args)
            else if (action === 'stop') await stop.mutateAsync(args)
            else await remove.mutateAsync(args)
            toast.success(
                action === 'restart'
                    ? 'Container restarted'
                    : action === 'stop'
                      ? 'Container stopped'
                      : 'Container deleted',
                container.name,
            )
        } catch (failure) {
            toast.error(`Could not ${action} the container`, describeError(failure))
        }
    }

    const copy = pending ? actionCopy(pending.action, pending.container) : null

    return { pending, copy, request, cancel, confirm }
}
