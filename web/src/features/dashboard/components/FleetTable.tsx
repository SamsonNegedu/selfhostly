import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowDown, ArrowUp, ExternalLink, Loader2, MoreHorizontal, Play, RefreshCw, Square, Trash2 } from 'lucide-react'
import { AppTile } from '@/shared/components/ui/AppTile'
import { Button } from '@/shared/components/ui/Button'
import { Card } from '@/shared/components/ui/Card'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/shared/components/ui/DropdownMenu'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/components/ui/Table'
import { formatBytes, formatPercent } from '@/shared/lib/format'
import { appHref } from '@/shared/lib/routes'
import { appStatusMeta } from '@/shared/lib/status'
import { cn } from '@/shared/lib/utils'
import type { AppMetrics } from '../hooks/useFleetMetrics'
import type { useFleetActions } from '../hooks/useFleetActions'
import type { FleetApp } from '../lib/fleet'

type SortKey = 'name' | 'status' | 'node' | 'cpu' | 'memory'
type SortDir = 'asc' | 'desc'

interface FleetTableProps {
    apps: FleetApp[]
    metricsFor: (nodeId: string, appName: string) => AppMetrics | undefined
    actions: ReturnType<typeof useFleetActions>
}

const COLUMNS: { key: SortKey; label: string; className?: string }[] = [
    { key: 'name', label: 'App' },
    { key: 'status', label: 'Status' },
    { key: 'node', label: 'Node', className: 'max-md:hidden' },
    { key: 'cpu', label: 'CPU', className: 'max-lg:hidden' },
    { key: 'memory', label: 'Memory', className: 'max-lg:hidden' },
]

const hostOf = (url: string) => {
    try {
        return new URL(url).host
    } catch {
        return url
    }
}

function sortValue(item: FleetApp, key: SortKey, metrics?: AppMetrics): string | number {
    switch (key) {
        case 'name':
            return item.app.name.toLowerCase()
        case 'status':
            return item.unreachable ? 'unreachable' : item.app.status
        case 'node':
            return (item.node?.name ?? '').toLowerCase()
        case 'cpu':
            return metrics?.cpuPercent ?? -1
        case 'memory':
            return metrics?.memoryBytes ?? -1
    }
}

// Every app as one row, for scanning and sorting a long list. Columns that need room drop away on narrow screens.
function FleetTable({ apps, metricsFor, actions }: FleetTableProps) {
    const [sort, setSort] = useState<{ key: SortKey; dir: SortDir }>({ key: 'name', dir: 'asc' })

    const rows = useMemo(() => {
        const withMetrics = apps.map((item) => ({ item, metrics: metricsFor(item.app.node_id, item.app.name) }))
        const direction = sort.dir === 'asc' ? 1 : -1
        return withMetrics.sort((a, b) => {
            const left = sortValue(a.item, sort.key, a.metrics)
            const right = sortValue(b.item, sort.key, b.metrics)
            if (left === right) return 0
            return (left < right ? -1 : 1) * direction
        })
    }, [apps, metricsFor, sort])

    const toggleSort = (key: SortKey) =>
        setSort((current) => (current.key === key ? { key, dir: current.dir === 'asc' ? 'desc' : 'asc' } : { key, dir: 'asc' }))

    return (
        <Card className="overflow-hidden p-0">
            <Table aria-label="Apps" className="max-md:[&_td]:px-3 max-md:[&_th]:px-3">
                <TableHeader>
                    <TableRow className="hover:bg-transparent">
                        {COLUMNS.map((column) => {
                            const active = sort.key === column.key
                            return (
                                <TableHead
                                    key={column.key}
                                    className={column.className}
                                    aria-sort={active ? (sort.dir === 'asc' ? 'ascending' : 'descending') : 'none'}
                                >
                                    <button
                                        type="button"
                                        onClick={() => toggleSort(column.key)}
                                        className="inline-flex min-h-[44px] items-center gap-1 uppercase tracking-wider hover:text-foreground md:min-h-0"
                                    >
                                        {column.label}
                                        {active && (sort.dir === 'asc' ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />)}
                                    </button>
                                </TableHead>
                            )
                        })}
                        <TableHead className="max-xl:hidden">Address</TableHead>
                        <TableHead className="w-[1%]">
                            <span className="sr-only">Actions</span>
                        </TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {rows.map(({ item, metrics }) => (
                        <FleetTableRow key={item.app.id} item={item} metrics={metrics} actions={actions} />
                    ))}
                </TableBody>
            </Table>
        </Card>
    )
}

function FleetTableRow({ item, metrics, actions }: { item: FleetApp; metrics?: AppMetrics; actions: FleetTableProps['actions'] }) {
    const { app, unreachable, node } = item
    const meta = unreachable ? ({ kind: 'warn', label: 'Unreachable' } as const) : appStatusMeta(app.status)
    const busy = actions.isBusy(app.id)
    const isRunning = !unreachable && app.status === 'running'
    const canStart = !unreachable && (app.status === 'stopped' || app.status === 'error')
    const inProgress = !unreachable && (app.status === 'updating' || app.status === 'pending')

    return (
        <TableRow data-app={app.name}>
            <TableCell>
                <div className="flex items-center gap-3">
                    <AppTile name={app.name} size="sm" />
                    <div className="flex flex-col">
                        <Link to={appHref(app)} className="min-h-[44px] content-center font-semibold hover:underline md:min-h-0">
                            {app.name}
                        </Link>
                        <span className="text-xs text-muted-foreground md:hidden">{node?.name}</span>
                        {isRunning && metrics && (
                            <span className="text-xs text-muted-foreground lg:hidden">
                                {formatPercent(metrics.cpuPercent)} CPU · {formatBytes(metrics.memoryBytes)}
                            </span>
                        )}
                    </div>
                </div>
            </TableCell>
            <TableCell>
                <StatusPill kind={meta.kind}>{meta.label}</StatusPill>
            </TableCell>
            <TableCell className="text-muted-foreground max-md:hidden">{node?.name ?? 'Unknown'}</TableCell>
            <TableCell className={cn('tabular-nums max-lg:hidden', !isRunning && 'text-muted-foreground')}>
                {isRunning && metrics ? formatPercent(metrics.cpuPercent) : '-'}
            </TableCell>
            <TableCell className={cn('tabular-nums max-lg:hidden', !isRunning && 'text-muted-foreground')}>
                {isRunning && metrics ? formatBytes(metrics.memoryBytes) : '-'}
            </TableCell>
            <TableCell className="font-mono text-[12.5px] text-muted-foreground max-xl:hidden">
                {app.public_url ? hostOf(app.public_url) : 'LAN only'}
            </TableCell>
            <TableCell>
                <div className="flex items-center justify-end gap-1">
                    {canStart && (
                        <Button variant="outline" size="sm" onClick={() => actions.start(app)} disabled={busy} aria-label={`Start ${app.name}`}>
                            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                            <span className="max-md:hidden">Start</span>
                        </Button>
                    )}
                    {isRunning && (
                        <Button variant="outline" size="sm" onClick={() => actions.requestStop(app)} disabled={busy} aria-label={`Stop ${app.name}`}>
                            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Square className="h-4 w-4" />}
                            <span className="max-md:hidden">Stop</span>
                        </Button>
                    )}
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon" aria-label={`More actions for ${app.name}`}>
                                <MoreHorizontal className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-44">
                            <DropdownMenuItem asChild>
                                <Link to={appHref(app)}>Details</Link>
                            </DropdownMenuItem>
                            {isRunning && app.public_url && (
                                <DropdownMenuItem asChild>
                                    <a href={app.public_url} target="_blank" rel="noopener noreferrer">
                                        <ExternalLink className="mr-2 h-4 w-4" />
                                        Open
                                    </a>
                                </DropdownMenuItem>
                            )}
                            {!unreachable && (
                                <DropdownMenuItem onSelect={() => actions.requestUpdate(app)} disabled={busy || inProgress}>
                                    <RefreshCw className="mr-2 h-4 w-4" />
                                    Update
                                </DropdownMenuItem>
                            )}
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                                onSelect={() => actions.requestDelete(app)}
                                disabled={busy}
                                className="text-destructive focus:text-destructive"
                            >
                                <Trash2 className="mr-2 h-4 w-4" />
                                Delete
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </TableCell>
        </TableRow>
    )
}

export default FleetTable
