import { useMemo } from 'react'
import { HardDrive } from 'lucide-react'
import { formatDayAgo } from '@/shared/lib/format'
import type { App } from '@/shared/types/api'
import { readComposeResources } from '../lib/compose-resources'
import OverviewCard from './OverviewCard'

// The networks and volumes the compose file declares, and when the app was last changed.
function ResourcesCard({ app }: { app: App }) {
    const composeInfo = useMemo(() => readComposeResources(app.compose_content), [app.compose_content])

    return (
        <OverviewCard
            icon={<HardDrive className="h-4 w-4 text-muted-foreground" />}
            title="Resources"
            contentClassName="flex flex-col gap-3 text-sm"
        >
            <div className="flex items-center justify-between gap-3">
                <span className="text-muted-foreground">Networks</span>
                <div className="flex flex-wrap justify-end gap-1">
                    {(composeInfo.networks.length > 0 ? composeInfo.networks : ['default']).map((network) => (
                        <span key={network} className="rounded-full bg-muted px-2.5 py-0.5 font-mono text-xs">
                            {network}
                        </span>
                    ))}
                </div>
            </div>
            <div className="flex items-center justify-between gap-3">
                <span className="text-muted-foreground">Volumes</span>
                <span className="text-right font-medium">
                    {composeInfo.volumes.length > 0 ? composeInfo.volumes.join(', ') : 'None'}
                </span>
            </div>
            <div className="flex items-center justify-between gap-3">
                <span className="text-muted-foreground">Last updated</span>
                <span className="font-medium">{formatDayAgo(app.updated_at)}</span>
            </div>
        </OverviewCard>
    )
}

export default ResourcesCard
