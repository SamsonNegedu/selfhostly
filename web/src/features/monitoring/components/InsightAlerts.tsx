import { AlertTriangle, CheckCircle2, XCircle } from 'lucide-react'
import { Card } from '@/shared/components/ui/Card'
import { cn } from '@/shared/lib/utils'
import type { StatusKind } from '@/shared/lib/status'
import type { InsightAlert } from '../lib/alerts'

const ICONS = { err: XCircle, warn: AlertTriangle }
const TILE: Record<'err' | 'warn', string> = {
    err: 'bg-status-err-bg text-status-err-fg',
    warn: 'bg-status-warn-bg text-status-warn-fg',
}
const WORD: Record<StatusKind, string> = { ok: 'Fine', warn: 'Warning', err: 'Critical', info: 'Info', idle: 'Info' }

// The short list of things to look at, or a plain statement that there is nothing.
function InsightAlerts({ alerts }: { alerts: InsightAlert[] }) {
    if (alerts.length === 0) {
        return (
            <Card className="flex items-center gap-3 p-4" aria-label="Alerts" role="region">
                <CheckCircle2 aria-hidden="true" className="h-5 w-5 shrink-0 text-status-ok-fg" />
                <p className="text-sm font-medium">
                    Everything looks fine. CPU, memory and disk are all within their limits.
                </p>
            </Card>
        )
    }

    return (
        <Card className="divide-y divide-border" aria-label="Alerts" role="region">
            {alerts.map((alert) => {
                const kind = alert.kind === 'err' ? 'err' : 'warn'
                const Icon = ICONS[kind]
                return (
                    <div key={alert.id} className="flex items-start gap-3 p-4">
                        <div
                            aria-hidden="true"
                            className={cn('flex h-8 w-8 shrink-0 items-center justify-center rounded-lg', TILE[kind])}
                        >
                            <Icon className="h-[17px] w-[17px]" />
                        </div>
                        <div className="min-w-0">
                            <p className="font-semibold">
                                <span className="sr-only">{WORD[alert.kind]}: </span>
                                {alert.title}
                            </p>
                            <p className="break-words text-[13px] text-muted-foreground">{alert.detail}</p>
                        </div>
                    </div>
                )
            })}
        </Card>
    )
}

export default InsightAlerts
