import type { ComposeCheck } from '@/features/app-details/lib/compose-checks'

export type Mode = 'template' | 'paste' | 'link'
export type Access = 'none' | 'quick' | 'custom'

const NAME_PATTERN = /^[a-z0-9-]+$/
const MIN_NAME = 3
const MAX_NAME = 63
const MAX_PORT = 65535

export function nameError(name: string): string | undefined {
    if (name === '') return undefined
    if (!NAME_PATTERN.test(name)) return 'Use lowercase letters, numbers and hyphens only.'
    if (name.length < MIN_NAME) return `Use at least ${MIN_NAME} characters.`
    if (name.length > MAX_NAME) return `Use ${MAX_NAME} characters or fewer.`
    return undefined
}

// Only templates ask for a port. Pasted files publish whatever they say.
export function portError(mode: Mode, port: number): string | undefined {
    return mode === 'template' && (!Number.isInteger(port) || port < 1 || port > MAX_PORT)
        ? `Enter a port from 1 to ${MAX_PORT}.`
        : undefined
}

// A warning, not an error: two apps can be set up on one port, but only one of them can run.
export function portSharedMessage(
    mode: Mode,
    port: number,
    hasPortError: boolean,
    usedPorts: Set<number>,
    nodeName: string,
): string | undefined {
    return mode === 'template' && !hasPortError && usedPorts.has(port)
        ? `Another app on ${nodeName} already publishes ${port}.`
        : undefined
}

// Everything the checks and the blocking reason depend on, already worked out by the form.
export interface NewAppRuleInput {
    mode: Mode
    hasTemplate: boolean
    content: string
    name: string
    nameProblem: string | undefined
    nameTaken: boolean
    nodeId: string
    nodeName: string
    portError: string | undefined
    portShared: string | undefined
    // What checking the compose file found, or none while there is no file.
    composeChecks: ComposeCheck[]
    access: Access
    hasExposedPort: boolean
    hostname: string
}

// The list shown beside the form.
export function buildChecks(input: NewAppRuleInput): ComposeCheck[] {
    const { name, nameProblem, nameTaken, nodeId, nodeName } = input
    const checks: ComposeCheck[] = []
    if (name !== '') {
        checks.push(
            nameProblem || nameTaken
                ? {
                      id: 'name',
                      level: 'err',
                      title: nameTaken ? 'Name already used' : 'Name is not valid',
                      detail: nameTaken ? `${nodeName} already has an app called ${name}.` : nameProblem,
                  }
                : { id: 'name', level: 'ok', title: 'Name is available' },
        )
    }
    if (nodeId) checks.push({ id: 'node', level: 'ok', title: `Deploys to ${nodeName}` })
    else checks.push({ id: 'node', level: 'err', title: 'No node is online', detail: 'Bring a node online to deploy.' })
    if (input.portShared)
        checks.push({ id: 'port-shared', level: 'warn', title: 'Port already published', detail: input.portShared })
    checks.push(...input.composeChecks)
    if (input.access === 'quick' && !input.hasExposedPort)
        checks.push({
            id: 'quick',
            level: 'err',
            title: 'Quick Tunnel needs a published port',
            detail: 'Add a ports entry so the tunnel knows where to send visitors.',
        })
    if (input.access === 'custom' && input.hostname.trim() === '')
        checks.push({ id: 'host', level: 'err', title: 'Custom domain needs a hostname' })
    return checks
}

// Why Create is not available yet, in the order the person would fix things, or undefined when it is.
export function blockedReason(input: NewAppRuleInput, checks: ComposeCheck[]): string | undefined {
    if (input.mode === 'template' && !input.hasTemplate) return 'Choose a template'
    if (input.content.trim() === '') return input.mode === 'link' ? 'Fetch a compose file first' : 'Add a compose file'
    if (input.name === '') return 'Give the app a name'
    if (checks.some((check) => check.level === 'err') || input.portError) return 'Fix the problems in the checks'
    if (!input.nodeId) return 'No node is online'
    return undefined
}
