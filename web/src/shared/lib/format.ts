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

// A calendar-ish age for a date: "Today", "Yesterday", "3 days ago", then the date itself.
export function formatDayAgo(dateString: string, now: Date = new Date()): string {
    const date = new Date(dateString)
    const diffMs = now.getTime() - date.getTime()
    const diffDays = Math.floor(diffMs / 86400000)

    if (diffDays === 0) return 'Today'
    if (diffDays === 1) return 'Yesterday'
    if (diffDays < 7) return `${diffDays} days ago`
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

// The host of an address, such as "app.example.com", or the text itself when it is not a valid URL.
export const hostOf = (url: string) => {
    try {
        return new URL(url).host
    } catch {
        return url
    }
}
