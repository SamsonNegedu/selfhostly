import { useLocation, useParams, useSearchParams } from 'react-router-dom'
import { APP_TAB_LABELS, ROUTES, resolveAppTab } from '@/shared/lib/routes'
import { useApps, useNodes } from '@/shared/services/api'

export interface Crumb {
    label: string
    to?: string
}

const ALL_NODES = ['all']

const ROOT_CRUMBS: Record<string, Crumb[]> = {
    [ROUTES.fleet]: [{ label: 'Fleet' }],
    [ROUTES.newApp]: [{ label: 'Fleet', to: ROUTES.fleet }, { label: 'New app' }],
    [ROUTES.nodes]: [{ label: 'Nodes' }],
    [ROUTES.registerNode]: [{ label: 'Nodes', to: ROUTES.nodes }, { label: 'Register' }],
    [ROUTES.access]: [{ label: 'Access' }],
    [ROUTES.insights]: [{ label: 'Insights' }],
    [ROUTES.settings]: [{ label: 'Settings' }],
    '/dev/ui': [{ label: 'UI gallery' }],
}

// Where you are. Static pages come from a table, and an app page reads the app's name and node from the
// list the Fleet screen already loads.
export function useCrumbs(): Crumb[] {
    const { pathname } = useLocation()
    const { id } = useParams()
    const [searchParams] = useSearchParams()
    const { data: apps = [] } = useApps(ALL_NODES)
    const { data: nodes = [] } = useNodes()

    const fixed = ROOT_CRUMBS[pathname]
    if (fixed) return fixed

    if (id && pathname.startsWith(`${ROUTES.fleet}/`)) {
        const app = apps.find((candidate) => candidate.id === id)
        const crumbs: Crumb[] = [{ label: 'Fleet', to: ROUTES.fleet }]
        const nodeName = app?.node_name ?? nodes.find((node) => node.id === app?.node_id)?.name
        if (nodeName) crumbs.push({ label: nodeName })
        const nodeId = searchParams.get('node_id')
        crumbs.push({ label: app?.name ?? 'App', to: `${ROUTES.fleet}/${id}${nodeId ? `?node_id=${nodeId}` : ''}` })
        const tab = resolveAppTab(searchParams.get('tab'))
        if (tab !== 'overview') crumbs.push({ label: APP_TAB_LABELS[tab] })
        return crumbs
    }

    return []
}
