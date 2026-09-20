import type { AuditEntry } from '@/shared/services/api'

// The phrase for each action the server records. The target's name follows it, for example "Started nextcloud".
const ACTION_PHRASES: Record<string, string> = {
    'app.create': 'Created',
    'app.edit': 'Edited',
    'app.delete': 'Deleted',
    'app.start': 'Started',
    'app.stop': 'Stopped',
    'app.update': 'Updated',
    'app.service_restart': 'Restarted a container of',
    'app.restore_version': 'Restored a version of',
    'schedule.save': 'Saved the schedule of',
    'schedule.delete': 'Deleted the schedule of',
    'tunnel.create': 'Created a tunnel for',
    'tunnel.quick_create': 'Created a Quick Tunnel for',
    'tunnel.switch_to_custom': 'Moved to a custom domain:',
    'tunnel.sync': 'Synced the tunnel of',
    'tunnel.routes_edit': 'Edited the routes of',
    'tunnel.dns_create': 'Created a DNS record for',
    'tunnel.delete': 'Deleted the tunnel of',
    'settings.update': 'Changed the settings',
    'container.restart': 'Restarted container',
    'container.stop': 'Stopped container',
    'container.delete': 'Deleted container',
    'node.register': 'Added node',
    'node.edit': 'Edited node',
    'node.remove': 'Removed node',
    'node.join_token': 'Made a join token',
    'node.check': 'Checked node',
    'node.join': 'A node joined:',
    'session.revoke_all': 'Signed everyone out',
}

// A sentence for an audit entry, or null when the server did not describe the request.
export function describeAudit(entry: AuditEntry): string | null {
    const phrase = ACTION_PHRASES[entry.action]
    if (!phrase) return null
    const target = entry.target_name || (entry.target_type === 'container' ? entry.target_id.slice(0, 12) : '')
    return target ? `${phrase} ${target}` : phrase
}
