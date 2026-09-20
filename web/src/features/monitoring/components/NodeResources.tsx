import { Card } from '@/shared/components/ui/Card'
import { AreaChart } from '@/shared/components/ui/Chart'
import { ProgressBar } from '@/shared/components/ui/ProgressBar'
import { formatBytes, formatPercent } from '@/shared/lib/format'
import { resourceTone, TONE_WORD, type Resource as ResourceKind } from '@/shared/lib/thresholds'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import type { SystemStats } from '@/shared/types/api'
import type { NodeHistory } from '../hooks/useStatsHistory'

function Resource({
    label,
    kind,
    percent,
    detail,
    history,
}: {
    label: string
    kind: ResourceKind
    percent: number
    detail: string
    history: number[]
}) {
    const tone = resourceTone(kind, percent)
    return (
        <div className="flex flex-col gap-2">
            <div className="flex items-baseline justify-between gap-2">
                <span className="text-[13px] text-muted-foreground">{label}</span>
                <span className="flex items-center gap-2">
                    <StatusPill kind={tone} size="sm">
                        {TONE_WORD[tone]}
                    </StatusPill>
                    <span className="text-xl font-semibold tabular-nums">{formatPercent(percent)}</span>
                </span>
            </div>
            <ProgressBar value={percent} tone={tone} aria-label={`${label} use`} />
            <p className="text-[12.5px] text-muted-foreground">{detail}</p>
            {history.length < 2 && (
                <p className="h-14 text-[12.5px] text-muted-foreground">
                    Collecting readings. The trend appears in a few seconds.
                </p>
            )}
            {history.length >= 2 && (
                <AreaChart data={history} tone={tone} height={56} label={`${label} since you opened this page`} />
            )}
        </div>
    )
}

// One machine: how much CPU, memory and disk it is using, and how that has moved while this page was open.
function NodeResources({ stats, history }: { stats: SystemStats; history?: NodeHistory }) {
    return (
        <Card className="flex flex-col gap-4 p-4" data-node-stats={stats.node_name}>
            <div className="flex flex-wrap items-baseline justify-between gap-2">
                <h2 className="text-[16px] font-semibold">{stats.node_name}</h2>
                <p className="text-[13px] text-muted-foreground">
                    {stats.docker.running} running of {stats.docker.total_containers} containers
                    {stats.docker.version ? ` · Docker ${stats.docker.version}` : ''}
                </p>
            </div>
            <div className="grid gap-5 sm:grid-cols-3">
                <Resource
                    label="CPU"
                    kind="cpu"
                    percent={stats.cpu.usage_percent}
                    detail={`${stats.cpu.cores} ${stats.cpu.cores === 1 ? 'core' : 'cores'}`}
                    history={history?.cpu ?? []}
                />
                <Resource
                    label="Memory"
                    kind="memory"
                    percent={stats.memory.usage_percent}
                    detail={`${formatBytes(stats.memory.used_bytes)} of ${formatBytes(stats.memory.total_bytes)}`}
                    history={history?.memory ?? []}
                />
                <Resource
                    label="Disk"
                    kind="disk"
                    percent={stats.disk.usage_percent}
                    detail={`${formatBytes(stats.disk.used_bytes)} of ${formatBytes(stats.disk.total_bytes)}`}
                    history={history?.disk ?? []}
                />
            </div>
        </Card>
    )
}

export default NodeResources
