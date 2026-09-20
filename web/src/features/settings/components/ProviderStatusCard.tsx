import { Card } from '@/shared/components/ui/Card'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import type { TunnelProvidersResponse } from '@/shared/types/api'

// Whether the chosen provider has credentials saved.
function ProviderStatusCard({ current }: { current: TunnelProvidersResponse['providers'][number] | undefined }) {
    return (
        <Card className="flex flex-wrap items-center gap-3 p-4" aria-label="Provider status" role="region">
            <div className="min-w-0 flex-1">
                <p className="font-semibold">{current?.display_name ?? 'No provider'}</p>
                <p className="text-compact text-muted-foreground">
                    {current?.is_configured
                        ? 'Credentials are saved. Custom domains are available.'
                        : 'Add credentials below to publish apps on your own domain.'}
                </p>
            </div>
            <StatusPill kind={current?.is_configured ? 'ok' : 'idle'}>
                {current?.is_configured ? 'Connected' : 'Not connected'}
            </StatusPill>
        </Card>
    )
}

export default ProviderStatusCard
