import { useState } from 'react'
import { Eye, EyeOff, Trash2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card } from '@/shared/components/ui/Card'
import { Input } from '@/shared/components/ui/Input'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/components/ui/Table'
import { isSecretEntry, type EnvEntry } from '../lib/compose-env'
import { envFlag, MASK, rowId } from '../lib/env-view'

interface EnvTableProps {
    entries: EnvEntry[]
    // The entries as saved, to mark what is new or changed.
    saved: EnvEntry[]
    // The rows whose secret value is showing.
    revealed: Set<string>
    onToggleReveal: (id: string) => void
    onChangeValue: (entry: EnvEntry, value: string) => void
    onRemove: (entry: EnvEntry) => void
}

// The variables as an editable table. Secrets are hidden until revealed.
function EnvTable({ entries, saved, revealed, onToggleReveal, onChangeValue, onRemove }: EnvTableProps) {
    return (
        <Card className="min-w-0 overflow-hidden p-0">
            <Table aria-label="Environment variables" className="max-md:[&_td]:px-3 max-md:[&_th]:px-3">
                <TableHeader>
                    <TableRow className="hover:bg-transparent">
                        <TableHead>Name</TableHead>
                        <TableHead>Value</TableHead>
                        <TableHead className="max-md:hidden">Service</TableHead>
                        <TableHead className="w-[1%]">
                            <span className="sr-only">Actions</span>
                        </TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {entries.map((entry) => {
                        const id = rowId(entry)
                        const secret = isSecretEntry(entry.key, entry.value)
                        const shown = !secret || revealed.has(id)
                        const flag = envFlag(saved, entry)
                        return (
                            <TableRow key={id} data-env={id}>
                                <TableCell className="align-top">
                                    <div className="flex flex-col gap-1">
                                        <span className="break-all font-mono text-compact font-medium">
                                            {entry.key}
                                        </span>
                                        <span className="text-xs text-muted-foreground md:hidden">{entry.service}</span>
                                        {flag && (
                                            <StatusPill
                                                kind={flag === 'new' ? 'info' : 'warn'}
                                                size="sm"
                                                className="w-fit"
                                            >
                                                {flag === 'new' ? 'New' : 'Changed'}
                                            </StatusPill>
                                        )}
                                    </div>
                                </TableCell>
                                <TableCell className="min-w-[140px]">
                                    {shown ? (
                                        <ValueInput
                                            label={`Value of ${entry.key}`}
                                            value={entry.value}
                                            onCommit={(value) => onChangeValue(entry, value)}
                                        />
                                    ) : (
                                        <span
                                            className="font-mono text-compact text-muted-foreground"
                                            aria-label={`Value of ${entry.key} is hidden`}
                                        >
                                            {MASK}
                                        </span>
                                    )}
                                </TableCell>
                                <TableCell className="text-muted-foreground max-md:hidden">{entry.service}</TableCell>
                                <TableCell>
                                    <div className="flex items-center justify-end gap-1">
                                        {secret && (
                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                aria-label={`${shown ? 'Hide' : 'Reveal'} ${entry.key}`}
                                                aria-pressed={shown}
                                                onClick={() => onToggleReveal(id)}
                                            >
                                                {shown ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                                            </Button>
                                        )}
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            aria-label={`Remove ${entry.key}`}
                                            onClick={() => onRemove(entry)}
                                        >
                                            <Trash2 className="h-4 w-4" />
                                        </Button>
                                    </div>
                                </TableCell>
                            </TableRow>
                        )
                    })}
                </TableBody>
            </Table>
        </Card>
    )
}

// The file is only rewritten when you leave the field or press Enter, not on every keystroke, so comments and
// layout in the compose file are touched as little as possible.
function ValueInput({ label, value, onCommit }: { label: string; value: string; onCommit: (value: string) => void }) {
    const [text, setText] = useState(value)
    const [shownValue, setShownValue] = useState(value)
    if (shownValue !== value) {
        setShownValue(value)
        setText(value)
    }
    const commit = () => text !== value && onCommit(text)
    return (
        <Input
            aria-label={label}
            value={text}
            onChange={(event) => setText(event.target.value)}
            onBlur={commit}
            onKeyDown={(event) => event.key === 'Enter' && commit()}
            className="font-mono text-compact"
        />
    )
}

export default EnvTable
