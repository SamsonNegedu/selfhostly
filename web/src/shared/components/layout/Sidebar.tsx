import { useEffect } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Server, Plus, Cloud, Activity, Settings, ChevronLeft, ChevronRight, X } from 'lucide-react';
import { Button } from '../ui/button';

interface SidebarProps {
  isOpen: boolean;
  onClose: () => void;
  isCollapsed: boolean;
  onToggleCollapse: () => void;
}

const navItems = [
  { path: '/dashboard', label: 'Apps', icon: Server },
  { path: '/apps/new', label: 'New App', icon: Plus },
  { path: '/cloudflare', label: 'Cloudflare', icon: Cloud },
  { path: '/monitoring', label: 'Monitoring', icon: Activity },
  { path: '/settings', label: 'Settings', icon: Settings },
];

function Sidebar({ isOpen, onClose, isCollapsed, onToggleCollapse }: SidebarProps) {
  const location = useLocation();

  // Close sidebar on ESC key (mobile only)
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, onClose]);

  // Prevent body scroll when mobile sidebar is open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  const isActive = (path: string) => {
    if (path === '/dashboard') {
      return location.pathname === '/dashboard' || location.pathname === '/';
    }
    return location.pathname.startsWith(path);
  };

  return (
    <>
      {/* Backdrop for mobile */}
      {isOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-30 md:hidden transition-opacity duration-200"
          onClick={onClose}
          aria-hidden="true"
        />
      )}

      {/* Sidebar */}
      <aside
        className={`
          fixed md:static inset-y-0 left-0 z-40
          bg-card border-r border-border
          transition-all duration-300 ease-in-out
          flex flex-col
          ${isCollapsed ? 'md:w-16' : 'md:w-60'}
          ${isOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}
          w-60
        `}
        role="navigation"
        aria-label="Main navigation"
      >
        {/* Mobile close button */}
        <div className="md:hidden flex items-center justify-between p-4 border-b border-border">
          <span className="font-semibold text-lg">Menu</span>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClose}
            aria-label="Close menu"
          >
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Navigation items */}
        <nav className="flex-1 overflow-y-auto p-3 space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = isActive(item.path);

            return (
              <Link
                key={item.path}
                to={item.path}
                onClick={() => {
                  // Close mobile sidebar after navigation
                  if (window.innerWidth < 768) {
                    onClose();
                  }
                }}
                className={`
                  flex items-center gap-3 px-3 py-2.5 rounded-lg
                  transition-colors duration-150
                  ${active
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                  }
                  ${isCollapsed ? 'md:justify-center' : ''}
                `}
                aria-label={item.label}
                aria-current={active ? 'page' : undefined}
              >
                <Icon className="h-5 w-5 flex-shrink-0" />
                <span
                  className={`
                    font-medium transition-opacity duration-200
                    ${isCollapsed ? 'md:hidden' : ''}
                  `}
                >
                  {item.label}
                </span>
              </Link>
            );
          })}
        </nav>

        {/* Collapse toggle (desktop only) */}
        <div className="hidden md:flex p-3 border-t border-border">
          <Button
            variant="ghost"
            size="sm"
            onClick={onToggleCollapse}
            className={`w-full ${isCollapsed ? 'justify-center' : 'justify-between'}`}
            aria-label={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {!isCollapsed && <span className="text-sm">Collapse</span>}
            {isCollapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <ChevronLeft className="h-4 w-4" />
            )}
          </Button>
        </div>
      </aside>
    </>
  );
}

export default Sidebar;
