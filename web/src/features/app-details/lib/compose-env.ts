import { isMap, isSeq, parseDocument, YAMLMap, type Document } from 'yaml'

export interface EnvEntry {
    service: string
    key: string
    value: string
}

const SECRET_KEY_PATTERN = /pass(word|wd)?|secret|token|api[_-]?key|private|credential|auth|salt|signing/i
const REFERENCE_PATTERN = /\$\{([A-Za-z_][A-Za-z0-9_]*)(:?[-?+][^}]*)?\}/g
const KEY_PATTERN = /^[A-Za-z_][A-Za-z0-9_.]*$/

export const isValidEnvKey = (key: string) => KEY_PATTERN.test(key)

const CREDENTIAL_URL_PATTERN = /:\/\/[^/\s:@]+:[^@\s]+@/

// Entries that look like they should not be shown on a shared screen: a name such as PASSWORD or TOKEN, or a value
// that is a URL with a login inside it. This is a guess, not a guarantee.
export const isSecretEntry = (key: string, value: string) =>
    SECRET_KEY_PATTERN.test(key) || CREDENTIAL_URL_PATTERN.test(value)

function servicesOf(doc: Document): YAMLMap | null {
    const services = doc.get('services', true)
    return isMap(services) ? services : null
}

function serviceMap(doc: Document, service: string): YAMLMap | null {
    const node = servicesOf(doc)?.get(service, true)
    return isMap(node) ? node : null
}

const stringify = (value: unknown) => (value === null || value === undefined ? '' : String(value))

export function readServiceNames(content: string): string[] {
    const doc = parseDocument(content)
    if (doc.errors.length > 0) return []
    return servicesOf(doc)?.items.map((pair) => String((pair.key as { value?: unknown })?.value ?? pair.key)) ?? []
}

// Every variable set in the `environment` of a service, in either the map or the list form.
export function readEnv(content: string): { entries: EnvEntry[]; error?: string } {
    const doc = parseDocument(content)
    if (doc.errors.length > 0) return { entries: [], error: doc.errors[0].message.split('\n')[0] }

    const entries: EnvEntry[] = []
    for (const service of readServiceNames(content)) {
        const env = serviceMap(doc, service)?.get('environment', true)
        if (isMap(env)) {
            for (const pair of env.items) {
                entries.push({
                    service,
                    key: stringify((pair.key as { value?: unknown })?.value ?? pair.key),
                    value: stringify((pair.value as { value?: unknown })?.value ?? pair.value),
                })
            }
        } else if (isSeq(env)) {
            for (const item of env.items) {
                const text = stringify((item as { value?: unknown })?.value ?? item)
                const split = text.indexOf('=')
                entries.push(
                    split === -1
                        ? { service, key: text, value: '' }
                        : { service, key: text.slice(0, split), value: text.slice(split + 1) },
                )
            }
        }
    }
    return { entries }
}

// Adds a variable, or changes its value if the service already sets it. Comments and layout are kept.
export function setEnv(content: string, service: string, key: string, value: string): string {
    const doc = parseDocument(content)
    const target = serviceMap(doc, service)
    if (!target) return content

    const env = target.get('environment', true)
    if (isSeq(env)) {
        const line = `${key}=${value}`
        const index = env.items.findIndex(
            (item) => stringify((item as { value?: unknown })?.value ?? item).split('=')[0] === key,
        )
        if (index === -1) env.add(doc.createNode(line))
        else env.set(index, doc.createNode(line))
    } else if (isMap(env)) {
        env.set(key, value)
    } else {
        const created = new YAMLMap()
        created.set(key, value)
        target.set('environment', created)
    }
    return doc.toString({ lineWidth: 0 })
}

export function removeEnv(content: string, service: string, key: string): string {
    const doc = parseDocument(content)
    const env = serviceMap(doc, service)?.get('environment', true)
    if (isMap(env)) {
        env.delete(key)
        if (env.items.length === 0) serviceMap(doc, service)?.delete('environment')
    } else if (isSeq(env)) {
        const index = env.items.findIndex(
            (item) => stringify((item as { value?: unknown })?.value ?? item).split('=')[0] === key,
        )
        if (index !== -1) env.delete(index)
        if (env.items.length === 0) serviceMap(doc, service)?.delete('environment')
    }
    return doc.toString({ lineWidth: 0 })
}

// KEY=value lines as found in a .env file. Comments, blank lines and an `export ` prefix are ignored, and
// matching quotes around a value are removed.
export function parseDotenv(text: string): { key: string; value: string }[] {
    const result: { key: string; value: string }[] = []
    for (const raw of text.split(/\r?\n/)) {
        const line = raw.trim().replace(/^export\s+/, '')
        if (!line || line.startsWith('#')) continue
        const split = line.indexOf('=')
        if (split < 1) continue
        const key = line.slice(0, split).trim()
        let value = line.slice(split + 1).trim()
        if (value.length >= 2 && (value[0] === '"' || value[0] === "'") && value.endsWith(value[0]))
            value = value.slice(1, -1)
        if (isValidEnvKey(key)) result.push({ key, value })
    }
    return result
}

// Variables the compose file refers to as ${NAME} that nothing sets and that have no default. Compose would
// start the container with an empty value, which is rarely what anyone wants.
export function findUnresolved(content: string, entries: EnvEntry[]): string[] {
    const defined = new Set(entries.map((entry) => entry.key))
    const missing = new Set<string>()
    for (const match of content.matchAll(REFERENCE_PATTERN)) {
        const [, name, modifier] = match
        const hasDefault = /^:?-/.test(modifier ?? '')
        if (!defined.has(name) && !hasDefault) missing.add(name)
    }
    return [...missing]
}
