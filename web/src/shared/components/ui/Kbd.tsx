import * as React from 'react'

import { cn } from '@/shared/lib/utils'

const Kbd = React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(({ className, ...props }, ref) => (
    <kbd
        ref={ref}
        className={cn(
            'inline-flex h-5 items-center rounded border border-border bg-background px-1.5 font-mono text-caption text-muted-foreground',
            className,
        )}
        {...props}
    />
))
Kbd.displayName = 'Kbd'

export { Kbd }
