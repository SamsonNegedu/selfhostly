import { Button } from '@/shared/components/ui/Button'
import { DiffBlock } from '@/shared/components/ui/DiffBlock'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/shared/components/ui/Dialog'

interface FormatPreviewDialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    // Why the file could not be formatted. When set, the preview is replaced by this message.
    error: string | null
    before: string
    after: string
    onApply: () => void
}

// Shows what Format would change, as a diff, and only touches the editor when the person applies it.
function FormatPreviewDialog({ open, onOpenChange, error, before, after, onApply }: FormatPreviewDialogProps) {
    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="flex max-h-[90vh] max-w-3xl flex-col overflow-hidden">
                <DialogHeader>
                    <DialogTitle>Format preview</DialogTitle>
                    <DialogDescription>
                        Red lines are removed and green lines are added. Nothing changes until you apply it.
                    </DialogDescription>
                </DialogHeader>
                {error ? (
                    <p role="alert" className="text-sm text-status-err-fg">
                        Could not format the file: {error}
                    </p>
                ) : (
                    <div className="min-h-0 flex-1 overflow-auto">
                        <DiffBlock before={before} after={after} aria-label="Formatting changes" />
                    </div>
                )}
                <div className="flex justify-end gap-2">
                    <Button variant="outline" onClick={() => onOpenChange(false)}>
                        Cancel
                    </Button>
                    {!error && <Button onClick={onApply}>Apply format</Button>}
                </div>
            </DialogContent>
        </Dialog>
    )
}

export default FormatPreviewDialog
