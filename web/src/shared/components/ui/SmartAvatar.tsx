import { Avatar, AvatarFallback } from './avatar'
import { getInitials, stringToColor } from '@/shared/lib/avatar'
import { User } from '@/shared/services/api'

interface SmartAvatarProps {
    user: User | null
    size?: 'sm' | 'md' | 'lg' | 'xl'
    className?: string
}

function SmartAvatar({ user, size = 'md', className }: SmartAvatarProps) {
    if (!user) {
        return (
            <Avatar size={size} className={className}>
                <AvatarFallback>
                    <div className="h-4 w-4 bg-muted rounded-full animate-pulse" />
                </AvatarFallback>
            </Avatar>
        )
    }

    // Generate initials for fallback
    const initials = getInitials(user.name)

    // Create a simpler fallback component with background color
    const customFallback = (
        <div
            className="h-full w-full rounded-full flex items-center justify-center text-xs font-medium text-white"
            style={{
                backgroundColor: stringToColor(user.name),
                color: 'white'
            }}
        >
            {initials}
        </div>
    )

    return (
        <Avatar
            src={user.picture}
            name={user.name}
            size={size}
            className={`${className}`}
            fallback={customFallback}
        />
    )
}

export default SmartAvatar
