import { Link } from 'react-router-dom';
import { Server, LogOut, Menu, Sun, Moon, Monitor } from 'lucide-react';
import { Button } from '../ui/button';
import SmartAvatar from '../ui/SmartAvatar';
import { useAuth } from '../auth/AuthProvider';
import { logout } from '@/shared/services/api';
import {
    SimpleDropdown,
    SimpleDropdownItem,
} from '../ui/simple-dropdown';
import { useTheme } from '../theme/ThemeProvider';

interface HeaderProps {
    onMenuToggle: () => void;
}

function Header({ onMenuToggle }: HeaderProps) {
    const { user, isAuthenticated } = useAuth();
    const { setTheme, theme } = useTheme();

    if (!isAuthenticated) {
        return null;
    }

    return (
        <header className="border-b bg-background flex-shrink-0 z-50">
            <div className="px-3 sm:px-4 py-3 sm:py-4 flex items-center justify-between">
                {/* Mobile hamburger menu */}
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={onMenuToggle}
                    className="md:hidden h-9 w-9"
                    aria-label="Open menu"
                >
                    <Menu className="h-5 w-5" />
                </Button>

                {/* Logo */}
                <Link to="/apps" className="flex items-center space-x-2 hover:opacity-80 transition-opacity">
                    <Server className="h-6 w-6 text-primary" />
                    <span className="font-bold text-lg sm:text-xl">Selfhostly</span>
                </Link>

                {/* User Menu */}
                <div className="flex items-center">
                    <SimpleDropdown
                        trigger={
                            <Button variant="ghost" size="icon" className="relative">
                                <SmartAvatar user={user} size="sm" />
                            </Button>
                        }
                    >
                        <div className="py-1 min-w-[200px]">
                            <div className="px-3 py-2 border-b">
                                <p className="font-medium text-sm">{user?.name}</p>
                            </div>
                            <div className="px-2 py-1">
                                <p className="text-xs font-medium text-muted-foreground px-2 py-1">Theme</p>
                                <SimpleDropdownItem onClick={() => setTheme('light')}>
                                    <div className="flex items-center w-full">
                                        <Sun className="mr-2 h-4 w-4" />
                                        <span>Light</span>
                                        {theme === 'light' && <span className="ml-auto text-primary">✓</span>}
                                    </div>
                                </SimpleDropdownItem>
                                <SimpleDropdownItem onClick={() => setTheme('dark')}>
                                    <div className="flex items-center w-full">
                                        <Moon className="mr-2 h-4 w-4" />
                                        <span>Dark</span>
                                        {theme === 'dark' && <span className="ml-auto text-primary">✓</span>}
                                    </div>
                                </SimpleDropdownItem>
                                <SimpleDropdownItem onClick={() => setTheme('system')}>
                                    <div className="flex items-center w-full">
                                        <Monitor className="mr-2 h-4 w-4" />
                                        <span>System</span>
                                        {theme === 'system' && <span className="ml-auto text-primary">✓</span>}
                                    </div>
                                </SimpleDropdownItem>
                            </div>
                            <div className="border-t mt-1 pt-1">
                                <SimpleDropdownItem onClick={() => logout()}>
                                    <div className="flex items-center text-destructive">
                                        <LogOut className="h-4 w-4 mr-2" />
                                        Logout
                                    </div>
                                </SimpleDropdownItem>
                            </div>
                        </div>
                    </SimpleDropdown>
                </div>
            </div>
        </header>
    );
}

export default Header;
