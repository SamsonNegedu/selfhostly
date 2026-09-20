import { useState } from 'react'
import { Download } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import {
    Dialog,
    DialogClose,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogOverlay,
    DialogPortal,
    DialogTitle,
} from '@/shared/components/ui/Dialog'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'

interface ConfirmUpdateDialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    version: string
    isLoading: boolean
    onConfirm: () => void
}

// An update restarts Selfhostly, so the person types the version they are installing before it starts. This is
// built from the dialog primitives rather than ConfirmationDialog because its two controls need test ids.
function ConfirmUpdateDialog({ open, onOpenChange, version, isLoading, onConfirm }: ConfirmUpdateDialogProps) {
    const [typed, setTyped] = useState('')

    // Clear what was typed when the dialog closes.
    const [wasOpen, setWasOpen] = useState(open)
    if (open !== wasOpen) {
        setWasOpen(open)
        if (!open) setTyped('')
    }

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogPortal>
                <DialogOverlay />
                <DialogContent className="sm:max-w-md">
                    <div className="flex items-start gap-4">
                        <div
                            aria-hidden="true"
                            className="flex h-10 w-10 shrink-0 items-center justify-center rounded-[10px] bg-status-info-bg text-status-info-fg"
                        >
                            <Download className="h-5 w-5" />
                        </div>
                        <DialogHeader className="space-y-1 pr-6 text-left">
                            <DialogTitle>Update Selfhostly to {version}?</DialogTitle>
                            <DialogDescription>
                                This page and the API are unavailable for 10 to 30 seconds while they restart. Your apps
                                keep running. If anything goes wrong, Selfhostly puts the old version back by itself.
                            </DialogDescription>
                        </DialogHeader>
                    </div>
                    <Field label={`Type ${version} to confirm`}>
                        <Input
                            data-testid="update-confirm-input"
                            value={typed}
                            onChange={(event) => setTyped(event.target.value)}
                            autoComplete="off"
                            spellCheck={false}
                        />
                    </Field>
                    <DialogFooter className="flex flex-row justify-end gap-2 space-x-0">
                        <DialogClose asChild>
                            <Button variant="outline" disabled={isLoading}>
                                Cancel
                            </Button>
                        </DialogClose>
                        <Button
                            data-testid="update-confirm-button"
                            onClick={onConfirm}
                            disabled={isLoading || typed !== version}
                        >
                            {isLoading ? 'Starting...' : 'Update now'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </DialogPortal>
        </Dialog>
    )
}

export default ConfirmUpdateDialog
