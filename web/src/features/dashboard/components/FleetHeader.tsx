import { Link } from 'react-router-dom'
import { LayoutGrid, List, Plus } from 'lucide-react'
import { buttonClasses } from '@/shared/components/ui/Button'
import { SegmentedControl } from '@/shared/components/ui/SegmentedControl'
import { ROUTES } from '@/shared/lib/routes'
import type { ViewMode } from '../hooks/useFleetView'

const VIEW_OPTIONS = [
    { value: 'grid', label: 'Cards', icon: <LayoutGrid className="h-4 w-4" /> },
    { value: 'list', label: 'Table', icon: <List className="h-4 w-4" /> },
]

interface FleetHeaderProps {
    total: number
    nodeCount: number
    phone: boolean
    // The view the person chose. On a phone the page shows cards whatever this says.
    viewMode: ViewMode
    onViewModeChange: (mode: ViewMode) => void
}

// The page title with how many apps there are, the cards or table switch, and the New app button.
function FleetHeader({ total, nodeCount, phone, viewMode, onViewModeChange }: FleetHeaderProps) {
    return (
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
                        value={viewMode}
                        onValueChange={(value) => onViewModeChange(value as ViewMode)}
                    />
                )}
                <Link to={ROUTES.newApp} className={buttonClasses({ className: 'max-md:hidden' })}>
                    <Plus className="h-4 w-4" />
                    New app
                </Link>
            </div>
        </div>
    )
}

export default FleetHeader
