import { Link } from 'react-router-dom';
import { ChevronLeft, Search, Server } from 'lucide-react';
import { Kbd } from '../ui/Kbd';
import { useCommandPalette } from './CommandPaletteContext';
import { useAuth } from '../auth/AuthProvider';
import NotificationsMenu from './NotificationsMenu';
import ScopeSwitcher from './ScopeSwitcher';
import TopbarBreadcrumbs from './TopbarBreadcrumbs';
import { useCrumbs } from './useCrumbs';
import UserMenu from './UserMenu';

const isApplePlatform = () => /Mac|iPhone|iPad/.test(navigator.platform);

function Header() {
    const { isAuthenticated } = useAuth();
    const { setOpen: openPalette } = useCommandPalette();
    const crumbs = useCrumbs();

    if (!isAuthenticated) {
        return null;
    }

    // On a phone a deep page shows a back arrow to its parent, and a top level page shows the logo.
    const title = crumbs[crumbs.length - 1]?.label ?? 'Selfhostly';
    // The back arrow goes to the nearest earlier crumb that links somewhere (a node name has no page).
    const parent = crumbs.slice(0, -1).reverse().find((crumb) => crumb.to);

    return (
        <header className="flex h-[60px] shrink-0 items-center gap-2 border-b border-border bg-background px-2 sm:gap-3 sm:px-8 md:px-8 z-30">
            {parent?.to ? (
                <Link
                    to={parent.to}
                    className="flex h-[44px] w-[44px] items-center justify-center rounded-lg hover:bg-accent md:hidden"
                    aria-label={`Back to ${parent.label}`}
                >
                    <ChevronLeft className="h-5 w-5" />
                </Link>
            ) : (
                <Link to="/apps" className="flex h-[44px] w-[44px] items-center justify-center md:hidden" aria-label="Selfhostly home">
                    <span className="flex h-[30px] w-[30px] items-center justify-center rounded-lg bg-primary text-primary-foreground">
                        <Server className="h-[17px] w-[17px]" />
                    </span>
                </Link>
            )}
            <p className="min-w-0 flex-1 truncate text-base font-semibold md:hidden">{title}</p>

            <TopbarBreadcrumbs className="hidden min-w-0 md:block" />

            <div className="ml-auto flex items-center gap-2 max-md:ml-0">
                <button
                    type="button"
                    onClick={() => openPalette(true)}
                    aria-haspopup="dialog"
                    className="hidden h-9 w-[340px] items-center gap-2.5 rounded-lg border border-input bg-card px-3 text-[13px] text-muted-foreground ring-offset-background transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 lg:flex"
                >
                    <Search className="h-[15px] w-[15px]" />
                    <span className="flex-1 text-left">Search apps, actions, pages</span>
                    <Kbd>{isApplePlatform() ? 'Cmd K' : 'Ctrl K'}</Kbd>
                </button>
                <button
                    type="button"
                    onClick={() => openPalette(true)}
                    aria-label="Search"
                    aria-haspopup="dialog"
                    className="flex h-[44px] w-[44px] items-center justify-center rounded-lg border border-input bg-card text-muted-foreground hover:bg-accent md:h-9 md:w-9 lg:hidden"
                >
                    <Search className="h-4 w-4" />
                </button>
                <ScopeSwitcher />
                <NotificationsMenu />
                <div className="hidden md:block">
                    <UserMenu />
                </div>
            </div>
        </header>
    );
}

export default Header;
