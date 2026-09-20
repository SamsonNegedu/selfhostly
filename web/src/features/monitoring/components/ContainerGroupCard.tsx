import { Link } from 'react-router-dom'
import { AppTile } from '@/shared/components/ui/AppTile'
import { Card } from '@/shared/components/ui/Card'
import { Table, TableBody, TableHead, TableHeader, TableRow } from '@/shared/components/ui/Table'
import { appHref } from '@/shared/lib/routes'
import type { ContainerInfo } from '@/shared/types/api'
import type { ContainerAction, ContainerGroup } from '../lib/container-groups'
import ContainerRow from './ContainerRow'

interface ContainerGroupCardProps {
    group: ContainerGroup
    nodeName: (id: string) => string
    onAction: (action: ContainerAction, container: ContainerInfo) => void
}

// The containers of one app (or the ones Selfhostly did not start), as a titled table.
function ContainerGroupCard({ group, nodeName, onAction }: ContainerGroupCardProps) {
    return (
        <Card className="overflow-hidden p-0" data-group={group.name}>
            <div className="flex items-center gap-3 border-b border-border p-4">
                {group.managed && <AppTile name={group.name} size="sm" />}
                {group.app ? (
                    <Link
                        to={appHref(group.app)}
                        className="min-h-[44px] content-center text-title font-semibold hover:underline md:min-h-0"
                    >
                        {group.name}
                    </Link>
                ) : (
                    <h3 className="text-title font-semibold">{group.name}</h3>
                )}
                <span className="text-compact text-muted-foreground">
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
                        <ContainerRow
                            key={container.id}
                            container={container}
                            nodeName={nodeName}
                            onAction={onAction}
                        />
                    ))}
                </TableBody>
            </Table>
        </Card>
    )
}

export default ContainerGroupCard
