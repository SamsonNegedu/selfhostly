const CONFLICT_PATTERN = /already allocated|address already in use|port is already|bind for .* failed/i
const ERROR_PORT_PATTERN = /:(\d{2,5})\b/
const PORT_MAPPING_PATTERN = /(["']?)((?:\d{1,3}(?:\.\d{1,3}){3}:)?)(\d{2,5}):(\d{1,5})(\/\w+)?\1/g
const FIRST_UNPRIVILEGED_PORT = 1024
const MAX_PORT = 65535

// The host port a "port is already allocated" style failure names, or null when the failure is about something else.
export function detectPortConflict(errorMessage: string | undefined): number | null {
    if (!errorMessage || !CONFLICT_PATTERN.test(errorMessage)) return null
    const match = errorMessage.match(ERROR_PORT_PATTERN)
    return match ? Number(match[1]) : null
}

// Every host port that the given compose files publish.
export function collectHostPorts(composeFiles: string[]): Set<number> {
    const ports = new Set<number>()
    for (const compose of composeFiles) {
        for (const match of compose.matchAll(PORT_MAPPING_PATTERN)) {
            ports.add(Number(match[3]))
        }
    }
    return ports
}

// The next port above `port` that none of the known apps publish. It cannot see ports taken by things
// outside Selfhostly, so the caller must say the suggestion is only checked against the apps.
export function suggestFreePort(port: number, used: Set<number>): number | null {
    for (let candidate = Math.max(port + 1, FIRST_UNPRIVILEGED_PORT); candidate <= MAX_PORT; candidate++) {
        if (!used.has(candidate)) return candidate
    }
    return null
}

// Rewrites the first mapping that publishes `from` so that it publishes `to`. Returns null if none does.
export function replaceHostPort(compose: string, from: number, to: number): string | null {
    let replaced = false
    const next = compose.replace(
        PORT_MAPPING_PATTERN,
        (whole, quote: string, ip: string, host: string, container: string, proto?: string) => {
            if (replaced || Number(host) !== from) return whole
            replaced = true
            return `${quote}${ip}${to}:${container}${proto ?? ''}${quote}`
        },
    )
    return replaced ? next : null
}
