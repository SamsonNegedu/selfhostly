import * as React from 'react'
import { AlertTriangle, RotateCw } from 'lucide-react'

import { cn } from '@/shared/lib/utils'
import { describeError } from '@/shared/lib/errors'
import { Button } from './Button'

interface ErrorStateProps {
    title?: string
    error?: unknown
    description?: string
    onRetry?: () => void
    retryLabel?: string
    className?: string
    children?: React.ReactNode
}

// A designed failure state. It shows a readable sentence, never a raw response body.
function ErrorState({
    title = 'Could not load this',
    error,
    description,
    onRetry,
    retryLabel = 'Try again',
    className,
    children,
}: ErrorStateProps) {
    return (
        <div
            role="alert"
            className={cn('flex flex-col items-start gap-4 rounded-xl border border-border bg-card p-6', className)}
        >
            <div
                aria-hidden="true"
                className="flex h-11 w-11 items-center justify-center rounded-xl bg-status-err-bg text-status-err-fg"
            >
                <AlertTriangle className="h-5 w-5" />
            </div>
            <div className="flex max-w-xl flex-col gap-1">
                <h2 className="text-lg font-semibold">{title}</h2>
                <p className="text-sm text-muted-foreground">{description ?? describeError(error)}</p>
            </div>
            {children}
            {onRetry && (
                <Button onClick={onRetry}>
                    <RotateCw className="h-4 w-4" />
                    {retryLabel}
                </Button>
            )}
        </div>
    )
}

export { ErrorState }
