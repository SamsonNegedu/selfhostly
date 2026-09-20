// Fails the build when a screen goes around the design tokens.
//
// The rules in .agents/rules/frontend-ui.md say colors and type sizes come from the tokens in globals.css and
// tailwind.config.js. Those slip one class at a time, so this keeps them from coming back. The dev gallery is
// exempt because it shows raw values on purpose. The theme preview swatches in AppearanceSection are exempt
// because they draw the other theme's colors.
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, sep } from 'node:path'

const SRC = new URL('../src', import.meta.url).pathname
const EXEMPT = new Set(['features/dev/ui-gallery.tsx', 'features/settings/sections/AppearanceSection.tsx'])

const PALETTE = 'red|green|amber|yellow|blue|orange|emerald|rose|sky|slate|gray|zinc|neutral|stone|indigo|purple|violet'
const RULES = [
    {
        name: 'arbitrary text size',
        pattern: /\btext-\[[0-9.]+(px|rem)\]/,
        fix: 'use a named size: text-caption, text-detail, text-compact, text-body, text-title or text-heading',
    },
    {
        name: 'palette color class',
        pattern: new RegExp(`\\b(text|bg|border|ring|fill|stroke)-(${PALETTE})-[0-9]{2,3}\\b`),
        fix: 'use a status or tint token such as bg-status-ok or text-status-err-fg',
    },
    {
        name: 'black or white utility',
        pattern: /\b(text|bg|border)-(black|white)\b/,
        fix: 'use bg-scrim for overlays, or a token such as bg-card and text-foreground',
    },
    {
        name: 'hex color',
        pattern: /#[0-9a-fA-F]{6}\b/,
        fix: 'use a token from globals.css',
    },
    {
        name: 'h-screen',
        pattern: /\b(min-h|h)-screen\b/,
        fix: 'use h-dvh or min-h-dvh so phone browser toolbars do not cover the bottom of the page',
    },
]

function* walk(dir) {
    for (const name of readdirSync(dir)) {
        const path = join(dir, name)
        if (statSync(path).isDirectory()) yield* walk(path)
        else if (/\.tsx?$/.test(name)) yield path
    }
}

const problems = []
for (const file of walk(SRC)) {
    const rel = relative(SRC, file).split(sep).join('/')
    if (EXEMPT.has(rel)) continue
    readFileSync(file, 'utf8')
        .split('\n')
        .forEach((line, index) => {
            for (const rule of RULES) {
                if (rule.pattern.test(line)) problems.push(`  ${rel}:${index + 1}  ${rule.name}: ${rule.fix}`)
            }
        })
}

if (problems.length > 0) {
    console.error(`These lines go around the design tokens:\n${problems.join('\n')}`)
    process.exit(1)
}
console.log('ui tokens ok: no raw sizes, palette colors or h-screen in src')
