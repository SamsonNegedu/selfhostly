import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
    Activity,
    FileText,
    Globe,
    LayoutGrid,
    Monitor,
    Moon,
    Play,
    Plus,
    Server,
    SlidersHorizontal,
    Sun,
    type LucideIcon,
} from 'lucide-react'
import { CommandDialog, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '../ui/Command'
import { AppTile } from '../ui/AppTile'
import { useToast } from '../ui/Toast'
import { useCommandPalette } from './CommandPaletteContext'
import { useTheme } from '../theme/ThemeProvider'
import { appStatusMeta } from '@/shared/lib/status'
import { describeError } from '@/shared/lib/errors'
import { useApps, useStartApp } from '@/shared/services/api'
import { ROUTES, appHref } from '@/shared/lib/routes'
import type { App } from '@/shared/types/api'

const ALL_NODES = ['all']
const SHORTCUT_KEY = 'k'

interface PaletteAction {
    id: string
    label: string
    icon: LucideIcon
    run: () => void
}

// Search for anything and jump to it from the keyboard. Cmd or Ctrl plus K opens it anywhere in the app.
function CommandPalette() {
    const { open, setOpen, toggle } = useCommandPalette()
    const navigate = useNavigate()
    const { setTheme } = useTheme()
    const { data: apps = [] } = useApps(ALL_NODES)
    const startApp = useStartApp()
    const { toast } = useToast()

    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key.toLowerCase() === SHORTCUT_KEY && (event.metaKey || event.ctrlKey)) {
                event.preventDefault()
                toggle()
            }
        }
        document.addEventListener('keydown', onKeyDown)
        return () => document.removeEventListener('keydown', onKeyDown)
    }, [toggle])

    const go = (path: string) => () => navigate(path)

    const actions: PaletteAction[] = [
        { id: 'new-app', label: 'New app', icon: Plus, run: go(ROUTES.newApp) },
        { id: 'register-node', label: 'Register node', icon: Server, run: go(ROUTES.registerNode) },
        { id: 'theme-dark', label: 'Use dark theme', icon: Moon, run: () => setTheme('dark') },
        { id: 'theme-light', label: 'Use light theme', icon: Sun, run: () => setTheme('light') },
        { id: 'theme-system', label: 'Follow system theme', icon: Monitor, run: () => setTheme('system') },
    ]

    const destinations: PaletteAction[] = [
        { id: 'go-fleet', label: 'Fleet', icon: LayoutGrid, run: go(ROUTES.fleet) },
        { id: 'go-nodes', label: 'Nodes', icon: Server, run: go(ROUTES.nodes) },
        { id: 'go-access', label: 'Access', icon: Globe, run: go(ROUTES.access) },
        { id: 'go-insights', label: 'Insights', icon: Activity, run: go(ROUTES.insights) },
        { id: 'go-settings', label: 'Settings', icon: SlidersHorizontal, run: go(ROUTES.settings) },
    ]

    const runAndClose = (run: () => void) => () => {
        setOpen(false)
        run()
    }

    const start = (app: App) => {
        startApp.mutate(
            { id: app.id, nodeId: app.node_id },
            {
                onSuccess: () => toast.success(`${app.name} is starting`),
                onError: (error) => toast.error(`Could not start ${app.name}`, describeError(error)),
            },
        )
    }

    const stoppedApps = apps.filter((app) => app.status === 'stopped')

    return (
        <>
            <CommandDialog
                open={open}
                onOpenChange={setOpen}
                title="Command palette"
                description="Search apps, actions and pages, then press Enter to go."
            >
                <CommandInput placeholder="Search apps, actions and pages" />
                <CommandList>
                    <CommandEmpty>Nothing matches. Try an app name or an action such as start.</CommandEmpty>

                    <CommandGroup heading="Actions">
                        {actions.map((action) => (
                            <CommandItem key={action.id} value={action.label} onSelect={runAndClose(action.run)}>
                                <action.icon className="h-[17px] w-[17px] text-muted-foreground" />
                                {action.label}
                            </CommandItem>
                        ))}
                        {stoppedApps.map((app) => (
                            <CommandItem
                                key={`start-${app.id}`}
                                value={`Start ${app.name}`}
                                keywords={[app.node_name ?? '', 'run', 'launch']}
                                onSelect={runAndClose(() => start(app))}
                            >
                                <Play className="h-[17px] w-[17px] text-muted-foreground" />
                                Start {app.name}
                            </CommandItem>
                        ))}
                    </CommandGroup>

                    {apps.length > 0 && (
                        <CommandGroup heading="Apps">
                            {apps.map((app) => {
                                const status = appStatusMeta(app.status)
                                return (
                                    <CommandItem
                                        key={app.id}
                                        value={`${app.name} ${app.node_name ?? ''}`}
                                        keywords={[status.label, app.description]}
                                        onSelect={runAndClose(() => navigate(appHref(app)))}
                                    >
                                        <AppTile name={app.name} size="sm" className="h-6 w-6 text-caption" />
                                        <span className="flex-1 truncate font-medium" title={app.name}>
                                            {app.name}
                                        </span>
                                        <span className="text-xs text-muted-foreground">{status.label}</span>
                                    </CommandItem>
                                )
                            })}
                        </CommandGroup>
                    )}

                    {apps.length > 0 && (
                        <CommandGroup heading="Logs">
                            {apps.map((app) => (
                                <CommandItem
                                    key={`logs-${app.id}`}
                                    value={`Logs ${app.name}`}
                                    keywords={['output', 'console']}
                                    onSelect={runAndClose(() => navigate(appHref(app, 'logs')))}
                                >
                                    <FileText className="h-[17px] w-[17px] text-muted-foreground" />
                                    View logs for {app.name}
                                </CommandItem>
                            ))}
                        </CommandGroup>
                    )}

                    <CommandGroup heading="Go to">
                        {destinations.map((destination) => (
                            <CommandItem
                                key={destination.id}
                                value={`Go to ${destination.label}`}
                                onSelect={runAndClose(destination.run)}
                            >
                                <destination.icon className="h-[17px] w-[17px] text-muted-foreground" />
                                {destination.label}
                            </CommandItem>
                        ))}
                    </CommandGroup>
                </CommandList>
                <div className="hidden h-10 items-center gap-4 border-t border-border bg-background px-4 text-xs text-muted-foreground sm:flex">
                    <span>Arrow keys to move</span>
                    <span>Enter to run</span>
                    <span className="ml-auto">Esc to close</span>
                </div>
            </CommandDialog>
        </>
    )
}

export default CommandPalette
