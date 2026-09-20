import type { UpdatePlan } from '@/shared/types/api'

// What stops an update, one plain sentence each.
function PlanBlockerList({ blockers }: { blockers: UpdatePlan['blockers'] }) {
    return (
        <ul aria-label="What stops this update" className="flex flex-col gap-2">
            {blockers.map((blocker) => (
                <li
                    key={`${blocker.code}-${blocker.message}`}
                    data-testid="update-blocker"
                    data-code={blocker.code}
                    className="rounded-lg bg-status-warn-bg px-3 py-2 text-compact text-status-warn-fg"
                >
                    {blocker.message}
                </li>
            ))}
        </ul>
    )
}

export default PlanBlockerList
