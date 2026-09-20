export const ROUTES = {
    fleet: '/apps',
    newApp: '/apps/new',
    nodes: '/nodes',
    registerNode: '/nodes/new',
    access: '/access',
    insights: '/insights',
    settings: '/settings',
} as const

// Old addresses that people may have bookmarked. They forward to the new ones.
export const LEGACY_REDIRECTS: { from: string; to: string }[] = [
    { from: '/cloudflare', to: ROUTES.access },
    { from: '/monitoring', to: ROUTES.insights },
]

const APP_TABS = ['overview', 'config', 'environment', 'logs', 'access', 'schedule', 'history'] as const
export type AppTab = (typeof APP_TABS)[number]

const DEFAULT_APP_TAB: AppTab = 'overview'

// The tab names used in earlier versions.
const LEGACY_TAB_ALIASES: Record<string, AppTab> = {
    compose: 'config',
    cloudflare: 'access',
}

// Reads the ?tab= value from an address. Unknown values fall back to the overview.
function normalizeAppTab(value: string | null): AppTab {
    if (!value) return DEFAULT_APP_TAB
    if ((APP_TABS as readonly string[]).includes(value)) return value as AppTab
    return LEGACY_TAB_ALIASES[value] ?? DEFAULT_APP_TAB
}

// The tabs that have content today. A tab is added here when its screen is built, so an address for one
// that is not ready yet falls back to the overview instead of showing an empty page.
export const AVAILABLE_APP_TABS: readonly AppTab[] = [
    'overview',
    'config',
    'environment',
    'logs',
    'access',
    'schedule',
    'history',
]

// The tab to show for an address: normalized, and only one that exists.
export function resolveAppTab(value: string | null): AppTab {
    const tab = normalizeAppTab(value)
    return AVAILABLE_APP_TABS.includes(tab) ? tab : DEFAULT_APP_TAB
}

export const APP_TAB_LABELS: Record<Exclude<AppTab, 'overview'>, string> = {
    config: 'Config',
    environment: 'Environment',
    logs: 'Logs',
    access: 'Access',
    schedule: 'Schedule',
    history: 'History',
}

// The address of an app, optionally on one of its tabs. The node is part of the address because app ids
// are looked up per node.
export function appHref(app: { id: string; node_id?: string }, tab?: AppTab): string {
    const params = new URLSearchParams()
    if (app.node_id) params.set('node_id', app.node_id)
    if (tab && tab !== DEFAULT_APP_TAB) params.set('tab', tab)
    const query = params.toString()
    return `${ROUTES.fleet}/${app.id}${query ? `?${query}` : ''}`
}
