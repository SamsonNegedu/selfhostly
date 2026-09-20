import { useState } from 'react'
import { LayoutGrid, List, Play, Plus, RotateCw, Square, Trash2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Checkbox } from '@/shared/components/ui/Checkbox'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { Kbd } from '@/shared/components/ui/Kbd'
import { RadioCard, RadioGroup } from '@/shared/components/ui/RadioGroup'
import { SegmentedControl } from '@/shared/components/ui/SegmentedControl'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/components/ui/Select'
import { Switch } from '@/shared/components/ui/Switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/components/ui/Tabs'
import { AppTile } from '@/shared/components/ui/AppTile'
import { CodeBlock } from '@/shared/components/ui/CodeBlock'
import { DiffBlock } from '@/shared/components/ui/DiffBlock'
import { Terminal, type TerminalLine } from '@/shared/components/ui/Terminal'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Sheet, SheetContent, SheetDescription, SheetTitle, SheetTrigger } from '@/shared/components/ui/Sheet'
import { CardSkeleton, Skeleton } from '@/shared/components/ui/Skeleton'
import { useToast } from '@/shared/components/ui/Toast'
import { TooltipContent, TooltipProvider, TooltipTrigger } from '@/shared/components/ui/Tooltip'
import { AreaChart, Sparkline } from '@/shared/components/ui/Chart'
import { Card } from '@/shared/components/ui/Card'
import { ProgressBar } from '@/shared/components/ui/ProgressBar'
import { StatusDot, StatusPill } from '@/shared/components/ui/StatusPill'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/components/ui/Table'
import { appStatusMeta, jobStatusMeta, nodeStatusMeta, type StatusKind } from '@/shared/lib/status'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/shared/components/ui/DropdownMenu'
import { Textarea } from '@/shared/components/ui/Textarea'
import { ErrorBoundary } from '@/shared/components/ErrorBoundary'
import { JobProgress } from '@/shared/components/ui/JobProgress'
import { FleetEmpty, FleetLoading } from '@/features/dashboard/components/FleetStates'

const BUTTON_VARIANTS = ['default', 'outline', 'secondary', 'ghost', 'danger', 'destructive', 'link'] as const
const BUTTON_SIZES = ['sm', 'default', 'lg'] as const
const APP_TABS = ['Overview', 'Config', 'Environment', 'Logs', 'Access', 'Schedule', 'History'] as const
const VIEW_OPTIONS = [
    { value: 'cards', label: 'Cards', icon: <LayoutGrid className="h-4 w-4" /> },
    { value: 'table', label: 'Table', icon: <List className="h-4 w-4" /> },
]
const NODE_NAMES = ['Pi Primary', 'Garage NAS', 'Hetzner VPS'] as const
const STATUS_KINDS: StatusKind[] = ['ok', 'warn', 'err', 'info', 'idle']
const APP_STATUSES = ['running', 'stopped', 'updating', 'pending', 'error', 'mystery'] as const
const NODE_STATUSES = ['online', 'offline', 'unreachable'] as const
const JOB_STATUSES = ['pending', 'running', 'completed', 'failed'] as const
const APP_NAMES = ['nextcloud', 'pihole', 'jellyfin', 'vaultwarden', 'homepage', 'gitea'] as const
const COMPOSE_BEFORE = `services:
  app:
    image: nextcloud:28
    ports:
      - "8081:80"
    # Data lives on the host
    volumes:
      - /data/nextcloud:/var/www/html
    restart: unless-stopped
`
const COMPOSE_AFTER = COMPOSE_BEFORE.replace('nextcloud:28', 'nextcloud:29')
const LONG_LINE = '      - LONG_VARIABLE=https://example.com/a/very/long/path/that/keeps/going/and/going/until/it/is/wider/than/any/phone/screen/could/hold'
const LOG_LINES: TerminalLine[] = [
    { time: '17:02:11', level: 'info', text: 'GET /status.php 200 3ms' },
    { time: '17:03:12', level: 'info', text: 'cron.php finished in 0.41s' },
    { time: '17:05:02', level: 'warn', text: 'Slow query detected (312ms) in oc_filecache' },
    { time: '17:07:03', level: 'error', text: 'Could not connect to redis at redis:6379 (retrying)' },
    { time: '17:07:11', text: 'A line with a time but no level' },
    { text: 'A plain line with neither' },
    ...Array.from({ length: 10 }, (_, i) => ({ time: `17:08:${10 + i}`, level: 'info' as const, text: `GET /status.php 200 ${i + 2}ms` })),
]
const CPU_SERIES = [22, 28, 25, 31, 40, 36, 30, 27, 33, 38, 34, 32]
const MEMORY_SERIES = [70, 72, 74, 77, 79, 80, 81, 82, 83, 83, 84, 84]
const PROGRESS_SAMPLES: { value: number; tone: StatusKind }[] = [
    { value: 0, tone: 'info' },
    { value: 45, tone: 'ok' },
    { value: 84, tone: 'warn' },
    { value: 100, tone: 'err' },
]

function Breaks({ broken }: { broken: boolean }) {
    if (broken) throw new Error('Deliberate failure to show the error boundary')
    return <p className="text-sm text-muted-foreground">This part of the page renders normally.</p>
}

function BoundaryDemo() {
    const [broken, setBroken] = useState(false)
    return (
        <div data-row="error-boundary" className="flex flex-col gap-3">
            <Button variant="outline" className="self-start" onClick={() => setBroken(true)}>Break this part</Button>
            <ErrorBoundary resetKey={broken ? 'broken' : 'ok'}>
                <Breaks broken={broken} />
            </ErrorBoundary>
        </div>
    )
}

function Section({ id, title, children }: { id: string; title: string; children: React.ReactNode }) {
    return (
        <section id={id} className="space-y-3">
            <h2 className="text-sm font-semibold text-muted-foreground">{title}</h2>
            <div className="rounded-xl border bg-card p-5">{children}</div>
        </section>
    )
}

/**
 * Development-only gallery of the shared UI primitives in every state.
 * It exists so each primitive can be checked in isolation, in both themes and at both viewports.
 */
export default function UiGallery() {
    const [view, setView] = useState('cards')
    const [exposure, setExposure] = useState('custom')
    const [node, setNode] = useState<string>(NODE_NAMES[0])
    const [notify, setNotify] = useState(true)
    const { toast } = useToast()
    const [retries, setRetries] = useState(0)
    const [confirmOpen, setConfirmOpen] = useState<'none' | 'restart' | 'delete'>('none')
    const [eventLog, setEventLog] = useState<string[]>([])
    const log = (entry: string) => setEventLog((prev) => [...prev, entry])

    return (
        <div className="space-y-8">
            <div>
                <h1 className="text-2xl font-semibold">UI gallery</h1>
                <p className="text-sm text-muted-foreground">Development only. Not part of the production build.</p>
            </div>

            <Section id="buttons" title="Buttons">
                <div className="space-y-4">
                    {BUTTON_SIZES.map((size) => (
                        <div key={size} className="flex flex-wrap items-center gap-3" data-size-row={size}>
                            {BUTTON_VARIANTS.map((variant) => (
                                <Button key={variant} variant={variant} size={size}>
                                    {variant}
                                </Button>
                            ))}
                        </div>
                    ))}
                    <div className="flex flex-wrap items-center gap-3" data-row="icons">
                        <Button><Play className="h-4 w-4" />With icon</Button>
                        <Button variant="outline"><RotateCw className="h-4 w-4" />Restart</Button>
                        <Button variant="outline"><Square className="h-4 w-4" />Stop</Button>
                        <Button variant="danger"><Trash2 className="h-4 w-4" />Delete</Button>
                        <Button variant="outline" size="icon" aria-label="Add"><Plus className="h-4 w-4" /></Button>
                        <Button variant="ghost" size="icon" aria-label="Restart"><RotateCw className="h-4 w-4" /></Button>
                    </div>
                    <div className="flex flex-wrap items-center gap-3" data-row="disabled">
                        <Button disabled>disabled</Button>
                        <Button variant="outline" disabled>disabled</Button>
                        <Button variant="danger" disabled>disabled</Button>
                    </div>
                </div>
            </Section>

            <Section id="fields" title="Fields">
                <div className="grid max-w-3xl gap-5 md:grid-cols-2">
                    <Field label="App name" hint="Lowercase letters, numbers and hyphens">
                        <Input placeholder="my-app" />
                    </Field>
                    <Field label="Node" error="That name is already taken">
                        <Input defaultValue="jellyfin" />
                    </Field>
                    <Field label="Disabled">
                        <Input defaultValue="Pi Primary" disabled />
                    </Field>
                    <Field label="Description" hint="Optional">
                        <Textarea placeholder="What does this app do?" />
                    </Field>
                    <Field label="Textarea with error" error="Required">
                        <Textarea />
                    </Field>
                </div>
            </Section>

            <Section id="checkbox" title="Checkbox and Kbd">
                <div className="flex flex-wrap items-center gap-6">
                    <label className="flex items-center gap-2 text-sm"><Checkbox aria-label="Unchecked" />Unchecked</label>
                    <label className="flex items-center gap-2 text-sm"><Checkbox aria-label="Checked" defaultChecked />Checked</label>
                    <label className="flex items-center gap-2 text-sm"><Checkbox aria-label="Disabled" disabled />Disabled</label>
                    <span className="flex items-center gap-1 text-sm">Search <Kbd>Cmd</Kbd><Kbd>K</Kbd></span>
                </div>
            </Section>

            <Section id="switch" title="Switch">
                <div className="flex flex-wrap items-center gap-6">
                    <label className="flex items-center gap-2 text-sm">
                        <Switch aria-label="Notifications" checked={notify} onCheckedChange={setNotify} />
                        <span data-testid="switch-state">{notify ? 'On' : 'Off'}</span>
                    </label>
                    <label className="flex items-center gap-2 text-sm"><Switch aria-label="Default on" defaultChecked />On</label>
                    <label className="flex items-center gap-2 text-sm"><Switch aria-label="Disabled off" disabled />Disabled</label>
                    <label className="flex items-center gap-2 text-sm"><Switch aria-label="Disabled on" disabled defaultChecked />Disabled on</label>
                </div>
            </Section>

            <Section id="tabs" title="Tabs">
                <Tabs defaultValue="Overview">
                    <TabsList aria-label="App sections">
                        {APP_TABS.map((tab) => (
                            <TabsTrigger key={tab} value={tab}>{tab}</TabsTrigger>
                        ))}
                    </TabsList>
                    {APP_TABS.map((tab) => (
                        <TabsContent key={tab} value={tab}>
                            <p className="text-sm text-muted-foreground">{tab} content</p>
                        </TabsContent>
                    ))}
                </Tabs>
            </Section>

            <Section id="radio" title="Radio cards">
                <RadioGroup value={exposure} onValueChange={setExposure} aria-label="Exposure" className="max-w-md">
                    <RadioCard value="lan" title="Private, LAN only" description="No public address" />
                    <RadioCard value="custom" title="Custom domain" description="Stable address through Cloudflare" />
                    <RadioCard value="quick" title="Quick tunnel" description="Temporary trycloudflare.com address" />
                    <RadioCard value="off" title="Disabled option" description="Not available" disabled />
                </RadioGroup>
                <p className="mt-3 text-xs text-muted-foreground" data-testid="radio-state">Selected: {exposure}</p>
            </Section>

            <Section id="segmented" title="Segmented control">
                <div className="flex items-center gap-4">
                    <SegmentedControl aria-label="Fleet view" options={VIEW_OPTIONS} value={view} onValueChange={setView} />
                    <span className="text-sm text-muted-foreground" data-testid="segmented-state">View: {view}</span>
                </div>
            </Section>

            <Section id="select" title="Select">
                <div className="grid max-w-3xl gap-5 md:grid-cols-2">
                    <Field label="Deploy to" hint="2 of 3 nodes online">
                        <Select value={node} onValueChange={setNode}>
                            <SelectTrigger>
                                <SelectValue placeholder="Choose a node" />
                            </SelectTrigger>
                            <SelectContent>
                                {NODE_NAMES.map((name) => (
                                    <SelectItem key={name} value={name}>{name}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </Field>
                    <Field label="Timezone" error="Pick a timezone">
                        <Select>
                            <SelectTrigger>
                                <SelectValue placeholder="Choose a timezone" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="Europe/London">Europe/London</SelectItem>
                                <SelectItem value="UTC">UTC</SelectItem>
                            </SelectContent>
                        </Select>
                    </Field>
                    <Field label="Disabled">
                        <Select disabled>
                            <SelectTrigger>
                                <SelectValue placeholder="Unavailable" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="x">x</SelectItem>
                            </SelectContent>
                        </Select>
                    </Field>
                </div>
                <p className="mt-3 text-xs text-muted-foreground" data-testid="select-state">Node: {node}</p>
            </Section>

            <Section id="status" title="Status pills and dots">
                <div className="space-y-4">
                    <div className="flex flex-wrap items-center gap-3" data-row="pills">
                        {STATUS_KINDS.map((kind) => (
                            <StatusPill key={kind} kind={kind}>{kind}</StatusPill>
                        ))}
                    </div>
                    <div className="flex flex-wrap items-center gap-3" data-row="pills-sm">
                        {STATUS_KINDS.map((kind) => (
                            <StatusPill key={kind} kind={kind} size="sm">{kind}</StatusPill>
                        ))}
                    </div>
                    <div className="flex flex-wrap items-center gap-3" data-row="dots">
                        {STATUS_KINDS.map((kind) => (
                            <span key={kind} className="flex items-center gap-2 text-sm"><StatusDot kind={kind} />{kind}</span>
                        ))}
                    </div>
                    <div className="space-y-2" data-row="mapped">
                        <div className="flex flex-wrap gap-3">
                            {APP_STATUSES.map((s) => (
                                <StatusPill key={s} kind={appStatusMeta(s).kind}>{appStatusMeta(s).label}</StatusPill>
                            ))}
                        </div>
                        <div className="flex flex-wrap gap-3">
                            {NODE_STATUSES.map((s) => (
                                <StatusPill key={s} kind={nodeStatusMeta(s).kind}>{nodeStatusMeta(s).label}</StatusPill>
                            ))}
                            {JOB_STATUSES.map((s) => (
                                <StatusPill key={s} kind={jobStatusMeta(s).kind}>{jobStatusMeta(s).label}</StatusPill>
                            ))}
                        </div>
                    </div>
                </div>
            </Section>

            <Section id="progress" title="Progress bars">
                <div className="grid max-w-xl gap-4">
                    {PROGRESS_SAMPLES.map(({ value, tone }) => (
                        <div key={value} className="flex items-center gap-3 text-xs">
                            <span className="w-10 text-muted-foreground">{value}%</span>
                            <ProgressBar value={value} tone={tone} aria-label={`Sample ${value} percent`} />
                        </div>
                    ))}
                </div>
            </Section>

            <Section id="charts" title="Sparkline and area chart">
                <div className="space-y-6">
                    <div className="flex flex-wrap items-center gap-6" data-row="sparklines">
                        {STATUS_KINDS.map((kind) => (
                            <Sparkline key={kind} data={CPU_SERIES} tone={kind} label={`CPU trend ${kind}`} />
                        ))}
                        <Sparkline data={[5]} label="Too short to draw" />
                    </div>
                    <div className="grid gap-4 md:grid-cols-2" data-row="areas">
                        <Card className="p-4">
                            <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">CPU</p>
                            <p className="text-2xl font-semibold">32%</p>
                            <AreaChart data={CPU_SERIES} tone="info" label="CPU over the last 6 hours" />
                        </Card>
                        <Card className="p-4">
                            <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Memory</p>
                            <p className="text-2xl font-semibold">84%</p>
                            <AreaChart data={MEMORY_SERIES} tone="warn" label="Memory over the last 6 hours" />
                        </Card>
                    </div>
                </div>
            </Section>

            <Section id="tiles" title="App tiles">
                <div className="flex flex-wrap items-center gap-4" data-row="tiles">
                    {APP_NAMES.map((name) => (
                        <div key={name} className="flex items-center gap-2 text-sm"><AppTile name={name} />{name}</div>
                    ))}
                    <AppTile name="nextcloud" size="sm" />
                    <AppTile name="nextcloud" size="lg" />
                </div>
            </Section>

            <Section id="table" title="Table and card">
                <Card className="overflow-hidden">
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>Application</TableHead>
                                <TableHead>Status</TableHead>
                                <TableHead>Node</TableHead>
                                <TableHead>CPU</TableHead>
                                <TableHead>Address</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {APP_NAMES.slice(0, 3).map((name, index) => (
                                <TableRow key={name}>
                                    <TableCell>
                                        <span className="flex items-center gap-3"><AppTile name={name} size="sm" /><span className="font-medium">{name}</span></span>
                                    </TableCell>
                                    <TableCell>
                                        <StatusPill kind={index === 2 ? 'idle' : 'ok'}>{index === 2 ? 'Stopped' : 'Running'}</StatusPill>
                                    </TableCell>
                                    <TableCell className="text-muted-foreground">{NODE_NAMES[index]}</TableCell>
                                    <TableCell><Sparkline data={CPU_SERIES} width={60} height={20} /></TableCell>
                                    <TableCell className="whitespace-nowrap font-mono text-xs text-muted-foreground">{name}.example.com</TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </Card>
            </Section>

            <Section id="code" title="Code, diff and terminal">
                <div className="space-y-5">
                    <CodeBlock code={COMPOSE_BEFORE.trimEnd()} aria-label="compose.yml" />
                    <div data-row="diff"><DiffBlock before={COMPOSE_BEFORE} after={COMPOSE_AFTER} aria-label="Changes to compose.yml" /></div>
                    <div data-row="long"><CodeBlock code={LONG_LINE} aria-label="Long line" /></div>
                    <Terminal lines={LOG_LINES} aria-label="Container logs" className="h-56" />
                </div>
            </Section>

            <Section id="feedback" title="Feedback: empty, error, loading">
                <div className="space-y-5">
                    <div data-row="empty">
                        <EmptyState
                            icon={<Plus className="h-5 w-5" />}
                            title="No apps yet"
                            description="Pick a template and Selfhostly writes, checks and starts it for you."
                            action={<Button>New app</Button>}
                        />
                    </div>
                    <div data-row="error">
                        <ErrorState
                            title="Could not load logs"
                            error={new Error('{"error":"Failed to get app logs","details":"container operation failed: get app logs"}')}
                            onRetry={() => setRetries((n) => n + 1)}
                        />
                        <p className="mt-2 text-xs text-muted-foreground" data-testid="retry-count">Retries: {retries}</p>
                    </div>
                    <div data-row="error-network">
                        <ErrorState title="Could not load nodes" error={{ message: 'Failed to fetch' }} />
                    </div>
                    <div data-row="skeleton" className="space-y-3">
                        <div className="flex items-center gap-3">
                            <Skeleton className="h-10 w-10 rounded-[9px]" />
                            <div className="flex-1 space-y-2"><Skeleton className="h-4 w-1/3" /><Skeleton className="h-3 w-2/3" /></div>
                        </div>
                        <CardSkeleton />
                    </div>
                </div>
            </Section>

            <Section id="fleet-states" title="Fleet: loading and first run">
                <div className="space-y-8">
                    <div data-row="fleet-loading"><FleetLoading /></div>
                    <div data-row="fleet-empty"><FleetEmpty /></div>
                </div>
            </Section>

            <Section id="error-boundary" title="Error boundary">
                <BoundaryDemo />
            </Section>

            <Section id="jobs" title="Job progress">
                <div data-row="jobs" className="grid gap-4 sm:grid-cols-2">
                    {(['pending', 'running', 'completed', 'failed'] as const).map((status) => (
                        <Card key={status} className="p-4">
                            <JobProgress
                                job={{
                                    id: status,
                                    type: 'app_update',
                                    app_id: 'a1',
                                    status,
                                    progress: status === 'running' ? 60 : status === 'completed' ? 100 : 0,
                                    progress_message: status === 'running' ? 'Pulling images' : undefined,
                                    error_message: status === 'failed' ? 'port is already allocated: 0.0.0.0:8082' : undefined,
                                    started_at: new Date(Date.now() - 90_000).toISOString(),
                                    completed_at: status === 'completed' || status === 'failed' ? new Date(Date.now() - 30_000).toISOString() : undefined,
                                    created_at: new Date(Date.now() - 100_000).toISOString(),
                                } as never}
                                compact={status === 'pending'}
                            />
                        </Card>
                    ))}
                </div>
            </Section>

            <Section id="overlays" title="Dialogs, sheets, tooltip, menu, toasts">
                <div className="space-y-5">
                    <div className="flex flex-wrap items-center gap-3" data-row="dialogs">
                        <Button variant="outline" onClick={() => setConfirmOpen('restart')}>Restart dialog</Button>
                        <Button variant="danger" onClick={() => setConfirmOpen('delete')}><Trash2 className="h-4 w-4" />Delete dialog</Button>
                        <Sheet>
                            <SheetTrigger asChild><Button variant="outline">Bottom sheet</Button></SheetTrigger>
                            <SheetContent side="bottom">
                                <SheetTitle>More</SheetTitle>
                                <SheetDescription>Settings, notifications and theme</SheetDescription>
                                <Button variant="outline">Settings</Button>
                            </SheetContent>
                        </Sheet>
                        <Sheet>
                            <SheetTrigger asChild><Button variant="outline">Side drawer</Button></SheetTrigger>
                            <SheetContent side="right">
                                <SheetTitle>nextcloud routes</SheetTitle>
                                <SheetDescription>cloud.example.com</SheetDescription>
                            </SheetContent>
                        </Sheet>
                        <TooltipProvider>
                            <TooltipTrigger asChild><Button variant="outline" aria-label="Restart app"><RotateCw className="h-4 w-4" /></Button></TooltipTrigger>
                            <TooltipContent>Restart app</TooltipContent>
                        </TooltipProvider>
                        <DropdownMenu>
                            <DropdownMenuTrigger asChild><Button variant="outline">Actions</Button></DropdownMenuTrigger>
                            <DropdownMenuContent>
                                <DropdownMenuItem onSelect={() => log('menu:restart')}>Restart</DropdownMenuItem>
                                <DropdownMenuItem onSelect={() => log('menu:stop')}>Stop</DropdownMenuItem>
                                <DropdownMenuItem onSelect={() => log('menu:delete')}>Delete</DropdownMenuItem>
                            </DropdownMenuContent>
                        </DropdownMenu>
                    </div>
                    <div className="flex flex-wrap items-center gap-3" data-row="toasts">
                        <Button variant="outline" onClick={() => toast.success('nextcloud restarted', 'Back online in 9 seconds')}>Success toast</Button>
                        <Button variant="outline" onClick={() => toast.error('Could not start vaultwarden', 'Port 8082 is already in use', { label: 'Fix port', onClick: () => log('toast:fix-port') })}>Error toast</Button>
                        <Button variant="outline" onClick={() => toast.warning('Memory is at 84%')}>Warning toast</Button>
                        <Button variant="outline" onClick={() => toast.info('Syncing tunnel routes')}>Info toast</Button>
                    </div>
                    <div
                        data-row="row-click"
                        className="flex items-center justify-between rounded-lg border p-3 text-sm"
                        onClick={() => log('row:clicked')}
                    >
                        <span>Clickable row (menu clicks must not bubble here)</span>
                        <DropdownMenu>
                            <DropdownMenuTrigger asChild><Button variant="ghost" size="icon" aria-label="Row actions"><Plus className="h-4 w-4" /></Button></DropdownMenuTrigger>
                            <DropdownMenuContent>
                                <DropdownMenuItem onSelect={() => log('rowmenu:edit')}>Edit</DropdownMenuItem>
                            </DropdownMenuContent>
                        </DropdownMenu>
                    </div>
                    <p className="text-xs text-muted-foreground" data-testid="event-log">{eventLog.join(',') || 'no events'}</p>
                </div>
                <ConfirmationDialog
                    open={confirmOpen === 'restart'}
                    onOpenChange={(open) => !open && setConfirmOpen('none')}
                    title="Restart nextcloud?"
                    description="It will be unavailable for about 10 seconds."
                    confirmText="Restart"
                    onConfirm={() => { log('dialog:restart'); setConfirmOpen('none') }}
                />
                <ConfirmationDialog
                    open={confirmOpen === 'delete'}
                    onOpenChange={(open) => !open && setConfirmOpen('none')}
                    title="Delete vaultwarden?"
                    description="This cannot be undone."
                    confirmText="Delete app"
                    variant="destructive"
                    confirmationText="vaultwarden"
                    onConfirm={() => { log('dialog:delete'); setConfirmOpen('none') }}
                />
            </Section>
        </div>
    )
}
