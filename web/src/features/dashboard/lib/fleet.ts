import type { App, Node } from '@/shared/types/api'

export type FleetFilter = 'all' | 'running' | 'stopped' | 'updating' | 'failed' | 'unreachable'

// An app together with its node. An app whose node is not online is unreachable: its stored status
// is the last one seen, so it is not trusted.
export interface FleetApp {
    app: App
    node?: Node
    unreachable: boolean
}

export interface NodeGroup {
    key: string
    node?: Node
    name: string
    apps: FleetApp[]
}

const STATUS_ORDER: Record<string, number> = { error: 0, updating: 1, pending: 2, running: 3, stopped: 4 }
const UNKNOWN_STATUS_RANK = 5
const NO_NODE_KEY = 'no-node'

export function toFleetApps(apps: App[], nodes: Node[]): FleetApp[] {
    const byId = new Map(nodes.map((node) => [node.id, node]))
    return apps.map((app) => {
        const node = byId.get(app.node_id)
        return { app, node, unreachable: node !== undefined && node.status !== 'online' }
    })
}

export function matchesFilter({ app, unreachable }: FleetApp, filter: FleetFilter): boolean {
    switch (filter) {
        case 'all':
            return true
        case 'unreachable':
            return unreachable
        case 'failed':
            return !unreachable && app.status === 'error'
        case 'running':
        case 'stopped':
        case 'updating':
            return !unreachable && app.status === filter
    }
}

export function countByFilter(list: FleetApp[]): Record<FleetFilter, number> {
    const filters: FleetFilter[] = ['all', 'running', 'stopped', 'updating', 'failed', 'unreachable']
    return Object.fromEntries(
        filters.map((filter) => [filter, list.filter((item) => matchesFilter(item, filter)).length]),
    ) as Record<FleetFilter, number>
}

export function matchesQuery({ app }: FleetApp, query: string): boolean {
    const text = query.trim().toLowerCase()
    if (!text) return true
    return app.name.toLowerCase().includes(text) || (app.description ?? '').toLowerCase().includes(text)
}

// Problems first, then what is running, then what is stopped, and by name within each.
export function sortFleetApps(list: FleetApp[]): FleetApp[] {
    const rank = (item: FleetApp) =>
        item.unreachable ? UNKNOWN_STATUS_RANK : (STATUS_ORDER[item.app.status] ?? UNKNOWN_STATUS_RANK)
    return [...list].sort((a, b) => rank(a) - rank(b) || a.app.name.localeCompare(b.app.name))
}

// The primary node first, other online nodes by name, and nodes that are down last.
export function groupByNode(list: FleetApp[], nodes: Node[]): NodeGroup[] {
    const order = new Map(
        [...nodes]
            .sort(
                (a, b) =>
                    Number(a.status !== 'online') - Number(b.status !== 'online') ||
                    Number(b.is_primary) - Number(a.is_primary) ||
                    a.name.localeCompare(b.name),
            )
            .map((node, index) => [node.id, index]),
    )

    const groups = new Map<string, NodeGroup>()
    for (const item of list) {
        const key = item.node?.id ?? NO_NODE_KEY
        const group = groups.get(key) ?? {
            key,
            node: item.node,
            name: item.node?.name ?? item.app.node_name ?? 'Unknown node',
            apps: [],
        }
        group.apps.push(item)
        groups.set(key, group)
    }

    return [...groups.values()].sort((a, b) => (order.get(a.key) ?? nodes.length) - (order.get(b.key) ?? nodes.length))
}
