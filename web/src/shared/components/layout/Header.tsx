import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Server, Settings, Plus, LogOut, Cloud, Menu, X, Home, ChevronDown } from 'lucide-react';
import { Button } from '../ui/button';
import SmartAvatar from '../ui/SmartAvatar';
import { useAuth } from '../auth/AuthProvider';
import { logout } from '@/shared/services/api';
import {
    SimpleDropdown,
    SimpleDropdownItem,
} from '../ui/simple-dropdown';

function Header() {
    const location = useLocation();
    const { user, isAuthenticated } = useAuth();
    const [isMobileMenuOpen, setIsMobileMenuOpen] = React.useState(false);

    if (!isAuthenticated) {
        return null;
    }

    return (
        <>
            <header className="border-b bg-background sticky top-0 z-50">
                <div className="container mx-auto px-4 py-4 flex items-center justify-between">
                    <div className="flex items-center space-x-2">
                        <Server className="h-6 w-6 text-primary" />
                        <span className="font-bold text-xl">Selfhostly</span>
                    </div>

                    {/* Desktop Navigation */}
                    <nav className="hidden md:flex items-center space-x-2">
                        {/* Primary Navigation */}
                        <Link to="/dashboard">
                            <Button
                                variant={location.pathname === '/dashboard' ? 'default' : 'ghost'}
                                className="flex items-center"
                            >
                                <Home className="h-4 w-4 mr-2" />
                                Dashboard
                            </Button>
                        </Link>

                        <Link to="/apps/new">
                            <Button className="flex items-center">
                                <Plus className="h-4 w-4 mr-2" />
                                New App
                            </Button>
                        </Link>

                        {/* Divider */}
                        <div className="h-6 w-px bg-border mx-1" />

                        {/* Secondary Navigation */}
                        <SimpleDropdown
                            trigger={
                                <Button variant="ghost" size="sm" className="flex items-center">
                                    <Settings className="h-4 w-4 mr-2" />
                                    More
                                    <ChevronDown className="ml-1 h-4 w-4" />
                                </Button>
                            }
                        >
                            <div className="py-1">
                                <SimpleDropdownItem>
                                    <Link to="/cloudflare" className="flex items-center">
                                        <Cloud className="h-4 w-4 mr-2" />
                                        Cloudflare
                                    </Link>
                                </SimpleDropdownItem>
                                <SimpleDropdownItem>
                                    <Link to="/settings" className="flex items-center">
                                        <Settings className="h-4 w-4 mr-2" />
                                        Settings
                                    </Link>
                                </SimpleDropdownItem>
                            </div>
                        </SimpleDropdown>

                        {/* User Menu */}
                        <div className="flex items-center space-x-3 ml-4 pl-4 border-l">
                            <SmartAvatar user={user} size="md" />
                            <Button
                                variant="ghost"
                                size="icon"
                                onClick={() => logout()}
                                title="Logout"
                            >
                                <LogOut className="h-4 w-4" />
                            </Button>
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
                        <Link to="/dashboard" onClick={() => setIsMobileMenuOpen(false)}>
                            <Button
                                variant={location.pathname === '/dashboard' ? 'default' : 'ghost'}
                                className="w-full justify-start"
                            >
                                <Home className="h-4 w-4 mr-2" />
                                Dashboard
                            </Button>
                        </Link>

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
                            <div className="flex items-center space-x-3 mb-3">
                                <SmartAvatar user={user} size="sm" />
                            </div>
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => {
                                    logout();
                                    setIsMobileMenuOpen(false);
                                }}
                                className="w-full justify-start"
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
