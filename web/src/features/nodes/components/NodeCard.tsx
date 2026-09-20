import { Globe, MoreHorizontal, Trash2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card } from '@/shared/components/ui/Card'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/shared/components/ui/DropdownMenu'
import { ProgressBar } from '@/shared/components/ui/ProgressBar'
import { StatusDot, StatusPill } from '@/shared/components/ui/StatusPill'
import { resourceTone, type Resource } from '@/shared/lib/thresholds'
import { formatAgo } from '@/shared/lib/attention'
import { formatPercent } from '@/shared/lib/format'
import { nodeStatusMeta } from '@/shared/lib/status'
import { cn } from '@/shared/lib/utils'
import type { NodeMetrics } from '@/features/dashboard/hooks/useFleetMetrics'
import type { Node } from '@/shared/types/api'

interface NodeCardProps {
    node: Node
    isCurrent: boolean
    appCount: number
    metrics?: NodeMetrics
    onRemove: (node: Node) => void
}

// When the node was last checked and how the checks are going, in one line.
function checkSummary(node: Node): string {
    if (!node.last_health_check) return node.status === 'unknown' ? 'First check running' : 'Not checked yet'
    const parts = [`Checked ${formatAgo(node.last_health_check)}`]
    if (node.status === 'online' && node.last_latency_ms) parts.push(`${node.last_latency_ms} ms`)
    if ((node.consecutive_failures ?? 0) > 0)
        parts.push(
            `${node.consecutive_failures} failed ${node.consecutive_failures === 1 ? 'check' : 'checks'} in a row`,
        )
    return parts.join(' · ')
}

const TONE_TEXT = { ok: '', warn: 'text-status-warn-fg', err: 'text-status-err-fg', info: '', idle: '' } as const

function Meter({ label, kind, percent }: { label: string; kind: Resource; percent: number }) {
    const tone = resourceTone(kind, percent)
    return (
        <div className="flex flex-col gap-1.5">
            <div className="flex items-baseline justify-between text-compact">
                <span className="text-muted-foreground">{label}</span>
                <span className="flex items-center gap-1.5 font-semibold tabular-nums">
                    <StatusDot kind={tone} />
                    <span className={TONE_TEXT[tone]}>{formatPercent(percent)}</span>
                </span>
            </div>
            <ProgressBar value={percent} tone={tone} aria-label={`${label} use`} />
        </div>
    )
}

// One machine in the cluster: how it is doing, what runs on it, and how to reach it.
function NodeCard({ node, isCurrent, appCount, metrics, onRemove }: NodeCardProps) {
    const meta = nodeStatusMeta(node.status)
    const online = node.status === 'online'

    return (
        <Card className={cn('flex flex-col gap-4 p-4', !online && 'border-status-warn/50')} data-node={node.name}>
            <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                        <h2 className="truncate text-heading font-semibold" title={node.name}>
                            {node.name}
                        </h2>
                        <StatusPill kind={meta.kind}>{meta.label}</StatusPill>
                    </div>
                    <p className="mt-1 text-compact text-muted-foreground">
                        {node.is_primary ? 'Primary' : 'Secondary'}
                        {isCurrent && ' · you are connected here'}
                        {' · '}
                        {appCount} {appCount === 1 ? 'app' : 'apps'}
                    </p>
                </div>
                {!isCurrent && (
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon" aria-label={`More actions for ${node.name}`}>
                                <MoreHorizontal className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-44">
                            <DropdownMenuItem
                                onSelect={() => onRemove(node)}
                                className="text-destructive focus:text-destructive"
                            >
                                <Trash2 className="mr-2 h-4 w-4" />
                                Remove node
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                )}
            </div>

            {online && metrics ? (
                <div className="grid gap-3 sm:grid-cols-2">
                    <Meter label="CPU" kind="cpu" percent={metrics.cpuPercent} />
                    <Meter label="Memory" kind="memory" percent={metrics.memoryPercent} />
                </div>
            ) : (
                <div className="rounded-lg bg-status-warn-bg px-3 py-2 text-compact text-status-warn-fg">
                    {online
                        ? 'Waiting for the first reading.'
                        : `${node.name} is not answering. Its apps keep their last known state${node.last_seen ? `. Last seen ${formatAgo(node.last_seen)}.` : '.'}`}
                </div>
            )}

            <div className="flex flex-col gap-1 text-compact text-muted-foreground">
                <p className="flex items-center gap-1.5">
                    <Globe aria-hidden="true" className="h-3.5 w-3.5 shrink-0" />
                    <span className="truncate font-mono" title={node.api_endpoint}>
                        {node.api_endpoint}
                    </span>
                </p>
                <p>{checkSummary(node)}</p>
            </div>
        </Card>
    )
}

export default NodeCard
