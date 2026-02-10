import { useState, useEffect, useRef } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/Card'
import { Button } from '@/shared/components/ui/Button'
import { Save, RotateCcw, FileCode, AlertTriangle, X, Download, PanelLeft, PanelLeftClose, Sparkles } from 'lucide-react'
import { useUpdateApp, useUpdateAppContainers, useComposeVersions } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/components/ui/Dialog'
import ComposeVersionHistory from './ComposeVersionHistory'
import EnvVarSidebar from './EnvVarSidebar'
import { parse, stringify } from 'yaml'
import ReactDiffViewer from 'react-diff-viewer-continued'
import { EditorView } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { oneDark } from '@codemirror/theme-one-dark'
import { yaml } from '@codemirror/lang-yaml'
import { searchKeymap, highlightSelectionMatches } from '@codemirror/search'
import { keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter, highlightSpecialChars, drawSelection, dropCursor, rectangularSelection, crosshairCursor } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { foldGutter, foldKeymap, syntaxHighlighting, defaultHighlightStyle, bracketMatching, indentOnInput } from '@codemirror/language'
import { autocompletion, completionKeymap, closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import type { ComposeVersion } from '@/shared/types/api'

interface ComposeEditorProps {
    appId: string;
    nodeId: string;
    initialComposeContent: string;
}

function ComposeEditor({ appId, nodeId, initialComposeContent }: ComposeEditorProps) {
    const [composeContent, setComposeContent] = useState(initialComposeContent)
    const [isSaving, setIsSaving] = useState(false)
    const [hasChanges, setHasChanges] = useState(false)
    const [lineCount, setLineCount] = useState(0)
    const [charCount, setCharCount] = useState(0)
    const [showUpdateDialog, setShowUpdateDialog] = useState(false)
    const [activeSidebarTab, setActiveSidebarTab] = useState<'env' | 'history' | null>('env')
    const [showFormatDialog, setShowFormatDialog] = useState(false)
    const [formattedContent, setFormattedContent] = useState('')
    const [formatError, setFormatError] = useState<string | null>(null)
    const [viewingVersion, setViewingVersion] = useState<ComposeVersion | null>(null)
    const editorRef = useRef<EditorView | null>(null)
    const editorContainerRef = useRef<HTMLDivElement | null>(null)
    const updateApp = useUpdateApp(appId, nodeId)
    const updateAppContainers = useUpdateAppContainers()
    const { data: versions } = useComposeVersions(appId, nodeId)
    const { toast } = useToast()

    // Initialize CodeMirror editor
    useEffect(() => {
        if (!editorContainerRef.current || editorRef.current) return

        const themeCompartment = new Compartment()
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
                    fontSize: '14px',
                    height: '500px',
                },
                '.cm-content': {
                    padding: '8px',
                    minHeight: '500px',
                },
                '.cm-scroller': {
                    fontFamily: 'monospace',
                },
                '.cm-editor': {
                    height: '100%',
                },
            }),
            themeCompartment.of(isDark ? [oneDark] : []),
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

    const handleSave = async () => {
        if (!hasChanges) return

        setIsSaving(true)
        try {
            await updateApp.mutateAsync({
                compose_content: composeContent
            })
            setHasChanges(false)
            toast.success('Saved', 'Docker Compose configuration updated successfully')
            // Prompt user to update containers with new config
            setShowUpdateDialog(true)
        } catch (error) {
            console.error('Failed to update compose file:', error)
            toast.error('Failed to save', 'Could not update compose configuration')
        } finally {
            setIsSaving(false)
        }
    }

    const handleUpdateContainers = () => {
        updateAppContainers.mutate({ id: appId, nodeId }, {
            onSuccess: () => {
                toast.success('Update Started', 'Containers are being updated with the new configuration')
                setShowUpdateDialog(false)
            },
            onError: (error) => {
                toast.error('Update Failed', error.message)
            }
        })
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
        toast.info('Reset', 'Changes have been reset to the original version')
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
                    return obj.map(item => fixNullValues(item, parentKey))
                }
                if (typeof obj === 'object') {
                    const result: any = {}
                    for (const [key, value] of Object.entries(obj)) {
                        if (value === null && (key === 'networks' || key === 'volumes' || key === 'services')) {
                            result[key] = {}
                        } else if (value === null && (parentKey === 'networks' || parentKey === 'volumes' || parentKey === 'services')) {
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
                if ((inNetworksSection || inVolumesSection || inServicesSection) && line.match(/^(\s+)(\w+):\s+(null|{})\s*$/)) {
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

    // Load sidebar preference from localStorage
    useEffect(() => {
        const saved = localStorage.getItem('composeEditorActiveSidebarTab')
        if (saved !== null) {
            setActiveSidebarTab(saved as 'env' | 'history' | null)
        }
    }, [])

    // Save sidebar preference to localStorage
    useEffect(() => {
        localStorage.setItem('composeEditorActiveSidebarTab', String(activeSidebarTab))
    }, [activeSidebarTab])

    return (
        <div className="space-y-6">
            {/* Main layout: Editor on left, Sidebars on right */}
            <div className="flex flex-col lg:flex-row gap-6">
                {/* Editor - takes most space */}
                <div className={`flex-1 ${activeSidebarTab ? 'lg:min-w-0' : ''}`}>
                    <Card>
                        <CardHeader>
                            <div className="flex items-center justify-between flex-wrap gap-4">
                                <div className="flex items-center gap-2">
                                    <FileCode className="h-5 w-5 text-primary" />
                                    <CardTitle className="text-xl">Docker Compose Editor</CardTitle>
                                </div>
                                <div className="flex gap-2 flex-wrap">
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={handleFormat}
                                        title="Format YAML"
                                        className="button-press"
                                    >
                                        <Sparkles className="h-4 w-4 mr-2" />
                                        Format
                                    </Button>
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={() => setActiveSidebarTab(activeSidebarTab ? null : 'env')}
                                        title={activeSidebarTab ? "Hide Sidebar" : "Show Sidebar"}
                                        className="button-press"
                                    >
                                        {activeSidebarTab ? (
                                            <PanelLeftClose className="h-4 w-4" />
                                        ) : (
                                            <PanelLeft className="h-4 w-4 mr-2" />
                                        )}
                                        <span className="hidden sm:inline">Sidebar</span>
                                    </Button>
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={handleReset}
                                        disabled={!hasChanges}
                                        title="Reset to original"
                                        className="button-press"
                                    >
                                        <RotateCcw className="h-4 w-4 mr-2" />
                                        Reset
                                    </Button>
                                    <Button
                                        size="sm"
                                        onClick={handleSave}
                                        disabled={!hasChanges || isSaving}
                                        className="button-press"
                                    >
                                        {isSaving ? (
                                            <>
                                                <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin mr-2" />
                                                Saving...
                                            </>
                                        ) : (
                                            <>
                                                <Save className="h-4 w-4 mr-2" />
                                                Save Changes
                                            </>
                                        )}
                                    </Button>
                                </div>
                            </div>
                        </CardHeader>
                        <CardContent>
                            <div className="space-y-4">
                                {/* CodeMirror Editor container */}
                                <div className="border rounded-md overflow-hidden">
                                    <div
                                        ref={editorContainerRef}
                                        className="min-h-[500px]"
                                    />
                                </div>

                                {/* Status bar */}
                                <div className="flex items-center justify-between text-xs text-muted-foreground pt-2 border-t">
                                    <div className="flex items-center gap-4">
                                        <span>{lineCount} lines</span>
                                        <span>{charCount} characters</span>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        {hasChanges && (
                                            <span className="flex items-center gap-1 text-amber-600 dark:text-amber-400">
                                                <AlertTriangle className="h-3 w-3" />
                                                Unsaved changes
                                            </span>
                                        )}
                                        <span>YAML</span>
                                    </div>
                                </div>
                            </div>
                        </CardContent>
                    </Card>
                </div>

                {/* Right Sidebar with Tabs */}
                {activeSidebarTab && (
                    <div className="flex flex-col lg:w-[32rem] lg:flex-shrink-0 lg:max-h-[calc(100vh-10rem)] min-h-0 space-y-0">
                        {/* Tabs */}
                        <div className="flex items-center gap-2 border-b bg-card rounded-t-lg flex-shrink-0">
                            <button
                                onClick={() => setActiveSidebarTab('env')}
                                className={`flex-1 px-4 py-3 text-sm font-medium transition-colors border-b-2 ${
                                    activeSidebarTab === 'env'
                                        ? 'border-primary text-primary'
                                        : 'border-transparent text-muted-foreground hover:text-foreground'
                                }`}
                            >
                                Environment Variables
                            </button>
                            <button
                                onClick={() => setActiveSidebarTab('history')}
                                className={`flex-1 px-4 py-3 text-sm font-medium transition-colors border-b-2 ${
                                    activeSidebarTab === 'history'
                                        ? 'border-primary text-primary'
                                        : 'border-transparent text-muted-foreground hover:text-foreground'
                                }`}
                            >
                                Version History
                            </button>
                        </div>
                        
                        {/* Tab Content - scrolls within sidebar */}
                        <div className="flex-1 min-h-0 overflow-auto rounded-b-lg">
                            {activeSidebarTab === 'env' && (
                                <EnvVarSidebar composeContent={composeContent} />
                            )}
                            {activeSidebarTab === 'history' && (
                                <ComposeVersionHistory
                                    appId={appId}
                                    nodeId={nodeId}
                                    onVersionSelect={handleVersionSelect}
                                />
                            )}
                        </div>
                    </div>
                )}
            </div>

            {/* Update Containers Dialog */}
            <ConfirmationDialog
                open={showUpdateDialog}
                onOpenChange={setShowUpdateDialog}
                title="Update Containers?"
                description="Your compose configuration has been saved. Would you like to update the running containers to apply these changes?"
                confirmText="Update Containers"
                cancelText="Not Now"
                onConfirm={handleUpdateContainers}
                isLoading={updateAppContainers.isPending}
            />

            {/* View Version Dialog */}
            <Dialog open={!!viewingVersion} onOpenChange={(open) => !open && setViewingVersion(null)}>
                <DialogContent className="max-w-[85vw] max-h-[95vh] overflow-hidden flex flex-col">
                    <DialogHeader>
                        <DialogTitle className="flex items-center gap-2">
                            <FileCode className="h-5 w-5 text-primary" />
                            Version {viewingVersion?.version} Comparison
                        </DialogTitle>
                    </DialogHeader>

                    {viewingVersion && (
                        <>
                            {/* Version metadata */}
                            <div className="p-4 bg-muted/50 rounded-lg space-y-3 text-sm mb-4 flex-shrink-0">
                                {/* Badges */}
                                <div className="flex items-center gap-2 flex-wrap">
                                    {viewingVersion.is_current && (
                                        <span className="inline-flex items-center gap-1 px-2 py-1 rounded bg-primary/10 text-primary text-xs font-medium">
                                            Current Version
                                        </span>
                                    )}
                                    {viewingVersion.rolled_back_from && (
                                        <span className="inline-flex items-center gap-1 px-2 py-1 rounded bg-amber-500/10 text-amber-600 dark:text-amber-400 text-xs font-medium">
                                            Rolled back from v{viewingVersion.rolled_back_from}
                                        </span>
                                    )}
                                </div>

                                {/* Metadata grid */}
                                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                                    {viewingVersion.created_at && (
                                        <div>
                                            <span className="text-muted-foreground">Created:</span>
                                            <span className="ml-2 font-medium">{new Date(viewingVersion.created_at).toLocaleString()}</span>
                                        </div>
                                    )}
                                    {viewingVersion.changed_by && (
                                        <div>
                                            <span className="text-muted-foreground">Changed by:</span>
                                            <span className="ml-2 font-medium">{viewingVersion.changed_by}</span>
                                        </div>
                                    )}
                                </div>

                                {/* Change reason */}
                                {viewingVersion.change_reason && (
                                    <div className="pt-2 border-t border-border">
                                        <span className="text-muted-foreground text-xs">Reason:</span>
                                        <p className="font-medium mt-1">{viewingVersion.change_reason}</p>
                                    </div>
                                )}
                            </div>

                            {/* Diff viewer */}
                            <div className="border rounded-md overflow-auto" style={{ height: '650px' }}>
                                <ReactDiffViewer
                                    oldValue={composeContent}
                                    newValue={viewingVersion.compose_content}
                                    splitView={true}
                                    useDarkTheme={true}
                                    leftTitle={`Current${versions ? ` (v${versions.find(v => v.is_current)?.version || '?'})` : ''}`}
                                    rightTitle={`Version ${viewingVersion.version}`}
                                    showDiffOnly={false}
                                />
                            </div>

                            {/* Actions */}
                            <div className="flex justify-between pt-4 border-t flex-shrink-0">
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => setViewingVersion(null)}
                                >
                                    <X className="h-4 w-4 mr-2" />
                                    Close
                                </Button>
                                <Button
                                    variant="default"
                                    size="sm"
                                    onClick={handleLoadVersion}
                                    disabled={viewingVersion.is_current}
                                >
                                    <Download className="h-4 w-4 mr-2" />
                                    {viewingVersion.is_current ? 'Current Version' : 'Load into Editor'}
                                </Button>
                            </div>
                        </>
                    )}
                </DialogContent>
            </Dialog>

            {/* Format Dialog */}
            <Dialog open={showFormatDialog} onOpenChange={setShowFormatDialog}>
                <DialogContent className="max-w-5xl max-h-[80vh] overflow-hidden flex flex-col">
                    <DialogHeader>
                        <DialogTitle className="flex items-center gap-2">
                            <Sparkles className="h-5 w-5 text-primary" />
                            Format Preview
                        </DialogTitle>
                    </DialogHeader>

                    <div className="flex-1 overflow-hidden">
                        {formatError ? (
                            <div className="flex items-center justify-center h-full">
                                <div className="text-center text-muted-foreground">
                                    <AlertTriangle className="h-12 w-12 text-amber-500 mx-auto mb-4" />
                                    <p>Failed to format YAML: {formatError}</p>
                                </div>
                            </div>
                        ) : (
                            <div className="flex-1 border rounded-md overflow-hidden">
                                <ReactDiffViewer
                                    oldValue={composeContent}
                                    newValue={formattedContent}
                                    splitView={true}
                                    useDarkTheme={true}
                                    leftTitle="Current"
                                    rightTitle="Formatted"
                                />
                            </div>
                        )}
                    </div>

                    {/* Actions */}
                    <div className="flex justify-between pt-4 border-t">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setShowFormatDialog(false)}
                        >
                            <X className="h-4 w-4 mr-2" />
                            Cancel
                        </Button>
                        {!formatError && (
                            <Button
                                variant="default"
                                size="sm"
                                onClick={handleApplyFormat}
                            >
                                <Sparkles className="h-4 w-4 mr-2" />
                                Apply Format
                            </Button>
                        )}
                    </div>
                </DialogContent>
            </Dialog>

        </div>
    )
}

export default ComposeEditor
