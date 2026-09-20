import { useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Activity, Globe, LayoutGrid, Plus, Server, SlidersHorizontal } from 'lucide-react'
import MoreSheet from './MoreSheet'
import { useAttention } from '@/shared/hooks/useAttention'
import { ROUTES } from '@/shared/lib/routes'

const TAB_ITEMS = [
    { path: ROUTES.fleet, label: 'Fleet', icon: LayoutGrid, showsAttention: true },
    { path: ROUTES.nodes, label: 'Nodes', icon: Server },
    { path: ROUTES.access, label: 'Access', icon: Globe },
    { path: ROUTES.insights, label: 'Insights', icon: Activity },
] as const

const FLEET_PATH = ROUTES.fleet

// The phone navigation: four destinations and a More sheet for the rest. It replaces the side drawer
// so the main places are always one tap away.
function MobileTabBar() {
    const { pathname } = useLocation()
    const { failedApps } = useAttention()
    const [moreOpen, setMoreOpen] = useState(false)

    const isActive = (path: string) =>
        path === FLEET_PATH ? pathname === '/' || pathname.startsWith(FLEET_PATH) : pathname.startsWith(path)
    const moreActive = pathname.startsWith(ROUTES.settings)

    return (
        <>
            {pathname === FLEET_PATH && (
                <Link
                    to={ROUTES.newApp}
                    className="fixed bottom-[calc(var(--mobile-nav-h)+16px)] right-4 z-30 flex h-[52px] items-center gap-2 rounded-full bg-primary px-5 font-semibold text-primary-foreground shadow-lg md:hidden"
                >
                    <Plus className="h-[18px] w-[18px]" />
                    New app
                </Link>
            )}

            <nav
                aria-label="Primary"
                className="fixed inset-x-0 bottom-0 z-40 flex h-[var(--mobile-nav-h)] pb-[env(safe-area-inset-bottom,0px)] border-t border-border bg-card md:hidden"
            >
                {TAB_ITEMS.map((item) => {
                    const Icon = item.icon
                    const active = isActive(item.path)
                    const showBadge = 'showsAttention' in item && item.showsAttention && failedApps > 0

                    return (
                        <Link
                            key={item.path}
                            to={item.path}
                            aria-current={active ? 'page' : undefined}
                            aria-label={showBadge ? `${item.label}, ${failedApps} failed` : item.label}
                            className={`relative flex min-h-[44px] min-w-[44px] flex-1 flex-col items-center justify-center gap-[3px] text-[11px] ${active ? 'font-semibold text-foreground' : 'font-medium text-muted-foreground'}`}
                        >
                            <span className="relative">
                                <Icon className="h-[22px] w-[22px]" strokeWidth={active ? 2.4 : 2} />
                                {showBadge && (
                                    <span
                                        aria-hidden="true"
                                        className="absolute -right-1 -top-0.5 h-2 w-2 rounded-full border-2 border-card bg-status-err"
                                    />
                                )}
                            </span>
                            {item.label}
                        </Link>
                    )
                })}
                <button
                    type="button"
                    onClick={() => setMoreOpen(true)}
                    aria-haspopup="dialog"
                    className={`flex min-h-[44px] min-w-[44px] flex-1 flex-col items-center justify-center gap-[3px] text-[11px] ${moreActive ? 'font-semibold text-foreground' : 'font-medium text-muted-foreground'}`}
                >
                    <SlidersHorizontal className="h-[22px] w-[22px]" strokeWidth={moreActive ? 2.4 : 2} />
                    More
                </button>
            </nav>

            <MoreSheet open={moreOpen} onOpenChange={setMoreOpen} />
        </>
    )
}

export default MobileTabBar
