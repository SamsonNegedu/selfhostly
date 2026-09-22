import { AlertTriangle, CheckCircle2, XCircle } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { cn } from '@/shared/lib/utils'
import type { CheckLevel, ComposeCheck } from '../lib/compose-checks'

const ICONS: Record<CheckLevel, typeof CheckCircle2> = { ok: CheckCircle2, warn: AlertTriangle, err: XCircle }
const ICON_CLASSES: Record<CheckLevel, string> = {
    ok: 'text-status-ok-fg',
    warn: 'text-status-warn-fg',
    err: 'text-status-err-fg',
}
const LEVEL_WORDS: Record<CheckLevel, string> = { ok: 'Passed', warn: 'Warning', err: 'Problem' }

// The rows themselves, so a caller that doesn't want the "Checks" card (a compact inline summary, say) can
// render just the list.
function ComposeCheckList({ checks, className }: { checks: ComposeCheck[]; className?: string }) {
    return (
        <ul className={cn('flex flex-col gap-3', className)}>
            {checks.map((check) => {
                const Icon = ICONS[check.level]
                return (
                    <li key={check.id} className="flex items-start gap-2.5 text-sm">
                        <Icon aria-hidden="true" className={cn('mt-0.5 h-4 w-4 shrink-0', ICON_CLASSES[check.level])} />
                        <div className="min-w-0">
                            <p className="font-medium">
                                <span className="sr-only">{LEVEL_WORDS[check.level]}: </span>
                                {check.title}
                            </p>
                            {check.detail && <p className="break-words text-compact text-muted-foreground">{check.detail}</p>}
                        </div>
                    </li>
                )
            })}
        </ul>
    )
}

// The list of things checked in the file as you type, in its own sidebar card. Each row says its result in
// words, not only by color.
function ComposeChecks({ checks }: { checks: ComposeCheck[] }) {
    return (
        <Card aria-label="Checks" role="region">
            <CardHeader>
                <CardTitle className="text-base">Checks</CardTitle>
            </CardHeader>
            <CardContent>
                <ComposeCheckList checks={checks} />
            </CardContent>
        </Card>
    )
}

export default ComposeChecks
export { ComposeCheckList }
