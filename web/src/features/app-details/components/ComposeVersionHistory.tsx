import { useState } from 'react'
import { Button } from '@/shared/components/ui/Button'
import { RotateCcw, Eye, AlertCircle, CheckCircle2 } from 'lucide-react'
import { useComposeVersions, useRollbackToVersion } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { ComposeVersion } from '@/shared/types/api'

interface ComposeVersionHistoryProps {
    appId: string;
    nodeId: string;
    onVersionSelect?: (version: ComposeVersion) => void;
}

function ComposeVersionHistory({ appId, nodeId, onVersionSelect }: ComposeVersionHistoryProps) {
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
                    toast.success('Rolled back', `Successfully rolled back to version ${selectedVersion.version}`)
                    setShowRollbackDialog(false)
                    setSelectedVersion(null)
                },
                onError: (error) => {
                    toast.error('Rollback failed', error.message)
                }
            }
        )
    }

    const formatDate = (dateString: string) => {
        const date = new Date(dateString)
        return new Intl.DateTimeFormat('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        }).format(date)
    }

    const formatRelativeTime = (dateString: string) => {
        const date = new Date(dateString)
        const now = new Date()
        const diffMs = now.getTime() - date.getTime()
        const diffMins = Math.floor(diffMs / 60000)
        const diffHours = Math.floor(diffMins / 60)
        const diffDays = Math.floor(diffHours / 24)

        if (diffMins < 1) return 'just now'
        if (diffMins < 60) return `${diffMins}m ago`
        if (diffHours < 24) return `${diffHours}h ago`
        if (diffDays < 7) return `${diffDays}d ago`
        return formatDate(dateString)
    }

    if (isLoading) {
        return (
            <div className="flex items-center justify-center py-8">
                <div className="h-8 w-8 border-4 border-primary border-t-transparent rounded-full animate-spin" />
            </div>
        )
    }

    if (error) {
        return (
            <div className="flex items-center gap-2 text-destructive py-4">
                <AlertCircle className="h-5 w-5" />
                <span>Failed to load version history</span>
            </div>
        )
    }

    if (!versions || versions.length === 0) {
        return (
            <div className="text-muted-foreground text-center py-8">
                No version history available yet
            </div>
        )
    }

    return (
        <>
            <div className="space-y-4">
                <div className="space-y-2">
                        {versions.map((version, index) => (
                            <div
                                key={version.id}
                                className={`relative border rounded-lg p-3 transition-all hover:shadow-sm ${version.is_current
                                    ? 'border-primary bg-primary/5'
                                    : 'border-border hover:border-primary/50'
                                    }`}
                            >
                                {/* Timeline connector */}
                                {index < versions.length - 1 && (
                                    <div className="absolute left-6 top-full h-2 w-px bg-border" />
                                )}

                                <div className="flex items-center gap-3">
                                    {/* Version indicator */}
                                    <div className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center text-xs font-semibold ${version.is_current
                                        ? 'bg-primary text-primary-foreground'
                                        : 'bg-muted text-muted-foreground'
                                        }`}>
                                        v{version.version}
                                    </div>

                                    {/* Version details */}
                                    <div className="flex-1 min-w-0 flex items-center justify-between gap-2">
                                        <div className="flex items-center gap-2 min-w-0 flex-1">
                                            {version.is_current && (
                                                <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-primary/10 text-primary text-xs font-medium flex-shrink-0">
                                                    <CheckCircle2 className="h-3 w-3" />
                                                    Current
                                                </span>
                                            )}
                                            {version.rolled_back_from && (
                                                <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-600 dark:text-amber-400 text-xs font-medium flex-shrink-0">
                                                    <RotateCcw className="h-3 w-3" />
                                                    Rollback
                                                </span>
                                            )}

                                            <span className="text-xs text-muted-foreground truncate" title={formatDate(version.created_at)}>
                                                {formatRelativeTime(version.created_at)}
                                            </span>
                                        </div>

                                        {/* Actions */}
                                        <div className="flex gap-1 flex-shrink-0">
                                            {onVersionSelect && (
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    onClick={() => onVersionSelect(version)}
                                                    className="h-7 px-2 text-xs"
                                                    title="View details"
                                                >
                                                    <Eye className="h-3 w-3" />
                                                </Button>
                                            )}

                                            {!version.is_current && (
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    onClick={() => handleRollback(version)}
                                                    className="h-7 px-2 text-xs"
                                                    disabled={rollback.isPending}
                                                    title="Rollback to this version"
                                                >
                                                    <RotateCcw className="h-3 w-3" />
                                                </Button>
                                            )}
                                        </div>
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>

                {/* Info message */}
                <div className="p-2 bg-blue-50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-900/30 rounded text-xs text-muted-foreground">
                    <strong className="text-foreground">Tip:</strong> Click the eye icon to view version details.
                </div>
            </div>

            {/* Rollback Confirmation Dialog */}
            <ConfirmationDialog
                open={showRollbackDialog}
                onOpenChange={setShowRollbackDialog}
                title="Rollback to Previous Version?"
                description={`Are you sure you want to rollback to version ${selectedVersion?.version}? This will create a new version with the previous configuration. You'll need to update the containers afterwards to apply the changes.`}
                confirmText="Rollback"
                cancelText="Cancel"
                onConfirm={confirmRollback}
                isLoading={rollback.isPending}
            />
        </>
    )
}

export default ComposeVersionHistory
