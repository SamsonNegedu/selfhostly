import { useCallback, useEffect, useRef } from 'react'
import { EditorView, keymap } from '@codemirror/view'
import {
    lineNumbers,
    highlightActiveLine,
    highlightActiveLineGutter,
    highlightSpecialChars,
    drawSelection,
    dropCursor,
    rectangularSelection,
    crosshairCursor,
} from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { yaml } from '@codemirror/lang-yaml'
import { searchKeymap, highlightSelectionMatches } from '@codemirror/search'
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
import { yamlEditorTheme } from '@/shared/components/ui/YamlEditor'

interface Options {
    // What the editor starts with. It is read once, when the editor is built.
    initialDoc: string
    // Called with the whole text after every edit, including edits made through `replaceDoc`.
    onDocChange: (doc: string) => void
}

const isDarkTheme = () => document.documentElement.classList.contains('dark')

// A CodeMirror editor for a compose file, mounted on the element `containerRef` is given. The editor is built once
// and follows the light and dark theme by itself.
export function useYamlEditorView({ initialDoc, onDocChange }: Options) {
    const containerRef = useRef<HTMLDivElement | null>(null)
    const viewRef = useRef<EditorView | null>(null)
    const themeCompartment = useRef(new Compartment())
    // The editor outlives renders, so it calls the latest handler and not the one from when it was built.
    const onDocChangeRef = useRef(onDocChange)
    useEffect(() => {
        onDocChangeRef.current = onDocChange
    })

    useEffect(() => {
        if (!containerRef.current || viewRef.current) return

        const extensions = [
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
                if (update.docChanged) onDocChangeRef.current(update.state.doc.toString())
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
            themeCompartment.current.of(yamlEditorTheme(isDarkTheme())),
        ]

        const view = new EditorView({
            state: EditorState.create({ doc: initialDoc, extensions }),
            parent: containerRef.current,
        })
        viewRef.current = view

        return () => {
            view.destroy()
            viewRef.current = null
        }
        // The editor is built once. Later changes to the text reach it through `replaceDoc`.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    // Follow the theme switch without rebuilding the editor.
    useEffect(() => {
        const observer = new MutationObserver(() => {
            viewRef.current?.dispatch({ effects: themeCompartment.current.reconfigure(yamlEditorTheme(isDarkTheme())) })
        })
        observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
        return () => observer.disconnect()
    }, [])

    // The text now in the editor, or undefined before it is built.
    const currentDoc = useCallback(() => viewRef.current?.state.doc.toString(), [])

    // Swaps the whole text. Returns false, and does nothing, when the editor is not built yet.
    const replaceDoc = useCallback((text: string) => {
        const view = viewRef.current
        if (!view) return false
        view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } })
        return true
    }, [])

    return { containerRef, currentDoc, replaceDoc }
}
