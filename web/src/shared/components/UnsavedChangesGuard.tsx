import { useEffect } from 'react'
import { useBlocker } from 'react-router-dom'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'

// Asks before an edit is thrown away. Closing or reloading the tab uses the browser's own prompt, and moving to
// another page or another tab of the app page (the tab is in the address) uses our dialog. Render it next to the
// editor with `when` set to whether there are unsaved changes.
export function UnsavedChangesGuard({ when }: { when: boolean }) {
    const blocker = useBlocker(
        ({ currentLocation, nextLocation }) =>
            when && currentLocation.pathname + currentLocation.search !== nextLocation.pathname + nextLocation.search,
    )

    useEffect(() => {
        if (!when) return
        const warn = (event: BeforeUnloadEvent) => event.preventDefault()
        window.addEventListener('beforeunload', warn)
        return () => window.removeEventListener('beforeunload', warn)
    }, [when])

    return (
        <ConfirmationDialog
            open={blocker.state === 'blocked'}
            onOpenChange={(open) => {
                if (!open && blocker.state === 'blocked') blocker.reset()
            }}
            title="Leave without saving?"
            description="You have changes that are not saved. If you leave now, they are lost."
            confirmText="Leave without saving"
            cancelText="Keep editing"
            variant="destructive"
            onConfirm={() => {
                if (blocker.state === 'blocked') blocker.proceed()
            }}
        />
    )
}
