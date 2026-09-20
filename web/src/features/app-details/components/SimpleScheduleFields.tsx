import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { cn } from '@/shared/lib/utils'
import {
    describeDays,
    formatTime,
    parseTime,
    PRESETS,
    windowHours,
    type SimpleSchedule,
    type SimpleTime,
} from '../lib/schedule-cron'
import type { ScheduleErrors, ScheduleFormState } from '../hooks/useScheduleForm'
import TimezoneSelect from './TimezoneSelect'
import WeekGrid from './WeekGrid'

const formatHours = (hours: number) => `${Number.isInteger(hours) ? hours : hours.toFixed(1)} hours`

interface SimpleScheduleFieldsProps {
    form: ScheduleFormState
    errors: ScheduleErrors
    start: SimpleSchedule
    stop: SimpleSchedule
    days: number[]
    onChange: (patch: Partial<ScheduleFormState>) => void
    onTimes: (start: SimpleTime, stop: SimpleTime, days: number[]) => void
    onToggleDay: (day: number) => void
}

// The everyday form: a start time, a stop time, the days, and a week view of when the app runs.
function SimpleScheduleFields({
    form,
    errors,
    start,
    stop,
    days,
    onChange,
    onTimes,
    onToggleDay,
}: SimpleScheduleFieldsProps) {
    return (
        <>
            <div className="flex flex-wrap items-start gap-x-4 gap-y-3">
                <Field label="Starts at" error={errors.start} className="w-[150px]">
                    <Input
                        type="time"
                        value={formatTime(start)}
                        onChange={(event) => {
                            const time = parseTime(event.target.value)
                            if (time) onTimes(time, stop, days)
                        }}
                    />
                </Field>
                <Field label="Stops at" error={errors.stop} className="w-[150px]">
                    <Input
                        type="time"
                        value={formatTime(stop)}
                        onChange={(event) => {
                            const time = parseTime(event.target.value)
                            if (time) onTimes(start, time, days)
                        }}
                    />
                </Field>
                <Field label="Timezone" className="min-w-[200px] flex-1">
                    <TimezoneSelect value={form.timezone} onChange={(timezone) => onChange({ timezone })} />
                </Field>
            </div>

            <div className="flex flex-col gap-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                    <p className="text-compact font-medium">Days it runs</p>
                    <div className="flex flex-wrap gap-1.5" role="group" aria-label="Day presets">
                        {PRESETS.map((preset) => {
                            const active =
                                preset.days.length === days.length && preset.days.every((day) => days.includes(day))
                            return (
                                <button
                                    key={preset.label}
                                    type="button"
                                    aria-pressed={active}
                                    onClick={() => onTimes(start, stop, [...preset.days])}
                                    className={cn(
                                        'min-h-[44px] rounded-full border px-3 text-compact font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring md:min-h-[30px]',
                                        active
                                            ? 'border-primary bg-primary text-primary-foreground'
                                            : 'border-input text-muted-foreground hover:text-foreground',
                                    )}
                                >
                                    {preset.label}
                                </button>
                            )
                        })}
                    </div>
                </div>
                <WeekGrid start={start} stop={stop} days={days} timezone={form.timezone} onToggle={onToggleDay} />
                <p className="text-sm text-muted-foreground">
                    Runs {formatHours(windowHours(start, stop))} a day, {describeDays(days)}.
                </p>
            </div>
        </>
    )
}

export default SimpleScheduleFields
