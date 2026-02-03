import { useState } from 'react'
import { Badge } from '@/shared/components/ui/Badge'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { Server, Trash2, Clock, Globe } from 'lucide-react'
import { useDeleteNode } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'
import { DataTable, ColumnDef, RowAction } from '@/shared/components/ui/DataTable'
import type { Node } from '@/shared/types/api'

interface NodesListViewProps {
    nodes: Node[]
}

// Status badge component
const getStatusBadge = (status: string) => {
    switch (status) {
        case 'online':
            return (
                <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md text-xs font-semibold text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20">
                    <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                    online
                </div>
            )
        case 'offline':
            return (
                <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md text-xs font-semibold text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-800">
                    <div className="w-2 h-2 bg-gray-400 rounded-full" />
                    offline
                </div>
            )
        case 'unreachable':
            return (
                <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md text-xs font-semibold text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20">
                    <div className="w-2 h-2 bg-amber-500 rounded-full" />
                    unreachable
                </div>
            )
        default:
            return (
                <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md text-xs font-semibold text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-800">
                    {status}
                </div>
            )
    }
}

function NodesListView({ nodes }: NodesListViewProps) {
    const deleteNode = useDeleteNode()
    const { toast } = useToast()

    const [nodeToDelete, setNodeToDelete] = useState<{ id: string; name: string } | null>(null)

    // Define columns
    const columns: ColumnDef<Node>[] = [
        {
            key: 'status',
            label: 'Status',
            width: 'w-28',
            render: (node) => getStatusBadge(node.status)
        },
        {
            key: 'name',
            label: 'Name',
            width: 'flex-1 min-w-0',
            render: (node) => (
                <div className="flex items-center gap-2">
                    <Server className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                    <span className="font-semibold text-sm truncate">{node.name}</span>
                </div>
            )
        },
        {
            key: 'role',
            label: 'Role',
            width: 'w-28',
            render: (node) => (
                node.is_primary ? (
                    <Badge variant="default" className="bg-blue-600">
                        Primary
                    </Badge>
                ) : (
                    <Badge variant="secondary">
                        Secondary
                    </Badge>
                )
            )
        },
        {
            key: 'endpoint',
            label: 'API Endpoint',
            width: 'w-64',
            render: (node) => (
                <div className="flex items-center gap-2">
                    <Globe className="h-3.5 w-3.5 text-muted-foreground flex-shrink-0" />
                    <span className="text-xs text-muted-foreground font-mono truncate">
                        {node.api_endpoint}
                    </span>
                </div>
            )
        },
        {
            key: 'last_seen',
            label: 'Last Seen',
            width: 'w-40',
            render: (node) => (
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Clock className="h-3 w-3 flex-shrink-0" />
                    <span>
                        {node.last_seen
                            ? new Date(node.last_seen).toLocaleDateString()
                            : 'Never'}
                    </span>
                </div>
            )
        }
    ]

    // Define actions
    const actions: RowAction<Node>[] = [
        {
            label: 'Remove Node',
            icon: <Trash2 className="h-4 w-4" />,
            onClick: (node) => setNodeToDelete({ id: node.id, name: node.name }),
            variant: 'destructive',
            show: (node) => !node.is_primary // Only show for secondary nodes
        }
    ]

    // Handle node deletion confirmation
    const confirmDelete = () => {
        if (nodeToDelete) {
            const nodeName = nodeToDelete.name
            const nodeId = nodeToDelete.id

            toast.info('Deleting node', `Deleting "${nodeName}"...`)

            deleteNode.mutate(nodeId, {
                onSuccess: () => {
                    toast.success('Node deleted', `"${nodeName}" has been deleted successfully`)
                },
                onError: (error) => {
                    toast.error('Failed to delete node', error.message)
                }
            })

            setNodeToDelete(null)
        }
    }

    // Empty state
    const emptyState = (
        <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-muted mb-4">
                <Server className="w-8 h-8 text-muted-foreground" />
            </div>
            <h3 className="text-lg font-semibold mb-2">No nodes found</h3>
            <p className="text-muted-foreground max-w-sm mx-auto">
                Register your first secondary node to enable multi-node deployment
            </p>
        </div>
    )

    return (
        <>
            {/* Data Table */}
            <DataTable
                data={nodes}
                columns={columns}
                getRowKey={(node) => node.id}
                actions={actions}
                emptyState={emptyState}
            />

            {/* Confirmation Dialog */}
            <ConfirmationDialog
                open={!!nodeToDelete}
                onOpenChange={(open: boolean) => !open && setNodeToDelete(null)}
                title="Delete Node"
                description={`Are you sure you want to delete "${nodeToDelete?.name}"? This action cannot be undone.`}
                confirmText="Delete"
                cancelText="Cancel"
                onConfirm={confirmDelete}
                isLoading={deleteNode.isPending}
                variant="destructive"
            />
        </>
    )
}

export default NodesListView
