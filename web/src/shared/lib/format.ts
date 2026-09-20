const BYTES_PER_UNIT = 1024
const BYTE_UNITS = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
const WHOLE_NUMBER_FROM_UNIT = 2

// A readable size such as "412 MiB". Small values keep a decimal, larger ones are rounded.
export function formatBytes(bytes: number): string {
    if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'

    let value = bytes
    let unit = 0
    while (value >= BYTES_PER_UNIT && unit < BYTE_UNITS.length - 1) {
        value /= BYTES_PER_UNIT
        unit += 1
    }

    const text = unit >= WHOLE_NUMBER_FROM_UNIT && value < 10 ? value.toFixed(1) : Math.round(value).toString()
    return `${text} ${BYTE_UNITS[unit]}`
}

export function formatPercent(value: number): string {
    return `${Math.round(value)}%`
}

const MS_PER_MINUTE = 60_000
const MINUTES_PER_HOUR = 60
const HOURS_PER_DAY = 24

// How long until a time, as "in 7h 46m", "in 2d 3h" or "in 12m". Times in the past read as "now".
export function formatUntil(iso: string, now: number = Date.now()): string {
    const minutes = Math.floor((new Date(iso).getTime() - now) / MS_PER_MINUTE)
    if (!Number.isFinite(minutes) || minutes < 1) return 'now'
    if (minutes < MINUTES_PER_HOUR) return `in ${minutes}m`
    const hours = Math.floor(minutes / MINUTES_PER_HOUR)
    if (hours < HOURS_PER_DAY) return `in ${hours}h ${minutes % MINUTES_PER_HOUR}m`
    return `in ${Math.floor(hours / HOURS_PER_DAY)}d ${hours % HOURS_PER_DAY}h`
}
