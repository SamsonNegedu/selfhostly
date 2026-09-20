import { parse, YAMLParseError } from 'yaml'
import { collectHostPorts } from './port-conflict'

export type CheckLevel = 'ok' | 'warn' | 'err'

export interface ComposeCheck {
    id: string
    level: CheckLevel
    title: string
    detail?: string
}

export interface ComposeCheckResult {
    checks: ComposeCheck[]
    // True when the file cannot be deployed as it is.
    blocked: boolean
}

interface ServiceShape {
    image?: unknown
    build?: unknown
}

// What the browser can tell about a compose file before it is saved. Nothing here talks to Docker, so it cannot
// promise the file will start, only that it is well formed and that nothing obvious is wrong.
export function checkCompose(
    content: string,
    otherApps: { name: string; compose_content: string }[],
): ComposeCheckResult {
    const checks: ComposeCheck[] = []

    let parsed: unknown
    try {
        parsed = parse(content)
    } catch (error) {
        const line = error instanceof YAMLParseError ? error.linePos?.[0]?.line : undefined
        const reason = error instanceof YAMLParseError ? error.message.split('\n')[0] : 'The file is not valid YAML'
        checks.push({
            id: 'yaml',
            level: 'err',
            title: 'Not valid YAML',
            detail: line ? `${reason} (line ${line})` : reason,
        })
        return { checks, blocked: true }
    }
    checks.push({ id: 'yaml', level: 'ok', title: 'Valid YAML' })

    const services =
        parsed && typeof parsed === 'object'
            ? (parsed as { services?: Record<string, ServiceShape | null> }).services
            : undefined
    const names = services && typeof services === 'object' ? Object.keys(services) : []
    if (names.length === 0) {
        checks.push({
            id: 'services',
            level: 'err',
            title: 'No services defined',
            detail: 'Add at least one entry under services.',
        })
        return { checks, blocked: true }
    }
    checks.push({
        id: 'services',
        level: 'ok',
        title: `${names.length} ${names.length === 1 ? 'service' : 'services'} defined`,
        detail: names.join(', '),
    })

    const noImage = names.filter((name) => {
        const service = services?.[name]
        return !service || (!service.image && !service.build)
    })
    if (noImage.length > 0) {
        checks.push({
            id: 'image',
            level: 'err',
            title: 'A service has no image',
            detail: `${noImage.join(', ')} needs an image or a build.`,
        })
    } else {
        checks.push({ id: 'image', level: 'ok', title: 'Every service has an image' })
    }

    const mine = collectHostPorts([content])
    const clashes: string[] = []
    for (const other of otherApps) {
        for (const port of collectHostPorts([other.compose_content])) {
            if (mine.has(port)) clashes.push(`${port} (${other.name})`)
        }
    }
    if (clashes.length > 0) {
        checks.push({ id: 'ports', level: 'warn', title: 'Port also used by another app', detail: clashes.join(', ') })
    } else if (mine.size > 0) {
        checks.push({
            id: 'ports',
            level: 'ok',
            title: 'No port shared with another app',
            detail: [...mine].join(', '),
        })
    }

    return { checks, blocked: checks.some((check) => check.level === 'err') }
}
