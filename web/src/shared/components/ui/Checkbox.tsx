import * as React from 'react'
import * as CheckboxPrimitive from '@radix-ui/react-checkbox'
import { Check } from 'lucide-react'

import { cn } from '@/shared/lib/utils'

interface CheckboxProps extends React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root> {
    checked?: boolean
    onCheckedChange?: (checked: boolean) => void
}

// The 16px box is what people see. On small screens an invisible 44px hit area surrounds it.
const Checkbox = React.forwardRef<React.ElementRef<typeof CheckboxPrimitive.Root>, CheckboxProps>(
    ({ className, ...props }, ref) => (
        <CheckboxPrimitive.Root
            ref={ref}
            className={cn(
                "peer relative h-4 w-4 shrink-0 rounded-[4px] border border-input bg-card ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-primary data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground max-sm:after:absolute max-sm:after:-inset-[16px] max-sm:after:content-['']",
                className,
            )}
            {...props}
        >
            <CheckboxPrimitive.Indicator className={cn('flex items-center justify-center text-current')}>
                <Check className="h-3.5 w-3.5" />
            </CheckboxPrimitive.Indicator>
        </CheckboxPrimitive.Root>
    ),
)
Checkbox.displayName = CheckboxPrimitive.Root.displayName

export { Checkbox }
