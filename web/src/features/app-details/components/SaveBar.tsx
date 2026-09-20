import { AlertTriangle, Loader2, Rocket, Save, Undo2 } from 'lucide-react'
import ActionBar from '@/shared/components/ui/ActionBar'
import { Button } from '@/shared/components/ui/Button'

interface SaveBarProps {
    saving: boolean
    // When set, saving is not allowed and this says why.
    blockedReason?: string
    onDiscard: () => void
    onSave: (redeploy: boolean) => void
}

// The bar that appears once a file has unsaved changes. Save keeps the change for the next start, and
// Save and redeploy also restarts the containers so it takes effect now.
function SaveBar({ saving, blockedReason, onDiscard, onSave }: SaveBarProps) {
    return (
        <ActionBar label="Unsaved changes">
            <span className="flex items-center gap-2 text-sm font-medium">
                <AlertTriangle className="h-4 w-4 text-status-warn-fg" />
                {blockedReason ?? 'Unsaved changes'}
            </span>
            <div className="flex flex-wrap items-center gap-2">
                <Button variant="ghost" onClick={onDiscard} disabled={saving}>
                    <Undo2 className="h-4 w-4" />
                    Discard
                </Button>
                <Button variant="outline" onClick={() => onSave(false)} disabled={saving || !!blockedReason}>
                    <Save className="h-4 w-4" />
                    Save
                </Button>
                <Button onClick={() => onSave(true)} disabled={saving || !!blockedReason}>
                    {saving ? (
                        <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                    ) : (
                        <Rocket className="h-4 w-4" />
                    )}
                    Save and redeploy
                </Button>
            </div>
        </ActionBar>
    )
}

export default SaveBar
