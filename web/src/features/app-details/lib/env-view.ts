import type { ComposeCheck } from './compose-checks'
import { findUnresolved, isSecretEntry, readEnv, type EnvEntry } from './compose-env'

// What a hidden secret looks like on screen.
export const MASK = '••••••••'

// One variable is identified by the service it belongs to and its name.
export const rowId = (entry: EnvEntry) => `${entry.service}:${entry.key}`

// Whether a variable is new, or changed, compared with what is saved.
export function envFlag(saved: EnvEntry[], entry: EnvEntry): 'new' | 'changed' | null {
    const before = saved.find((other) => rowId(other) === rowId(entry))
    if (!before) return 'new'
    return before.value !== entry.value ? 'changed' : null
}

// The list beside the table: how many variables, and anything that looks wrong.
export function envChecks(current: ReturnType<typeof readEnv>, draft: string): ComposeCheck[] {
    if (current.error)
        return [{ id: 'yaml', level: 'err', title: 'The compose file is not valid YAML', detail: current.error }]
    const list: ComposeCheck[] = [
        {
            id: 'count',
            level: 'ok',
            title: `${current.entries.length} ${current.entries.length === 1 ? 'variable' : 'variables'} across ${new Set(current.entries.map((entry) => entry.service)).size} ${new Set(current.entries.map((entry) => entry.service)).size === 1 ? 'service' : 'services'}`,
        },
    ]
    const unresolved = findUnresolved(draft, current.entries)
    if (unresolved.length > 0) {
        list.push({
            id: 'unresolved',
            level: 'warn',
            title: 'Referenced but not set',
            detail: `${unresolved.map((name) => '${' + name + '}').join(', ')} has no value and no default.`,
        })
    }
    const empty = current.entries.filter((entry) => entry.value === '')
    if (empty.length > 0) {
        list.push({
            id: 'empty',
            level: 'warn',
            title: 'Empty values',
            detail: empty.map((entry) => entry.key).join(', '),
        })
    }
    return list
}

// The variables as .env lines, grouped by service, with secrets hidden until revealed.
export function rawEnvLines(services: string[], entries: EnvEntry[], revealed: Set<string>): { text: string }[] {
    const lines: { text: string }[] = []
    for (const service of services) {
        const own = entries.filter((entry) => entry.service === service)
        if (own.length === 0) continue
        if (lines.length > 0) lines.push({ text: '' })
        lines.push({ text: `# ${service}` })
        for (const entry of own) {
            const hidden = isSecretEntry(entry.key, entry.value) && !revealed.has(rowId(entry))
            lines.push({ text: `${entry.key}=${hidden ? MASK : entry.value}` })
        }
    }
    return lines
}
