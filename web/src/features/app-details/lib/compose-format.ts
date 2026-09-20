import { Document, parse, Scalar, visit } from 'yaml'

// The compose keys whose entries are often written as a bare name with nothing after it (`data:` under volumes).
// The YAML parser reads those as null, which would print back as `data: null`.
const EMPTY_OK_KEYS = ['networks', 'volumes', 'services']
const isEmptyOkKey = (key?: string) => key !== undefined && EMPTY_OK_KEYS.includes(key)

// Text such as `22:22`, `8080:80/tcp` or `127.0.0.1:8080:80`. Compose files are read by parsers that take an unquoted
// `22:22` as a base 60 number, so port mappings (and times like `08:30`) stay quoted.
const COLON_NUMBERS = /^[\d.-]+(:[\d.-]+)+(\/[a-z]+)?$/i

const PRINT_OPTIONS = {
    indent: 2,
    lineWidth: 0,
    sortMapEntries: false,
    defaultStringType: 'PLAIN',
    defaultKeyType: 'PLAIN',
} as const

const SECTION_HEADER = /^\s*(networks|volumes|services):\s*$/
const BARE_KEY = /^\s*\w+:\s*$/
const EMPTY_VALUE_LINE = /^(\s+)(\w+):\s+(null|{})\s*$/
const EMPTY_VALUE_TAIL = /:\s+(null|{})\s*$/

// Turns null into an empty object wherever the compose file keeps empty definitions.
function fixNullValues(value: unknown, parentKey?: string): unknown {
    if (value === null) return isEmptyOkKey(parentKey) ? {} : null
    if (Array.isArray(value)) return value.map((item) => fixNullValues(item, parentKey))
    if (typeof value === 'object') {
        const result: Record<string, unknown> = {}
        for (const [key, child] of Object.entries(value)) {
            result[key] =
                child === null && (isEmptyOkKey(key) || isEmptyOkKey(parentKey)) ? {} : fixNullValues(child, key)
        }
        return result
    }
    return value
}

// Prints `key: {}` and `key: null` as a bare `key:` inside the networks, volumes and services sections.
function tidyEmptyEntries(text: string): string {
    let inSection = false
    return text
        .split('\n')
        .map((line) => {
            if (SECTION_HEADER.test(line)) inSection = true
            else if (BARE_KEY.test(line)) inSection = false

            return inSection && EMPTY_VALUE_LINE.test(line) ? line.replace(EMPTY_VALUE_TAIL, ':') : line
        })
        .join('\n')
}

// The compose file with consistent indentation, no `null` placeholders and colon separated numbers kept in quotes. Throws with the parser's message when
// the file is not valid YAML.
export function formatCompose(content: string): string {
    const fixed = fixNullValues(parse(content))
    const doc = new Document(fixed, PRINT_OPTIONS)
    visit(doc, {
        Scalar(_key, node) {
            if (typeof node.value === 'string' && COLON_NUMBERS.test(node.value)) node.type = Scalar.QUOTE_DOUBLE
        },
    })
    const formatted = doc.toString(PRINT_OPTIONS)
    return tidyEmptyEntries(formatted)
}
