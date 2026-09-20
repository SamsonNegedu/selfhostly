import * as React from 'react'
import * as RadioGroupPrimitive from '@radix-ui/react-radio-group'

import { cn } from '@/shared/lib/utils'

interface SegmentedControlOption {
    value: string
    label: string
    icon?: React.ReactNode
}

interface SegmentedControlProps {
    options: SegmentedControlOption[]
    value: string
    onValueChange: (value: string) => void
    'aria-label': string
    className?: string
}

// A small set of mutually exclusive choices, such as Cards or Table. It is a radio group underneath,
// so arrow keys move the selection and screen readers announce it as a group of options.
function SegmentedControl({ options, value, onValueChange, className, ...props }: SegmentedControlProps) {
    return (
        <RadioGroupPrimitive.Root
            value={value}
            onValueChange={onValueChange}
            orientation="horizontal"
            className={cn(
                'inline-flex items-center gap-0 rounded-[9px] border border-border bg-card p-[3px]',
                className,
            )}
            {...props}
        >
            {options.map((option) => (
                <RadioGroupPrimitive.Item
                    key={option.value}
                    value={option.value}
                    className="inline-flex h-[30px] min-h-[44px] min-w-[44px] items-center justify-center gap-2 whitespace-nowrap rounded-md px-3 text-[13px] font-medium text-muted-foreground ring-offset-background transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground sm:min-h-0 sm:min-w-0"
                >
                    {option.icon}
                    {option.label}
                </RadioGroupPrimitive.Item>
            ))}
        </RadioGroupPrimitive.Root>
    )
}

export { SegmentedControl }
