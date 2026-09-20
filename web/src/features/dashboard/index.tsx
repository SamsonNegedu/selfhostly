import { useApps, useNodes } from '@/shared/services/api'
import { Button } from '@/shared/components/ui/Button'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { SearchX } from 'lucide-react'
import { useNodeContext } from '@/shared/contexts/NodeContext'
import AttentionStrip from './components/AttentionStrip'
import { FleetEmpty, FleetLoading } from './components/FleetStates'
import FleetHeader from './components/FleetHeader'
import FleetTable from './components/FleetTable'
import FleetToolbar from './components/FleetToolbar'
import NodeGroupSection from './components/NodeGroupSection'
import { useFleetActions } from './hooks/useFleetActions'
import { useFleetMetrics } from './hooks/useFleetMetrics'
import { useFleetView } from './hooks/useFleetView'
import { useSyncAppStore } from './hooks/useSyncAppStore'

function Dashboard() {
    const { selectedNodeIds } = useNodeContext()
    const { data: apps, error, refetch } = useApps(selectedNodeIds)
    const { data: nodes = [] } = useNodes()
    const metrics = useFleetMetrics(nodes)
    const actions = useFleetActions()
    useSyncAppStore(apps)
    const {
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
        total,
        counts,
        visible,
        groups,
        showHeaders,
        isFiltering,
        clearFilters,
    } = useFleetView(apps, nodes)

    const nodeCount = selectedNodeIds.length

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
            <FleetHeader
                total={total}
                nodeCount={nodeCount}
                phone={phone}
                viewMode={savedViewMode}
                onViewModeChange={setViewMode}
            />

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
