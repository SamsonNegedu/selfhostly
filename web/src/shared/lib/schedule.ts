import type { ScheduleNextRuns } from '@/shared/types/api'

export interface UpcomingRun {
    action: 'start' | 'stop'
    at: string
}

// The next starts and stops in time order. Older servers only send the next start and the next stop, so those
// are used when the full list is missing.
export function upcomingRuns(nextRuns: ScheduleNextRuns | null | undefined): UpcomingRun[] {
    if (!nextRuns) return []
    if (nextRuns.upcoming && nextRuns.upcoming.length > 0) return nextRuns.upcoming
    return [
        ...(nextRuns.next_start ? [{ action: 'start' as const, at: nextRuns.next_start }] : []),
        ...(nextRuns.next_stop ? [{ action: 'stop' as const, at: nextRuns.next_stop }] : []),
    ].sort((a, b) => a.at.localeCompare(b.at))
}

// "Sun 20 Sept, 08:00" on the clock of the schedule's own timezone.
export function formatRunTime(iso: string, timeZone?: string): string {
    return new Date(iso).toLocaleString(undefined, {
        timeZone,
        weekday: 'short',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
    })
}
