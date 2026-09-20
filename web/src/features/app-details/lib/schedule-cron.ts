export interface SimpleTime {
    hour: number
    minute: number
}

export interface SimpleSchedule extends SimpleTime {
    // 0 is Sunday and 6 is Saturday, as in cron.
    days: number[]
}

export const DAY_LABELS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
export const ALL_DAYS = [0, 1, 2, 3, 4, 5, 6]
const MINUTES_PER_DAY = 24 * 60
const MINUTES_PER_HOUR = 60
const SIMPLE_CRON = /^(\d{1,2})\s+(\d{1,2})\s+\*\s+\*\s+(\S+)$/
const DAY_PART = /^([0-7])(?:-([0-7]))?$/

function parseDays(field: string): number[] | null {
    if (field === '*') return [...ALL_DAYS]
    const days = new Set<number>()
    for (const part of field.split(',')) {
        const match = part.match(DAY_PART)
        if (!match) return null
        const from = Number(match[1]) % 7
        const to = match[2] === undefined ? from : Number(match[2]) % 7
        if (to < from && match[2] !== '7') return null
        for (let day = from; day <= (match[2] === '7' ? 7 : to); day++) days.add(day % 7)
    }
    return [...days].sort((a, b) => a - b)
}

// A cron expression the simple builder can show: one time of day on chosen weekdays. Anything else returns
// null and is edited as text.
export function parseSimpleCron(expression: string): SimpleSchedule | null {
    const match = expression.trim().match(SIMPLE_CRON)
    if (!match) return null
    const minute = Number(match[1])
    const hour = Number(match[2])
    if (minute > 59 || hour > 23) return null
    const days = parseDays(match[3])
    return days ? { hour, minute, days } : null
}

// 1,2,3,4,5 becomes 1-5. Runs of two stay as a list.
function compressDays(days: number[]): string {
    if (days.length === ALL_DAYS.length) return '*'
    const parts: string[] = []
    let index = 0
    while (index < days.length) {
        let end = index
        while (end + 1 < days.length && days[end + 1] === days[end] + 1) end++
        parts.push(end - index >= 2 ? `${days[index]}-${days[end]}` : days.slice(index, end + 1).join(','))
        index = end + 1
    }
    return parts.join(',')
}

export function buildCron({ hour, minute, days }: SimpleSchedule): string {
    return `${minute} ${hour} * * ${compressDays([...days].sort((a, b) => a - b))}`
}

export const formatTime = ({ hour, minute }: SimpleTime) =>
    `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`

export function parseTime(value: string): SimpleTime | null {
    const match = value.match(/^(\d{2}):(\d{2})$/)
    return match ? { hour: Number(match[1]), minute: Number(match[2]) } : null
}

// The part of a 24 hour day the app runs, as segments in minutes. A stop earlier than the start means the
// app runs overnight, so the window wraps past midnight.
export function runningWindow(start: SimpleTime, stop: SimpleTime): { from: number; to: number }[] {
    const from = start.hour * MINUTES_PER_HOUR + start.minute
    const to = stop.hour * MINUTES_PER_HOUR + stop.minute
    if (from === to) return []
    return from < to
        ? [{ from, to }]
        : [
              { from, to: MINUTES_PER_DAY },
              { from: 0, to },
          ]
}

export function windowHours(start: SimpleTime, stop: SimpleTime): number {
    return (
        runningWindow(start, stop).reduce((total, segment) => total + (segment.to - segment.from), 0) / MINUTES_PER_HOUR
    )
}

export const MINUTES_IN_DAY = MINUTES_PER_DAY

// "every day", "Mon to Fri", "Sat and Sun", or a list.
export function describeDays(days: number[]): string {
    if (days.length === ALL_DAYS.length) return 'every day'
    const sorted = [...days].sort((a, b) => a - b)
    if (sorted.length >= 3 && sorted.every((day, i) => i === 0 || day === sorted[i - 1] + 1)) {
        return `${DAY_LABELS[sorted[0]]} to ${DAY_LABELS[sorted[sorted.length - 1]]}`
    }
    if (sorted.length === 2) return `${DAY_LABELS[sorted[0]]} and ${DAY_LABELS[sorted[1]]}`
    return sorted.map((day) => DAY_LABELS[day]).join(', ')
}

// Quick choices for which days a schedule runs.
export const PRESETS: { label: string; days: number[] }[] = [
    { label: 'Every day', days: ALL_DAYS },
    { label: 'Weekdays', days: [1, 2, 3, 4, 5] },
    { label: 'Weekends', days: [6, 0] },
]

// The weekday (0 is Sunday) and minutes since midnight right now in a given timezone. A schedule runs on that
// zone's clock, so "today" and "now" must come from it and not from the browser's own zone.
export function nowInZone(timezone: string, date: Date = new Date()): { day: number; minutes: number } {
    try {
        const parts = new Intl.DateTimeFormat('en-US', {
            timeZone: timezone,
            weekday: 'short',
            hour: '2-digit',
            minute: '2-digit',
            hourCycle: 'h23',
        }).formatToParts(date)
        const get = (type: string) => parts.find((part) => part.type === type)?.value ?? ''
        const day = DAY_LABELS.indexOf(get('weekday'))
        return {
            day: day === -1 ? date.getDay() : day,
            minutes: Number(get('hour')) * MINUTES_PER_HOUR + Number(get('minute')),
        }
    } catch {
        return { day: date.getDay(), minutes: date.getHours() * MINUTES_PER_HOUR + date.getMinutes() }
    }
}

export const browserZone = () => Intl.DateTimeFormat().resolvedOptions().timeZone

// Every time zone the browser knows, with UTC and the schedule's current zone always included.
export function timeZones(current: string): string[] {
    const intl = Intl as typeof Intl & { supportedValuesOf?: (key: 'timeZone') => string[] }
    const list = intl.supportedValuesOf?.('timeZone') ?? []
    const zones = new Set(['UTC', ...list])
    zones.add(current)
    return [...zones].sort()
}
