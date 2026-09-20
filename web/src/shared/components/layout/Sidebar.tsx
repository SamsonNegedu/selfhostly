import { Link, useLocation } from 'react-router-dom'
import { Activity, ChevronLeft, ChevronRight, Globe, LayoutGrid, Server, SlidersHorizontal } from 'lucide-react'
import { Button } from '../ui/Button'
import ClusterStatus from './ClusterStatus'
import { useAttention } from '@/shared/hooks/useAttention'
import { ROUTES } from '@/shared/lib/routes'

interface SidebarProps {
    isCollapsed: boolean
    onToggleCollapse: () => void
}

const navItems = [
    { path: ROUTES.fleet, label: 'Fleet', icon: LayoutGrid, showsAttention: true },
    { path: ROUTES.nodes, label: 'Nodes', icon: Server },
    { path: ROUTES.access, label: 'Access', icon: Globe },
    { path: ROUTES.insights, label: 'Insights', icon: Activity },
    { path: ROUTES.settings, label: 'Settings', icon: SlidersHorizontal },
]

// The desktop navigation. Phones use the bottom tab bar instead.
function Sidebar({ isCollapsed, onToggleCollapse }: SidebarProps) {
    const location = useLocation()
    const { failedApps } = useAttention()

    const isActive = (path: string) => {
        if (path === ROUTES.fleet) {
            return location.pathname === '/' || location.pathname.startsWith('/apps')
        }
        return location.pathname.startsWith(path)
    }

    return (
        <aside
            className={`
        hidden md:flex flex-col shrink-0
        bg-background border-r border-border
        transition-all duration-300 ease-in-out
        ${isCollapsed ? 'w-16' : 'w-[216px]'}
      `}
            aria-label="Main navigation"
        >
            <Link
                to="/apps"
                className={`flex items-center gap-2.5 pb-4 pt-5 ${isCollapsed ? 'justify-center' : 'px-4'}`}
                aria-label="Selfhostly home"
            >
                <span className="flex h-[30px] w-[30px] shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                    <Server className="h-[17px] w-[17px]" />
                </span>
                {!isCollapsed && <span className="text-base font-semibold">Selfhostly</span>}
            </Link>

            <nav className="flex-1 overflow-y-auto px-3 space-y-1">
                {navItems.map((item) => {
                    const Icon = item.icon
                    const active = isActive(item.path)
                    const showBadge = item.showsAttention && failedApps > 0

                    return (
                        <Link
                            key={item.path}
                            to={item.path}
                            className={`
                flex min-h-[38px] items-center gap-3 rounded-lg px-3
                text-sm transition-colors duration-150
                ${
                    active
                        ? 'bg-primary text-primary-foreground font-semibold'
                        : 'font-medium text-foreground hover:bg-accent hover:text-accent-foreground'
                }
                ${isCollapsed ? 'justify-center px-0' : ''}
              `}
                            aria-label={showBadge ? `${item.label}, ${failedApps} failed` : item.label}
                            aria-current={active ? 'page' : undefined}
                        >
                            <Icon className="h-[17px] w-[17px] flex-shrink-0" />
                            {!isCollapsed && <span className="flex-1">{item.label}</span>}
                            {showBadge && !isCollapsed && (
                                <span
                                    aria-hidden="true"
                                    className="flex h-5 min-w-[20px] items-center justify-center rounded-full bg-destructive px-1 text-caption font-semibold text-destructive-foreground"
                                >
                                    {failedApps}
                                </span>
                            )}
                        </Link>
                    )
                })}
            </nav>

            <div className="space-y-2 p-3">
                <ClusterStatus collapsed={isCollapsed} />
                <Button
                    variant="ghost"
                    size="sm"
                    onClick={onToggleCollapse}
                    className={`w-full ${isCollapsed ? 'justify-center' : 'justify-between'}`}
                    aria-label={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
                >
                    {!isCollapsed && <span className="text-sm">Collapse</span>}
                    {isCollapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
                </Button>
            </div>
        </aside>
    )
}

export default Sidebar
