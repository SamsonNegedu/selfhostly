import { ChevronDown, Server } from 'lucide-react'
import { Button } from '../ui/Button'
import {
    DropdownMenu,
    DropdownMenuCheckboxItem,
    DropdownMenuContent,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '../ui/DropdownMenu'
import { StatusPill } from '../ui/StatusPill'
import { useNodeContext } from '@/shared/contexts/NodeContext'
import { nodeStatusMeta } from '@/shared/lib/status'
import { useNodes } from '@/shared/services/api'

const MAX_LABEL_NAMES = 1

// Chooses which nodes the whole app is showing. It lives in the topbar because it scopes every screen.
function ScopeSwitcher() {
    const { data: nodes = [] } = useNodes()
    const { selectedNodeIds, setSelectedNodeIds } = useNodeContext()

    if (nodes.length === 0) return null

    const allSelected = nodes.length > 0 && nodes.every((node) => selectedNodeIds.includes(node.id))
    const selectedNodes = nodes.filter((node) => selectedNodeIds.includes(node.id))
    const label = allSelected
        ? 'All nodes'
        : selectedNodes.length <= MAX_LABEL_NAMES
          ? (selectedNodes[0]?.name ?? 'No node')
          : `${selectedNodes.length} nodes`

    const toggleNode = (nodeId: string, checked: boolean) => {
        const next = checked ? [...selectedNodeIds, nodeId] : selectedNodeIds.filter((id) => id !== nodeId)
        // Keep at least one node selected so a screen is never empty by accident.
        if (next.length > 0) setSelectedNodeIds(next)
    }

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button variant="outline" aria-label={`Showing ${label}. Change nodes`}>
                    <Server className="h-4 w-4" />
                    <span className="max-w-[140px] truncate max-sm:sr-only">{label}</span>
                    <ChevronDown className="h-4 w-4 text-muted-foreground" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-64">
                <DropdownMenuLabel>Show apps from</DropdownMenuLabel>
                <DropdownMenuCheckboxItem
                    checked={allSelected}
                    onSelect={(event) => event.preventDefault()}
                    onCheckedChange={() => setSelectedNodeIds(nodes.map((node) => node.id))}
                >
                    All nodes
                </DropdownMenuCheckboxItem>
                <DropdownMenuSeparator />
                {nodes.map((node) => {
                    const meta = nodeStatusMeta(node.status)
                    return (
                        <DropdownMenuCheckboxItem
                            key={node.id}
                            checked={selectedNodeIds.includes(node.id)}
                            onSelect={(event) => event.preventDefault()}
                            onCheckedChange={(checked) => toggleNode(node.id, checked)}
                        >
                            <span className="flex w-full items-center justify-between gap-2">
                                <span className="truncate">{node.name}</span>
                                {node.status !== 'online' && (
                                    <StatusPill kind={meta.kind} size="sm">
                                        {meta.label}
                                    </StatusPill>
                                )}
                            </span>
                        </DropdownMenuCheckboxItem>
                    )
                })}
            </DropdownMenuContent>
        </DropdownMenu>
    )
}

export default ScopeSwitcher
