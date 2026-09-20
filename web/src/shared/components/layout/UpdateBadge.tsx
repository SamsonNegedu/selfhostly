import { Link } from 'react-router-dom'
import { Download } from 'lucide-react'
import { useUpdateStatus } from '@/shared/hooks/useUpdate'
import { ROUTES } from '@/shared/lib/routes'

const UPDATES_ADDRESS = `${ROUTES.settings}?section=updates`

// Appears in the header when a newer version exists and this server can install it from the browser.
function UpdateBadge() {
    const { data } = useUpdateStatus()
    if (!data?.enabled || !data.available) return null

    return (
        <Link
            to={UPDATES_ADDRESS}
            data-testid="update-badge"
            aria-label={`Update available: version ${data.available.version}`}
            className="flex h-[44px] items-center gap-2 rounded-lg border border-status-info-fg/30 bg-status-info-bg px-3 text-[13px] font-medium text-status-info-fg ring-offset-background transition-colors hover:bg-status-info-bg/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 md:h-9"
        >
            <Download aria-hidden="true" className="h-4 w-4" />
            <span className="hidden sm:inline">Update available</span>
        </Link>
    )
}

export default UpdateBadge
