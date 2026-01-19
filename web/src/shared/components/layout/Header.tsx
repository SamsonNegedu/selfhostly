import { Link, useLocation } from 'react-router-dom';
import { Server, Settings, Plus, LogOut, Cloud } from 'lucide-react';
import { Button } from '../ui/button';
import SmartAvatar from '../ui/SmartAvatar';
import { useAuth } from '../auth/AuthProvider';
import { logout } from '@/shared/services/api';

function Header() {
    const location = useLocation();
    const { user, isAuthenticated } = useAuth();

    if (!isAuthenticated) {
        return null;
    }

    return (
        <header className="border-b bg-background">
            <div className="container mx-auto px-4 py-4 flex items-center justify-between">
                <div className="flex items-center space-x-2">
                    <Server className="h-6 w-6 text-primary" />
                    <span className="font-bold text-xl">Self-Host Automaton</span>
                </div>

                <nav className="flex items-center space-x-4">
                    <Link to="/dashboard">
                        <Button
                            variant={location.pathname === '/dashboard' ? 'default' : 'ghost'}
                        >
                            Dashboard
                        </Button>
                    </Link>
                    <Link to="/apps/new">
                        <Button>
                            <Plus className="h-4 w-4 mr-2" />
                            New App
                        </Button>
                    </Link>
                    <Link to="/cloudflare">
                        <Button
                            variant={location.pathname === '/cloudflare' ? 'default' : 'ghost'}
                            size="icon"
                        >
                            <Cloud className="h-5 w-5" />
                        </Button>
                    </Link>
                    <Link to="/settings">
                        <Button
                            variant={location.pathname === '/settings' ? 'default' : 'ghost'}
                            size="icon"
                        >
                            <Settings className="h-5 w-5" />
                        </Button>
                    </Link>

                    {/* User menu */}
                    <div className="flex items-center space-x-3 ml-4 pl-4 border-l">
                        <SmartAvatar user={user} size="md" />
                        <span className="text-sm font-medium hidden sm:inline">
                            {user?.name}
                        </span>
                        <Button variant="ghost" size="icon" onClick={() => logout()} title="Logout">
                            <LogOut className="h-4 w-4" />
                        </Button>
                    </div>
                </nav>
            </div>
        </header>
    );
}

export default Header;
