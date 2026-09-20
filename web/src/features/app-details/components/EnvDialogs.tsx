import { useMemo, useState } from 'react'
import { Button } from '@/shared/components/ui/Button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/shared/components/ui/Dialog'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/components/ui/Select'
import { Textarea } from '@/shared/components/ui/Textarea'
import { isValidEnvKey, parseDotenv, type EnvEntry } from '../lib/compose-env'

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

export function AddVariableDialog({
    open,
    services,
    existing,
    onOpenChange,
    onAdd,
}: {
    open: boolean
    services: string[]
    existing: EnvEntry[]
    onOpenChange: (open: boolean) => void
    onAdd: (service: string, key: string, value: string) => void
}) {
    const [service, setService] = useState(services[0] ?? '')
    const [key, setKey] = useState('')
    const [value, setValue] = useState('')

    // Start fresh each time the dialog opens, and when the list of services changes while it is open.
    const [seen, setSeen] = useState({ open, services })
    if (seen.open !== open || seen.services !== services) {
        setSeen({ open, services })
        if (open) {
            setService((current) => (services.includes(current) ? current : (services[0] ?? '')))
            setKey('')
            setValue('')
        }
    }

    const trimmed = key.trim()
    const error =
        trimmed === ''
            ? undefined
            : !isValidEnvKey(trimmed)
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
                        <Input
                            value={key}
                            onChange={(event) => setKey(event.target.value)}
                            placeholder="DATABASE_URL"
                            className="font-mono"
                            autoFocus
                        />
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

export function ImportDialog({
    open,
    services,
    onOpenChange,
    onImport,
}: {
    open: boolean
    services: string[]
    onOpenChange: (open: boolean) => void
    onImport: (service: string, pairs: { key: string; value: string }[]) => void
}) {
    const [service, setService] = useState(services[0] ?? '')
    const [text, setText] = useState('')

    const [seen, setSeen] = useState({ open, services })
    if (seen.open !== open || seen.services !== services) {
        setSeen({ open, services })
        if (open) {
            setService((current) => (services.includes(current) ? current : (services[0] ?? '')))
            setText('')
        }
    }

    const pairs = useMemo(() => parseDotenv(text), [text])

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-w-lg">
                <DialogHeader>
                    <DialogTitle>Import .env</DialogTitle>
                    <DialogDescription>
                        Paste the contents of a .env file. Comments and blank lines are skipped, and existing names are
                        overwritten.
                    </DialogDescription>
                </DialogHeader>
                <div className="flex flex-col gap-4">
                    {services.length > 1 && (
                        <Field label="Add to service">
                            <ServiceSelect services={services} value={service} onChange={setService} />
                        </Field>
                    )}
                    <Field
                        label=".env contents"
                        hint={
                            text.trim() === ''
                                ? undefined
                                : `${pairs.length} ${pairs.length === 1 ? 'variable' : 'variables'} found`
                        }
                    >
                        <Textarea
                            value={text}
                            onChange={(event) => setText(event.target.value)}
                            rows={8}
                            className="font-mono text-compact"
                            placeholder={'DATABASE_URL=postgres://...\nSECRET_KEY=...'}
                        />
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
