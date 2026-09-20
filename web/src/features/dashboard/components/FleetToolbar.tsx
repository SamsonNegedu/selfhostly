import { Search } from 'lucide-react'
import { Input } from '@/shared/components/ui/Input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/components/ui/Select'
import { cn } from '@/shared/lib/utils'
import type { FleetFilter } from '../lib/fleet'

export type GroupBy = 'node' | 'none'

interface FleetToolbarProps {
    counts: Record<FleetFilter, number>
    filter: FleetFilter
    onFilterChange: (filter: FleetFilter) => void
    query: string
    onQueryChange: (query: string) => void
    groupBy: GroupBy
    onGroupByChange: (groupBy: GroupBy) => void
    showGroupBy: boolean
}

const CHIPS: { value: FleetFilter; label: string; alwaysShow: boolean }[] = [
    { value: 'all', label: 'All', alwaysShow: true },
    { value: 'running', label: 'Running', alwaysShow: true },
    { value: 'stopped', label: 'Stopped', alwaysShow: true },
    { value: 'updating', label: 'Updating', alwaysShow: false },
    { value: 'failed', label: 'Failed', alwaysShow: true },
    { value: 'unreachable', label: 'Unreachable', alwaysShow: false },
]

// Count chips that double as filters, plus a text filter and the grouping choice.
function FleetToolbar({
    counts,
    filter,
    onFilterChange,
    query,
    onQueryChange,
    groupBy,
    onGroupByChange,
    showGroupBy,
}: FleetToolbarProps) {
    const chips = CHIPS.filter((chip) => chip.alwaysShow || counts[chip.value] > 0 || chip.value === filter)

    return (
        <div className="flex flex-col gap-3 xl:flex-row xl:items-center">
            <div
                role="group"
                aria-label="Filter by status"
                className="flex gap-2 overflow-x-auto scrollbar-hide md:flex-wrap md:overflow-visible xl:flex-1"
            >
                {chips.map((chip) => {
                    const active = filter === chip.value
                    return (
                        <button
                            key={chip.value}
                            type="button"
                            aria-pressed={active}
                            onClick={() => onFilterChange(chip.value)}
                            className={cn(
                                'inline-flex h-[34px] min-h-[44px] shrink-0 items-center gap-2 rounded-full border px-3.5 text-[13px] font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 md:min-h-0',
                                active
                                    ? 'border-primary bg-primary text-primary-foreground'
                                    : 'border-border bg-card text-foreground hover:bg-accent',
                            )}
                        >
                            {chip.label}
                            <span className={cn('text-xs', active ? 'opacity-80' : 'text-muted-foreground')}>
                                {counts[chip.value]}
                            </span>
                        </button>
                    )
                })}
            </div>

            <div className="flex items-center gap-2">
                <div className="relative flex-1 md:w-[220px] md:flex-none">
                    <Search
                        aria-hidden="true"
                        className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground"
                    />
                    <Input
                        aria-label="Filter apps"
                        placeholder="Filter apps"
                        value={query}
                        onChange={(event) => onQueryChange(event.target.value)}
                        className="pl-9"
                    />
                </div>
                {showGroupBy && (
                    <Select value={groupBy} onValueChange={(value) => onGroupByChange(value as GroupBy)}>
                        <SelectTrigger aria-label="Group apps by" className="w-[150px]">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="node">Group by node</SelectItem>
                            <SelectItem value="none">No grouping</SelectItem>
                        </SelectContent>
                    </Select>
                )}
            </div>
        </div>
    )
}

export default FleetToolbar
