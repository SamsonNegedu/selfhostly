import { useState, useEffect, useRef } from 'react'
import Sidebar from './Sidebar'
import Header from './Header'
import MobileTabBar from './MobileTabBar'
import JobBanner from './JobBanner'
import UpdateProgress from './UpdateProgress'
import OfflineBanner from './OfflineBanner'
import CommandPalette from './CommandPalette'
import { CommandPaletteProvider } from './CommandPaletteContext'
import { ErrorBoundary } from '@/shared/components/ErrorBoundary'
import { useLocation } from 'react-router-dom'
import { useEventStream } from '@/shared/hooks/useEventStream'
import { useMediaQuery } from '@/shared/hooks/useMediaQuery'
import { useDocumentTitle } from '@/shared/hooks/useDocumentTitle'
import { useCrumbs } from './useCrumbs'

const TABLET_QUERY = '(min-width: 768px) and (max-width: 1023px)'
const MAIN_ID = 'main-content'
// How many of the deepest crumbs name the page in the tab title. The root "Fleet" crumb is left out on app pages.
const TITLE_CRUMBS = 2
const NOT_FOUND_TITLE = 'Page not found'

interface MainLayoutProps {
    children: React.ReactNode
}

function MainLayout({ children }: MainLayoutProps) {
    useEventStream()
    const { pathname } = useLocation()
    // On a tablet the full sidebar leaves too little room for the page, so it shows as the narrow icon rail.
    const isTablet = useMediaQuery(TABLET_QUERY)
    const mainRef = useRef<HTMLElement>(null)
    const crumbs = useCrumbs()
    const names = crumbs.map((crumb) => crumb.label).filter((label) => label !== 'Fleet' || crumbs.length === 1)
    // No crumbs means the address matches no page.
    useDocumentTitle(...(names.length > 0 ? names.slice(-TITLE_CRUMBS).reverse() : [NOT_FOUND_TITLE]))

    // A new page starts at the top with focus on the page, so keyboard and screen reader users are not left on the
    // link they used. The first load is skipped so the browser's own focus handling still applies.
    const previousPath = useRef(pathname)
    useEffect(() => {
        if (previousPath.current === pathname) return
        previousPath.current = pathname
        mainRef.current?.scrollTo({ top: 0 })
        mainRef.current?.focus({ preventScroll: true })
    }, [pathname])
    const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(() => {
        // Load collapsed state from localStorage
        const saved = localStorage.getItem('sidebar-collapsed')
        return saved ? JSON.parse(saved) : false
    })

    // Persist collapsed state to localStorage
    useEffect(() => {
        localStorage.setItem('sidebar-collapsed', JSON.stringify(isSidebarCollapsed))
    }, [isSidebarCollapsed])

    return (
        <CommandPaletteProvider>
            <a
                href={`#${MAIN_ID}`}
                onClick={(event) => {
                    event.preventDefault()
                    mainRef.current?.focus()
                }}
                className="sr-only focus:not-sr-only focus:fixed focus:left-3 focus:top-3 focus:z-[60] focus:rounded-lg focus:bg-card focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:shadow-lg"
            >
                Skip to content
            </a>
            <div className="flex h-dvh overflow-hidden">
                <Sidebar
                    isCollapsed={isSidebarCollapsed || isTablet}
                    onToggleCollapse={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
                />

                <div className="flex-1 flex flex-col overflow-hidden">
                    <Header />

                    {/* On phones the bottom tab bar covers the last 68px, so leave room for it. */}
                    <main
                        id={MAIN_ID}
                        ref={mainRef}
                        tabIndex={-1}
                        className="flex-1 overflow-y-auto bg-background focus:outline-none pb-[calc(var(--mobile-nav-h)+var(--mobile-content-gap))] md:pb-0"
                    >
                        <div className="sticky top-0 z-20">
                            <OfflineBanner />
                            <UpdateProgress />
                            <JobBanner />
                        </div>
                        <div className="container mx-auto px-3 sm:px-12 py-4 sm:py-6 md:py-8">
                            <ErrorBoundary resetKey={pathname}>{children}</ErrorBoundary>
                        </div>
                    </main>
                </div>

                <MobileTabBar />
                <CommandPalette />
            </div>
        </CommandPaletteProvider>
    )
}

export default MainLayout
