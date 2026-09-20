import * as React from 'react'

type Variant = 'default' | 'destructive' | 'danger' | 'outline' | 'secondary' | 'ghost' | 'link'
type Size = 'default' | 'sm' | 'lg' | 'icon'

interface ButtonVariants {
    variant: Record<Variant, string>
    size: Record<Size, string>
}

// Every size keeps a 44px minimum on small screens so it stays tappable, and relaxes at sm and up.
const TAP_TARGET = 'min-h-[44px] min-w-[44px] sm:min-h-0 sm:min-w-0'

const buttonVariants: ButtonVariants = {
    variant: {
        default: 'bg-primary text-primary-foreground hover:bg-primary/90',
        destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90',
        // Tinted red for destructive actions that should not shout, such as Stop exposing.
        danger: 'border border-status-err-fg/30 bg-status-err-bg text-status-err-fg hover:bg-status-err-bg/70',
        outline: 'border border-border bg-card text-foreground hover:bg-accent hover:text-accent-foreground',
        secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
        ghost: 'hover:bg-accent hover:text-accent-foreground',
        link: 'text-primary underline-offset-4 hover:underline',
    },
    size: {
        default: `h-[34px] px-3.5 ${TAP_TARGET}`,
        sm: `h-[30px] px-3 text-xs ${TAP_TARGET}`,
        lg: 'h-11 min-h-[44px] min-w-[44px] px-5 text-sm',
        icon: `h-[34px] w-[34px] ${TAP_TARGET}`,
    },
}

export interface ButtonProps extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'variant' | 'size'> {
    variant?: Variant
    size?: Size
}

// The button look as a class string, so a link can wear it too: <Link className={buttonClasses({ variant: "outline" })}>.
export function buttonClasses({
    variant = 'default',
    size = 'default',
    className,
}: { variant?: Variant; size?: Size; className?: string } = {}) {
    return [
        'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-lg text-[13px] font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 [&_svg]:shrink-0',
        buttonVariants.variant[variant],
        buttonVariants.size[size],
        className,
    ]
        .filter(Boolean)
        .join(' ')
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
    ({ className, variant = 'default', size = 'default', ...props }, ref) => {
        const classes = buttonClasses({ variant, size, className })

        return (
            <button ref={ref} className={classes} {...props}>
                {props.children}
            </button>
        )
    },
)
Button.displayName = 'Button'
