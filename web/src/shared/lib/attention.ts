import type { App, Node } from '@/shared/types/api'
import type { StatusKind } from '@/shared/lib/status'

type AttentionKind = 'app-error' | 'node-offline'

export interface AttentionItem {
    id: string
    kind: AttentionKind
    title: string
    detail: string
    href: string
    tone: StatusKind
    // Set for failed apps, so the strip can act on the app without looking it up again.
    app?: App
}

const MS_PER_MINUTE = 60_000
const MINUTES_PER_HOUR = 60
const HOURS_PER_DAY = 24

// A short, human sentence such as "3h ago". Missing or unparseable times read as unknown.
export function formatAgo(iso: string | undefined, now: number = Date.now()): string {
    if (!iso) return 'unknown'
    const then = new Date(iso).getTime()
    if (Number.isNaN(then)) return 'unknown'

    const minutes = Math.max(0, Math.floor((now - then) / MS_PER_MINUTE))
    if (minutes < 1) return 'just now'
    if (minutes < MINUTES_PER_HOUR) return `${minutes}m ago`
    const hours = Math.floor(minutes / MINUTES_PER_HOUR)
    if (hours < HOURS_PER_DAY) return `${hours}h ago`
    return `${Math.floor(hours / HOURS_PER_DAY)}d ago`
}

// Everything that needs a person to look at it, worst first: failed apps, then nodes that are not online.
export function getAttentionItems(apps: App[], nodes: Node[], now: number = Date.now()): AttentionItem[] {
    const appItems: AttentionItem[] = apps
        .filter((app) => app.status === 'error')
        .map((app) => ({
            id: `app-${app.id}`,
            kind: 'app-error',
            title: `${app.name} needs attention`,
            detail: app.error_message || 'The app reported an error',
            href: `/apps/${app.id}${app.node_id ? `?node_id=${app.node_id}` : ''}`,
            tone: 'err',
            app,
        }))

    const nodeItems: AttentionItem[] = nodes
        .filter((node) => node.status !== 'online')
        .map((node) => ({
            id: `node-${node.id}`,
            kind: 'node-offline',
            title: `${node.name} is ${node.status}`,
            detail: node.last_seen ? `Last seen ${formatAgo(node.last_seen, now)}` : 'No heartbeat yet',
            href: '/nodes',
            tone: 'warn',
        }))

    return [...appItems, ...nodeItems]
}
