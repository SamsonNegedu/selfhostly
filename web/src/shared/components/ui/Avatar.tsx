import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/shared/lib/utils'

const avatarVariants = cva(
    'inline-flex items-center justify-center rounded-full text-sm font-medium bg-muted text-muted-foreground',
    {
        variants: {
            size: {
                sm: 'h-8 w-8 text-xs',
                md: 'h-10 w-10 text-sm',
                lg: 'h-12 w-12 text-base',
                xl: 'h-16 w-16 text-lg',
            },
        },
        defaultVariants: {
            size: 'md',
        },
    },
)

interface AvatarProps extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof avatarVariants> {
    src?: string
    name?: string
    fallback?: React.ReactNode
}

const Avatar = React.forwardRef<HTMLDivElement, AvatarProps>(
    ({ className, src, name, fallback, size, ...props }, ref) => {
        // Remembers which address failed, so a new src is tried again.
        const [failedSrc, setFailedSrc] = React.useState<string | undefined>()

        // If src is not provided, always use fallback
        const shouldUseFallback = !src || failedSrc === src

        if (shouldUseFallback) {
            // If custom fallback is provided, don't apply the default avatar styles
            const fallbackClassName = fallback
                ? cn(
                      size === 'sm'
                          ? 'h-8 w-8'
                          : size === 'md'
                            ? 'h-10 w-10'
                            : size === 'lg'
                              ? 'h-12 w-12'
                              : 'h-16 w-16',
                      className,
                  )
                : cn(avatarVariants({ size }), className)

            return (
                <div ref={ref} className={fallbackClassName} {...props}>
                    {fallback}
                </div>
            )
        }

        return (
            <div ref={ref} className={cn('relative', className)} {...props}>
                <img
                    src={src}
                    alt={name || 'User avatar'}
                    className="h-full w-full rounded-full object-cover"
                    onError={() => setFailedSrc(src)}
                />
            </div>
        )
    },
)
Avatar.displayName = 'Avatar'

const AvatarFallback = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
    ({ className, ...props }, ref) => (
        <div ref={ref} className={cn('h-full w-full flex items-center justify-center', className)} {...props} />
    ),
)
AvatarFallback.displayName = 'AvatarFallback'

export { Avatar, AvatarFallback }
