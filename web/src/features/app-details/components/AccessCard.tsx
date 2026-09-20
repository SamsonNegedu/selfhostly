import { Globe } from 'lucide-react'
import { Link } from 'react-router-dom'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { appHref } from '@/shared/lib/routes'
import type { App } from '@/shared/types/api'
import OverviewCard from './OverviewCard'

// Where the app can be reached, and a link to change it.
function AccessCard({ app }: { app: App }) {
    return (
        <OverviewCard
            icon={<Globe className="h-4 w-4 text-muted-foreground" />}
            title="Access"
            contentClassName="flex flex-col gap-3"
        >
            {app.public_url ? (
                <div className="flex flex-wrap items-center gap-2">
                    <a
                        href={app.public_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex min-h-[44px] items-center break-all font-mono text-sm text-status-info-fg hover:underline md:min-h-0"
                    >
                        {app.public_url.replace(/^https?:\/\//, '')}
                    </a>
                    {app.tunnel_mode === 'quick' && (
                        <StatusPill kind="warn" size="sm">
                            Temporary
                        </StatusPill>
                    )}
                </div>
            ) : (
                <p className="text-sm text-muted-foreground">
                    Only reachable on your network. Add a public address when you want to share it.
                </p>
            )}
            <Link
                to={appHref(app, 'access')}
                className="inline-flex min-h-[44px] items-center text-sm font-medium hover:underline md:min-h-0"
            >
                Manage access
            </Link>
        </OverviewCard>
    )
}

export default AccessCard
