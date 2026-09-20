import { useEffect, useMemo, useState } from 'react'
import { Copy, Eye, EyeOff, FileUp, List, Plus, Trash2, FileText } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent } from '@/shared/components/ui/Card'
import { CodeBlock } from '@/shared/components/ui/CodeBlock'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/shared/components/ui/Dialog'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/components/ui/Select'
import { SegmentedControl } from '@/shared/components/ui/SegmentedControl'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/components/ui/Table'
import { Textarea } from '@/shared/components/ui/Textarea'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { useUpdateApp, useUpdateAppContainers } from '@/shared/services/api'
import type { App } from '@/shared/types/api'
import { findUnresolved, isSecretEntry, isValidEnvKey, parseDotenv, readEnv, readServiceNames, removeEnv, setEnv, type EnvEntry } from '../lib/compose-env'
import type { ComposeCheck } from '../lib/compose-checks'
import ComposeChecks from './ComposeChecks'
import SaveBar from './SaveBar'

type View = 'table' | 'raw'

const VIEW_OPTIONS = [
    { value: 'table', label: 'Table', icon: <List className="h-4 w-4" /> },
    { value: 'raw', label: 'Raw', icon: <FileText className="h-4 w-4" /> },
]
const MASK = '••••••••'
const rowId = (entry: EnvEntry) => `${entry.service}:${entry.key}`

// The variables an app's containers are started with. They live in the compose file, so editing them here
// edits that file, and the same Save and Save and redeploy apply.
function EnvironmentTab({ app }: { app: App }) {
    const { toast } = useToast()
    const updateApp = useUpdateApp(app.id, app.node_id)
    const updateContainers = useUpdateAppContainers()

    const [draft, setDraft] = useState(app.compose_content)
    const [view, setView] = useState<View>('table')
    const [revealed, setRevealed] = useState<Set<string>>(new Set())
    const [saving, setSaving] = useState(false)
    const [adding, setAdding] = useState(false)
    const [importing, setImporting] = useState(false)

    // Show the saved file again whenever it changes underneath, for example after a rollback.
    useEffect(() => setDraft(app.compose_content), [app.compose_content])

    const services = useMemo(() => readServiceNames(draft), [draft])
    const current = useMemo(() => readEnv(draft), [draft])
    const saved = useMemo(() => readEnv(app.compose_content).entries, [app.compose_content])
    const dirty = draft !== app.compose_content

    const flagFor = (entry: EnvEntry): 'new' | 'changed' | null => {
        const before = saved.find((other) => rowId(other) === rowId(entry))
        if (!before) return 'new'
        return before.value !== entry.value ? 'changed' : null
    }

    const checks = useMemo<ComposeCheck[]>(() => {
        if (current.error) return [{ id: 'yaml', level: 'err', title: 'The compose file is not valid YAML', detail: current.error }]
        const list: ComposeCheck[] = [
            {
                id: 'count',
                level: 'ok',
                title: `${current.entries.length} ${current.entries.length === 1 ? 'variable' : 'variables'} across ${new Set(current.entries.map((entry) => entry.service)).size} ${new Set(current.entries.map((entry) => entry.service)).size === 1 ? 'service' : 'services'}`,
            },
        ]
        const unresolved = findUnresolved(draft, current.entries)
        if (unresolved.length > 0) {
            list.push({ id: 'unresolved', level: 'warn', title: 'Referenced but not set', detail: `${unresolved.map((name) => '${' + name + '}').join(', ')} has no value and no default.` })
        }
        const empty = current.entries.filter((entry) => entry.value === '')
        if (empty.length > 0) {
            list.push({ id: 'empty', level: 'warn', title: 'Empty values', detail: empty.map((entry) => entry.key).join(', ') })
        }
        return list
    }, [current, draft])

    const toggleReveal = (id: string) =>
        setRevealed((currentSet) => {
            const next = new Set(currentSet)
            if (next.has(id)) next.delete(id)
            else next.add(id)
            return next
        })

    const save = async (redeploy: boolean) => {
        setSaving(true)
        try {
            await updateApp.mutateAsync({ compose_content: draft })
            if (redeploy) {
                await updateContainers.mutateAsync({ id: app.id, nodeId: app.node_id })
                toast.success('Saved and redeploying', 'The containers are restarting with the new variables')
            } else {
                toast.success('Saved', 'The new variables apply the next time the app starts or updates')
            }
        } catch (error) {
            toast.error('Could not save', describeError(error))
        } finally {
            setSaving(false)
        }
    }

    const copyEnv = async () => {
        const text = current.entries.map((entry) => `${entry.key}=${entry.value}`).join('\n')
        try {
            await navigator.clipboard.writeText(text)
            toast.success('Copied', 'Variables copied as a .env file')
        } catch {
            toast.error('Could not copy', 'Your browser blocked access to the clipboard')
        }
    }

    const rawLines = useMemo(() => {
        const lines: { text: string }[] = []
        for (const service of services) {
            const own = current.entries.filter((entry) => entry.service === service)
            if (own.length === 0) continue
            if (lines.length > 0) lines.push({ text: '' })
            lines.push({ text: `# ${service}` })
            for (const entry of own) {
                const hidden = isSecretEntry(entry.key, entry.value) && !revealed.has(rowId(entry))
                lines.push({ text: `${entry.key}=${hidden ? MASK : entry.value}` })
            }
        }
        return lines
    }, [services, current.entries, revealed])

    return (
        <div className="flex flex-col gap-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <SegmentedControl aria-label="Environment view" options={VIEW_OPTIONS} value={view} onValueChange={(value) => setView(value as View)} />
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

            <div className="grid grid-cols-1 gap-5 lg:grid-cols-[minmax(0,1fr)_340px]">
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
                            <p className="mt-3 text-[13px] text-muted-foreground">Secrets stay hidden here until you reveal them in the table.</p>
                        </CardContent>
                    </Card>
                ) : (
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
                                {current.entries.map((entry) => {
                                    const id = rowId(entry)
                                    const secret = isSecretEntry(entry.key, entry.value)
                                    const shown = !secret || revealed.has(id)
                                    const flag = flagFor(entry)
                                    return (
                                        <TableRow key={id} data-env={id}>
                                            <TableCell className="align-top">
                                                <div className="flex flex-col gap-1">
                                                    <span className="break-all font-mono text-[13px] font-medium">{entry.key}</span>
                                                    <span className="text-xs text-muted-foreground md:hidden">{entry.service}</span>
                                                    {flag && <StatusPill kind={flag === 'new' ? 'info' : 'warn'} size="sm" className="w-fit">{flag === 'new' ? 'New' : 'Changed'}</StatusPill>}
                                                </div>
                                            </TableCell>
                                            <TableCell className="min-w-[140px]">
                                                {shown ? (
                                                    <ValueInput label={`Value of ${entry.key}`} value={entry.value} onCommit={(value) => setDraft(setEnv(draft, entry.service, entry.key, value))} />
                                                ) : (
                                                    <span className="font-mono text-[13px] text-muted-foreground" aria-label={`Value of ${entry.key} is hidden`}>{MASK}</span>
                                                )}
                                            </TableCell>
                                            <TableCell className="text-muted-foreground max-md:hidden">{entry.service}</TableCell>
                                            <TableCell>
                                                <div className="flex items-center justify-end gap-1">
                                                    {secret && (
                                                        <Button variant="ghost" size="icon" aria-label={`${shown ? 'Hide' : 'Reveal'} ${entry.key}`} aria-pressed={shown} onClick={() => toggleReveal(id)}>
                                                            {shown ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                                                        </Button>
                                                    )}
                                                    <Button variant="ghost" size="icon" aria-label={`Remove ${entry.key}`} onClick={() => setDraft(removeEnv(draft, entry.service, entry.key))}>
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
                )}

                <ComposeChecks checks={checks} />
            </div>

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
                    toast.success('Imported', `${pairs.length} ${pairs.length === 1 ? 'variable' : 'variables'} added to ${service}. Save to keep them.`)
                }}
            />
        </div>
    )
}

// The file is only rewritten when you leave the field or press Enter, not on every keystroke, so comments and
// layout in the compose file are touched as little as possible.
function ValueInput({ label, value, onCommit }: { label: string; value: string; onCommit: (value: string) => void }) {
    const [text, setText] = useState(value)
    useEffect(() => setText(value), [value])
    const commit = () => text !== value && onCommit(text)
    return (
        <Input
            aria-label={label}
            value={text}
            onChange={(event) => setText(event.target.value)}
            onBlur={commit}
            onKeyDown={(event) => event.key === 'Enter' && commit()}
            className="font-mono text-[13px]"
        />
    )
}

interface ServiceSelectProps {
    services: string[]
    value: string
    onChange: (value: string) => void
    id?: string
}

function ServiceSelect({ services, value, onChange, id }: ServiceSelectProps) {
    return (
        <Select value={value} onValueChange={onChange}>
            <SelectTrigger id={id} aria-label="Service">
                <SelectValue />
            </SelectTrigger>
            <SelectContent>
                {services.map((service) => (
                    <SelectItem key={service} value={service}>
                        {service}
                    </SelectItem>
                ))}
            </SelectContent>
        </Select>
    )
}

function AddVariableDialog({ open, services, existing, onOpenChange, onAdd }: {
    open: boolean
    services: string[]
    existing: EnvEntry[]
    onOpenChange: (open: boolean) => void
    onAdd: (service: string, key: string, value: string) => void
}) {
    const [service, setService] = useState(services[0] ?? '')
    const [key, setKey] = useState('')
    const [value, setValue] = useState('')

    useEffect(() => {
        if (open) {
            setService((current) => (services.includes(current) ? current : services[0] ?? ''))
            setKey('')
            setValue('')
        }
    }, [open, services])

    const trimmed = key.trim()
    const error = trimmed === '' ? undefined : !isValidEnvKey(trimmed)
        ? 'Use letters, numbers and underscores, and do not start with a number.'
        : existing.some((entry) => entry.service === service && entry.key === trimmed)
          ? `${service} already sets ${trimmed}. Edit it in the table.`
          : undefined
    const canAdd = trimmed !== '' && !error

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-w-md">
                <DialogHeader>
                    <DialogTitle>Add variable</DialogTitle>
                    <DialogDescription>It is added to the service's environment in the compose file.</DialogDescription>
                </DialogHeader>
                <form
                    className="flex flex-col gap-4"
                    onSubmit={(event) => {
                        event.preventDefault()
                        if (!canAdd) return
                        onAdd(service, trimmed, value)
                        onOpenChange(false)
                    }}
                >
                    {services.length > 1 && (
                        <Field label="Service">
                            <ServiceSelect services={services} value={service} onChange={setService} />
                        </Field>
                    )}
                    <Field label="Name" error={error}>
                        <Input value={key} onChange={(event) => setKey(event.target.value)} placeholder="DATABASE_URL" className="font-mono" autoFocus />
                    </Field>
                    <Field label="Value">
                        <Input value={value} onChange={(event) => setValue(event.target.value)} className="font-mono" />
                    </Field>
                    <div className="flex justify-end gap-2">
                        <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                            Cancel
                        </Button>
                        <Button type="submit" disabled={!canAdd}>
                            Add
                        </Button>
                    </div>
                </form>
            </DialogContent>
        </Dialog>
    )
}

function ImportDialog({ open, services, onOpenChange, onImport }: {
    open: boolean
    services: string[]
    onOpenChange: (open: boolean) => void
    onImport: (service: string, pairs: { key: string; value: string }[]) => void
}) {
    const [service, setService] = useState(services[0] ?? '')
    const [text, setText] = useState('')

    useEffect(() => {
        if (open) {
            setService((current) => (services.includes(current) ? current : services[0] ?? ''))
            setText('')
        }
    }, [open, services])

    const pairs = useMemo(() => parseDotenv(text), [text])

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-w-lg">
                <DialogHeader>
                    <DialogTitle>Import .env</DialogTitle>
                    <DialogDescription>Paste the contents of a .env file. Comments and blank lines are skipped, and existing names are overwritten.</DialogDescription>
                </DialogHeader>
                <div className="flex flex-col gap-4">
                    {services.length > 1 && (
                        <Field label="Add to service">
                            <ServiceSelect services={services} value={service} onChange={setService} />
                        </Field>
                    )}
                    <Field label=".env contents" hint={text.trim() === '' ? undefined : `${pairs.length} ${pairs.length === 1 ? 'variable' : 'variables'} found`}>
                        <Textarea value={text} onChange={(event) => setText(event.target.value)} rows={8} className="font-mono text-[13px]" placeholder={'DATABASE_URL=postgres://...\nSECRET_KEY=...'} />
                    </Field>
                    <div className="flex justify-end gap-2">
                        <Button variant="outline" onClick={() => onOpenChange(false)}>
                            Cancel
                        </Button>
                        <Button
                            disabled={pairs.length === 0}
                            onClick={() => {
                                onImport(service, pairs)
                                onOpenChange(false)
                            }}
                        >
                            Import {pairs.length > 0 ? pairs.length : ''}
                        </Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    )
}

export default EnvironmentTab
