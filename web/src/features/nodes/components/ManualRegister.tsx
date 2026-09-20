import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { Button, buttonClasses } from '@/shared/components/ui/Button'
import { Card, CardContent } from '@/shared/components/ui/Card'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError, fieldError } from '@/shared/lib/errors'
import { ROUTES } from '@/shared/lib/routes'
import { useNodeHealth, useNodes, useRegisterNode } from '@/shared/services/api'
import type { RegisterNodeRequest } from '@/shared/types/api'
import { findConflicts, isRegistrationValid } from '../lib/register-rules'

// For a machine that cannot reach this one, or that already has its own ID and key.
function ManualRegister() {
    const navigate = useNavigate()
    const { toast } = useToast()
    const register = useRegisterNode()
    const [form, setForm] = useState<RegisterNodeRequest>({ id: '', name: '', api_endpoint: '', api_key: '' })
    const [registeredId, setRegisteredId] = useState<string | null>(null)
    const health = useNodeHealth(registeredId ?? '')
    const { data: nodes = [] } = useNodes()

    const { idTaken, nameTaken } = findConflicts(form, nodes)
    const valid = isRegistrationValid(form, nodes)
    const set = (patch: Partial<RegisterNodeRequest>) => setForm({ ...form, ...patch })

    const submit = (event: React.FormEvent) => {
        event.preventDefault()
        if (!valid) return
        register.mutate(form, {
            onSuccess: (node) => {
                setRegisteredId(node.id)
                toast.success('Node added', `${node.name} is registered. Checking the connection.`)
            },
        })
    }

    useEffect(() => {
        if (registeredId) health.mutate()
        // Check once, right after registering.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [registeredId])

    if (registeredId) {
        return (
            <Card aria-label="Connection" role="region">
                <CardContent className="flex flex-wrap items-center gap-3 p-4">
                    {health.isPending ? (
                        <Loader2 aria-hidden="true" className="h-5 w-5 animate-spin text-muted-foreground" />
                    ) : health.isError ? (
                        <StatusPill kind="warn">Not answering</StatusPill>
                    ) : (
                        <StatusPill kind="ok">Connected</StatusPill>
                    )}
                    <p className="flex-1 text-sm">
                        {health.isPending
                            ? 'Checking the connection.'
                            : health.isError
                              ? `Registered, but it did not answer (${describeError(health.error).replace(/[.\s]+$/, '')}). Check the address, and that the node is running.`
                              : `${form.name} answered. You can put apps on it now.`}
                    </p>
                    {health.isError && (
                        <Button variant="outline" onClick={() => health.mutate()}>
                            Check again
                        </Button>
                    )}
                    <Button onClick={() => navigate(ROUTES.nodes)}>See nodes</Button>
                </CardContent>
            </Card>
        )
    }

    return (
        <Card>
            <CardContent className="p-4 sm:p-6">
                <form onSubmit={submit} className="flex flex-col gap-4">
                    <p className="text-sm text-muted-foreground">
                        Copy the ID and API key from the new machine's start-up log or its .env.
                    </p>
                    <Field
                        label="Node ID"
                        hint="From the new machine. Needed so its heartbeats are accepted."
                        error={
                            idTaken
                                ? 'A node with this ID is already in the cluster.'
                                : fieldError(register.error, 'id')
                        }
                    >
                        <Input
                            value={form.id}
                            onChange={(event) => set({ id: event.target.value })}
                            className="font-mono"
                            autoComplete="off"
                        />
                    </Field>
                    <Field
                        label="Name"
                        hint="Shown on cards and in the node picker, for example garage-nas."
                        error={
                            nameTaken
                                ? 'A node with this name is already in the cluster.'
                                : fieldError(register.error, 'name')
                        }
                    >
                        <Input
                            value={form.name}
                            onChange={(event) => set({ name: event.target.value })}
                            placeholder="garage-nas"
                            autoComplete="off"
                        />
                    </Field>
                    <Field
                        label="Address"
                        hint="Where this server reaches the machine."
                        error={fieldError(register.error, 'api_endpoint')}
                    >
                        <Input
                            type="url"
                            value={form.api_endpoint}
                            onChange={(event) => set({ api_endpoint: event.target.value })}
                            placeholder="http://192.168.1.50:8080"
                            className="font-mono"
                        />
                    </Field>
                    <Field label="API key" hint="The NODE_API_KEY set on the new machine.">
                        <Input
                            type="password"
                            value={form.api_key}
                            onChange={(event) => set({ api_key: event.target.value })}
                            autoComplete="off"
                        />
                    </Field>
                    {register.error &&
                        !['id', 'name', 'api_endpoint'].some((field) => fieldError(register.error, field)) && (
                            <p
                                role="alert"
                                className="rounded-lg bg-status-err-bg px-3.5 py-3 text-sm text-status-err-fg"
                            >
                                Could not add the node. {describeError(register.error)}
                            </p>
                        )}
                    <div className="flex justify-end gap-2">
                        <Link to={ROUTES.nodes} className={buttonClasses({ variant: 'ghost' })}>
                            Cancel
                        </Link>
                        <Button type="submit" disabled={!valid || register.isPending}>
                            {register.isPending && <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />}
                            {register.isPending ? 'Adding' : 'Add node'}
                        </Button>
                    </div>
                </form>
            </CardContent>
        </Card>
    )
}

export default ManualRegister
