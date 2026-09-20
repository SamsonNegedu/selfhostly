import * as React from 'react'

import { cn } from '@/shared/lib/utils'

interface EmptyStateProps {
    icon?: React.ReactNode
    title: string
    description?: string
    action?: React.ReactNode
    className?: string
    // 2 inside a page that has its own title. 1 when this is the whole page.
    headingLevel?: 1 | 2
}

// Shown where a list or panel has nothing to show yet. Always say what to do next.
function EmptyState({ icon, title, description, action, className, headingLevel = 2 }: EmptyStateProps) {
    const Heading = headingLevel === 1 ? 'h1' : 'h2'
    return (
        <div
            className={cn(
                'flex flex-col items-center gap-3 rounded-xl border border-dashed border-border px-6 py-10 text-center',
                className,
            )}
        >
            {icon && (
                <div
                    aria-hidden="true"
                    className="flex h-11 w-11 items-center justify-center rounded-full bg-muted text-muted-foreground"
                >
                    {icon}
                </div>
            )}
            <div className="flex max-w-md flex-col gap-1">
                <Heading className="text-base font-semibold">{title}</Heading>
                {description && <p className="text-sm text-muted-foreground">{description}</p>}
            </div>
            {action}
        </div>
    )
}

export { EmptyState }
