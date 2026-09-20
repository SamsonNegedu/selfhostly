import { MoreHorizontal, RotateCw, Square, Trash2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/shared/components/ui/DropdownMenu'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { TableCell, TableRow } from '@/shared/components/ui/Table'
import { formatBytes, formatPercent } from '@/shared/lib/format'
import type { ContainerInfo } from '@/shared/types/api'
import type { ContainerAction } from '../lib/container-groups'

const STATE_KIND = { running: 'ok', stopped: 'idle', paused: 'warn' } as const

interface ContainerRowProps {
    container: ContainerInfo
    nodeName: (id: string) => string
    onAction: (action: ContainerAction, container: ContainerInfo) => void
}

// One container: its state and usage, and a menu to restart, stop or delete it.
function ContainerRow({ container, nodeName, onAction }: ContainerRowProps) {
    return (
        <TableRow>
            <TableCell className="max-w-[180px]">
                <p className="truncate font-mono text-compact font-medium" title={container.name}>
                    {container.name}
                </p>
                <p className="text-xs text-muted-foreground md:hidden">
                    {container.state === 'running'
                        ? `${formatPercent(container.cpu_percent)} CPU · ${formatBytes(container.memory_usage_bytes)}`
                        : container.status}
                </p>
            </TableCell>
            <TableCell>
                <StatusPill kind={STATE_KIND[container.state] ?? 'idle'}>
                    {container.state === 'running' ? 'Running' : container.state === 'paused' ? 'Paused' : 'Stopped'}
                </StatusPill>
            </TableCell>
            <TableCell className="tabular-nums max-md:hidden">
                {container.state === 'running' ? formatPercent(container.cpu_percent) : '-'}
            </TableCell>
            <TableCell className="tabular-nums max-md:hidden">
                {container.state === 'running' ? formatBytes(container.memory_usage_bytes) : '-'}
                {container.state === 'running' && container.memory_limit_bytes > 0 && (
                    <span className="text-muted-foreground"> / {formatBytes(container.memory_limit_bytes)}</span>
                )}
            </TableCell>
            <TableCell className="text-muted-foreground max-lg:hidden">{nodeName(container.node_id)}</TableCell>
            <TableCell>
                <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" aria-label={`Actions for ${container.name}`}>
                            <MoreHorizontal className="h-4 w-4" />
                        </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="w-44">
                        <DropdownMenuItem onSelect={() => onAction('restart', container)}>
                            <RotateCw className="mr-2 h-4 w-4" />
                            {container.state === 'stopped' ? 'Start' : 'Restart'}
                        </DropdownMenuItem>
                        <DropdownMenuItem
                            onSelect={() => onAction('stop', container)}
                            disabled={container.state !== 'running'}
                        >
                            <Square className="mr-2 h-4 w-4" />
                            Stop
                        </DropdownMenuItem>
                        {container.state === 'stopped' && (
                            <DropdownMenuItem
                                onSelect={() => onAction('delete', container)}
                                className="text-destructive focus:text-destructive"
                            >
                                <Trash2 className="mr-2 h-4 w-4" />
                                Delete
                            </DropdownMenuItem>
                        )}
                    </DropdownMenuContent>
                </DropdownMenu>
            </TableCell>
        </TableRow>
    )
}

export default ContainerRow
