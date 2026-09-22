import { useMemo, useState } from 'react'
import { Copy, FileUp, List, Plus, FileText } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { CardContent, Card } from '@/shared/components/ui/Card'
import { CodeBlock } from '@/shared/components/ui/CodeBlock'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { SegmentedControl } from '@/shared/components/ui/SegmentedControl'
import { useToast } from '@/shared/components/ui/Toast'
import { useCopyToClipboard } from '@/shared/hooks/useCopyToClipboard'
import { UnsavedChangesGuard } from '@/shared/components/UnsavedChangesGuard'
import type { App } from '@/shared/types/api'
import { readEnv, readServiceNames, removeEnv, setEnv } from '../lib/compose-env'
import { envChecks, rawEnvLines } from '../lib/env-view'
import { useSaveCompose } from '../hooks/useSaveCompose'
import { ComposeCheckList } from './ComposeChecks'
import { AddVariableDialog, ImportDialog } from './EnvDialogs'
import EnvTable from './EnvTable'
import SaveBar from './SaveBar'

type View = 'table' | 'raw'

const VIEW_OPTIONS = [
    { value: 'table', label: 'Table', icon: <List className="h-4 w-4" /> },
    { value: 'raw', label: 'Raw', icon: <FileText className="h-4 w-4" /> },
]
const SAVE_COPY = {
    redeploying: 'The containers are restarting with the new variables',
    saved: 'The new variables apply the next time the app starts or updates',
}

// The variables an app's containers are started with. They live in the compose file, so editing them here
// edits that file, and the same Save and Save and redeploy apply.
function EnvironmentTab({ app }: { app: App }) {
    const { toast } = useToast()
    const copyText = useCopyToClipboard()
    const { save: saveCompose, saving } = useSaveCompose(app.id, app.node_id, SAVE_COPY)

    const [draft, setDraft] = useState(app.compose_content)
    const [view, setView] = useState<View>('table')
    const [revealed, setRevealed] = useState<Set<string>>(new Set())
    const [adding, setAdding] = useState(false)
    const [importing, setImporting] = useState(false)

    // Show the saved file again whenever it changes underneath, for example after a rollback.
    const [savedContent, setSavedContent] = useState(app.compose_content)
    if (savedContent !== app.compose_content) {
        setSavedContent(app.compose_content)
        setDraft(app.compose_content)
    }

    const services = useMemo(() => readServiceNames(draft), [draft])
    const current = useMemo(() => readEnv(draft), [draft])
    const saved = useMemo(() => readEnv(app.compose_content).entries, [app.compose_content])
    const dirty = draft !== app.compose_content

    const checks = useMemo(() => envChecks(current, draft), [current, draft])

    const toggleReveal = (id: string) =>
        setRevealed((currentSet) => {
            const next = new Set(currentSet)
            if (next.has(id)) next.delete(id)
            else next.add(id)
            return next
        })

    const save = (redeploy: boolean) => saveCompose(draft, redeploy)

    const copyEnv = () =>
        copyText(
            current.entries.map((entry) => `${entry.key}=${entry.value}`).join('\n'),
            'Variables copied as a .env file',
        )

    const rawLines = useMemo(
        () => rawEnvLines(services, current.entries, revealed),
        [services, current.entries, revealed],
    )

    return (
        <div className="flex flex-col gap-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <SegmentedControl
                    aria-label="Environment view"
                    options={VIEW_OPTIONS}
                    value={view}
                    onValueChange={(value) => setView(value as View)}
                />
                <div className="flex flex-wrap items-center gap-2">
                    <Button variant="outline" onClick={() => setImporting(true)} disabled={services.length === 0}>
                        <FileUp className="h-4 w-4" />
                        Import .env
                    </Button>
                    {view === 'raw' && (
                        <Button variant="outline" onClick={copyEnv} disabled={current.entries.length === 0}>
                            <Copy className="h-4 w-4" />
                            Copy
                        </Button>
                    )}
                    <Button onClick={() => setAdding(true)} disabled={services.length === 0}>
                        <Plus className="h-4 w-4" />
                        Add variable
                    </Button>
                </div>
            </div>

            <ComposeCheckList checks={checks} className="gap-1.5" />

            <div className="flex flex-col gap-5">
                {current.entries.length === 0 && !current.error ? (
                    <EmptyState
                        icon={<Plus className="h-5 w-5" />}
                        title="No variables yet"
                        description="Variables set here go to the app's containers. Add one, or import a .env file."
                        className="min-w-0 py-12"
                    />
                ) : view === 'raw' ? (
                    <Card className="min-w-0">
                        <CardContent className="p-4">
                            <CodeBlock lines={rawLines} aria-label="Variables as a .env file" />
                            <p className="mt-3 text-compact text-muted-foreground">
                                Secrets stay hidden here until you reveal them in the table.
                            </p>
                        </CardContent>
                    </Card>
                ) : (
                    <EnvTable
                        entries={current.entries}
                        saved={saved}
                        revealed={revealed}
                        onToggleReveal={toggleReveal}
                        onChangeValue={(entry, value) => setDraft(setEnv(draft, entry.service, entry.key, value))}
                        onRemove={(entry) => setDraft(removeEnv(draft, entry.service, entry.key))}
                    />
                )}
            </div>

            <UnsavedChangesGuard when={dirty} />

            {dirty && (
                <SaveBar
                    saving={saving}
                    blockedReason={current.error ? 'Fix the compose file to save' : undefined}
                    onDiscard={() => setDraft(app.compose_content)}
                    onSave={save}
                />
            )}

            <AddVariableDialog
                open={adding}
                services={services}
                existing={current.entries}
                onOpenChange={setAdding}
                onAdd={(service, key, value) => setDraft(setEnv(draft, service, key, value))}
            />
            <ImportDialog
                open={importing}
                services={services}
                onOpenChange={setImporting}
                onImport={(service, pairs) => {
                    let next = draft
                    for (const pair of pairs) next = setEnv(next, service, pair.key, pair.value)
                    setDraft(next)
                    toast.success(
                        'Imported',
                        `${pairs.length} ${pairs.length === 1 ? 'variable' : 'variables'} added to ${service}. Save to keep them.`,
                    )
                }}
            />
        </div>
    )
}

export default EnvironmentTab
