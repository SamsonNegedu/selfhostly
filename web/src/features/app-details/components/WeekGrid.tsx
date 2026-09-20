import { useEffect, useState } from 'react'
import { Check } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { DAY_LABELS, MINUTES_IN_DAY, nowInZone, runningWindow } from '../lib/schedule-cron'

const HOUR_TICKS = [0, 6, 12, 18, 24]
const WEEK_ORDER = [1, 2, 3, 4, 5, 6, 0]
const NOW_REFRESH_MS = 60_000

// One row per weekday with a 24 hour bar, so the week can be read at a glance: the day is on or off, and when it
// runs is the filled part of its bar. A day is switched on and off by pressing its name.
function WeekGrid({
    start,
    stop,
    days,
    timezone,
    onToggle,
}: {
    start: { hour: number; minute: number }
    stop: { hour: number; minute: number }
    days: number[]
    timezone: string
    onToggle: (day: number) => void
}) {
    const segments = runningWindow(start, stop)
    // "Now" moves once a minute, on the schedule's own clock.
    const [now, setNow] = useState(() => nowInZone(timezone))
    const [shownZone, setShownZone] = useState(timezone)
    if (shownZone !== timezone) {
        setShownZone(timezone)
        setNow(nowInZone(timezone))
    }
    useEffect(() => {
        const timer = setInterval(() => setNow(nowInZone(timezone)), NOW_REFRESH_MS)
        return () => clearInterval(timer)
    }, [timezone])
    const today = now.day

    return (
        <div className="flex flex-col gap-1.5" role="group" aria-label="Days and running hours">
            <div aria-hidden="true" className="relative ml-[88px] h-4 text-caption text-muted-foreground">
                {HOUR_TICKS.map((hour) => (
                    <span
                        key={hour}
                        className={cn('absolute', hour === 24 ? 'right-0' : hour === 0 ? 'left-0' : '-translate-x-1/2')}
                        style={hour > 0 && hour < 24 ? { left: `${(hour / 24) * 100}%` } : undefined}
                    >
                        {String(hour).padStart(2, '0')}
                    </span>
                ))}
            </div>
            {WEEK_ORDER.map((day) => {
                const on = days.includes(day)
                return (
                    <div key={day} className="flex items-center gap-3">
                        <button
                            type="button"
                            aria-pressed={on}
                            aria-label={`${DAY_LABELS[day]}${day === today ? ' (today)' : ''}, ${on ? 'runs' : 'off'}`}
                            onClick={() => onToggle(day)}
                            className={cn(
                                'flex min-h-[44px] w-[76px] shrink-0 items-center gap-1.5 rounded-lg border px-2.5 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring md:min-h-[32px]',
                                on
                                    ? 'border-primary bg-primary text-primary-foreground'
                                    : 'border-dashed border-input text-muted-foreground hover:text-foreground',
                            )}
                        >
                            {on ? (
                                <Check aria-hidden="true" className="h-3.5 w-3.5" />
                            ) : (
                                <span aria-hidden="true" className="h-3.5 w-3.5" />
                            )}
                            {DAY_LABELS[day]}
                        </button>
                        <div
                            aria-hidden="true"
                            className={cn(
                                'relative h-5 flex-1 overflow-hidden rounded-md',
                                on ? 'bg-muted' : 'bg-muted/40',
                            )}
                        >
                            {on &&
                                segments.map((segment) => (
                                    <div
                                        key={segment.from}
                                        className="absolute inset-y-0 rounded-md bg-status-ok"
                                        style={{
                                            left: `${(segment.from / MINUTES_IN_DAY) * 100}%`,
                                            width: `${((segment.to - segment.from) / MINUTES_IN_DAY) * 100}%`,
                                        }}
                                    />
                                ))}
                            {day === today && (
                                <div
                                    aria-hidden="true"
                                    title="Now"
                                    className="absolute inset-y-0 w-0.5 bg-foreground"
                                    style={{ left: `${(now.minutes / MINUTES_IN_DAY) * 100}%` }}
                                />
                            )}
                        </div>
                    </div>
                )
            })}
            <p className="ml-[88px] flex items-center gap-1.5 text-detail text-muted-foreground">
                <span aria-hidden="true" className="h-3 w-0.5 bg-foreground" />
                Now, {String(Math.floor(now.minutes / 60)).padStart(2, '0')}:{String(now.minutes % 60).padStart(2, '0')}{' '}
                in {timezone.replace(/_/g, ' ')}
            </p>
        </div>
    )
}

export default WeekGrid
