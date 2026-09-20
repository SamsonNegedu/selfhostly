import React, { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { LayoutGrid, List, Plus, SearchX } from 'lucide-react'
import { useApps, useNodes, useQueryClient } from '@/shared/services/api'
import { useAppStore } from '@/shared/stores/app-store'
import { buttonClasses, Button } from '@/shared/components/ui/Button'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { SegmentedControl } from '@/shared/components/ui/SegmentedControl'
import { useNodeContext } from '@/shared/contexts/NodeContext'
import { useIsPhone } from '@/shared/hooks/useMediaQuery'
import { FLEET_GROUP_KEY, FLEET_VIEW_KEY } from '@/shared/lib/preferences'
import { ROUTES } from '@/shared/lib/routes'
import type { App } from '@/shared/types/api'
import AttentionStrip from './components/AttentionStrip'
import { FleetEmpty, FleetLoading } from './components/FleetStates'
import FleetTable from './components/FleetTable'
import FleetToolbar, { type GroupBy } from './components/FleetToolbar'
import NodeGroupSection from './components/NodeGroupSection'
import { useFleetActions } from './hooks/useFleetActions'
import { useFleetMetrics } from './hooks/useFleetMetrics'
import {
    countByFilter,
    groupByNode,
    matchesFilter,
    matchesQuery,
    sortFleetApps,
    toFleetApps,
    type FleetFilter,
} from './lib/fleet'

type ViewMode = 'grid' | 'list'

const VIEW_STORAGE_KEY = FLEET_VIEW_KEY
const GROUP_STORAGE_KEY = FLEET_GROUP_KEY

const VIEW_OPTIONS = [
    { value: 'grid', label: 'Cards', icon: <LayoutGrid className="h-4 w-4" /> },
    { value: 'list', label: 'Table', icon: <List className="h-4 w-4" /> },
]

function readStored<T extends string>(key: string, allowed: readonly T[], fallback: T): T {
    const saved = localStorage.getItem(key)
    return allowed.includes(saved as T) ? (saved as T) : fallback
}

function Dashboard() {
    const { selectedNodeIds } = useNodeContext()
    const { data: apps, error, refetch } = useApps(selectedNodeIds)
    const { data: nodes = [] } = useNodes()
    const setApps = useAppStore((state) => state.setApps)
    const queryClient = useQueryClient()
    const metrics = useFleetMetrics(nodes)
    const actions = useFleetActions()

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

    // Keep the shared store in step with the query cache, as the other views expect.
    React.useEffect(() => {
        const unsubscribe = queryClient.getQueryCache().subscribe(() => {
            const appsQuery = queryClient.getQueryCache().findAll({ queryKey: ['apps'] })
            if (appsQuery.length > 0) {
                const appsData = appsQuery[0].state.data as App[]
                if (appsData) setApps(appsData)
            }
        })
        if (apps) setApps(apps)
        return () => unsubscribe()
    }, [apps, setApps, queryClient])

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

    const nodeCount = selectedNodeIds.length
    const total = fleetApps.length
    const showHeaders = groupBy === 'node' && groups.length > 1
    const isFiltering = filter !== 'all' || query.trim() !== ''

    const clearFilters = () => {
        setFilter('all')
        setQuery('')
    }

    // Until there is data, show the error or the skeleton and never the empty state. A query that is paused
    // (offline, or waiting to retry) has no data, no error and is not loading, and must not read as "no apps".
    // A failed background refresh keeps showing the last apps.
    if (apps === undefined) {
        if (error) {
            return <ErrorState title="Could not load your apps" error={error} onRetry={() => refetch()} />
        }
        return <FleetLoading />
    }

    return (
        <div className="flex flex-col gap-5">
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-semibold tracking-tight">Fleet</h1>
                    <p className="text-muted-foreground">
                        {total === 0
                            ? 'No apps yet'
                            : `${total} ${total === 1 ? 'app' : 'apps'} across ${nodeCount} ${nodeCount === 1 ? 'node' : 'nodes'}`}
                    </p>
                </div>
                <div className="flex items-center gap-2">
                    {total > 0 && !phone && (
                        <SegmentedControl
                            aria-label="View"
                            options={VIEW_OPTIONS}
                            value={savedViewMode}
                            onValueChange={(value) => setViewMode(value as ViewMode)}
                        />
                    )}
                    <Link to={ROUTES.newApp} className={buttonClasses({ className: 'max-md:hidden' })}>
                        <Plus className="h-4 w-4" />
                        New app
                    </Link>
                </div>
            </div>

            {total === 0 ? (
                <FleetEmpty />
            ) : (
                <>
                    <AttentionStrip actions={actions} />

                    <FleetToolbar
                        counts={counts}
                        filter={filter}
                        onFilterChange={setFilter}
                        query={query}
                        onQueryChange={setQuery}
                        groupBy={groupBy}
                        onGroupByChange={setGroupBy}
                        showGroupBy={viewMode === 'grid' && nodes.length > 1}
                    />

                    {visible.length === 0 ? (
                        <EmptyState
                            icon={<SearchX className="h-5 w-5" />}
                            title="No apps match"
                            description={
                                isFiltering
                                    ? 'Try a different filter or clear it to see everything.'
                                    : 'Nothing to show.'
                            }
                            action={
                                isFiltering ? (
                                    <Button variant="outline" onClick={clearFilters}>
                                        Clear filters
                                    </Button>
                                ) : undefined
                            }
                        />
                    ) : viewMode === 'list' ? (
                        <FleetTable apps={visible} metricsFor={metrics.forApp} actions={actions} />
                    ) : (
                        <div className="flex flex-col gap-7">
                            {groups.map((group) => (
                                <NodeGroupSection
                                    key={group.key}
                                    group={group}
                                    showHeader={showHeaders}
                                    nodeMetrics={group.node ? metrics.forNode(group.node.id) : undefined}
                                    metricsFor={metrics.forApp}
                                    actions={actions}
                                />
                            ))}
                        </div>
                    )}
                </>
            )}

            {actions.dialog}
        </div>
    )
}

export default Dashboard
