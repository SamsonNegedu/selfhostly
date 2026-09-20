import { useState } from 'react'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { useDeleteApp, useStartApp, useStopApp, useUpdateAppContainers } from '@/shared/services/api'
import { useAppStore } from '@/shared/stores/app-store'
import type { App } from '@/shared/types/api'

type ConfirmedAction = 'stop' | 'update' | 'delete'

interface PendingAction {
    type: ConfirmedAction
    app: App
}

interface DialogCopy {
    title: (name: string) => string
    description: (name: string) => string
    confirmText: string
    destructive: boolean
}

const DIALOG_COPY: Record<ConfirmedAction, DialogCopy> = {
    stop: {
        title: (name) => `Stop ${name}?`,
        description: (name) => `${name} will be unavailable until you start it again. Its data is kept.`,
        confirmText: 'Stop app',
        destructive: false,
    },
    update: {
        title: (name) => `Update ${name}?`,
        description: (name) => `Pulls the latest images and restarts ${name}. It is unavailable for a short time.`,
        confirmText: 'Update',
        destructive: false,
    },
    delete: {
        title: (name) => `Delete ${name}?`,
        description: () => 'This removes the app, its compose file and its versions. It cannot be undone.',
        confirmText: 'Delete app',
        destructive: true,
    },
}

// Everything you can do to an app from the Fleet screen, with the confirmation dialog for the actions
// that interrupt or remove it. The screen renders `dialog` once, wherever it likes.
export function useFleetActions({ onDeleted }: { onDeleted?: (app: App) => void } = {}) {
    const startApp = useStartApp()
    const stopApp = useStopApp()
    const updateApp = useUpdateAppContainers()
    const deleteApp = useDeleteApp()
    const { toast } = useToast()
    const [pending, setPending] = useState<PendingAction | null>(null)
    const [busyIds, setBusyIds] = useState<Set<string>>(new Set())

    const setBusy = (id: string, busy: boolean) =>
        setBusyIds((current) => {
            const next = new Set(current)
            if (busy) next.add(id)
            else next.delete(id)
            return next
        })

    const run = (
        app: App,
        mutate: (callbacks: { onSuccess: () => void; onError: (error: Error) => void }) => void,
        success: [string, string],
        failure: string,
    ) => {
        setBusy(app.id, true)
        mutate({
            onSuccess: () => {
                toast.success(success[0], success[1])
                setBusy(app.id, false)
            },
            onError: (error) => {
                toast.error(failure, describeError(error))
                setBusy(app.id, false)
            },
        })
    }

    const start = (app: App) =>
        run(
            app,
            (callbacks) => startApp.mutate({ id: app.id, nodeId: app.node_id }, callbacks),
            ['App starting', `${app.name} is starting`],
            'Could not start app',
        )

    const confirm = () => {
        if (!pending) return
        const { type, app } = pending
        setPending(null)

        if (type === 'stop') {
            run(
                app,
                (callbacks) => stopApp.mutate({ id: app.id, nodeId: app.node_id }, callbacks),
                ['App stopped', `${app.name} has been stopped`],
                'Could not stop app',
            )
        } else if (type === 'update') {
            run(
                app,
                (callbacks) => updateApp.mutate({ id: app.id, nodeId: app.node_id }, callbacks),
                ['Update started', `${app.name} is updating`],
                'Could not start update',
            )
        } else {
            run(
                app,
                (callbacks) =>
                    deleteApp.mutate(
                        { id: app.id, nodeId: app.node_id },
                        {
                            ...callbacks,
                            onSuccess: () => {
                                useAppStore.getState().removeApp(app.id)
                                callbacks.onSuccess()
                                onDeleted?.(app)
                            },
                        },
                    ),
                ['App deleted', `${app.name} has been deleted`],
                'Could not delete app',
            )
        }
    }

    const copy = pending ? DIALOG_COPY[pending.type] : null

    const dialog = (
        <ConfirmationDialog
            open={pending !== null}
            onOpenChange={(open) => !open && setPending(null)}
            title={pending && copy ? copy.title(pending.app.name) : ''}
            description={pending && copy ? copy.description(pending.app.name) : ''}
            confirmText={copy?.confirmText}
            variant={copy?.destructive ? 'destructive' : 'default'}
            confirmationText={pending?.type === 'delete' ? pending.app.name : undefined}
            onConfirm={confirm}
        />
    )

    return {
        start,
        requestStop: (app: App) => setPending({ type: 'stop', app }),
        requestUpdate: (app: App) => setPending({ type: 'update', app }),
        requestDelete: (app: App) => setPending({ type: 'delete', app }),
        isBusy: (appId: string) => busyIds.has(appId),
        dialog,
    }
}
