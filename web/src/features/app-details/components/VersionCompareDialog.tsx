import { X } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { DiffBlock } from '@/shared/components/ui/DiffBlock'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/shared/components/ui/Dialog'
import type { ComposeVersion } from '@/shared/types/api'

interface VersionCompareDialogProps {
    version: ComposeVersion | null
    // What the version is compared with: the editor, or the saved file.
    against: string
    againstLabel: string
    onClose: () => void
    // Shown only when a version can be loaded from this place.
    primaryLabel?: string
    onPrimary?: (version: ComposeVersion) => void
}

// One version next to the file it is compared with. Red lines are in the file now, green lines are in the version.
function VersionCompareDialog({
    version,
    against,
    againstLabel,
    onClose,
    primaryLabel,
    onPrimary,
}: VersionCompareDialogProps) {
    return (
        <Dialog open={!!version} onOpenChange={(open) => !open && onClose()}>
            <DialogContent className="flex max-h-[90vh] max-w-3xl flex-col overflow-hidden">
                <DialogHeader>
                    <DialogTitle>Version {version?.version}</DialogTitle>
                    <DialogDescription>
                        {version?.is_current
                            ? 'This is the version that is running now.'
                            : `Red lines are in ${againstLabel} now. Green lines are in this version.`}
                    </DialogDescription>
                </DialogHeader>
                {version && (
                    <>
                        <div className="flex flex-wrap gap-x-4 gap-y-1 text-compact text-muted-foreground">
                            {version.created_at && <span>{new Date(version.created_at).toLocaleString()}</span>}
                            {version.changed_by && <span>by {version.changed_by}</span>}
                            {version.rolled_back_from && <span>Rolled back from v{version.rolled_back_from}</span>}
                            {version.change_reason && <span>{version.change_reason}</span>}
                        </div>
                        <div className="min-h-0 flex-1 overflow-auto">
                            <DiffBlock
                                before={against}
                                after={version.compose_content}
                                aria-label={`Version ${version.version} compared with ${againstLabel}`}
                            />
                        </div>
                        <div className="flex justify-end gap-2">
                            <Button variant="outline" onClick={onClose}>
                                <X className="h-4 w-4" />
                                Close
                            </Button>
                            {primaryLabel && onPrimary && (
                                <Button onClick={() => onPrimary(version)} disabled={version.is_current}>
                                    {primaryLabel}
                                </Button>
                            )}
                        </div>
                    </>
                )}
            </DialogContent>
        </Dialog>
    )
}

export default VersionCompareDialog
