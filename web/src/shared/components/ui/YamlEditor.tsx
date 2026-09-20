import { useEffect, useRef } from 'react'
import { Compartment, EditorState } from '@codemirror/state'
import { EditorView, keymap, lineNumbers } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { yaml } from '@codemirror/lang-yaml'
import { bracketMatching, defaultHighlightStyle, indentOnInput, syntaxHighlighting } from '@codemirror/language'
import { oneDark } from '@codemirror/theme-one-dark'
import { cn } from '@/shared/lib/utils'

// The editor takes its colors from the app tokens, so it matches in both themes. Dark also gets the dark
// syntax colors.
export const yamlEditorTheme = (isDark: boolean) => [
    EditorView.theme(
        {
            '&': { backgroundColor: 'hsl(var(--card))', color: 'hsl(var(--foreground))' },
            '.cm-gutters': {
                backgroundColor: 'hsl(var(--card))',
                color: 'hsl(var(--muted-foreground))',
                border: 'none',
            },
            '.cm-activeLine, .cm-activeLineGutter': { backgroundColor: 'hsl(var(--muted) / 0.5)' },
            '&.cm-focused': { outline: '2px solid hsl(var(--ring))', outlineOffset: '-2px' },
        },
        { dark: isDark },
    ),
    ...(isDark ? [oneDark] : []),
]

interface YamlEditorProps {
    value: string
    onChange?: (value: string) => void
    readOnly?: boolean
    height?: number
    'aria-label': string
    className?: string
}

const isDarkMode = () => document.documentElement.classList.contains('dark')

// A small YAML editor for places that only need to show or change one file. The compose editor on the app
// detail page has more features and builds its own.
function YamlEditor({ value, onChange, readOnly = false, height = 320, className, ...props }: YamlEditorProps) {
    const container = useRef<HTMLDivElement>(null)
    const view = useRef<EditorView | null>(null)
    const theme = useRef(new Compartment())
    const latestOnChange = useRef(onChange)
    latestOnChange.current = onChange

    useEffect(() => {
        if (!container.current) return
        const editor = new EditorView({
            parent: container.current,
            state: EditorState.create({
                doc: value,
                extensions: [
                    lineNumbers(),
                    history(),
                    indentOnInput(),
                    bracketMatching(),
                    yaml(),
                    syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
                    keymap.of([...defaultKeymap, ...historyKeymap]),
                    EditorState.readOnly.of(readOnly),
                    EditorView.editable.of(!readOnly),
                    EditorView.contentAttributes.of({ 'aria-label': props['aria-label'] }),
                    EditorView.updateListener.of((update) => {
                        if (update.docChanged) latestOnChange.current?.(update.state.doc.toString())
                    }),
                    EditorView.theme({
                        '&': { fontSize: '13px', height: `${height}px` },
                        '.cm-content': { padding: '8px' },
                        '.cm-scroller': { fontFamily: 'var(--font-mono, ui-monospace, monospace)' },
                    }),
                    theme.current.of(yamlEditorTheme(isDarkMode())),
                ],
            }),
        })
        view.current = editor

        const observer = new MutationObserver(() =>
            editor.dispatch({ effects: theme.current.reconfigure(yamlEditorTheme(isDarkMode())) }),
        )
        observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })

        return () => {
            observer.disconnect()
            editor.destroy()
            view.current = null
        }
        // The editor is built once. Later value changes are applied by the effect below.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    // Apply a value that changed from outside, such as choosing a template, without disturbing typing.
    useEffect(() => {
        const editor = view.current
        if (editor && editor.state.doc.toString() !== value) {
            editor.dispatch({ changes: { from: 0, to: editor.state.doc.length, insert: value } })
        }
    }, [value])

    return <div ref={container} className={cn('overflow-hidden rounded-lg border border-border', className)} />
}

export { YamlEditor }
