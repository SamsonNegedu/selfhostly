import React from 'react'
import { Card, CardContent } from '@/shared/components/ui/Card'
import { Button } from '@/shared/components/ui/Button'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import {
    RefreshCw,
    AlertCircle,
    CheckCircle2,
    Clock,
    Plus,
    Search,
    Filter,
    Activity,
    Link2,
    ArrowUpDown
} from 'lucide-react'
import { useTunnels } from '@/shared/services/api'
import { useNodeContext } from '@/shared/contexts/NodeContext'
import { useAppStore } from '@/shared/stores/app-store'
import { useState } from 'react'
import TunnelsListView from './components/TunnelsListView'

type SortField = 'name' | 'status' | 'created' | 'updated'
type SortOrder = 'asc' | 'desc'
type StatusFilter = 'all' | 'active' | 'inactive' | 'error'

function CloudflareManagement() {
    // Get global node context for filtering tunnels by selected nodes
    const { selectedNodeIds } = useNodeContext()

    const { data: tunnelsData, isLoading, error, refetch } = useTunnels(selectedNodeIds)

    const [searchQuery, setSearchQuery] = useState('')
    const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
    const [sortField, setSortField] = useState<SortField>('name')
    const [sortOrder, setSortOrder] = useState<SortOrder>('asc')

    const tunnels = tunnelsData?.tunnels || []

    // Filter, sort, and search tunnels
    const processedTunnels = React.useMemo(() => {
        let result = [...tunnels]

        // Apply status filter
        if (statusFilter !== 'all') {
            result = result.filter(tunnel => {
                switch (statusFilter) {
                    case 'active':
                        return tunnel.is_active
                    case 'inactive':
                        return !tunnel.is_active
                    case 'error':
                        return tunnel.status === 'error'
                    default:
                        return true
                }
            })
        }

        // Apply search filter
        if (searchQuery) {
            const query = searchQuery.toLowerCase()
            result = result.filter(tunnel =>
                tunnel.tunnel_name.toLowerCase().includes(query) ||
                tunnel.tunnel_id.toLowerCase().includes(query) ||
                (tunnel.public_url && tunnel.public_url.toLowerCase().includes(query))
            )
        }

        // Apply sorting
        result.sort((a, b) => {
            let aValue: any
            let bValue: any

            switch (sortField) {
                case 'name':
                    aValue = a.tunnel_name.toLowerCase()
                    bValue = b.tunnel_name.toLowerCase()
                    break
                case 'status':
                    aValue = a.is_active ? 1 : 0
                    bValue = b.is_active ? 1 : 0
                    break
                case 'created':
                    aValue = new Date(a.created_at).getTime()
                    bValue = new Date(b.created_at).getTime()
                    break
                case 'updated':
                    aValue = new Date(a.updated_at).getTime()
                    bValue = new Date(b.updated_at).getTime()
                    break
                default:
                    return 0
            }

            if (aValue < bValue) return sortOrder === 'asc' ? -1 : 1
            if (aValue > bValue) return sortOrder === 'asc' ? 1 : -1
            return 0
        })

        return result
    }, [tunnels, searchQuery, statusFilter, sortField, sortOrder])

    const toggleSort = (field: SortField) => {
        if (sortField === field) {
            setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
        } else {
            setSortField(field)
            setSortOrder('asc')
        }
    }

    // Get apps from store to find node_id for tunnels
    const apps = useAppStore((state) => state.apps)

    if (isLoading) {
        return (
            <div className="space-y-6 fade-in">
                <div className="flex items-center justify-between">
                    <Skeleton className="h-10 w-64" />
                    <Skeleton className="h-9 w-24" />
                </div>
                <Skeleton className="h-12 w-full" />
                <div className="space-y-4">
                    {[1, 2, 3].map((i) => (
                        <Card key={i} className="border-2">
                            <CardContent className="p-6">
                                <div className="space-y-4">
                                    <div className="flex items-start justify-between">
                                        <div className="space-y-2 flex-1">
                                            <Skeleton className="h-6 w-48" />
                                            <Skeleton className="h-5 w-32" />
                                        </div>
                                        <Skeleton className="h-9 w-24" />
                                    </div>
                                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                                        {[1, 2, 3, 4].map((j) => (
                                            <div key={j}>
                                                <Skeleton className="h-4 w-20 mb-1" />
                                                <Skeleton className="h-5 w-32" />
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </div>
        )
    }

    if (error) {
        return (
            <div className="space-y-6 fade-in">
                <div>
                    <h1 className="flex items-center gap-3 text-3xl font-bold">
                        <div className="p-2 rounded-lg bg-primary/10">
                            <Activity className="h-6 w-6 text-primary" />
                        </div>
                        Cloudflare Tunnels
                    </h1>
                </div>
                <Card className="border-2">
                    <CardContent className="py-12">
                        <div className="flex flex-col items-center gap-4">
                            <AlertCircle className="h-12 w-12 text-red-500" />
                            <div className="text-center">
                                <h2 className="text-xl font-semibold mb-2">Failed to load tunnels</h2>
                                <p className="text-muted-foreground mb-4">{error.message}</p>
                                <Button onClick={() => refetch()} className="button-press">
                                    <RefreshCw className="h-4 w-4 mr-2" />
                                    Retry
                                </Button>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </div>
        )
    }

    const activeCount = tunnels.filter(t => t.is_active).length

    return (
        <div className="space-y-4 sm:space-y-6 fade-in">
            {/* Header */}
            <div className="flex items-center justify-between flex-wrap gap-3 sm:gap-4">
                <div className="space-y-1">
                    <h1 className="flex items-center gap-2 sm:gap-3 text-2xl sm:text-3xl font-bold">
                        <div className="p-1.5 sm:p-2 rounded-lg bg-primary/10">
                            <Activity className="h-6 w-6 text-primary" />
                        </div>
                        Cloudflare Tunnels
                    </h1>
                </div>
                <Button
                    onClick={() => refetch()}
                    variant="outline"
                    size="sm"
                    className="button-press h-9 text-xs sm:text-sm"
                >
                    <RefreshCw className={`h-4 w-4 mr-1.5 sm:mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                    Refresh
                </Button>
            </div>

            <div className="space-y-4 sm:space-y-6">
                {/* Info Banner */}
                {tunnels.length > 0 && (
                    <div className="flex items-start gap-2 sm:gap-3 p-3 sm:p-4 rounded-lg bg-muted/50 border-2">
                        <Activity className="h-5 w-5 text-primary flex-shrink-0 mt-0.5" />
                        <div className="flex-1">
                            <p className="text-xs sm:text-sm font-medium mb-1">
                                Cloudflare Tunnel Status
                            </p>
                            <p className="text-xs sm:text-sm text-muted-foreground">
                                {activeCount > 0
                                    ? `${activeCount} tunnel${activeCount !== 1 ? 's' : ''} actively routing traffic to your applications.`
                                    : 'No active tunnels. Start your applications to establish secure connections.'
                                }
                            </p>
                        </div>
                    </div>
                )}

                {/* Filters Row - Status and Sort */}
                <div className="overflow-x-auto scrollbar-hide -mx-3 sm:mx-0 px-3 sm:px-0 pb-1">
                    <div className="flex items-center gap-2 w-fit">
                        {/* Status Filter */}
                        <div className="flex items-center gap-1 rounded-lg p-1 bg-muted/50 flex-shrink-0">
                            <Button
                                variant={statusFilter === 'all' ? 'default' : 'ghost'}
                                size="sm"
                                onClick={() => setStatusFilter('all')}
                                className="h-8 px-3 text-xs whitespace-nowrap"
                            >
                                All
                            </Button>
                            <Button
                                variant={statusFilter === 'active' ? 'default' : 'ghost'}
                                size="sm"
                                onClick={() => setStatusFilter('active')}
                                className="h-8 px-3 text-xs whitespace-nowrap"
                            >
                                <CheckCircle2 className="h-3.5 w-3.5 mr-1" />
                                Active
                            </Button>
                            <Button
                                variant={statusFilter === 'inactive' ? 'default' : 'ghost'}
                                size="sm"
                                onClick={() => setStatusFilter('inactive')}
                                className="h-8 px-3 text-xs whitespace-nowrap"
                            >
                                <Clock className="h-3.5 w-3.5 mr-1" />
                                Inactive
                            </Button>
                            <Button
                                variant={statusFilter === 'error' ? 'default' : 'ghost'}
                                size="sm"
                                onClick={() => setStatusFilter('error')}
                                className="h-8 px-3 text-xs whitespace-nowrap"
                            >
                                <AlertCircle className="h-3.5 w-3.5 mr-1" />
                                Error
                            </Button>
                        </div>

                        {/* Sort Button */}
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={() => toggleSort('name')}
                            className="h-8 px-3 text-xs whitespace-nowrap flex-shrink-0"
                        >
                            <ArrowUpDown className="h-4 w-4 mr-1.5" />
                            Sort
                        </Button>
                    </div>
                </div>

                {/* Search Bar - Full width */}
                <div className="relative w-full">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
                    <input
                        type="text"
                        placeholder="Search tunnels..."
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="flex h-10 w-full rounded-lg border-2 border-input bg-background px-3 py-2 pl-10 pr-10 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:border-primary transition-colors"
                    />
                    {searchQuery && (
                        <button
                            onClick={() => setSearchQuery('')}
                            className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                            aria-label="Clear search"
                        >
                            <span className="text-xl font-light">×</span>
                        </button>
                    )}
                </div>

                {/* Empty State */}
                {tunnels.length === 0 && (
                    <Card className="border-2 border-dashed">
                        <CardContent className="py-16 text-center">
                            <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-primary/10 mb-6">
                                <Link2 className="h-10 w-10 text-primary" />
                            </div>
                            <h3 className="text-2xl font-semibold mb-3">No Tunnels Yet</h3>
                            <p className="text-muted-foreground mb-6 max-w-md mx-auto text-base">
                                You don't have any Cloudflare tunnels configured. Create your first app with Cloudflare tunnel enabled to establish secure public access.
                            </p>
                            <Button onClick={() => window.location.href = '/apps/new'} size="lg" className="button-press">
                                <Plus className="h-5 w-5 mr-2" />
                                Create Your First App
                            </Button>
                        </CardContent>
                    </Card>
                )}

                {/* No Search/Filter Results */}
                {tunnels.length > 0 && processedTunnels.length === 0 && (
                    <Card className="border-2 border-dashed">
                        <CardContent className="py-16 text-center">
                            <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-muted mb-4">
                                <Search className="h-8 w-8 text-muted-foreground" />
                            </div>
                            <h3 className="text-xl font-semibold mb-2">No Matching Tunnels</h3>
                            <p className="text-muted-foreground mb-4">
                                {searchQuery
                                    ? `No tunnels found matching "${searchQuery}"`
                                    : 'No tunnels match the selected filters'
                                }
                            </p>
                            <Button
                                variant="outline"
                                onClick={() => {
                                    setSearchQuery('')
                                    setStatusFilter('all')
                                }}
                                className="button-press"
                            >
                                <Filter className="h-4 w-4 mr-2" />
                                Clear Filters
                            </Button>
                        </CardContent>
                    </Card>
                )}

                {/* Tunnels List */}
                {processedTunnels.length > 0 && (
                    <div className="space-y-3 sm:space-y-4">
                        <div className="flex items-center justify-between">
                            <p className="text-xs sm:text-sm font-medium text-muted-foreground">
                                Showing <span className="text-foreground font-semibold">{processedTunnels.length}</span> of <span className="text-foreground font-semibold">{tunnels.length}</span> tunnel{tunnels.length !== 1 ? 's' : ''}
                            </p>
                        </div>

                        <TunnelsListView
                            tunnels={processedTunnels}
                            apps={apps}
                        />
                    </div>
                )}
            </div>
        </div>
    )
}

export default CloudflareManagement
