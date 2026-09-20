import * as React from 'react'
import * as DialogPrimitive from '@radix-ui/react-dialog'
import { X } from 'lucide-react'

import { cn } from '@/shared/lib/utils'

const Sheet = DialogPrimitive.Root
const SheetTrigger = DialogPrimitive.Trigger
const SheetClose = DialogPrimitive.Close

type SheetSide = 'bottom' | 'right'

const SIDE_CLASSES: Record<SheetSide, string> = {
    bottom: 'inset-x-0 bottom-0 rounded-t-[20px] border-t px-4 pb-7 pt-3',
    right: 'inset-y-0 right-0 h-full w-full max-w-sm border-l p-6',
}

interface SheetContentProps extends React.ComponentPropsWithoutRef<typeof DialogPrimitive.Content> {
    side?: SheetSide
}

// A panel that slides in from an edge: a bottom sheet on phones, a side drawer for details on desktop.
const SheetContent = React.forwardRef<React.ElementRef<typeof DialogPrimitive.Content>, SheetContentProps>(
    ({ side = 'bottom', className, children, ...props }, ref) => (
        <DialogPrimitive.Portal>
            <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/65" />
            <DialogPrimitive.Content
                ref={ref}
                className={cn(
                    'fixed z-50 flex flex-col gap-3 border-border bg-card shadow-2xl focus-visible:outline-none',
                    SIDE_CLASSES[side],
                    className,
                )}
                {...props}
            >
                {side === 'bottom' && (
                    <div aria-hidden="true" className="mx-auto mb-1 h-1 w-10 rounded-full bg-border" />
                )}
                {children}
                <DialogPrimitive.Close className="absolute right-3 top-3 flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground ring-offset-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 max-sm:h-[44px] max-sm:w-[44px]">
                    <X className="h-4 w-4" />
                    <span className="sr-only">Close</span>
                </DialogPrimitive.Close>
            </DialogPrimitive.Content>
        </DialogPrimitive.Portal>
    ),
)
SheetContent.displayName = 'SheetContent'

const SheetTitle = React.forwardRef<
    React.ElementRef<typeof DialogPrimitive.Title>,
    React.ComponentPropsWithoutRef<typeof DialogPrimitive.Title>
>(({ className, ...props }, ref) => (
    <DialogPrimitive.Title ref={ref} className={cn('text-base font-semibold', className)} {...props} />
))
SheetTitle.displayName = 'SheetTitle'

const SheetDescription = React.forwardRef<
    React.ElementRef<typeof DialogPrimitive.Description>,
    React.ComponentPropsWithoutRef<typeof DialogPrimitive.Description>
>(({ className, ...props }, ref) => (
    <DialogPrimitive.Description ref={ref} className={cn('text-sm text-muted-foreground', className)} {...props} />
))
SheetDescription.displayName = 'SheetDescription'

export { Sheet, SheetTrigger, SheetClose, SheetContent, SheetTitle, SheetDescription }
