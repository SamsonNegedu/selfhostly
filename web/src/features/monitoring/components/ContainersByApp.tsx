import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { MoreHorizontal, RotateCw, Square, Trash2 } from 'lucide-react'
import { AppTile } from '@/shared/components/ui/AppTile'
import { Button } from '@/shared/components/ui/Button'
import { Card } from '@/shared/components/ui/Card'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/shared/components/ui/DropdownMenu'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/components/ui/Table'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { formatBytes, formatPercent } from '@/shared/lib/format'
import { appHref } from '@/shared/lib/routes'
import { useDeleteContainer, useRestartContainer, useStopContainer } from '@/shared/services/api'
import type { App, ContainerInfo } from '@/shared/types/api'

type Action = 'restart' | 'stop' | 'delete'

interface Group {
    key: string
    name: string
    app?: App
    managed: boolean
    containers: ContainerInfo[]
}

interface ContainersByAppProps {
    containers: ContainerInfo[]
    apps: App[]
    nodeName: (id: string) => string
}

const STATE_KIND = { running: 'ok', stopped: 'idle', paused: 'warn' } as const

function groupContainers(containers: ContainerInfo[], apps: App[]): Group[] {
    const groups = new Map<string, Group>()
    for (const container of containers) {
        const managed = container.is_managed
        const key = managed ? `${container.node_id}:${container.app_name}` : 'external'
        const existing = groups.get(key)
        if (existing) existing.containers.push(container)
        else {
            groups.set(key, {
                key,
                name: managed ? container.app_name : 'Not managed by Selfhostly',
                app: managed
                    ? apps.find((app) => app.name === container.app_name && app.node_id === container.node_id)
                    : undefined,
                managed,
                containers: [container],
            })
        }
    }
    return [...groups.values()].sort((a, b) =>
        a.managed === b.managed ? a.name.localeCompare(b.name) : a.managed ? -1 : 1,
    )
}

// Every container that is running on the nodes, grouped under the app it belongs to. Containers that Selfhostly
// did not start are kept apart, because acting on them can break something else.
function ContainersByApp({ containers, apps, nodeName }: ContainersByAppProps) {
    const { toast } = useToast()
    const restart = useRestartContainer()
    const stop = useStopContainer()
    const remove = useDeleteContainer()
    const [pending, setPending] = useState<{ action: Action; container: ContainerInfo } | null>(null)
    const groups = useMemo(() => groupContainers(containers, apps), [containers, apps])

    const run = async () => {
        if (!pending) return
        const { action, container } = pending
        setPending(null)
        const args = { containerId: container.id, nodeId: container.node_id }
        try {
            if (action === 'restart') await restart.mutateAsync(args)
            else if (action === 'stop') await stop.mutateAsync(args)
            else await remove.mutateAsync(args)
            toast.success(
                action === 'restart'
                    ? 'Container restarted'
                    : action === 'stop'
                      ? 'Container stopped'
                      : 'Container deleted',
                container.name,
            )
        } catch (failure) {
            toast.error(`Could not ${action} the container`, describeError(failure))
        }
    }

    const copy = pending
        ? {
              restart: {
                  title: `${pending.container.state === 'stopped' ? 'Start' : 'Restart'} ${pending.container.name}?`,
                  text: pending.container.state === 'stopped' ? 'Start' : 'Restart',
                  body:
                      pending.container.state === 'stopped'
                          ? 'It starts running again.'
                          : 'It is unavailable for a moment while it restarts.',
              },
              stop: {
                  title: `Stop ${pending.container.name}?`,
                  text: 'Stop',
                  body: 'It stays stopped until you start it again.',
              },
              delete: {
                  title: `Delete ${pending.container.name}?`,
                  text: 'Delete container',
                  body: pending.container.is_managed
                      ? `It belongs to ${pending.container.app_name}. Deleting it can break that app. Stopping the whole app from Fleet is safer.`
                      : 'This removes the container. Volumes may stay behind.',
              },
          }[pending.action]
        : null

    return (
        <div className="flex flex-col gap-4">
            {groups.map((group) => (
                <Card key={group.key} className="overflow-hidden p-0" data-group={group.name}>
                    <div className="flex items-center gap-3 border-b border-border p-4">
                        {group.managed && <AppTile name={group.name} size="sm" />}
                        {group.app ? (
                            <Link
                                to={appHref(group.app)}
                                className="min-h-[44px] content-center text-[15px] font-semibold hover:underline md:min-h-0"
                            >
                                {group.name}
                            </Link>
                        ) : (
                            <h3 className="text-[15px] font-semibold">{group.name}</h3>
                        )}
                        <span className="text-[13px] text-muted-foreground">
                            {group.containers.length} {group.containers.length === 1 ? 'container' : 'containers'}
                        </span>
                    </div>
                    <Table aria-label={`Containers of ${group.name}`} className="max-md:[&_td]:px-3 max-md:[&_th]:px-3">
                        <TableHeader>
                            <TableRow className="hover:bg-transparent">
                                <TableHead>Container</TableHead>
                                <TableHead>State</TableHead>
                                <TableHead className="max-md:hidden">CPU</TableHead>
                                <TableHead className="max-md:hidden">Memory</TableHead>
                                <TableHead className="max-lg:hidden">Node</TableHead>
                                <TableHead className="w-[1%]">
                                    <span className="sr-only">Actions</span>
                                </TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {group.containers.map((container) => (
                                <TableRow key={container.id}>
                                    <TableCell className="max-w-[180px]">
                                        <p className="truncate font-mono text-[13px] font-medium">{container.name}</p>
                                        <p className="text-xs text-muted-foreground md:hidden">
                                            {container.state === 'running'
                                                ? `${formatPercent(container.cpu_percent)} CPU · ${formatBytes(container.memory_usage_bytes)}`
                                                : container.status}
                                        </p>
                                    </TableCell>
                                    <TableCell>
                                        <StatusPill kind={STATE_KIND[container.state] ?? 'idle'}>
                                            {container.state === 'running'
                                                ? 'Running'
                                                : container.state === 'paused'
                                                  ? 'Paused'
                                                  : 'Stopped'}
                                        </StatusPill>
                                    </TableCell>
                                    <TableCell className="tabular-nums max-md:hidden">
                                        {container.state === 'running' ? formatPercent(container.cpu_percent) : '-'}
                                    </TableCell>
                                    <TableCell className="tabular-nums max-md:hidden">
                                        {container.state === 'running'
                                            ? formatBytes(container.memory_usage_bytes)
                                            : '-'}
                                        {container.state === 'running' && container.memory_limit_bytes > 0 && (
                                            <span className="text-muted-foreground">
                                                {' '}
                                                / {formatBytes(container.memory_limit_bytes)}
                                            </span>
                                        )}
                                    </TableCell>
                                    <TableCell className="text-muted-foreground max-lg:hidden">
                                        {nodeName(container.node_id)}
                                    </TableCell>
                                    <TableCell>
                                        <DropdownMenu>
                                            <DropdownMenuTrigger asChild>
                                                <Button
                                                    variant="ghost"
                                                    size="icon"
                                                    aria-label={`Actions for ${container.name}`}
                                                >
                                                    <MoreHorizontal className="h-4 w-4" />
                                                </Button>
                                            </DropdownMenuTrigger>
                                            <DropdownMenuContent align="end" className="w-44">
                                                <DropdownMenuItem
                                                    onSelect={() => setPending({ action: 'restart', container })}
                                                >
                                                    <RotateCw className="mr-2 h-4 w-4" />
                                                    {container.state === 'stopped' ? 'Start' : 'Restart'}
                                                </DropdownMenuItem>
                                                <DropdownMenuItem
                                                    onSelect={() => setPending({ action: 'stop', container })}
                                                    disabled={container.state !== 'running'}
                                                >
                                                    <Square className="mr-2 h-4 w-4" />
                                                    Stop
                                                </DropdownMenuItem>
                                                {container.state === 'stopped' && (
                                                    <DropdownMenuItem
                                                        onSelect={() => setPending({ action: 'delete', container })}
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
                            ))}
                        </TableBody>
                    </Table>
                </Card>
            ))}

            <ConfirmationDialog
                open={pending !== null}
                onOpenChange={(open) => !open && setPending(null)}
                title={copy?.title ?? ''}
                description={copy?.body ?? ''}
                confirmText={copy?.text}
                variant={pending?.action === 'restart' ? 'default' : 'destructive'}
                confirmationText={pending?.action === 'delete' ? pending.container.name : undefined}
                onConfirm={run}
            />
        </div>
    )
}

export default ContainersByApp
