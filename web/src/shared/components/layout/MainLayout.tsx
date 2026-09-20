import { useState, useEffect } from 'react';
import Sidebar from './Sidebar';
import Header from './Header';
import MobileTabBar from './MobileTabBar';
import JobBanner from './JobBanner';
import CommandPalette from './CommandPalette';
import { CommandPaletteProvider } from './CommandPaletteContext';
import { ErrorBoundary } from '@/shared/components/ErrorBoundary';
import { useLocation } from 'react-router-dom';
import { useEventStream } from '@/shared/hooks/useEventStream';
import { useMediaQuery } from '@/shared/hooks/useMediaQuery';

const TABLET_QUERY = '(min-width: 768px) and (max-width: 1023px)';

interface MainLayoutProps {
  children: React.ReactNode;
}

function MainLayout({ children }: MainLayoutProps) {
  useEventStream();
  const { pathname } = useLocation();
  // On a tablet the full sidebar leaves too little room for the page, so it shows as the narrow icon rail.
  const isTablet = useMediaQuery(TABLET_QUERY);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(() => {
    // Load collapsed state from localStorage
    const saved = localStorage.getItem('sidebar-collapsed');
    return saved ? JSON.parse(saved) : false;
  });

  // Persist collapsed state to localStorage
  useEffect(() => {
    localStorage.setItem('sidebar-collapsed', JSON.stringify(isSidebarCollapsed));
  }, [isSidebarCollapsed]);

  return (
    <CommandPaletteProvider>
    <div className="flex h-screen overflow-hidden">
      <Sidebar
        isCollapsed={isSidebarCollapsed || isTablet}
        onToggleCollapse={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
      />

      <div className="flex-1 flex flex-col overflow-hidden">
        <Header />

        {/* On phones the bottom tab bar covers the last 68px, so leave room for it. */}
        <main className="flex-1 overflow-y-auto bg-background pb-[calc(var(--mobile-nav-h)+32px)] md:pb-0">
          <div className="sticky top-0 z-20">
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
  );
}

export default MainLayout;
