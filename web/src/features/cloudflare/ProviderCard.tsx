import { Link } from 'react-router-dom'
import { Globe } from 'lucide-react'
import { buttonClasses } from '@/shared/components/ui/Button'
import { Card } from '@/shared/components/ui/Card'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { ROUTES } from '@/shared/lib/routes'

// Whether Cloudflare is connected, with a link to Settings to connect or manage it.
function ProviderCard({ connected }: { connected: boolean }) {
    return (
        <Card className="flex flex-wrap items-center gap-4 p-4" aria-label="Tunnel provider" role="region">
            <div
                aria-hidden="true"
                className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground"
            >
                <Globe className="h-5 w-5" />
            </div>
            <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                    <p className="font-semibold">Cloudflare</p>
                    <StatusPill kind={connected ? 'ok' : 'idle'}>
                        {connected ? 'Connected' : 'Not connected'}
                    </StatusPill>
                </div>
                <p className="text-compact text-muted-foreground">
                    {connected
                        ? 'Custom domains are available. Quick Tunnels also work without it.'
                        : 'Connect it to publish apps on your own domain. Quick Tunnels work without it.'}
                </p>
            </div>
            <Link to={ROUTES.settings} className={buttonClasses({ variant: 'outline' })}>
                {connected ? 'Manage' : 'Connect'}
            </Link>
        </Card>
    )
}

export default ProviderCard
