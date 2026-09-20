import { useEffect } from 'react'
import { ErrorState } from '@/shared/components/ui/ErrorState'

const RETRY_INTERVAL_MS = 15_000

interface ServerUnavailableProps {
    onRetry: () => void
}

// Shown instead of the sign-in page when the server cannot be reached. Being signed out and the server
// being down are different problems, and mixing them up makes people think they were logged out.
function ServerUnavailable({ onRetry }: ServerUnavailableProps) {
    useEffect(() => {
        const timer = setInterval(onRetry, RETRY_INTERVAL_MS)
        return () => clearInterval(timer)
    }, [onRetry])

    return (
        <div className="flex min-h-screen items-center justify-center bg-background p-4">
            <ErrorState
                title="Can't reach the Selfhostly server"
                description="Your apps keep running, only this dashboard is affected. Check that the server is running and that this address is right. It tries again every 15 seconds."
                onRetry={onRetry}
                className="w-full max-w-lg"
            />
        </div>
    )
}

export default ServerUnavailable
