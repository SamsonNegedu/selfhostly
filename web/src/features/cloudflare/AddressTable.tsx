import { Link } from 'react-router-dom'
import { ExternalLink, Route } from 'lucide-react'
import { AppTile } from '@/shared/components/ui/AppTile'
import { Button } from '@/shared/components/ui/Button'
import { Card } from '@/shared/components/ui/Card'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/components/ui/Table'
import { appHref } from '@/shared/lib/routes'
import { hostOf } from '@/shared/lib/format'
import type { AddressRow } from './lib/address-rows'

interface AddressTableProps {
    rows: AddressRow[]
    nodeName: (nodeId: string) => string
    onRoutes: (row: AddressRow) => void
}

// The public addresses, one row per app, with its status and a button to see its routes.
function AddressTable({ rows, nodeName, onRoutes }: AddressTableProps) {
    return (
        <Card className="overflow-hidden p-0">
            <Table aria-label="Addresses" className="max-md:[&_td]:px-3 max-md:[&_th]:px-3">
                <TableHeader>
                    <TableRow className="hover:bg-transparent">
                        <TableHead>App</TableHead>
                        <TableHead>Address</TableHead>
                        <TableHead className="max-md:hidden">Status</TableHead>
                        <TableHead className="max-lg:hidden">Type</TableHead>
                        <TableHead className="max-lg:hidden">Node</TableHead>
                        <TableHead className="w-[1%]">
                            <span className="sr-only">Actions</span>
                        </TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {rows.map((row) => (
                        <TableRow key={row.app.id} data-address={row.app.name}>
                            <TableCell>
                                <div className="flex items-center gap-3">
                                    <AppTile name={row.app.name} size="sm" className="max-md:hidden" />
                                    <div className="flex flex-col items-start">
                                        <Link
                                            to={appHref(row.app, 'access')}
                                            className="inline-flex min-h-[44px] min-w-[44px] items-center font-semibold hover:underline md:min-h-0 md:min-w-0"
                                        >
                                            {row.app.name}
                                        </Link>
                                        <StatusPill kind={row.status.kind} size="sm" className="md:hidden">
                                            {row.status.label}
                                        </StatusPill>
                                        <span className="mb-1 mt-1 text-xs text-muted-foreground lg:hidden">
                                            {row.kind === 'quick' ? 'Quick Tunnel' : 'Custom domain'} ·{' '}
                                            {nodeName(row.app.node_id)}
                                        </span>
                                    </div>
                                </div>
                            </TableCell>
                            <TableCell>
                                <a
                                    href={row.app.public_url}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="inline-flex min-h-[44px] max-w-[140px] items-center gap-1.5 md:max-w-[220px] font-mono text-compact text-status-info-fg hover:underline md:min-h-0"
                                >
                                    <span className="truncate" title={row.app.public_url}>
                                        {hostOf(row.app.public_url ?? '')}
                                    </span>
                                    <ExternalLink className="h-3 w-3 shrink-0" aria-hidden="true" />
                                    <span className="sr-only">(opens in a new tab)</span>
                                </a>
                            </TableCell>
                            <TableCell className="max-md:hidden">
                                <StatusPill kind={row.status.kind}>{row.status.label}</StatusPill>
                            </TableCell>
                            <TableCell className="text-muted-foreground max-lg:hidden">
                                {row.kind === 'quick' ? 'Quick Tunnel' : 'Custom domain'}
                            </TableCell>
                            <TableCell className="text-muted-foreground max-lg:hidden">
                                {nodeName(row.app.node_id)}
                            </TableCell>
                            <TableCell>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => onRoutes(row)}
                                    aria-label={`Routes for ${row.app.name}`}
                                >
                                    <Route className="h-4 w-4" />
                                    <span className="max-md:hidden">Routes</span>
                                </Button>
                            </TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        </Card>
    )
}

export default AddressTable
