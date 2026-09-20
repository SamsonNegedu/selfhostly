import type { StatusKind } from '@/shared/lib/status'

export type Resource = 'cpu' | 'memory' | 'disk'

// Where a resource turns from fine to high to critical. Meters, alerts and node cards all use these, so a bar
// is never green while an alert says the same thing is high.
const THRESHOLDS: Record<Resource, { warn: number; err: number }> = {
    cpu: { warn: 80, err: 90 },
    memory: { warn: 85, err: 95 },
    disk: { warn: 85, err: 95 },
}

export function resourceTone(resource: Resource, percent: number): StatusKind {
    const { warn, err } = THRESHOLDS[resource]
    return percent > err ? 'err' : percent > warn ? 'warn' : 'ok'
}

export const TONE_WORD: Record<StatusKind, string> = { ok: 'Healthy', warn: 'High', err: 'Critical', info: 'Info', idle: 'Idle' }
