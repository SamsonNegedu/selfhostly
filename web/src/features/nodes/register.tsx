import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, CheckCircle2, Copy, KeyRound, Loader2 } from 'lucide-react'
import { Button, buttonClasses } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { CodeBlock } from '@/shared/components/ui/CodeBlock'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { SegmentedControl } from '@/shared/components/ui/SegmentedControl'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError, fieldError } from '@/shared/lib/errors'
import { ROUTES } from '@/shared/lib/routes'
import { useCreateJoinToken, useNodeHealth, useNodes, useRegisterNode } from '@/shared/services/api'
import type { RegisterNodeRequest } from '@/shared/types/api'

type Mode = 'automatic' | 'manual'

const MODE_OPTIONS = [
    { value: 'automatic', label: 'Automatic' },
    { value: 'manual', label: 'Manual' },
]
const POLL_MS = 3000

async function copy(text: string, toast: ReturnType<typeof useToast>['toast'], what: string) {
    try {
        await navigator.clipboard.writeText(text)
        toast.success('Copied', `${what} copied to your clipboard`)
    } catch {
        toast.error('Could not copy', 'Your browser blocked access to the clipboard')
    }
}

function RegisterNodePage() {
    const [mode, setMode] = useState<Mode>('automatic')

    return (
        <div className="flex max-w-3xl flex-col gap-5">
            <div>
                <Link to={ROUTES.nodes} className={buttonClasses({ variant: 'ghost', className: '-ml-3' })}>
                    <ArrowLeft className="h-4 w-4" />
                    Nodes
                </Link>
                <h1 className="mt-1 text-2xl font-semibold tracking-tight">Add a node</h1>
                <p className="text-muted-foreground">Run apps on another machine and manage them from here.</p>
            </div>

            <SegmentedControl aria-label="How to add" options={MODE_OPTIONS} value={mode} onValueChange={(value) => setMode(value as Mode)} className="self-start" />

            {mode === 'automatic' ? <AutomaticJoin /> : <ManualRegister />}
        </div>
    )
}

// The new machine adds itself with a single-use token. This page makes the token and the settings to paste,
// then waits for the machine to show up.
function AutomaticJoin() {
    const { toast } = useToast()
    const createToken = useCreateJoinToken()
    const [primaryUrl, setPrimaryUrl] = useState(window.location.origin)
    const [nodeUrl, setNodeUrl] = useState('')
    const issued = createToken.data
    const { data: nodes } = useNodes({ refetchInterval: issued ? POLL_MS : false })
    const knownAtStart = useRef<Set<string> | null>(null)

    // Remember who was in the cluster when the token was made, so anyone new after that is the machine that joined.
    useEffect(() => {
        if (issued && nodes && knownAtStart.current === null) knownAtStart.current = new Set(nodes.map((node) => node.id))
    }, [issued, nodes])
    const joined = issued && knownAtStart.current ? nodes?.find((node) => !knownAtStart.current?.has(node.id)) : undefined

    const env = useMemo(
        () =>
            [
                'NODE_IS_PRIMARY=false',
                `PRIMARY_NODE_URL=${primaryUrl.trim()}`,
                `REGISTRATION_TOKEN=${issued?.token ?? ''}`,
                `NODE_API_ENDPOINT=${nodeUrl.trim() || 'http://<this-machine-address>:8080'}`,
            ].join('\n'),
        [primaryUrl, nodeUrl, issued]
    )

    return (
        <div className="flex flex-col gap-5">
            <Card>
                <CardHeader>
                    <CardTitle className="text-base">1. Make a join token</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-col gap-4">
                    <p className="text-sm text-muted-foreground">The token works once and expires after an hour, so it is safe to paste into the new machine's settings.</p>
                    <div className="grid gap-4 sm:grid-cols-2">
                        <Field label="This server's address" hint="What the new machine will use to reach this one.">
                            <Input value={primaryUrl} onChange={(event) => setPrimaryUrl(event.target.value)} inputMode="url" className="font-mono" />
                        </Field>
                        <Field label="The new machine's address" hint="Where this server can reach the new machine.">
                            <Input value={nodeUrl} onChange={(event) => setNodeUrl(event.target.value)} placeholder="http://192.168.1.50:8080" inputMode="url" className="font-mono" />
                        </Field>
                    </div>
                    <div className="flex flex-wrap items-center gap-3">
                        <Button onClick={() => createToken.mutate()} disabled={createToken.isPending}>
                            {createToken.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <KeyRound className="h-4 w-4" />}
                            {issued ? 'Make a new token' : 'Make a join token'}
                        </Button>
                        {issued && <span className="text-[13px] text-muted-foreground">Expires {new Date(issued.expires_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>}
                    </div>
                    {createToken.error && (
                        <p role="alert" className="text-sm text-status-err-fg">
                            Could not make a token. {describeError(createToken.error)}
                        </p>
                    )}
                </CardContent>
            </Card>

            {issued && (
                <Card>
                    <CardHeader>
                        <CardTitle className="text-base">2. Put this in the new machine's .env</CardTitle>
                    </CardHeader>
                    <CardContent className="flex flex-col gap-4">
                        <CodeBlock lines={env.split('\n').map((text) => ({ text }))} aria-label="Settings for the new machine" />
                        <div className="flex flex-wrap items-center gap-3">
                            <Button variant="outline" onClick={() => void copy(env, toast, 'The settings')}>
                                <Copy className="h-4 w-4" />
                                Copy settings
                            </Button>
                            <span className="text-[13px] text-muted-foreground">Install Selfhostly there, add these lines to its .env, and start it.</span>
                        </div>
                    </CardContent>
                </Card>
            )}

            {issued && (
                <Card aria-label="Connection" role="region">
                    <CardContent className="flex items-center gap-3 p-4">
                        {joined ? (
                            <>
                                <CheckCircle2 aria-hidden="true" className="h-5 w-5 shrink-0 text-status-ok-fg" />
                                <div className="flex-1">
                                    <p className="font-semibold">{joined.name} joined</p>
                                    <p className="text-[13px] text-muted-foreground">It is {joined.status}. You can put apps on it now.</p>
                                </div>
                                <Link to={ROUTES.nodes} className={buttonClasses()}>
                                    See nodes
                                </Link>
                            </>
                        ) : (
                            <>
                                <Loader2 aria-hidden="true" className="h-5 w-5 shrink-0 animate-spin text-muted-foreground" />
                                <div className="flex-1">
                                    <p className="font-semibold">Waiting for the new machine</p>
                                    <p className="text-[13px] text-muted-foreground">This page checks every few seconds and shows it here as soon as it starts.</p>
                                </div>
                            </>
                        )}
                    </CardContent>
                </Card>
            )}
        </div>
    )
}

// For a machine that cannot reach this one, or that already has its own ID and key.
function ManualRegister() {
    const navigate = useNavigate()
    const { toast } = useToast()
    const register = useRegisterNode()
    const [form, setForm] = useState<RegisterNodeRequest>({ id: '', name: '', api_endpoint: '', api_key: '' })
    const [registeredId, setRegisteredId] = useState<string | null>(null)
    const health = useNodeHealth(registeredId ?? '')
    const { data: nodes = [] } = useNodes()

    const idTaken = form.id.trim() !== '' && nodes.some((node) => node.id === form.id.trim())
    const nameTaken = form.name.trim() !== '' && nodes.some((node) => node.name.toLowerCase() === form.name.trim().toLowerCase())
    const valid = form.id.trim() !== '' && form.name.trim() !== '' && form.api_endpoint.trim() !== '' && form.api_key.trim() !== '' && !idTaken && !nameTaken
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
                    <p className="text-sm text-muted-foreground">Copy the ID and API key from the new machine's start-up log or its .env.</p>
                    <Field label="Node ID" hint="From the new machine. Needed so its heartbeats are accepted." error={idTaken ? 'A node with this ID is already in the cluster.' : fieldError(register.error, 'id')}>
                        <Input value={form.id} onChange={(event) => set({ id: event.target.value })} className="font-mono" autoComplete="off" />
                    </Field>
                    <Field label="Name" hint="Shown on cards and in the node picker, for example garage-nas." error={nameTaken ? 'A node with this name is already in the cluster.' : fieldError(register.error, 'name')}>
                        <Input value={form.name} onChange={(event) => set({ name: event.target.value })} placeholder="garage-nas" autoComplete="off" />
                    </Field>
                    <Field label="Address" hint="Where this server reaches the machine." error={fieldError(register.error, 'api_endpoint')}>
                        <Input type="url" value={form.api_endpoint} onChange={(event) => set({ api_endpoint: event.target.value })} placeholder="http://192.168.1.50:8080" className="font-mono" />
                    </Field>
                    <Field label="API key" hint="The NODE_API_KEY set on the new machine.">
                        <Input type="password" value={form.api_key} onChange={(event) => set({ api_key: event.target.value })} autoComplete="off" />
                    </Field>
                    {register.error && !['id', 'name', 'api_endpoint'].some((field) => fieldError(register.error, field)) && (
                        <p role="alert" className="rounded-lg bg-status-err-bg px-3.5 py-3 text-sm text-status-err-fg">
                            Could not add the node. {describeError(register.error)}
                        </p>
                    )}
                    <div className="flex justify-end gap-2">
                        <Link to={ROUTES.nodes} className={buttonClasses({ variant: 'ghost' })}>
                            Cancel
                        </Link>
                        <Button type="submit" disabled={!valid || register.isPending}>
                            {register.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                            {register.isPending ? 'Adding' : 'Add node'}
                        </Button>
                    </div>
                </form>
            </CardContent>
        </Card>
    )
}

export default RegisterNodePage
