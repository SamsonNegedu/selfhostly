import { AlertTriangle, Globe, RefreshCw } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { StatusPill } from '@/shared/components/ui/StatusPill'

interface QuickTunnelCardProps {
    publicUrl: string
    onSwitch: () => void
    onNewAddress: () => void
}

// A temporary trycloudflare.com address, with what to know about its limits and the way out to your own domain.
function QuickTunnelCard({ publicUrl, onSwitch, onNewAddress }: QuickTunnelCardProps) {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="flex flex-wrap items-center gap-2.5 text-base">
                    <Globe className="h-4 w-4 text-muted-foreground" />
                    Quick Tunnel
                    <StatusPill kind="warn">Temporary</StatusPill>
                </CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
                <a
                    href={publicUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex min-h-[44px] items-center break-all font-mono text-sm font-medium text-status-info-fg hover:underline md:min-h-0"
                >
                    {publicUrl}
                </a>
                <div className="flex items-start gap-2 rounded-lg bg-status-warn-bg px-3 py-2.5 text-compact text-status-warn-fg">
                    <AlertTriangle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0" />
                    <p>
                        The address can change when the app restarts, it is limited to 200 requests at a time, and it
                        does not support server-sent events. Use your own domain for anything you rely on.
                    </p>
                </div>
                <div className="flex flex-wrap gap-2">
                    <Button onClick={() => onSwitch()}>Switch to my own domain</Button>
                    <Button variant="outline" onClick={() => onNewAddress()}>
                        <RefreshCw className="h-4 w-4" />
                        Get a new address
                    </Button>
                </div>
            </CardContent>
        </Card>
    )
}

export default QuickTunnelCard
