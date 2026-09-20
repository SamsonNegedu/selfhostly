import { useState, useEffect, useMemo, useRef } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/Card'
import { Button } from '@/shared/components/ui/Button'
import { FileCode, Sparkles } from 'lucide-react'
import { useApps, useUpdateApp, useUpdateAppContainers } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { DiffBlock } from '@/shared/components/ui/DiffBlock'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/shared/components/ui/Dialog'
import ComposeChecks from './ComposeChecks'
import SaveBar from './SaveBar'
import VersionCompareDialog from './VersionCompareDialog'
import ComposeVersionHistory from './ComposeVersionHistory'
import { Link } from 'react-router-dom'
import { readEnv } from '../lib/compose-env'
import { buttonClasses } from '@/shared/components/ui/Button'
import { appHref } from '@/shared/lib/routes'
import { useApp } from '@/shared/services/api'
import { checkCompose } from '../lib/compose-checks'
import { parse, stringify } from 'yaml'
import { EditorView } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { yamlEditorTheme } from '@/shared/components/ui/YamlEditor'
import { yaml } from '@codemirror/lang-yaml'
import { searchKeymap, highlightSelectionMatches } from '@codemirror/search'
import {
    keymap,
    lineNumbers,
    highlightActiveLine,
    highlightActiveLineGutter,
    highlightSpecialChars,
    drawSelection,
    dropCursor,
    rectangularSelection,
    crosshairCursor,
} from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import {
    foldGutter,
    foldKeymap,
    syntaxHighlighting,
    defaultHighlightStyle,
    bracketMatching,
    indentOnInput,
} from '@codemirror/language'
import { autocompletion, completionKeymap, closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import type { ComposeVersion } from '@/shared/types/api'

interface ComposeEditorProps {
    appId: string
    nodeId: string
    initialComposeContent: string
}

function ComposeEditor({ appId, nodeId, initialComposeContent }: ComposeEditorProps) {
    const [composeContent, setComposeContent] = useState(initialComposeContent)
    const [isSaving, setIsSaving] = useState(false)
    const [hasChanges, setHasChanges] = useState(false)
    const [lineCount, setLineCount] = useState(0)
    const [charCount, setCharCount] = useState(0)
    const [showFormatDialog, setShowFormatDialog] = useState(false)
    const [formattedContent, setFormattedContent] = useState('')
    const [formatError, setFormatError] = useState<string | null>(null)
    const [viewingVersion, setViewingVersion] = useState<ComposeVersion | null>(null)
    const editorRef = useRef<EditorView | null>(null)
    const editorContainerRef = useRef<HTMLDivElement | null>(null)
    const updateApp = useUpdateApp(appId, nodeId)
    const updateAppContainers = useUpdateAppContainers()
    const { data: nodeApps = [] } = useApps([nodeId])
    const themeCompartment = useRef(new Compartment())
    const checkResult = useMemo(
        () =>
            checkCompose(
                composeContent,
                nodeApps.filter((other) => other.id !== appId),
            ),
        [composeContent, nodeApps, appId],
    )
    const { data: app } = useApp(appId, nodeId)
    const envCount = useMemo(() => readEnv(composeContent).entries.length, [composeContent])
    const { toast } = useToast()

    // Initialize CodeMirror editor
    useEffect(() => {
        if (!editorContainerRef.current || editorRef.current) return

        const isDark = document.documentElement.classList.contains('dark')

        // Basic editor setup
        const basicExtensions = [
            lineNumbers(),
            highlightActiveLineGutter(),
            highlightSpecialChars(),
            history(),
            foldGutter(),
            drawSelection(),
            dropCursor(),
            EditorState.allowMultipleSelections.of(true),
            indentOnInput(),
            bracketMatching(),
            closeBrackets(),
            autocompletion(),
            rectangularSelection(),
            crosshairCursor(),
            highlightActiveLine(),
            highlightSelectionMatches(),
            keymap.of([
                ...closeBracketsKeymap,
                ...defaultKeymap,
                ...searchKeymap,
                ...historyKeymap,
                ...foldKeymap,
                ...completionKeymap,
            ]),
            yaml(),
            syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
            EditorView.updateListener.of((update) => {
                if (update.docChanged) {
                    const newContent = update.state.doc.toString()
                    setComposeContent(newContent)
                    setHasChanges(newContent !== initialComposeContent)
                    setLineCount(update.state.doc.lines)
                    setCharCount(newContent.length)
                }
            }),
            EditorView.theme({
                '&': {
                    fontSize: '13px',
                    height: '460px',
                },
                '.cm-content': {
                    padding: '8px',
                },
                '.cm-scroller': {
                    fontFamily: 'var(--font-mono, ui-monospace, monospace)',
                },
                '.cm-editor': {
                    height: '100%',
                },
            }),
            themeCompartment.current.of(yamlEditorTheme(isDark)),
        ]

        const startState = EditorState.create({
            doc: composeContent,
            extensions: basicExtensions,
        })

        const view = new EditorView({
            state: startState,
            parent: editorContainerRef.current,
        })

        editorRef.current = view

        return () => {
            view.destroy()
            editorRef.current = null
        }
        // The editor is built once. Later changes to the content reach it through the effects below.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    // Follow the theme switch without rebuilding the editor.
    useEffect(() => {
        const observer = new MutationObserver(() => {
            editorRef.current?.dispatch({
                effects: themeCompartment.current.reconfigure(
                    yamlEditorTheme(document.documentElement.classList.contains('dark')),
                ),
            })
        })
        observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
        return () => observer.disconnect()
    }, [])

    // Update editor content when initialComposeContent changes externally
    useEffect(() => {
        if (editorRef.current && initialComposeContent !== editorRef.current.state.doc.toString()) {
            const transaction = editorRef.current.state.update({
                changes: {
                    from: 0,
                    to: editorRef.current.state.doc.length,
                    insert: initialComposeContent,
                },
            })
            editorRef.current.dispatch(transaction)
            setComposeContent(initialComposeContent)
            setHasChanges(false)
        }
    }, [initialComposeContent])

    // Update line and character counts
    useEffect(() => {
        const lines = composeContent.split('\n').length
        setLineCount(lines)
        setCharCount(composeContent.length)
    }, [composeContent])

    const handleSave = async (redeploy: boolean) => {
        if (!hasChanges || checkResult.blocked) return

        setIsSaving(true)
        try {
            await updateApp.mutateAsync({ compose_content: composeContent })
            setHasChanges(false)
            if (redeploy) {
                await updateAppContainers.mutateAsync({ id: appId, nodeId })
                toast.success('Saved and redeploying', 'The containers are restarting with the new config')
            } else {
                toast.success('Saved', 'The new config applies the next time the app starts or updates')
            }
        } catch (error) {
            toast.error('Could not save', describeError(error))
        } finally {
            setIsSaving(false)
        }
    }

    const handleReset = () => {
        if (!editorRef.current) return

        // Update CodeMirror editor content
        const transaction = editorRef.current.state.update({
            changes: {
                from: 0,
                to: editorRef.current.state.doc.length,
                insert: initialComposeContent,
            },
        })
        editorRef.current.dispatch(transaction)

        // Update state
        setComposeContent(initialComposeContent)
        setHasChanges(false)
        toast.info('Changes discarded', 'The editor shows the saved config again')
    }

    const handleVersionSelect = (version: ComposeVersion) => {
        setViewingVersion(version)
    }

    const handleLoadVersion = () => {
        if (!viewingVersion || !editorRef.current) return

        // Update CodeMirror editor content
        const transaction = editorRef.current.state.update({
            changes: {
                from: 0,
                to: editorRef.current.state.doc.length,
                insert: viewingVersion.compose_content,
            },
        })
        editorRef.current.dispatch(transaction)

        // Update state
        setComposeContent(viewingVersion.compose_content)
        setHasChanges(viewingVersion.compose_content !== initialComposeContent)
        setViewingVersion(null)
        toast.success('Version loaded', `Loaded version ${viewingVersion.version} into editor`)
    }

    const handleFormat = () => {
        try {
            const parsed = parse(composeContent)

            // Recursively fix null values in networks/volumes/services (empty definitions)
            const fixNullValues = (obj: any, parentKey?: string): any => {
                if (obj === null) {
                    // If null and parent is networks/volumes/services, return empty object
                    if (parentKey === 'networks' || parentKey === 'volumes' || parentKey === 'services') {
                        return {}
                    }
                    return null
                }
                if (Array.isArray(obj)) {
                    return obj.map((item) => fixNullValues(item, parentKey))
                }
                if (typeof obj === 'object') {
                    const result: any = {}
                    for (const [key, value] of Object.entries(obj)) {
                        if (value === null && (key === 'networks' || key === 'volumes' || key === 'services')) {
                            result[key] = {}
                        } else if (
                            value === null &&
                            (parentKey === 'networks' || parentKey === 'volumes' || parentKey === 'services')
                        ) {
                            // Null value inside networks/volumes/services - convert to empty object
                            result[key] = {}
                        } else {
                            result[key] = fixNullValues(value, key)
                        }
                    }
                    return result
                }
                return obj
            }

            const fixed = fixNullValues(parsed)
            let formatted = stringify(fixed, {
                indent: 2,
                lineWidth: 0,
                sortMapEntries: false,
                defaultStringType: 'PLAIN',
                defaultKeyType: 'PLAIN',
            })

            // Post-process: replace `key: {}` or `key: null` with `key:` in networks/volumes/services sections
            const lines = formatted.split('\n')
            const processedLines: string[] = []
            let inNetworksSection = false
            let inVolumesSection = false
            let inServicesSection = false

            for (let i = 0; i < lines.length; i++) {
                let line = lines[i]

                // Track which section we're in
                if (line.match(/^\s*networks:\s*$/)) {
                    inNetworksSection = true
                    inVolumesSection = false
                    inServicesSection = false
                } else if (line.match(/^\s*volumes:\s*$/)) {
                    inNetworksSection = false
                    inVolumesSection = true
                    inServicesSection = false
                } else if (line.match(/^\s*services:\s*$/)) {
                    inNetworksSection = false
                    inVolumesSection = false
                    inServicesSection = true
                } else if (line.match(/^\s*\w+:\s*$/) && !line.match(/^\s+(networks|volumes|services):/)) {
                    // Reset sections when we hit a top-level key
                    inNetworksSection = false
                    inVolumesSection = false
                    inServicesSection = false
                }

                // Fix null or empty object values in these sections
                if (
                    (inNetworksSection || inVolumesSection || inServicesSection) &&
                    line.match(/^(\s+)(\w+):\s+(null|{})\s*$/)
                ) {
                    line = line.replace(/:\s+(null|{})\s*$/, ':')
                }

                processedLines.push(line)
            }

            formatted = processedLines.join('\n')

            setFormattedContent(formatted)
            setFormatError(null)
            setShowFormatDialog(true)
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : 'Failed to format YAML'
            toast.error('Format failed', errorMessage)
            setFormatError(errorMessage)
        }
    }

    const handleApplyFormat = () => {
        if (!editorRef.current) {
            toast.error('Editor not ready', 'Please wait for the editor to load')
            return
        }

        // Update CodeMirror editor content
        const transaction = editorRef.current.state.update({
            changes: {
                from: 0,
                to: editorRef.current.state.doc.length,
                insert: formattedContent,
            },
        })
        editorRef.current.dispatch(transaction)

        // Update state
        setComposeContent(formattedContent)
        setHasChanges(formattedContent !== initialComposeContent)
        setShowFormatDialog(false)
        toast.success('Formatted', 'YAML has been formatted')
    }

    // Update content when initialComposeContent changes (e.g., after rollback)
    useEffect(() => {
        setComposeContent(initialComposeContent)
        setHasChanges(false)
    }, [initialComposeContent])

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
                            <div ref={editorContainerRef} aria-label="Compose file editor" />
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
                            <ComposeVersionHistory
                                appId={appId}
                                nodeId={nodeId}
                                onVersionSelect={handleVersionSelect}
                            />
                        </CardContent>
                    </Card>
                </div>
            </div>

            {hasChanges && (
                <SaveBar
                    saving={isSaving}
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

            <Dialog open={showFormatDialog} onOpenChange={setShowFormatDialog}>
                <DialogContent className="flex max-h-[90vh] max-w-3xl flex-col overflow-hidden">
                    <DialogHeader>
                        <DialogTitle>Format preview</DialogTitle>
                        <DialogDescription>
                            Red lines are removed and green lines are added. Nothing changes until you apply it.
                        </DialogDescription>
                    </DialogHeader>
                    {formatError ? (
                        <p role="alert" className="text-sm text-status-err-fg">
                            Could not format the file: {formatError}
                        </p>
                    ) : (
                        <div className="min-h-0 flex-1 overflow-auto">
                            <DiffBlock
                                before={composeContent}
                                after={formattedContent}
                                aria-label="Formatting changes"
                            />
                        </div>
                    )}
                    <div className="flex justify-end gap-2">
                        <Button variant="outline" onClick={() => setShowFormatDialog(false)}>
                            Cancel
                        </Button>
                        {!formatError && <Button onClick={handleApplyFormat}>Apply format</Button>}
                    </div>
                </DialogContent>
            </Dialog>
        </div>
    )
}

export default ComposeEditor
