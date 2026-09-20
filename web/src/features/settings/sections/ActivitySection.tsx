import { History } from 'lucide-react'
import { Card } from '@/shared/components/ui/Card'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/components/ui/Table'
import { describeAudit } from '@/shared/lib/audit'
import { formatAgo } from '@/shared/lib/attention'
import { useAuditLog } from '@/shared/services/api'

const HTTP_ERROR = 400
const HTTP_SERVER_ERROR = 500

// Everything that changed something on this server recently: who, what, and whether it worked.
function ActivitySection() {
    const { data: entries, error, refetch } = useAuditLog(50)

    if (entries === undefined) {
        if (error) return <ErrorState title="Could not load the activity log" error={error} onRetry={() => refetch()} />
        return <Skeleton role="status" aria-label="Loading activity" className="h-64 rounded-xl" />
    }

    if (entries.length === 0) {
        return (
            <EmptyState
                icon={<History className="h-5 w-5" />}
                title="Nothing has changed yet"
                description="Starting, stopping, editing or deleting anything shows up here."
                className="py-14"
            />
        )
    }

    return (
        <Card className="overflow-hidden p-0">
            <Table aria-label="Recent changes" className="max-md:[&_td]:px-3 max-md:[&_th]:px-3">
                <TableHeader>
                    <TableRow className="hover:bg-transparent">
                        <TableHead>When</TableHead>
                        <TableHead>What happened</TableHead>
                        <TableHead>Result</TableHead>
                        <TableHead className="max-md:hidden">By</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {entries.map((entry) => (
                        <TableRow key={entry.id}>
                            <TableCell className="whitespace-nowrap text-muted-foreground">
                                {formatAgo(entry.time)}
                            </TableCell>
                            <TableCell className="w-full">
                                {describeAudit(entry) && <p className="font-medium">{describeAudit(entry)}</p>}
                                <p className="font-mono text-detail text-muted-foreground">
                                    <span className="font-semibold">{entry.method}</span>{' '}
                                    <span className="break-all">{entry.path}</span>
                                </p>
                            </TableCell>
                            <TableCell>
                                <StatusPill
                                    kind={
                                        entry.status >= HTTP_SERVER_ERROR
                                            ? 'err'
                                            : entry.status >= HTTP_ERROR
                                              ? 'warn'
                                              : 'ok'
                                    }
                                    size="sm"
                                >
                                    {entry.status}
                                </StatusPill>
                            </TableCell>
                            <TableCell
                                className="max-w-[200px] truncate text-muted-foreground max-md:hidden"
                                title={entry.actor || undefined}
                            >
                                {entry.actor || 'system'}
                            </TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        </Card>
    )
}

export default ActivitySection
