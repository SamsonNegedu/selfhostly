import React, { useState, useMemo, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useApps, useQueryClient } from '@/shared/services/api'
import { useAppStore } from '@/shared/stores/app-store'
import AppList from './components/AppList'
import AppListView from './components/AppListView'
import { AlertCircle, Search, Filter, X, TrendingUp, Server, Activity, AlertTriangle, Plus, LayoutGrid, List } from 'lucide-react'
import { DashboardSkeleton } from '@/shared/components/ui/Skeleton'
import { Button } from '@/shared/components/ui/Button'
import { useNodeContext } from '@/shared/contexts/NodeContext'
import type { App } from '@/shared/types/api'

type SortOption = 'name' | 'date' | 'status'
type FilterStatus = 'all' | 'running' | 'stopped' | 'updating' | 'error'
type ViewMode = 'grid' | 'list'

function Dashboard() {
    // Get global node context
    const { selectedNodeIds } = useNodeContext()

    const { data: apps, isLoading, error } = useApps(selectedNodeIds)
    const setApps = useAppStore((state) => state.setApps)
    const queryClient = useQueryClient()

    // Search and filter state
    const [searchQuery, setSearchQuery] = useState('')
    const [statusFilter, setStatusFilter] = useState<FilterStatus>('all')
    const [sortBy, setSortBy] = useState<SortOption>('date')
    const [showFilters, setShowFilters] = useState(false)

    // View mode state with localStorage persistence
    const [viewMode, setViewMode] = useState<ViewMode>(() => {
        const saved = localStorage.getItem('apps-view-mode')
        return (saved === 'grid' || saved === 'list') ? saved : 'list' // Default to list view
    })

    // Save view mode preference
    useEffect(() => {
        localStorage.setItem('apps-view-mode', viewMode)
    }, [viewMode])

    // Subscribe to query cache updates and sync with Zustand store
    React.useEffect(() => {
        const unsubscribe = queryClient.getQueryCache().subscribe(() => {
            // Only update when apps data changes
            const appsQuery = queryClient.getQueryCache().findAll({ queryKey: ['apps'] })
            if (appsQuery.length > 0) {
                const appsData = appsQuery[0].state.data as App[]
                if (appsData) {
                    setApps(appsData)
                }
            }
        })

        // Initial sync
        if (apps) {
            setApps(apps)
        }

        return () => unsubscribe()
    }, [apps, setApps, queryClient])

    // Filter and sort apps
    const filteredAndSortedApps = useMemo(() => {
        let result = apps || []

        // Filter by search query
        if (searchQuery) {
            const query = searchQuery.toLowerCase()
            result = result.filter(app =>
                app.name.toLowerCase().includes(query) ||
                (app.description && app.description.toLowerCase().includes(query))
            )
        }

        // Filter by status
        if (statusFilter !== 'all') {
            result = result.filter(app => app.status === statusFilter)
        }

        // Sort
        result = [...result].sort((a, b) => {
            switch (sortBy) {
                case 'name':
                    return a.name.localeCompare(b.name)
                case 'date':
                    return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
                case 'status':
                    const statusOrder = { error: 0, updating: 1, running: 2, stopped: 3 }
                    return (statusOrder[a.status as keyof typeof statusOrder] || 4) -
                        (statusOrder[b.status as keyof typeof statusOrder] || 4)
                default:
                    return 0
            }
        })

        return result
    }, [apps, searchQuery, statusFilter, sortBy])

    // Calculate statistics
    const stats = useMemo(() => {
        const total = apps?.length || 0
        const running = apps?.filter(a => a.status === 'running').length || 0
        const stopped = apps?.filter(a => a.status === 'stopped').length || 0
        const errors = apps?.filter(a => a.status === 'error').length || 0
        const updating = apps?.filter(a => a.status === 'updating').length || 0

        return { total, running, stopped, errors, updating }
    }, [apps])

    // Clear all filters
    const clearFilters = () => {
        setSearchQuery('')
        setStatusFilter('all')
    }

    if (isLoading) {
        return <DashboardSkeleton />
    }

    if (error) {
        return (
            <div className="flex items-center justify-center min-h-[400px]">
                <div className="text-center max-w-md fade-in">
                    <AlertCircle className="h-12 w-12 text-destructive mx-auto mb-4" />
                    <h2 className="text-xl font-semibold mb-2">Failed to load apps</h2>
                    <p className="text-muted-foreground mb-4">
                        There was an error loading your applications. Please try again.
                    </p>
                    <button
                        onClick={() => window.location.reload()}
                        className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:opacity-90 transition-opacity button-press"
                    >
                        Retry
                    </button>
                </div>
            </div>
        )
    }

    return (
        <div className="fade-in space-y-4 sm:space-y-6">
            {/* Header */}
            <div className="flex flex-col sm:flex-row items-start gap-3 sm:gap-0 sm:justify-between">
                <div className="flex-1">
                    <h1 className="text-2xl sm:text-3xl font-bold">Applications</h1>
                    <p className="text-muted-foreground mt-1 sm:mt-2 text-sm sm:text-base">
                        Manage and monitor your self-hosted apps
                    </p>
                </div>
                <div className="flex items-center gap-2 w-full sm:w-auto">
                    {/* View Toggle */}
                    <div className="flex items-center gap-1 bg-muted rounded-lg p-1">
                        <Button
                            variant={viewMode === 'list' ? 'default' : 'ghost'}
                            size="sm"
                            onClick={() => setViewMode('list')}
                            className="h-8 px-3"
                        >
                            <List className="h-4 w-4" />
                            <span className="hidden sm:inline ml-1.5">List</span>
                        </Button>
                        <Button
                            variant={viewMode === 'grid' ? 'default' : 'ghost'}
                            size="sm"
                            onClick={() => setViewMode('grid')}
                            className="h-8 px-3"
                        >
                            <LayoutGrid className="h-4 w-4" />
                            <span className="hidden sm:inline ml-1.5">Grid</span>
                        </Button>
                    </div>
                    <Link to="/apps/new" className="flex-1 sm:flex-none">
                        <Button className="button-press w-full sm:w-auto">
                            <Plus className="h-4 w-4 mr-2" />
                            <span className="sm:inline">New App</span>
                        </Button>
                    </Link>
                </div>
            </div>

            {/* Statistics Cards */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4">
                <div className="rounded-lg border bg-card text-card-foreground shadow-sm p-4 sm:p-6 card-hover">
                    <div className="flex items-center justify-between mb-1 sm:mb-2">
                        <span className="text-xs sm:text-sm font-medium text-muted-foreground">Total Apps</span>
                        <Server className="h-4 w-4 sm:h-5 sm:w-5 text-muted-foreground" />
                    </div>
                    <div className="text-xl sm:text-2xl font-bold">{stats.total}</div>
                    <div className="text-xs text-muted-foreground mt-0.5 sm:mt-1 hidden sm:block">Across all environments</div>
                </div>

                <div className="rounded-lg border bg-card text-card-foreground shadow-sm p-4 sm:p-6 card-hover">
                    <div className="flex items-center justify-between mb-1 sm:mb-2">
                        <span className="text-xs sm:text-sm font-medium text-muted-foreground">Running</span>
                        <Activity className="h-4 w-4 sm:h-5 sm:w-5 text-green-500" />
                    </div>
                    <div className="text-xl sm:text-2xl font-bold text-green-600 dark:text-green-400">{stats.running}</div>
                    <div className="text-xs text-muted-foreground mt-0.5 sm:mt-1">
                        {stats.total > 0 ? `${Math.round((stats.running / stats.total) * 100)}%` : '0%'}
                    </div>
                </div>

                <div className="rounded-lg border bg-card text-card-foreground shadow-sm p-4 sm:p-6 card-hover">
                    <div className="flex items-center justify-between mb-1 sm:mb-2">
                        <span className="text-xs sm:text-sm font-medium text-muted-foreground">Updating</span>
                        <TrendingUp className="h-4 w-4 sm:h-5 sm:w-5 text-blue-500" />
                    </div>
                    <div className="text-xl sm:text-2xl font-bold text-blue-600 dark:text-blue-400">{stats.updating}</div>
                    <div className="text-xs text-muted-foreground mt-0.5 sm:mt-1 hidden sm:block">In progress</div>
                </div>

                <div className="rounded-lg border bg-card text-card-foreground shadow-sm p-4 sm:p-6 card-hover">
                    <div className="flex items-center justify-between mb-1 sm:mb-2">
                        <span className="text-xs sm:text-sm font-medium text-muted-foreground">Errors</span>
                        <AlertTriangle className="h-4 w-4 sm:h-5 sm:w-5 text-red-500" />
                    </div>
                    <div className="text-xl sm:text-2xl font-bold text-red-600 dark:text-red-400">{stats.errors}</div>
                    <div className="text-xs text-muted-foreground mt-0.5 sm:mt-1 hidden sm:block">Require attention</div>
                </div>
            </div>

            {/* Search and Filters - only show if apps exist */}
            {stats.total > 0 && (
                <div className="flex flex-col gap-3 sm:gap-4">
                    {/* Search bar and Filters side by side */}
                    <div className="flex items-center gap-2 flex-wrap">
                        <div className="relative flex-1 min-w-0">
                            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                            <input
                                type="text"
                                placeholder="Search apps..."
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 pl-10 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                            />
                            {searchQuery && (
                                <button
                                    onClick={() => setSearchQuery('')}
                                    className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                                >
                                    <X className="h-4 w-4" />
                                </button>
                            )}
                        </div>
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setShowFilters(!showFilters)}
                            className="button-press flex-shrink-0 h-10"
                        >
                            <Filter className="h-4 w-4 mr-2" />
                            Filters
                            {(statusFilter !== 'all' || sortBy !== 'date') && (
                                <span className="ml-2 px-1.5 py-0.5 text-xs bg-primary text-primary-foreground rounded-full">
                                    {(statusFilter !== 'all' ? 1 : 0) + (sortBy !== 'date' ? 1 : 0)}
                                </span>
                            )}
                        </Button>
                        {(searchQuery || statusFilter !== 'all' || sortBy !== 'date') && (
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={clearFilters}
                                className="button-press flex-shrink-0 h-10"
                            >
                                Clear all
                            </Button>
                        )}
                    </div>
                </div>
            )}

            {/* Filter Options Panel */}
            {showFilters && (
                <div className="rounded-lg border bg-card p-4 space-y-4 slide-in">
                    <div className="flex flex-col sm:flex-row gap-4">
                        {/* Status Filter */}
                        <div className="flex-1">
                            <label className="text-sm font-medium mb-2 block">Status</label>
                            <select
                                value={statusFilter}
                                onChange={(e) => setStatusFilter(e.target.value as FilterStatus)}
                                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                            >
                                <option value="all">All Statuses</option>
                                <option value="running">Running</option>
                                <option value="stopped">Stopped</option>
                                <option value="updating">Updating</option>
                                <option value="error">Error</option>
                            </select>
                        </div>

                        {/* Sort By */}
                        <div className="flex-1">
                            <label className="text-sm font-medium mb-2 block">Sort By</label>
                            <select
                                value={sortBy}
                                onChange={(e) => setSortBy(e.target.value as SortOption)}
                                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                            >
                                <option value="date">Last Updated</option>
                                <option value="name">Name</option>
                                <option value="status">Status</option>
                            </select>
                        </div>
                    </div>
                </div>
            )}

            {/* Results Count */}
            {searchQuery || statusFilter !== 'all' || sortBy !== 'date' ? (
                <div className="text-sm text-muted-foreground">
                    Showing {filteredAndSortedApps.length} of {stats.total} app{stats.total !== 1 ? 's' : ''}
                </div>
            ) : null}

            {/* Render appropriate view based on mode */}
            {viewMode === 'list' ? (
                <AppListView filteredApps={filteredAndSortedApps} />
            ) : (
                <AppList filteredApps={filteredAndSortedApps} />
            )}
        </div>
    )
}

export default Dashboard
