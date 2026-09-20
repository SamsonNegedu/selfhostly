import type { StatusKind } from '@/shared/lib/status'
import type { App, CloudflareTunnel } from '@/shared/types/api'

export interface AddressRow {
    app: App
    tunnel?: CloudflareTunnel
    kind: 'custom' | 'quick'
    status: { kind: StatusKind; label: string }
}

export function statusFor(app: App, tunnel?: CloudflareTunnel): AddressRow['status'] {
    if (app.tunnel_mode === 'quick') return { kind: 'warn', label: 'Temporary' }
    if (!tunnel) return { kind: 'idle', label: 'Unknown' }
    if (tunnel.status === 'error') return { kind: 'err', label: 'Error' }
    return tunnel.is_active && tunnel.status === 'active'
        ? { kind: 'ok', label: 'Active' }
        : { kind: 'idle', label: 'Inactive' }
}

// The apps that have a public address, each with its tunnel and how it is doing, by name.
export function buildAddressRows(apps: App[], tunnels: CloudflareTunnel[]): AddressRow[] {
    return apps
        .filter((app) => app.public_url)
        .map((app) => {
            const tunnel = tunnels.find((candidate) => candidate.app_id === app.id)
            return {
                app,
                tunnel,
                kind: app.tunnel_mode === 'quick' ? ('quick' as const) : ('custom' as const),
                status: statusFor(app, tunnel),
            }
        })
        .sort((a, b) => a.app.name.localeCompare(b.app.name))
}
