// Fails the build when an output file could be mistaken for an API route.
//
// In production the tunnel decides where a request goes by matching its path against rules such as /api/*, and
// those rules match anywhere in the path, so /assets/api-abc123.js (or chunk-api-abc123.js) can be sent to the
// gateway and answer 404.
// vite.config.ts prefixes every file name to avoid that. This check keeps it from breaking quietly, for example
// when a new page or a renamed file produces a name that starts with one of the gateway's prefixes.
import { readdirSync, statSync } from 'node:fs'
import { join, relative, sep } from 'node:path'

const DIST = new URL('../dist', import.meta.url).pathname
// What the gateway serves. Keep in step with internal/gateway/router.go.
const GATEWAY_PREFIXES = ['api', 'auth', 'avatar']
// Anywhere in the path, in any case: the tunnel's rules are loose regular expressions.
const RISKY = new RegExp(GATEWAY_PREFIXES.join('|'), 'i')

function* walk(dir) {
    for (const name of readdirSync(dir)) {
        const path = join(dir, name)
        if (statSync(path).isDirectory()) yield* walk(path)
        else yield path
    }
}

const risky = [...walk(DIST)]
    .map((file) => '/' + relative(DIST, file).split(sep).join('/'))
    .filter((url) => RISKY.test(url))

if (risky.length > 0) {
    console.error(`These build files could be routed to the gateway instead of the frontend:\n  ${risky.join('\n  ')}`)
    console.error(`No file path may contain ${GATEWAY_PREFIXES.join(', ')}. Adjust the file names in vite.config.ts. (If it is only the random hash, change any source file and rebuild.)`)
    process.exit(1)
}
console.log(`asset names ok: no file path contains ${GATEWAY_PREFIXES.join(', ')}`)
