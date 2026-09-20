import * as React from 'react'

import { cn } from '@/shared/lib/utils'
import type { StatusKind } from '@/shared/lib/status'

const PILL_CLASSES: Record<StatusKind, string> = {
    ok: 'bg-status-ok-bg text-status-ok-fg',
    warn: 'bg-status-warn-bg text-status-warn-fg',
    err: 'bg-status-err-bg text-status-err-fg',
    info: 'bg-status-info-bg text-status-info-fg',
    idle: 'bg-status-idle-bg text-status-idle-fg',
}

const DOT_CLASSES: Record<StatusKind, string> = {
    ok: 'bg-status-ok',
    warn: 'bg-status-warn',
    err: 'bg-status-err',
    info: 'bg-status-info',
    idle: 'bg-status-idle',
}

interface StatusDotProps extends React.HTMLAttributes<HTMLSpanElement> {
    kind: StatusKind
}

// Color alone never carries the meaning: a dot always sits next to a text label.
function StatusDot({ kind, className, ...props }: StatusDotProps) {
    return (
        <span
            aria-hidden="true"
            className={cn('inline-block h-2 w-2 shrink-0 rounded-full', DOT_CLASSES[kind], className)}
            {...props}
        />
    )
}

interface StatusPillProps extends React.HTMLAttributes<HTMLSpanElement> {
    kind: StatusKind
    size?: 'sm' | 'md'
}

function StatusPill({ kind, size = 'md', className, children, ...props }: StatusPillProps) {
    return (
        <span
            className={cn(
                'inline-flex items-center gap-1.5 whitespace-nowrap rounded-full font-medium',
                size === 'sm' ? 'px-2.5 py-0.5 text-[11px]' : 'px-2.5 py-[3px] text-xs',
                PILL_CLASSES[kind],
                className,
            )}
            {...props}
        >
            <StatusDot kind={kind} className="h-1.5 w-1.5" />
            {children}
        </span>
    )
}

export { StatusPill, StatusDot }
