import * as React from 'react'
import { Link } from 'react-router-dom'
import { MoreHorizontal, RefreshCw, Trash2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/shared/components/ui/DropdownMenu'
import { appHref } from '@/shared/lib/routes'
import type { App } from '@/shared/types/api'
import type { useFleetActions } from '../hooks/useFleetActions'

interface FleetAppMenuProps {
    app: App
    actions: ReturnType<typeof useFleetActions>
    // The app's node is not answering, so there is nothing to update.
    unreachable: boolean
    // An action on this app is already under way.
    busy: boolean
    // The app is being updated or started right now.
    inProgress: boolean
    // Extra items for the view, shown after Details.
    children?: React.ReactNode
}

// The "more actions" menu of an app on Fleet, the same on the cards, the rows and the table.
function FleetAppMenu({ app, actions, unreachable, busy, inProgress, children }: FleetAppMenuProps) {
    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" aria-label={`More actions for ${app.name}`}>
                    <MoreHorizontal className="h-4 w-4" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44">
                <DropdownMenuItem asChild>
                    <Link to={appHref(app)}>Details</Link>
                </DropdownMenuItem>
                {children}
                {!unreachable && (
                    <DropdownMenuItem onSelect={() => actions.requestUpdate(app)} disabled={busy || inProgress}>
                        <RefreshCw className="mr-2 h-4 w-4" />
                        Update
                    </DropdownMenuItem>
                )}
                <DropdownMenuSeparator />
                <DropdownMenuItem
                    onSelect={() => actions.requestDelete(app)}
                    disabled={busy}
                    className="text-destructive focus:text-destructive"
                >
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete
                </DropdownMenuItem>
            </DropdownMenuContent>
        </DropdownMenu>
    )
}

export default FleetAppMenu
