import { useState } from 'react'
import { Loader2, Plus, Trash2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/shared/components/ui/Dialog'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { useCreateQuickTunnelForApp, useCreateTunnelForApp, useSwitchAppToCustomTunnel } from '@/shared/services/api'
import type { IngressRule } from '@/shared/types/api'
import { firstExposedService } from '@/features/create-app/lib/compose-access'

const MAX_PORT = 65535
const EMPTY_RULE: IngressRule = { service: '', hostname: null, path: null }

interface DialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    appId: string
    nodeId: string
    // Used to suggest the service and port, so most people only have to confirm them.
    composeContent?: string
}

// A temporary trycloudflare.com address that needs no account.
export function QuickTunnelDialog({
    open,
    onOpenChange,
    appId,
    nodeId,
    composeContent,
    recreate = false,
}: DialogProps & { recreate?: boolean }) {
    const { toast } = useToast()
    const create = useCreateQuickTunnelForApp()
    const [service, setService] = useState('')
    const [port, setPort] = useState('80')
    const [error, setError] = useState<string | null>(null)

    // Fill in a guess from the compose file each time the dialog opens.
    const [seen, setSeen] = useState({ open, composeContent })
    if (seen.open !== open || seen.composeContent !== composeContent) {
        setSeen({ open, composeContent })
        if (open) {
            const guess = composeContent ? firstExposedService(composeContent) : null
            setService(guess?.service ?? '')
            setPort(String(guess?.port ?? 80))
            setError(null)
        }
    }

    const submit = (event: React.FormEvent) => {
        event.preventDefault()
        const portNumber = Number(port)
        if (!service.trim()) return setError('Enter the name of the service that serves the app.')
        if (!Number.isInteger(portNumber) || portNumber < 1 || portNumber > MAX_PORT)
            return setError(`Enter a port from 1 to ${MAX_PORT}.`)
        setError(null)
        create.mutate(
            { appId, nodeId, service: service.trim(), port: portNumber },
            {
                onSuccess: () => {
                    onOpenChange(false)
                    toast.info('Quick Tunnel starting', 'Your temporary address appears here in a moment.')
                },
                onError: (failure) => setError(describeError(failure)),
            },
        )
    }

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-w-md">
                <DialogHeader>
                    <DialogTitle>{recreate ? 'Get a new Quick Tunnel address' : 'Create a Quick Tunnel'}</DialogTitle>
                    <DialogDescription>
                        A temporary public address on trycloudflare.com. It can change when the app restarts, so it
                        suits trying things out, not production.
                    </DialogDescription>
                </DialogHeader>
                <form onSubmit={submit} className="flex flex-col gap-4">
                    <Field label="Service" hint="The service name in your compose file that listens for web traffic.">
                        <Input
                            value={service}
                            onChange={(event) => setService(event.target.value)}
                            placeholder="web"
                            autoComplete="off"
                        />
                    </Field>
                    <Field label="Port" hint="The port that service listens on inside its container.">
                        <Input
                            value={port}
                            onChange={(event) => setPort(event.target.value.replace(/\D/g, ''))}
                            inputMode="numeric"
                            className="font-mono"
                        />
                    </Field>
                    {error && (
                        <p role="alert" className="text-sm text-status-err-fg">
                            {error}
                        </p>
                    )}
                    <div className="flex justify-end gap-2">
                        <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                            Cancel
                        </Button>
                        <Button type="submit" disabled={create.isPending}>
                            {create.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                            {recreate ? 'Get a new address' : 'Create Quick Tunnel'}
                        </Button>
                    </div>
                </form>
            </DialogContent>
        </Dialog>
    )
}

// A stable address on your own domain, through a named Cloudflare tunnel.
export function CustomDomainDialog({
    open,
    onOpenChange,
    appId,
    nodeId,
    composeContent,
    switching = false,
}: DialogProps & { switching?: boolean }) {
    const { toast } = useToast()
    const createTunnel = useCreateTunnelForApp()
    const switchTunnel = useSwitchAppToCustomTunnel()
    const mutation = switching ? switchTunnel : createTunnel
    const [rules, setRules] = useState<IngressRule[]>([EMPTY_RULE])
    const [error, setError] = useState<string | null>(null)

    const [seen, setSeen] = useState({ open, composeContent })
    if (seen.open !== open || seen.composeContent !== composeContent) {
        setSeen({ open, composeContent })
        if (open) {
            const guess = composeContent ? firstExposedService(composeContent) : null
            setRules([{ ...EMPTY_RULE, service: guess ? `http://${guess.service}:${guess.port}` : '' }])
            setError(null)
        }
    }

    const change = (index: number, patch: Partial<IngressRule>) =>
        setRules(rules.map((rule, i) => (i === index ? { ...rule, ...patch } : rule)))

    const submit = (event: React.FormEvent) => {
        event.preventDefault()
        const filled = rules.filter((rule) => rule.service.trim() !== '')
        if (filled.length === 0) return setError('Add at least one route with a service address.')
        if (!filled.some((rule) => rule.hostname && rule.hostname.trim() !== ''))
            return setError('At least one route needs a hostname, your own domain, such as app.example.com.')
        setError(null)
        mutation.mutate(
            {
                appId,
                nodeId,
                ingressRules: filled.map((rule) => ({
                    hostname: rule.hostname || undefined,
                    service: rule.service.trim(),
                    path: rule.path || undefined,
                })),
            },
            {
                onSuccess: () => {
                    onOpenChange(false)
                    toast.info(
                        switching ? 'Switching to your domain' : 'Creating your tunnel',
                        'This runs in the background. The new address appears here when it is ready.',
                    )
                },
                onError: (failure) => setError(describeError(failure)),
            },
        )
    }

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="flex max-h-[90vh] max-w-md flex-col">
                <DialogHeader>
                    <DialogTitle>{switching ? 'Switch to your own domain' : 'Use your own domain'}</DialogTitle>
                    <DialogDescription>
                        Say which hostname goes to which service, for example app.example.com to http://web:80.
                        {switching ? ' This replaces the temporary Quick Tunnel address.' : ''}
                    </DialogDescription>
                </DialogHeader>
                <form onSubmit={submit} className="flex min-h-0 flex-1 flex-col gap-4">
                    {/* The padding and matching negative margin leave room for the focus ring, which the scroll area would clip. */}
                    <div className="-m-1 flex min-h-0 flex-col gap-5 overflow-y-auto p-1">
                        {rules.map((rule, index) => (
                            <fieldset
                                key={index}
                                className="flex flex-col gap-3 border-t border-border pt-4 first:border-t-0 first:pt-0"
                            >
                                <legend className="sr-only">Route {index + 1}</legend>
                                <Field label={rules.length > 1 ? `Hostname (route ${index + 1})` : 'Hostname'}>
                                    <Input
                                        value={rule.hostname ?? ''}
                                        onChange={(event) =>
                                            change(index, { hostname: event.target.value.trim() || null })
                                        }
                                        placeholder="app.example.com"
                                        inputMode="url"
                                    />
                                </Field>
                                <Field label="Service address" hint="Where the app answers, for example http://web:80.">
                                    <Input
                                        value={rule.service}
                                        onChange={(event) => change(index, { service: event.target.value.trim() })}
                                        placeholder="http://web:80"
                                        inputMode="url"
                                        className="font-mono"
                                    />
                                </Field>
                                <Field label="Path (optional)">
                                    <Input
                                        value={rule.path ?? ''}
                                        onChange={(event) => change(index, { path: event.target.value.trim() || null })}
                                        placeholder="/"
                                    />
                                </Field>
                                {rules.length > 1 && (
                                    <Button
                                        type="button"
                                        variant="ghost"
                                        size="sm"
                                        className="self-start text-destructive"
                                        onClick={() => setRules(rules.filter((_, i) => i !== index))}
                                    >
                                        <Trash2 className="h-4 w-4" />
                                        Remove route
                                    </Button>
                                )}
                            </fieldset>
                        ))}
                        <Button
                            type="button"
                            variant="outline"
                            className="self-start"
                            onClick={() => setRules([...rules, EMPTY_RULE])}
                        >
                            <Plus className="h-4 w-4" />
                            Add a route
                        </Button>
                    </div>
                    {error && (
                        <p role="alert" className="text-sm text-status-err-fg">
                            {error}
                        </p>
                    )}
                    <div className="flex justify-end gap-2">
                        <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                            Cancel
                        </Button>
                        <Button type="submit" disabled={mutation.isPending}>
                            {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                            {switching ? 'Switch to my domain' : 'Create tunnel'}
                        </Button>
                    </div>
                </form>
            </DialogContent>
        </Dialog>
    )
}
