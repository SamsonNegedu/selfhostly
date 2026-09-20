import { useState, useEffect, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { FileCode, Sparkles } from 'lucide-react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/Card'
import { Button, buttonClasses } from '@/shared/components/ui/Button'
import { useToast } from '@/shared/components/ui/Toast'
import { UnsavedChangesGuard } from '@/shared/components/UnsavedChangesGuard'
import { appHref } from '@/shared/lib/routes'
import { useApp, useApps } from '@/shared/services/api'
import type { ComposeVersion } from '@/shared/types/api'
import { checkCompose } from '../lib/compose-checks'
import { readEnv } from '../lib/compose-env'
import { formatCompose } from '../lib/compose-format'
import { useSaveCompose } from '../hooks/useSaveCompose'
import { useYamlEditorView } from '../hooks/useYamlEditorView'
import ComposeChecks from './ComposeChecks'
import ComposeVersionHistory from './ComposeVersionHistory'
import FormatPreviewDialog from './FormatPreviewDialog'
import SaveBar from './SaveBar'
import VersionCompareDialog from './VersionCompareDialog'

interface ComposeEditorProps {
    appId: string
    nodeId: string
    initialComposeContent: string
}

const SAVE_COPY = {
    redeploying: 'The containers are restarting with the new config',
    saved: 'The new config applies the next time the app starts or updates',
}

function ComposeEditor({ appId, nodeId, initialComposeContent }: ComposeEditorProps) {
    const [composeContent, setComposeContent] = useState(initialComposeContent)
    const [hasChanges, setHasChanges] = useState(false)
    const [showFormatDialog, setShowFormatDialog] = useState(false)
    const [formattedContent, setFormattedContent] = useState('')
    const [formatError, setFormatError] = useState<string | null>(null)
    const [viewingVersion, setViewingVersion] = useState<ComposeVersion | null>(null)
    const { data: nodeApps = [] } = useApps([nodeId])
    const { data: app } = useApp(appId, nodeId)
    const { toast } = useToast()
    const { save, saving } = useSaveCompose(appId, nodeId, SAVE_COPY)

    const checkResult = useMemo(
        () =>
            checkCompose(
                composeContent,
                nodeApps.filter((other) => other.id !== appId),
            ),
        [composeContent, nodeApps, appId],
    )
    const envCount = useMemo(() => readEnv(composeContent).entries.length, [composeContent])
    const lineCount = composeContent.split('\n').length
    const charCount = composeContent.length

    const { containerRef, currentDoc, replaceDoc } = useYamlEditorView({
        initialDoc: composeContent,
        onDocChange: (doc) => {
            setComposeContent(doc)
            setHasChanges(doc !== initialComposeContent)
        },
    })

    // The saved file changed outside the editor (a rollback, or a save that came back): drop the edits and show it.
    const [seenInitial, setSeenInitial] = useState(initialComposeContent)
    if (seenInitial !== initialComposeContent) {
        setSeenInitial(initialComposeContent)
        setComposeContent(initialComposeContent)
        setHasChanges(false)
    }
    useEffect(() => {
        if (initialComposeContent !== currentDoc()) replaceDoc(initialComposeContent)
    }, [initialComposeContent, currentDoc, replaceDoc])

    const handleSave = async (redeploy: boolean) => {
        if (!hasChanges || checkResult.blocked) return
        await save(composeContent, redeploy, () => setHasChanges(false))
    }

    const handleReset = () => {
        if (!replaceDoc(initialComposeContent)) return

        setComposeContent(initialComposeContent)
        setHasChanges(false)
        toast.info('Changes discarded', 'The editor shows the saved config again')
    }

    const handleLoadVersion = () => {
        if (!viewingVersion || !replaceDoc(viewingVersion.compose_content)) return

        setComposeContent(viewingVersion.compose_content)
        setHasChanges(viewingVersion.compose_content !== initialComposeContent)
        setViewingVersion(null)
        toast.success('Version loaded', `Loaded version ${viewingVersion.version} into editor`)
    }

    const handleFormat = () => {
        try {
            setFormattedContent(formatCompose(composeContent))
            setFormatError(null)
            setShowFormatDialog(true)
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : 'Failed to format YAML'
            toast.error('Format failed', errorMessage)
            setFormatError(errorMessage)
        }
    }

    const handleApplyFormat = () => {
        if (!replaceDoc(formattedContent)) {
            toast.error('Editor not ready', 'Please wait for the editor to load')
            return
        }

        setComposeContent(formattedContent)
        setHasChanges(formattedContent !== initialComposeContent)
        setShowFormatDialog(false)
        toast.success('Formatted', 'YAML has been formatted')
    }

    return (
        <div className="flex flex-col gap-5">
            <div className="grid grid-cols-1 gap-5 lg:grid-cols-[minmax(0,1fr)_340px]">
                <Card className="min-w-0">
                    <CardHeader>
                        <div className="flex flex-wrap items-center justify-between gap-3">
                            <CardTitle className="flex items-center gap-2 font-mono text-sm font-medium">
                                <FileCode className="h-4 w-4 text-muted-foreground" />
                                docker-compose.yml
                            </CardTitle>
                            <Button variant="outline" size="sm" onClick={handleFormat} disabled={checkResult.blocked}>
                                <Sparkles className="h-4 w-4" />
                                Format
                            </Button>
                        </div>
                    </CardHeader>
                    <CardContent className="flex flex-col gap-3">
                        <div className="overflow-hidden rounded-lg border border-border">
                            <div ref={containerRef} aria-label="Compose file editor" />
                        </div>
                        <div className="flex items-center justify-between text-xs text-muted-foreground">
                            <span>
                                {lineCount} lines · {charCount} characters
                            </span>
                            <span>YAML</span>
                        </div>
                    </CardContent>
                </Card>

                <div className="flex min-w-0 flex-col gap-5">
                    <ComposeChecks checks={checkResult.checks} />
                    <Card>
                        <CardHeader>
                            <CardTitle className="text-base">Environment</CardTitle>
                        </CardHeader>
                        <CardContent className="flex flex-col gap-3">
                            <p className="text-sm text-muted-foreground">
                                {envCount === 0
                                    ? 'No variables set. Add them in the Environment tab.'
                                    : `${envCount} ${envCount === 1 ? 'variable' : 'variables'} set. Secrets are hidden in the Environment tab.`}
                            </p>
                            {app && (
                                <Link
                                    to={appHref(app, 'environment')}
                                    className={buttonClasses({ variant: 'outline' })}
                                >
                                    Open Environment
                                </Link>
                            )}
                        </CardContent>
                    </Card>
                    <Card>
                        <CardHeader>
                            <CardTitle className="text-base">Versions</CardTitle>
                        </CardHeader>
                        <CardContent>
                            <ComposeVersionHistory appId={appId} nodeId={nodeId} onVersionSelect={setViewingVersion} />
                        </CardContent>
                    </Card>
                </div>
            </div>

            <UnsavedChangesGuard when={hasChanges} />

            {hasChanges && (
                <SaveBar
                    saving={saving}
                    blockedReason={checkResult.blocked ? 'Fix the problems above to save' : undefined}
                    onDiscard={handleReset}
                    onSave={handleSave}
                />
            )}

            <VersionCompareDialog
                version={viewingVersion}
                against={composeContent}
                againstLabel="your editor"
                onClose={() => setViewingVersion(null)}
                primaryLabel="Load into editor"
                onPrimary={handleLoadVersion}
            />

            <FormatPreviewDialog
                open={showFormatDialog}
                onOpenChange={setShowFormatDialog}
                error={formatError}
                before={composeContent}
                after={formattedContent}
                onApply={handleApplyFormat}
            />
        </div>
    )
}

export default ComposeEditor
