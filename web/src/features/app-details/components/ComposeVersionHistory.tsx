import { useState } from 'react'
import { Button } from '@/shared/components/ui/Button'
import { RotateCcw, Eye } from 'lucide-react'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { formatAgo } from '@/shared/lib/attention'
import { useComposeVersions, useRollbackToVersion } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { ComposeVersion } from '@/shared/types/api'

interface ComposeVersionHistoryProps {
    appId: string;
    nodeId: string;
    onVersionSelect?: (version: ComposeVersion) => void;
    // Text buttons instead of icons, for the History tab where there is room.
    showLabels?: boolean;
}

function ComposeVersionHistory({ appId, nodeId, onVersionSelect, showLabels = false }: ComposeVersionHistoryProps) {
    const { data: versions, isLoading, error } = useComposeVersions(appId, nodeId)
    const rollback = useRollbackToVersion(appId, nodeId)
    const { toast } = useToast()
    const [selectedVersion, setSelectedVersion] = useState<ComposeVersion | null>(null)
    const [showRollbackDialog, setShowRollbackDialog] = useState(false)

    const handleRollback = (version: ComposeVersion) => {
        setSelectedVersion(version)
        setShowRollbackDialog(true)
    }

    const confirmRollback = () => {
        if (!selectedVersion) return

        rollback.mutate(
            { version: selectedVersion.version },
            {
                onSuccess: () => {
                    toast.success('Version restored', `Version ${selectedVersion.version} is now the current file`)
                    setShowRollbackDialog(false)
                    setSelectedVersion(null)
                },
                onError: (error) => {
                    toast.error('Could not restore', describeError(error))
                }
            }
        )
    }

    if (isLoading) {
        return (
            <div role="status" aria-label="Loading versions" className="flex flex-col gap-3">
                <Skeleton className="h-10" />
                <Skeleton className="h-10" />
            </div>
        )
    }

    if (error) return <ErrorState title="Could not load the versions" error={error} className="p-4" />

    if (!versions || versions.length === 0) {
        return <p className="py-6 text-center text-sm text-muted-foreground">No versions yet. Each save adds one.</p>
    }

    return (
        <>
            <ul className="flex max-h-[min(70vh,720px)] flex-col overflow-y-auto">
                {versions.map((version) => (
                    <li key={version.id} className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-border py-3 first:pt-0 last:border-b-0 last:pb-0">
                        <span className="w-9 shrink-0 font-mono text-sm font-semibold">v{version.version}</span>
                        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
                            {version.is_current && <StatusPill kind="ok" size="sm">Current</StatusPill>}
                            {version.rolled_back_from && <StatusPill kind="warn" size="sm">Restored</StatusPill>}
                            <span className="text-[13px] text-muted-foreground">{formatAgo(version.created_at)}</span>
                        </div>
                        <div className="flex shrink-0 gap-1">
                            {onVersionSelect && (
                                <Button variant="ghost" size="sm" onClick={() => onVersionSelect(version)} aria-label={`View version ${version.version}`} title="Compare with the current file">
                                    <Eye className="h-4 w-4" />
                                    {showLabels && 'Compare'}
                                </Button>
                            )}
                            {!version.is_current && (
                                <Button variant="ghost" size="sm" onClick={() => handleRollback(version)} disabled={rollback.isPending} aria-label={`Roll back to version ${version.version}`} title="Restore this version">
                                    <RotateCcw className="h-4 w-4" />
                                    {showLabels && 'Restore'}
                                </Button>
                            )}
                        </div>
                    </li>
                ))}
            </ul>

            <ConfirmationDialog
                open={showRollbackDialog}
                onOpenChange={setShowRollbackDialog}
                title={`Restore version ${selectedVersion?.version}?`}
                description="This saves that version as the newest one. Update the app afterwards to run it."
                confirmText="Restore"
                cancelText="Cancel"
                onConfirm={confirmRollback}
                isLoading={rollback.isPending}
            />
        </>
    )
}

export default ComposeVersionHistory
