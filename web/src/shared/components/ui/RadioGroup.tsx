import * as React from 'react'
import * as RadioGroupPrimitive from '@radix-ui/react-radio-group'

import { cn } from '@/shared/lib/utils'

const RadioGroup = React.forwardRef<
    React.ElementRef<typeof RadioGroupPrimitive.Root>,
    React.ComponentPropsWithoutRef<typeof RadioGroupPrimitive.Root>
>(({ className, ...props }, ref) => (
    <RadioGroupPrimitive.Root className={cn('grid gap-2', className)} {...props} ref={ref} />
))
RadioGroup.displayName = RadioGroupPrimitive.Root.displayName

const RadioGroupItem = React.forwardRef<
    React.ElementRef<typeof RadioGroupPrimitive.Item>,
    React.ComponentPropsWithoutRef<typeof RadioGroupPrimitive.Item>
>(({ className, ...props }, ref) => (
    <RadioGroupPrimitive.Item
        ref={ref}
        className={cn(
            'relative aspect-square h-[16px] w-[16px] shrink-0 rounded-full border-2 border-input bg-card ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-primary',
            className,
        )}
        {...props}
    >
        <RadioGroupPrimitive.Indicator className="flex items-center justify-center">
            <span className="block h-[6px] w-[6px] rounded-full bg-primary" />
        </RadioGroupPrimitive.Indicator>
    </RadioGroupPrimitive.Item>
))
RadioGroupItem.displayName = RadioGroupPrimitive.Item.displayName

interface RadioCardProps extends Omit<
    React.ComponentPropsWithoutRef<typeof RadioGroupPrimitive.Item>,
    'children' | 'title'
> {
    title: string
    description?: string
}

// A selectable card: the whole card is the click target and the selected one gets a strong border.
const RadioCard = React.forwardRef<React.ElementRef<typeof RadioGroupPrimitive.Item>, RadioCardProps>(
    ({ title, description, className, id, ...props }, ref) => {
        const autoId = React.useId()
        const itemId = id ?? autoId

        return (
            <label
                htmlFor={itemId}
                className={cn(
                    'flex min-h-[44px] cursor-pointer items-start gap-3 rounded-[10px] border border-border bg-card p-3.5 transition-colors hover:bg-accent/50 has-[[data-state=checked]]:border-primary has-[[data-state=checked]]:ring-1 has-[[data-state=checked]]:ring-primary has-[:disabled]:cursor-not-allowed has-[:disabled]:opacity-50',
                    className,
                )}
            >
                <RadioGroupItem ref={ref} id={itemId} className="mt-0.5" {...props} />
                <span className="flex flex-col">
                    <span className="text-[13.5px] font-medium">{title}</span>
                    {description && <span className="text-[12.5px] text-muted-foreground">{description}</span>}
                </span>
            </label>
        )
    },
)
RadioCard.displayName = 'RadioCard'

export { RadioGroup, RadioCard }
