import * as React from 'react'
import * as TooltipPrimitive from '@radix-ui/react-tooltip'

import { cn } from '@/shared/lib/utils'

const OPEN_DELAY_MS = 200

// Existing call sites use TooltipProvider as the wrapper of a single tooltip, so it provides the
// Radix provider and root together. Content is portaled, so it is never clipped by a table or card.
const TooltipProvider = ({ children }: { children: React.ReactNode }) => (
    <TooltipPrimitive.Provider delayDuration={OPEN_DELAY_MS}>
        <TooltipPrimitive.Root>{children}</TooltipPrimitive.Root>
    </TooltipPrimitive.Provider>
)

const TooltipTrigger = TooltipPrimitive.Trigger

const TooltipContent = React.forwardRef<
    React.ElementRef<typeof TooltipPrimitive.Content>,
    React.ComponentPropsWithoutRef<typeof TooltipPrimitive.Content>
>(({ className, sideOffset = 6, ...props }, ref) => (
    <TooltipPrimitive.Portal>
        <TooltipPrimitive.Content
            ref={ref}
            sideOffset={sideOffset}
            className={cn(
                'z-50 max-w-xs rounded-md border border-border bg-popover px-2.5 py-1.5 text-xs text-popover-foreground shadow-md',
                className,
            )}
            {...props}
        />
    </TooltipPrimitive.Portal>
))
TooltipContent.displayName = TooltipPrimitive.Content.displayName

export { TooltipProvider, TooltipTrigger, TooltipContent }
