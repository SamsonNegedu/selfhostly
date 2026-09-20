import { useEffect, useMemo, useState } from 'react'
import { useIsPhone } from '@/shared/hooks/useMediaQuery'
import { FLEET_GROUP_KEY, FLEET_VIEW_KEY } from '@/shared/lib/preferences'
import type { App, Node } from '@/shared/types/api'
import type { GroupBy } from '../components/FleetToolbar'
import {
    countByFilter,
    groupByNode,
    matchesFilter,
    matchesQuery,
    sortFleetApps,
    toFleetApps,
    type FleetFilter,
} from '../lib/fleet'

export type ViewMode = 'grid' | 'list'

const VIEW_STORAGE_KEY = FLEET_VIEW_KEY
const GROUP_STORAGE_KEY = FLEET_GROUP_KEY

function readStored<T extends string>(key: string, allowed: readonly T[], fallback: T): T {
    const saved = localStorage.getItem(key)
    return allowed.includes(saved as T) ? (saved as T) : fallback
}

// What the Fleet page shows and how: the filter, the search, the view and the grouping (the last two are remembered
// between visits), and the apps that result.
export function useFleetView(apps: App[] | undefined, nodes: Node[]) {
    const [filter, setFilter] = useState<FleetFilter>('all')
    const [query, setQuery] = useState('')
    const phone = useIsPhone()
    const [savedViewMode, setViewMode] = useState<ViewMode>(() =>
        readStored<ViewMode>(VIEW_STORAGE_KEY, ['grid', 'list'], 'grid'),
    )
    const [groupBy, setGroupBy] = useState<GroupBy>(() =>
        readStored<GroupBy>(GROUP_STORAGE_KEY, ['node', 'none'], 'node'),
    )

    // A table does not fit a phone, so phones always get the list of rows. The saved choice comes back on a wider screen.
    const viewMode: ViewMode = phone ? 'grid' : savedViewMode
    useEffect(() => localStorage.setItem(VIEW_STORAGE_KEY, savedViewMode), [savedViewMode])
    useEffect(() => localStorage.setItem(GROUP_STORAGE_KEY, groupBy), [groupBy])

    const fleetApps = useMemo(() => toFleetApps(apps ?? [], nodes), [apps, nodes])
    const counts = useMemo(() => countByFilter(fleetApps), [fleetApps])
    const visible = useMemo(
        () => sortFleetApps(fleetApps.filter((item) => matchesFilter(item, filter) && matchesQuery(item, query))),
        [fleetApps, filter, query],
    )
    const groups = useMemo(
        () => (groupBy === 'node' ? groupByNode(visible, nodes) : [{ key: 'all', name: 'Apps', apps: visible }]),
        [visible, nodes, groupBy],
    )

    const total = fleetApps.length
    const showHeaders = groupBy === 'node' && groups.length > 1
    const isFiltering = filter !== 'all' || query.trim() !== ''

    const clearFilters = () => {
        setFilter('all')
        setQuery('')
    }

    return {
        phone,
        filter,
        setFilter,
        query,
        setQuery,
        savedViewMode,
        setViewMode,
        viewMode,
        groupBy,
        setGroupBy,
        fleetApps,
        total,
        counts,
        visible,
        groups,
        showHeaders,
        isFiltering,
        clearFilters,
    }
}
