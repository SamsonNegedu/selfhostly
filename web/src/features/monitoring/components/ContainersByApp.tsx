import { useMemo } from 'react'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import type { App, ContainerInfo } from '@/shared/types/api'
import { useContainerActions } from '../hooks/useContainerActions'
import { groupContainers } from '../lib/container-groups'
import ContainerGroupCard from './ContainerGroupCard'

interface ContainersByAppProps {
    containers: ContainerInfo[]
    apps: App[]
    nodeName: (id: string) => string
}

// Every container that is running on the nodes, grouped under the app it belongs to. Containers that Selfhostly
// did not start are kept apart, because acting on them can break something else.
function ContainersByApp({ containers, apps, nodeName }: ContainersByAppProps) {
    const { pending, copy, request, cancel, confirm } = useContainerActions()
    const groups = useMemo(() => groupContainers(containers, apps), [containers, apps])

    return (
        <div className="flex flex-col gap-4">
            {groups.map((group) => (
                <ContainerGroupCard key={group.key} group={group} nodeName={nodeName} onAction={request} />
            ))}

            <ConfirmationDialog
                open={pending !== null}
                onOpenChange={(open) => !open && cancel()}
                title={copy?.title ?? ''}
                description={copy?.body ?? ''}
                confirmText={copy?.text}
                variant={pending?.action === 'restart' ? 'default' : 'destructive'}
                confirmationText={pending?.action === 'delete' ? pending.container.name : undefined}
                onConfirm={confirm}
            />
        </div>
    )
}

export default ContainersByApp
