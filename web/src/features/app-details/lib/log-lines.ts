import type { TerminalLine } from '@/shared/components/ui/Terminal'

const ERROR_PATTERN = /\b(error|err|fatal|panic|failed)\b/i
const WARN_PATTERN = /\b(warn|warning)\b/i

// Container output has no structure, so the level is a best guess from the words in the line.
export function toTerminalLines(text: string): TerminalLine[] {
    return text
        .replace(/\n$/, '')
        .split('\n')
        .map((line) => ({ text: line, level: ERROR_PATTERN.test(line) ? 'error' : WARN_PATTERN.test(line) ? 'warn' : undefined }))
}
