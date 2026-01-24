import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Server, Settings, Plus, LogOut, Cloud, Menu, X, Sun, Moon, Monitor } from 'lucide-react';
import { Button } from '../ui/button';
import SmartAvatar from '../ui/SmartAvatar';
import { useAuth } from '../auth/AuthProvider';
import { logout } from '@/shared/services/api';
import {
    SimpleDropdown,
    SimpleDropdownItem,
} from '../ui/simple-dropdown';
import { useTheme } from '../theme/ThemeProvider';

function Header() {
    const location = useLocation();
    const { user, isAuthenticated } = useAuth();
    const { setTheme, theme } = useTheme();
    const [isMobileMenuOpen, setIsMobileMenuOpen] = React.useState(false);

    if (!isAuthenticated) {
        return null;
    }

    return (
        <>
            <header className="border-b bg-background sticky top-0 z-50">
                <div className="container mx-auto px-4 py-4 flex items-center justify-between">
                    <Link to="/dashboard" className="flex items-center space-x-2 hover:opacity-80 transition-opacity">
                        <Server className="h-6 w-6 text-primary" />
                        <span className="font-bold text-xl">Selfhostly</span>
                    </Link>

                    {/* Desktop Navigation */}
                    <nav className="hidden md:flex items-center space-x-2">
                        {/* Primary Navigation */}
                        <Link to="/apps/new">
                            <Button className="flex items-center">
                                <Plus className="h-4 w-4 mr-2" />
                                New App
                            </Button>
                        </Link>

                        <Link to="/cloudflare">
                            <Button
                                variant={location.pathname === '/cloudflare' ? 'default' : 'ghost'}
                                className="flex items-center"
                            >
                                <Cloud className="h-4 w-4 mr-2" />
                                Cloudflare
                            </Button>
                        </Link>

                        <Link to="/settings">
                            <Button
                                variant={location.pathname === '/settings' ? 'default' : 'ghost'}
                                className="flex items-center"
                            >
                                <Settings className="h-4 w-4 mr-2" />
                                Settings
                            </Button>
                        </Link>

                        {/* User Menu - Combined */}
                        <div className="flex items-center ml-4 pl-4 border-l">
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
                    </nav>

                    {/* Mobile Menu Button */}
                    <div className="md:hidden">
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
                        >
                            {isMobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
                        </Button>
                    </div>
                </div>
            </header>

            {/* Mobile Navigation Menu */}
            {isMobileMenuOpen && (
                <div className="md:hidden bg-background border-b">
                    <nav className="container mx-auto px-4 py-2 space-y-1">
                        <Link to="/apps/new" onClick={() => setIsMobileMenuOpen(false)}>
                            <Button variant="ghost" className="w-full justify-start">
                                <Plus className="h-4 w-4 mr-2" />
                                New App
                            </Button>
                        </Link>

                        <Link to="/cloudflare" onClick={() => setIsMobileMenuOpen(false)}>
                            <Button
                                variant={location.pathname === '/cloudflare' ? 'default' : 'ghost'}
                                className="w-full justify-start"
                            >
                                <Cloud className="h-4 w-4 mr-2" />
                                Cloudflare
                            </Button>
                        </Link>

                        <Link to="/settings" onClick={() => setIsMobileMenuOpen(false)}>
                            <Button
                                variant={location.pathname === '/settings' ? 'default' : 'ghost'}
                                className="w-full justify-start"
                            >
                                <Settings className="h-4 w-4 mr-2" />
                                Settings
                            </Button>
                        </Link>

                        <div className="border-t my-2 pt-2">
                            <p className="text-xs font-medium text-muted-foreground px-3 py-2">Theme</p>
                            <Button
                                variant={theme === 'light' ? 'secondary' : 'ghost'}
                                onClick={() => {
                                    setTheme('light');
                                    setIsMobileMenuOpen(false);
                                }}
                                className="w-full justify-start"
                            >
                                <Sun className="h-4 w-4 mr-2" />
                                Light
                            </Button>
                            <Button
                                variant={theme === 'dark' ? 'secondary' : 'ghost'}
                                onClick={() => {
                                    setTheme('dark');
                                    setIsMobileMenuOpen(false);
                                }}
                                className="w-full justify-start"
                            >
                                <Moon className="h-4 w-4 mr-2" />
                                Dark
                            </Button>
                            <Button
                                variant={theme === 'system' ? 'secondary' : 'ghost'}
                                onClick={() => {
                                    setTheme('system');
                                    setIsMobileMenuOpen(false);
                                }}
                                className="w-full justify-start"
                            >
                                <Monitor className="h-4 w-4 mr-2" />
                                System
                            </Button>
                        </div>

                        <div className="border-t my-2 pt-2">
                            <div className="flex items-center space-x-3 mb-3 px-3">
                                <SmartAvatar user={user} size="sm" />
                                <div>
                                    <p className="text-sm font-medium">{user?.name}</p>
                                </div>
                            </div>
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => {
                                    logout();
                                    setIsMobileMenuOpen(false);
                                }}
                                className="w-full justify-start text-destructive hover:text-destructive"
                                title="Logout"
                            >
                                <LogOut className="h-4 w-4 mr-2" />
                                Logout
                            </Button>
                        </div>
                    </nav>
                </div>
            )}
        </>
    );
}

export default Header;
