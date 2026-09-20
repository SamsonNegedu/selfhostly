import { Link } from 'react-router-dom'
import { AlertTriangle, Bell, Server } from 'lucide-react'
import { Button } from '../ui/Button'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '../ui/DropdownMenu'
import { useAttention } from '@/shared/hooks/useAttention'
import type { AttentionItem } from '@/shared/lib/attention'

const TILE_CLASSES: Record<AttentionItem['kind'], string> = {
    'app-error': 'bg-status-err-bg text-status-err-fg',
    'node-offline': 'bg-status-warn-bg text-status-warn-fg',
}

// The bell lists whatever needs a person right now. It is worked out from live app and node status,
// so there is nothing to mark as read: an item disappears when the problem does.
function NotificationsMenu() {
    const { items } = useAttention()
    const count = items.length

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button
                    variant="outline"
                    size="icon"
                    className="relative"
                    aria-label={count > 0 ? `Notifications, ${count} need attention` : 'Notifications, none'}
                >
                    <Bell className="h-4 w-4" />
                    {count > 0 && (
                        <span
                            aria-hidden="true"
                            className="absolute right-1.5 top-1.5 h-2 w-2 rounded-full border-2 border-card bg-status-err"
                        />
                    )}
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-[360px] max-w-[calc(100vw-2rem)]">
                <DropdownMenuLabel>Needs attention</DropdownMenuLabel>
                <DropdownMenuSeparator />
                {count === 0 ? (
                    <p className="px-2 py-6 text-center text-sm text-muted-foreground">
                        All clear. Nothing needs you right now.
                    </p>
                ) : (
                    items.map((item) => {
                        const Icon = item.kind === 'app-error' ? AlertTriangle : Server
                        return (
                            <DropdownMenuItem key={item.id} asChild className="h-auto items-start gap-3 py-2.5">
                                <Link to={item.href}>
                                    <span
                                        aria-hidden="true"
                                        className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-lg ${TILE_CLASSES[item.kind]}`}
                                    >
                                        <Icon className="h-4 w-4" />
                                    </span>
                                    <span className="flex min-w-0 flex-col">
                                        <span className="text-[13.5px] font-medium">{item.title}</span>
                                        <span className="text-xs text-muted-foreground">{item.detail}</span>
                                    </span>
                                </Link>
                            </DropdownMenuItem>
                        )
                    })
                )}
            </DropdownMenuContent>
        </DropdownMenu>
    )
}

export default NotificationsMenu
