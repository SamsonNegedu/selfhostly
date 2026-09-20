import { useMemo } from 'react'
import { Checkbox } from '@/shared/components/ui/Checkbox'
import { CodeBlock } from '@/shared/components/ui/CodeBlock'
import { diffToLines } from '@/shared/lib/update'
import type { UpdatePlan } from '@/shared/types/api'

interface ComposePlanSectionProps {
    compose: UpdatePlan['compose']
    approved: boolean
    onApprovedChange: (approved: boolean) => void
}

// What this release does to your compose file, with the changes to read and the box that approves them.
function ComposePlanSection({ compose, approved, onApprovedChange }: ComposePlanSectionProps) {
    const diffLines = useMemo(() => diffToLines(compose.diff), [compose.diff])
    const needsApproval = compose.state === 'behind'

    return (
        <section aria-label="Compose file" className="flex flex-col gap-2">
            <h4 className="text-compact font-semibold">Your compose file</h4>
            {compose.state === 'current' && (
                <p className="text-compact text-muted-foreground">Your compose file is already up to date.</p>
            )}
            {compose.state === 'behind' && (
                <p className="text-compact text-muted-foreground">
                    This release changes the compose file. Nothing you set is lost. Review the changes and approve them
                    to continue.
                </p>
            )}
            {compose.state === 'customized' && (
                <p className="text-compact text-status-warn-fg">
                    Your compose file has edits this release would lose. Merge them by hand first, then review again.
                </p>
            )}
            {compose.report && (
                <p className="whitespace-pre-wrap rounded-lg bg-muted px-3 py-2 font-mono text-xs">{compose.report}</p>
            )}
            {diffLines.length > 0 && (
                <div data-testid="update-compose-diff">
                    <CodeBlock lines={diffLines} language="text" aria-label="Compose file changes" />
                </div>
            )}
            {needsApproval && (
                <label className="flex min-h-[44px] items-center gap-2.5 text-compact sm:min-h-0">
                    <Checkbox
                        data-testid="update-approve-compose"
                        checked={approved}
                        onCheckedChange={onApprovedChange}
                    />
                    I have reviewed the compose file changes
                </label>
            )}
        </section>
    )
}

export default ComposePlanSection
