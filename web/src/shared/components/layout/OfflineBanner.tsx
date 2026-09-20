import { WifiOff } from 'lucide-react'
import { useOnlineStatus } from '@/shared/hooks/useOnlineStatus'

// Shown while this device has no network, so a screen that stops updating does not look like it is working.
// The data on screen is what was loaded last, and it refreshes by itself once the connection is back.
function OfflineBanner() {
    const online = useOnlineStatus()
    if (online) return null

    return (
        <div role="status" className="border-b border-border bg-status-warn-bg text-status-warn-fg">
            <p className="container mx-auto flex items-center gap-2 px-3 py-2.5 text-sm font-medium sm:px-12">
                <WifiOff aria-hidden="true" className="h-4 w-4 shrink-0" />
                You are offline. What you see is from the last time it loaded, and it will refresh when you reconnect.
            </p>
        </div>
    )
}

export default OfflineBanner
