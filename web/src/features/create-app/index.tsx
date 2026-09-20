import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ClipboardPaste, LayoutGrid, Link2, Loader2, Rocket } from 'lucide-react'
import { AppTile } from '@/shared/components/ui/AppTile'
import { Button, buttonClasses } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { RadioCard, RadioGroup } from '@/shared/components/ui/RadioGroup'
import { SegmentedControl } from '@/shared/components/ui/SegmentedControl'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/components/ui/Select'
import { Textarea } from '@/shared/components/ui/Textarea'
import { YamlEditor } from '@/shared/components/ui/YamlEditor'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { appHref, ROUTES } from '@/shared/lib/routes'
import { cn } from '@/shared/lib/utils'
import { useApps, useCreateApp, useNodes } from '@/shared/services/api'
import ComposeChecks from '@/features/app-details/components/ComposeChecks'
import { checkCompose, type ComposeCheck } from '@/features/app-details/lib/compose-checks'
import { collectHostPorts, suggestFreePort } from '@/features/app-details/lib/port-conflict'
import { firstExposedService } from './lib/compose-access'
import { TEMPLATES, templateCompose, toRawUrl } from './lib/templates'

type Mode = 'template' | 'paste' | 'link'
type Access = 'none' | 'quick' | 'custom'
type Fetch = { status: 'idle' } | { status: 'loading' } | { status: 'ok' } | { status: 'error'; message: string }

const MODE_OPTIONS = [
    { value: 'template', label: 'Templates', icon: <LayoutGrid className="h-4 w-4" /> },
    { value: 'paste', label: 'Paste compose', icon: <ClipboardPaste className="h-4 w-4" /> },
    { value: 'link', label: 'From a link', icon: <Link2 className="h-4 w-4" /> },
]
const NAME_PATTERN = /^[a-z0-9-]+$/
const MIN_NAME = 3
const MAX_NAME = 63

function nameError(name: string): string | undefined {
    if (name === '') return undefined
    if (!NAME_PATTERN.test(name)) return 'Use lowercase letters, numbers and hyphens only.'
    if (name.length < MIN_NAME) return `Use at least ${MIN_NAME} characters.`
    if (name.length > MAX_NAME) return `Use ${MAX_NAME} characters or fewer.`
    return undefined
}

function NewApp() {
    const navigate = useNavigate()
    const { toast } = useToast()
    const createApp = useCreateApp()
    const { data: nodes = [] } = useNodes()
    const { data: apps = [] } = useApps(undefined)

    const [mode, setMode] = useState<Mode>('template')
    const [templateId, setTemplateId] = useState<string | null>(null)
    const [hostPort, setHostPort] = useState('')
    const [name, setName] = useState('')
    const [description, setDescription] = useState('')
    const [nodeId, setNodeId] = useState('')
    const [pasted, setPasted] = useState('')
    const [link, setLink] = useState('')
    const [fetched, setFetched] = useState<Fetch>({ status: 'idle' })
    const [access, setAccess] = useState<Access>('none')
    const [hostname, setHostname] = useState('')

    const onlineNodes = nodes.filter((node) => node.status === 'online')
    useEffect(() => {
        if (!nodeId && onlineNodes.length > 0) setNodeId((onlineNodes.find((node) => node.is_primary) ?? onlineNodes[0]).id)
    }, [nodeId, onlineNodes])

    const template = TEMPLATES.find((item) => item.id === templateId) ?? null
    const nodeApps = useMemo(() => apps.filter((app) => app.node_id === nodeId), [apps, nodeId])
    const usedPorts = useMemo(() => collectHostPorts(nodeApps.map((app) => app.compose_content)), [nodeApps])

    // A template starts on its usual port, or the next one no other app on the node publishes.
    useEffect(() => {
        if (!template) return
        const free = suggestFreePort(template.defaultHostPort - 1, usedPorts)
        setHostPort(String(free ?? template.defaultHostPort))
        // Only the template and the node choose the starting port. Typing in the field must not reset it.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [templateId, nodeId, apps.length])

    const port = Number(hostPort)
    const content = mode === 'template' ? (template ? templateCompose(template, name, Number.isInteger(port) && port > 0 ? port : template.defaultHostPort) : '') : pasted
    const exposed = useMemo(() => firstExposedService(content), [content])
    const composeResult = useMemo(() => (content.trim() === '' ? null : checkCompose(content, nodeApps)), [content, nodeApps])

    const error = nameError(name)
    const taken = name !== '' && !error && nodeApps.some((app) => app.name === name)
    const nodeName = nodes.find((node) => node.id === nodeId)?.name ?? nodeId
    const portError = mode === 'template' && (!Number.isInteger(port) || port < 1 || port > 65535) ? 'Enter a port from 1 to 65535.' : undefined
    const portShared = mode === 'template' && !portError && usedPorts.has(port) ? `Another app on ${nodeName} already publishes ${port}.` : undefined

    const checks: ComposeCheck[] = []
    if (name !== '') {
        checks.push(
            error || taken
                ? { id: 'name', level: 'err', title: taken ? 'Name already used' : 'Name is not valid', detail: taken ? `${nodeName} already has an app called ${name}.` : error }
                : { id: 'name', level: 'ok', title: 'Name is available' }
        )
    }
    if (nodeId) checks.push({ id: 'node', level: 'ok', title: `Deploys to ${nodeName}` })
    else checks.push({ id: 'node', level: 'err', title: 'No node is online', detail: 'Bring a node online to deploy.' })
    if (portShared) checks.push({ id: 'port-shared', level: 'warn', title: 'Port already published', detail: portShared })
    if (composeResult) checks.push(...composeResult.checks)
    if (access === 'quick' && !exposed) checks.push({ id: 'quick', level: 'err', title: 'Quick Tunnel needs a published port', detail: 'Add a ports entry so the tunnel knows where to send visitors.' })
    if (access === 'custom' && hostname.trim() === '') checks.push({ id: 'host', level: 'err', title: 'Custom domain needs a hostname' })

    let blocked: string | undefined
    if (mode === 'template' && !template) blocked = 'Choose a template'
    else if (content.trim() === '') blocked = mode === 'link' ? 'Fetch a compose file first' : 'Add a compose file'
    else if (name === '') blocked = 'Give the app a name'
    else if (checks.some((check) => check.level === 'err') || portError) blocked = 'Fix the problems in the checks'
    else if (!nodeId) blocked = 'No node is online'

    const fetchLink = async () => {
        setFetched({ status: 'loading' })
        try {
            const response = await fetch(toRawUrl(link))
            if (!response.ok) throw new Error(`The address answered ${response.status}.`)
            const text = await response.text()
            if (text.trim() === '') throw new Error('The file is empty.')
            setPasted(text)
            setFetched({ status: 'ok' })
        } catch (failure) {
            const network = failure instanceof TypeError
            setFetched({ status: 'error', message: network ? 'Could not reach that address. Check the link, or paste the file instead.' : describeError(failure) })
        }
    }

    const create = () => {
        if (blocked) return
        createApp.mutate(
            {
                name,
                description,
                compose_content: content,
                node_id: nodeId,
                tunnel_mode: access === 'none' ? undefined : access,
                quick_tunnel_service: access === 'quick' ? exposed?.service : undefined,
                quick_tunnel_port: access === 'quick' ? exposed?.port : undefined,
                ingress_rules: access === 'custom' && exposed ? [{ hostname: hostname.trim(), service: `http://${exposed.service}:${exposed.port}`, path: null }] : undefined,
            },
            {
                onSuccess: (created) => {
                    toast.success('App created', `${created.name} is being deployed`)
                    navigate(appHref(created, 'logs'))
                },
            }
        )
    }

    const showConfigure = mode === 'template' ? template !== null : content.trim() !== '' || mode === 'paste'

    return (
        <div className="flex flex-col gap-5">
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-semibold tracking-tight">New app</h1>
                    <p className="text-muted-foreground">Pick a template, or bring your own compose file.</p>
                </div>
                <Link to={ROUTES.fleet} className={buttonClasses({ variant: 'ghost' })}>
                    Cancel
                </Link>
            </div>

            <SegmentedControl aria-label="How to start" options={MODE_OPTIONS} value={mode} onValueChange={(value) => setMode(value as Mode)} className="self-start" />

            <div className="grid grid-cols-1 gap-5 lg:grid-cols-[minmax(0,1fr)_360px]">
                <div className="flex min-w-0 flex-col gap-5">
                    {mode === 'template' && (
                        <div role="group" aria-label="Templates" className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
                            {TEMPLATES.map((item) => (
                                <button
                                    key={item.id}
                                    type="button"
                                    aria-pressed={item.id === templateId}
                                    onClick={() => {
                                        setTemplateId(item.id)
                                        if (name === '' || TEMPLATES.some((other) => other.id === name)) setName(item.id)
                                    }}
                                    className={cn(
                                        'flex min-h-[44px] items-start gap-3 rounded-xl border bg-card p-3.5 text-left transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                                        item.id === templateId ? 'border-primary ring-1 ring-primary' : 'border-border'
                                    )}
                                >
                                    <AppTile name={item.name} size="md" />
                                    <span className="flex min-w-0 flex-col">
                                        <span className="text-[14px] font-semibold">{item.name}</span>
                                        <span className="text-[12.5px] text-muted-foreground">{item.description}</span>
                                    </span>
                                </button>
                            ))}
                        </div>
                    )}

                    {mode === 'paste' && (
                        <Card>
                            <CardHeader>
                                <CardTitle className="text-base">Compose file</CardTitle>
                            </CardHeader>
                            <CardContent className="flex flex-col gap-3">
                                <YamlEditor value={pasted} onChange={setPasted} aria-label="Compose file" height={360} />
                                {pasted.trim() === '' && <p className="text-[13px] text-muted-foreground">Paste a docker-compose.yml. It is checked as you type.</p>}
                            </CardContent>
                        </Card>
                    )}

                    {mode === 'link' && (
                        <Card>
                            <CardHeader>
                                <CardTitle className="text-base">Fetch a compose file</CardTitle>
                            </CardHeader>
                            <CardContent className="flex flex-col gap-4">
                                <form
                                    className="flex flex-col gap-3 sm:flex-row sm:items-end"
                                    onSubmit={(event) => {
                                        event.preventDefault()
                                        if (link.trim() !== '') void fetchLink()
                                    }}
                                >
                                    <Field label="Link to the file" hint="A GitHub file page or a raw address, for example github.com/you/repo/blob/main/docker-compose.yml" className="flex-1">
                                        <Input value={link} onChange={(event) => setLink(event.target.value)} placeholder="https://github.com/..." inputMode="url" />
                                    </Field>
                                    <Button type="submit" disabled={link.trim() === '' || fetched.status === 'loading'}>
                                        {fetched.status === 'loading' && <Loader2 className="h-4 w-4 animate-spin" />}
                                        Fetch
                                    </Button>
                                </form>
                                {fetched.status === 'error' && (
                                    <p role="alert" className="text-sm text-status-err-fg">
                                        {fetched.message}
                                    </p>
                                )}
                                {fetched.status === 'ok' && (
                                    <>
                                        <p role="status" className="text-sm text-status-ok-fg">Fetched. You can still edit it before deploying.</p>
                                        <YamlEditor value={pasted} onChange={setPasted} aria-label="Fetched compose file" height={320} />
                                    </>
                                )}
                                {fetched.status === 'idle' && (
                                    <EmptyState title="Nothing fetched yet" description="Private repositories and links that need a login cannot be fetched from the browser. Paste the file instead." className="py-8" />
                                )}
                            </CardContent>
                        </Card>
                    )}

                    {showConfigure && (
                        <Card>
                            <CardHeader>
                                <CardTitle className="text-base">Configure</CardTitle>
                            </CardHeader>
                            <CardContent className="flex flex-col gap-5">
                                <div className="grid gap-4 sm:grid-cols-2">
                                    <Field label="Name" error={error ?? (taken ? `${nodeName} already has an app called ${name}.` : undefined)} hint="Lowercase letters, numbers and hyphens.">
                                        <Input value={name} onChange={(event) => setName(event.target.value)} placeholder="my-app" autoComplete="off" />
                                    </Field>
                                    <Field label="Node">
                                        <Select value={nodeId} onValueChange={setNodeId}>
                                            <SelectTrigger aria-label="Node">
                                                <SelectValue placeholder="No node is online" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                {onlineNodes.map((node) => (
                                                    <SelectItem key={node.id} value={node.id}>
                                                        {node.name}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                    </Field>
                                </div>
                                {mode === 'template' && (
                                    <Field label="Port on the node" error={portError} hint={portShared ?? 'The address you open it on inside your network.'} className="sm:max-w-[200px]">
                                        <Input value={hostPort} onChange={(event) => setHostPort(event.target.value.replace(/\D/g, ''))} inputMode="numeric" className="font-mono" />
                                    </Field>
                                )}
                                <Field label="Description">
                                    <Textarea value={description} onChange={(event) => setDescription(event.target.value)} rows={2} placeholder="Optional" />
                                </Field>

                                <fieldset className="flex flex-col gap-2">
                                    <legend className="mb-1 text-[13px] font-medium">Who can reach it</legend>
                                    <RadioGroup value={access} onValueChange={(value) => setAccess(value as Access)} aria-label="Who can reach it">
                                        <RadioCard value="none" title="Only my network" description="No public address. You can add one later." />
                                        <RadioCard value="quick" title="Quick Tunnel" description="A temporary public address on trycloudflare.com. No account needed." />
                                        <RadioCard value="custom" title="Your own domain" description="A stable address through Cloudflare. Needs Cloudflare connected in Settings." />
                                    </RadioGroup>
                                    {access === 'custom' && (
                                        <Field label="Hostname" hint={exposed ? `Sends visitors to ${exposed.service} on port ${exposed.port}.` : undefined} className="mt-2 sm:max-w-sm">
                                            <Input value={hostname} onChange={(event) => setHostname(event.target.value)} placeholder="app.example.com" inputMode="url" />
                                        </Field>
                                    )}
                                </fieldset>
                            </CardContent>
                        </Card>
                    )}
                </div>

                <div className="flex min-w-0 flex-col gap-5 lg:sticky lg:top-4 lg:self-start">
                    {mode === 'template' && template && (
                        <Card>
                            <CardHeader>
                                <CardTitle className="text-base">Compose file</CardTitle>
                            </CardHeader>
                            <CardContent>
                                <YamlEditor value={content} readOnly aria-label="Generated compose file" height={260} />
                                <p className="mt-2 text-[13px] text-muted-foreground">Generated from the template. You can edit it after the app is created.</p>
                            </CardContent>
                        </Card>
                    )}
                    {checks.length > 0 && <ComposeChecks checks={checks} />}
                </div>
            </div>

            {createApp.error && (
                <p role="alert" className="rounded-lg bg-status-err-bg px-3.5 py-3 text-sm text-status-err-fg">
                    Could not create the app. {describeError(createApp.error)}
                </p>
            )}

            <div role="region" aria-label="Create" className="sticky bottom-[calc(var(--mobile-nav-h)+12px)] z-20 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-card p-3 shadow-lg md:bottom-4">
                <span className="text-sm font-medium">{blocked ?? `Ready to deploy ${name} to ${nodeName}`}</span>
                <div className="flex items-center gap-2">
                    <Link to={ROUTES.fleet} className={buttonClasses({ variant: 'ghost' })}>
                        Cancel
                    </Link>
                    <Button onClick={create} disabled={!!blocked || createApp.isPending}>
                        {createApp.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />}
                        Create app
                    </Button>
                </div>
            </div>
        </div>
    )
}

export default NewApp
