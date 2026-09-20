import * as React from 'react'
import { AlertTriangle, Info } from 'lucide-react'
import {
    Dialog,
    DialogPortal,
    DialogOverlay,
    DialogClose,
    DialogContent,
    DialogHeader,
    DialogFooter,
    DialogTitle,
    DialogDescription,
} from './Dialog'
import { Button } from './Button'
import { Field } from './Field'
import { Input } from './Input'

interface ConfirmationDialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    title: string
    description: string
    confirmText?: string
    cancelText?: string
    onConfirm: () => void
    isLoading?: boolean
    variant?: 'default' | 'destructive'
    icon?: React.ReactNode
    /** When set, the confirm button stays disabled until this exact text is typed. */
    confirmationText?: string
    /** Extra content between the description and the buttons, such as an option to tick. */
    children?: React.ReactNode
}

function ConfirmationDialog({
    open,
    onOpenChange,
    title,
    description,
    confirmText = 'Confirm',
    cancelText = 'Cancel',
    onConfirm,
    isLoading = false,
    variant = 'default',
    icon,
    confirmationText,
    children,
}: ConfirmationDialogProps) {
    const [typed, setTyped] = React.useState('')
    const isDestructive = variant === 'destructive'
    const needsTyping = confirmationText !== undefined
    const typingMatches = !needsTyping || typed === confirmationText

    React.useEffect(() => {
        if (!open) setTyped('')
    }, [open])

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogPortal>
                <DialogOverlay />
                <DialogContent className="sm:max-w-md">
                    <div className="flex items-start gap-4">
                        <div
                            aria-hidden="true"
                            className={
                                isDestructive
                                    ? 'flex h-10 w-10 shrink-0 items-center justify-center rounded-[10px] bg-status-err-bg text-status-err-fg'
                                    : 'flex h-10 w-10 shrink-0 items-center justify-center rounded-[10px] bg-status-info-bg text-status-info-fg'
                            }
                        >
                            {icon ?? (isDestructive ? <AlertTriangle className="h-5 w-5" /> : <Info className="h-5 w-5" />)}
                        </div>
                        <DialogHeader className="space-y-1 pr-6 text-left">
                            <DialogTitle>{title}</DialogTitle>
                            <DialogDescription>{description}</DialogDescription>
                        </DialogHeader>
                    </div>
                    {children}
                    {needsTyping && (
                        <Field label={`Type ${confirmationText} to confirm`}>
                            <Input
                                value={typed}
                                onChange={(event) => setTyped(event.target.value)}
                                autoComplete="off"
                                spellCheck={false}
                            />
                        </Field>
                    )}
                    <DialogFooter className="flex flex-row justify-end gap-2 space-x-0">
                        <DialogClose asChild>
                            <Button variant="outline" disabled={isLoading}>
                                {cancelText}
                            </Button>
                        </DialogClose>
                        <Button
                            variant={isDestructive ? 'danger' : 'default'}
                            onClick={onConfirm}
                            disabled={isLoading || !typingMatches}
                        >
                            {isLoading ? 'Processing...' : confirmText}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </DialogPortal>
        </Dialog>
    )
}

export default ConfirmationDialog
