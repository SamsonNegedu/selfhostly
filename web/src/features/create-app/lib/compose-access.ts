import { parse } from 'yaml'

interface ComposeService {
    ports?: unknown[]
}

// The first service that publishes a port and the container port it listens on. A tunnel needs both to know
// where to send visitors.
export function firstExposedService(content: string): { service: string; port: number } | null {
    try {
        const services = (parse(content) as { services?: Record<string, ComposeService | null> } | null)?.services
        if (!services) return null
        for (const [service, config] of Object.entries(services)) {
            for (const entry of config?.ports ?? []) {
                const text = typeof entry === 'object' && entry !== null ? String((entry as { target?: unknown }).target ?? '') : String(entry)
                const match = text.match(/(\d+)(?:\/\w+)?$/)
                if (match) return { service, port: Number(match[1]) }
            }
        }
    } catch {
        return null
    }
    return null
}
