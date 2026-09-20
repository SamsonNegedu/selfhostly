import { Link } from 'react-router-dom'
import { ChevronRight, LogOut, Monitor, Moon, SlidersHorizontal, Sun } from 'lucide-react'
import { Sheet, SheetClose, SheetContent, SheetDescription, SheetTitle } from '../ui/Sheet'
import { SegmentedControl } from '../ui/SegmentedControl'
import ClusterStatus from './ClusterStatus'
import { useAuth } from '../auth/AuthProvider'
import { useTheme } from '../theme/ThemeProvider'
import { logout } from '@/shared/services/api'

interface MoreSheetProps {
    open: boolean
    onOpenChange: (open: boolean) => void
}

type ThemeChoice = 'light' | 'dark' | 'system'

const THEME_OPTIONS = [
    { value: 'dark', label: 'Dark', icon: <Moon className="h-4 w-4" /> },
    { value: 'light', label: 'Light', icon: <Sun className="h-4 w-4" /> },
    { value: 'system', label: 'System', icon: <Monitor className="h-4 w-4" /> },
]

const rowClasses = 'flex min-h-[52px] items-center gap-3 rounded-lg px-2 text-[15px]'

// Everything that is not one of the four main destinations, in a sheet that rises from the bottom.
function MoreSheet({ open, onOpenChange }: MoreSheetProps) {
    const { user, authEnabled } = useAuth()
    const { theme, setTheme } = useTheme()

    return (
        <Sheet open={open} onOpenChange={onOpenChange}>
            <SheetContent side="bottom" className="md:hidden">
                <SheetTitle>{user?.name ?? 'More'}</SheetTitle>
                <SheetDescription className="sr-only">Settings, theme and account</SheetDescription>

                <SheetClose asChild>
                    <Link to="/settings" className={rowClasses}>
                        <SlidersHorizontal className="h-5 w-5" />
                        <span className="flex-1">Settings</span>
                        <ChevronRight className="h-4 w-4 text-muted-foreground" />
                    </Link>
                </SheetClose>

                <div className={rowClasses}>
                    <span className="flex-1">Theme</span>
                    <SegmentedControl
                        aria-label="Theme"
                        options={THEME_OPTIONS}
                        value={theme}
                        onValueChange={(value) => setTheme(value as ThemeChoice)}
                    />
                </div>

                {authEnabled && (
                    <button type="button" onClick={() => logout()} className={`${rowClasses} text-destructive`}>
                        <LogOut className="h-5 w-5" />
                        Sign out
                    </button>
                )}

                <div className="mt-1">
                    <ClusterStatus collapsed={false} />
                </div>
            </SheetContent>
        </Sheet>
    )
}

export default MoreSheet
