import * as React from 'react'

import { cn } from '@/shared/lib/utils'

type LineMark = 'add' | 'del'

interface CodeLine {
    text: string
    mark?: LineMark
}

const MARK_ROW_CLASSES: Record<LineMark, string> = {
    add: 'bg-status-ok-bg',
    del: 'bg-status-err-bg',
}

const MARK_SIGN: Record<LineMark, string> = {
    add: '+',
    del: '-',
}

const MARK_SIGN_CLASSES: Record<LineMark, string> = {
    add: 'text-status-ok-fg',
    del: 'text-status-err-fg',
}

const MARK_ANNOUNCEMENT: Record<LineMark, string> = {
    add: 'Added: ',
    del: 'Removed: ',
}

const YAML_KEY_PATTERN = /^(\s*(?:- )?)([^\s:#][^:#]*?)(:)(\s.*)?$/
const YAML_LIST_ITEM_PATTERN = /^(\s*- )(.*)$/
const YAML_COMMENT_PATTERN = /^(\s*)(#.*)$/

const keyClass = 'text-status-info-fg'
const valueClass = 'text-status-ok-fg'

// A light touch of YAML coloring for display. Editing uses CodeMirror, which has full highlighting.
function highlightYaml(line: string): React.ReactNode {
    const comment = line.match(YAML_COMMENT_PATTERN)
    if (comment) {
        return (
            <>
                {comment[1]}
                <span className="text-muted-foreground">{comment[2]}</span>
            </>
        )
    }

    const pair = line.match(YAML_KEY_PATTERN)
    if (pair) {
        const [, indent, key, colon, value] = pair
        return (
            <>
                {indent}
                <span className={keyClass}>{key}</span>
                {colon}
                {value && <span className={valueClass}>{value}</span>}
            </>
        )
    }

    const item = line.match(YAML_LIST_ITEM_PATTERN)
    if (item) {
        return (
            <>
                {item[1]}
                <span className={valueClass}>{item[2]}</span>
            </>
        )
    }

    return line
}

interface CodeBlockProps {
    code?: string
    lines?: CodeLine[]
    language?: 'yaml' | 'text'
    'aria-label'?: string
    className?: string
}

// Read-only code with line numbers. Long lines scroll inside the block, and the block is
// focusable so keyboard users can scroll it.
function CodeBlock({ code, lines, language = 'yaml', className, ...props }: CodeBlockProps) {
    const rows: CodeLine[] = lines ?? (code ?? '').split('\n').map((text) => ({ text }))

    return (
        <div
            tabIndex={0}
            role="region"
            aria-label={props['aria-label'] ?? 'Code'}
            className={cn(
                'overflow-x-auto rounded-[10px] border border-border bg-card font-mono text-detail leading-[22px] ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 sm:text-compact',
                className,
            )}
        >
            <pre className="m-0 min-w-max py-1">
                <code>
                    {rows.map((line, index) => (
                        <div key={index} className={cn('flex', line.mark && MARK_ROW_CLASSES[line.mark])}>
                            <span
                                aria-hidden="true"
                                className="w-11 shrink-0 select-none pr-3 text-right text-muted-foreground"
                            >
                                {index + 1}
                            </span>
                            <span
                                aria-hidden="true"
                                className={cn('w-4 shrink-0 select-none', line.mark && MARK_SIGN_CLASSES[line.mark])}
                            >
                                {line.mark ? MARK_SIGN[line.mark] : ''}
                            </span>
                            <span className="whitespace-pre pr-4">
                                {line.mark && <span className="sr-only">{MARK_ANNOUNCEMENT[line.mark]}</span>}
                                {language === 'yaml' ? highlightYaml(line.text) : line.text}
                            </span>
                        </div>
                    ))}
                </code>
            </pre>
        </div>
    )
}

export { CodeBlock }
export type { CodeLine }
