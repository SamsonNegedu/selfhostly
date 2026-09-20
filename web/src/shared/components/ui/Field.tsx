import * as React from 'react'

import { cn } from '@/shared/lib/utils'

interface FieldProps {
    label: string
    hint?: string
    error?: string
    id?: string
    className?: string
    children: React.ReactElement<Record<string, unknown>>
}

// Pairs a label, hint and error message with a single control. It wires up the id,
// aria-describedby and aria-invalid so the control needs no extra props.
function Field({ label, hint, error, id, className, children }: FieldProps) {
    const autoId = React.useId()
    const controlId = id ?? autoId
    const messageId = `${controlId}-message`
    const message = error ?? hint

    const control = React.cloneElement(children, {
        id: controlId,
        'aria-describedby': message ? messageId : undefined,
        'aria-invalid': error ? true : undefined,
    })

    return (
        <div className={cn('flex flex-col gap-1.5', className)}>
            <label htmlFor={controlId} className="text-compact font-medium">
                {label}
            </label>
            {control}
            {message && (
                <p id={messageId} className={cn('text-xs', error ? 'text-status-err-fg' : 'text-muted-foreground')}>
                    {message}
                </p>
            )}
        </div>
    )
}

export { Field }
