import { useEffect, useMemo, useState } from 'react'
import { CalendarClock, Check, Loader2, Play, Save, Square, Trash2, Undo2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { Switch } from '@/shared/components/ui/Switch'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { formatUntil } from '@/shared/lib/format'
import { formatRunTime, upcomingRuns } from '@/shared/lib/schedule'
import { cn } from '@/shared/lib/utils'
import { useAppSchedule, useDeleteAppSchedule, useTestSchedule, useUpdateAppSchedule } from '@/shared/services/api'
import type { ScheduleNextRuns } from '@/shared/types/api'
import {
    ALL_DAYS,
    buildCron,
    DAY_LABELS,
    describeDays,
    PRESETS,
    formatTime,
    MINUTES_IN_DAY,
    nowInZone,
    parseSimpleCron,
    parseTime,
    runningWindow,
    windowHours,
} from '../lib/schedule-cron'

interface ScheduleEditorProps {
    appId: string
    nodeId: string
}

interface FormState {
    enabled: boolean
    start_cron: string
    stop_cron: string
    timezone: string
}

const DEFAULT_START = { hour: 8, minute: 0 }
const DEFAULT_STOP = { hour: 22, minute: 0 }
const HOUR_TICKS = [0, 6, 12, 18, 24]

const browserZone = () => Intl.DateTimeFormat().resolvedOptions().timeZone

function timeZones(current: string): string[] {
    const intl = Intl as typeof Intl & { supportedValuesOf?: (key: 'timeZone') => string[] }
    const list = intl.supportedValuesOf?.('timeZone') ?? []
    const zones = new Set(['UTC', ...list])
    zones.add(current)
    return [...zones].sort()
}


// When the app starts and stops by itself. Most schedules are a start time, a stop time and some days, so that is
// what is shown. A schedule the simple form cannot express is edited as cron text instead.
export function ScheduleEditor({ appId, nodeId }: ScheduleEditorProps) {
    const { data: schedule, isLoading, error } = useAppSchedule(appId, nodeId)
    const updateSchedule = useUpdateAppSchedule(appId, nodeId)
    const deleteSchedule = useDeleteAppSchedule(appId, nodeId)
    const testSchedule = useTestSchedule()
    const { toast } = useToast()

    const saved: FormState | null = useMemo(
        () => (schedule ? { enabled: schedule.enabled, start_cron: schedule.start_cron || '', stop_cron: schedule.stop_cron || '', timezone: schedule.timezone || 'UTC' } : null),
        [schedule]
    )
    const [form, setForm] = useState<FormState | null>(saved)
    const [custom, setCustom] = useState(false)
    const [nextRuns, setNextRuns] = useState<ScheduleNextRuns | null>(null)
    const [errors, setErrors] = useState<{ start?: string; stop?: string }>({})
    const [confirmDelete, setConfirmDelete] = useState(false)

    const reset = (state: FormState | null) => {
        setForm(state)
        if (state) setCustom(!(parseSimpleCron(state.start_cron) && parseSimpleCron(state.stop_cron)))
    }

    useEffect(() => reset(saved), [saved])

    // Ask the server what the expressions mean, so wrong ones are caught before saving.
    useEffect(() => {
        if (!form || !form.enabled || (!form.start_cron && !form.stop_cron)) {
            setNextRuns(null)
            setErrors({})
            return
        }
        testSchedule.mutate(
            { ...form, app_id: appId, node_id: nodeId },
            {
                onSuccess: (data) => {
                    setNextRuns(data)
                    setErrors({})
                },
                onError: (failure) => {
                    const message = describeError(failure)
                    setNextRuns(null)
                    if (/must occur after|cannot be the same/i.test(message)) setErrors({ stop: message })
                    else if (/stop/i.test(message)) setErrors({ stop: message })
                    else setErrors({ start: message })
                },
            }
        )
        // The mutation object changes on every render, so only the form and the ids decide when to test.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [form?.enabled, form?.start_cron, form?.stop_cron, form?.timezone, appId, nodeId])

    if (isLoading) {
        return (
            <div role="status" aria-label="Loading schedule" className="flex flex-col gap-4">
                <Skeleton className="h-24 rounded-xl" />
                <Skeleton className="h-48 rounded-xl" />
            </div>
        )
    }

    if (error) {
        return <p role="alert" className="text-sm text-status-err-fg">Could not load the schedule. {describeError(error)}</p>
    }

    if (!form) {
        return (
            <EmptyState
                icon={<CalendarClock className="h-5 w-5" />}
                title="Runs all the time"
                description="Give it a start and a stop time and it turns itself on and off. Handy for apps you only need during the day, and for saving power."
                action={
                    <Button
                        onClick={() => {
                            setCustom(false)
                            setForm({ enabled: true, start_cron: buildCron({ ...DEFAULT_START, days: ALL_DAYS }), stop_cron: buildCron({ ...DEFAULT_STOP, days: ALL_DAYS }), timezone: browserZone() })
                        }}
                    >
                        Create schedule
                    </Button>
                }
                className="py-14"
            />
        )
    }

    const start = parseSimpleCron(form.start_cron)
    const stop = parseSimpleCron(form.stop_cron)
    const simple = !custom && start !== null && stop !== null
    const days = start?.days ?? ALL_DAYS
    const dirty = saved === null || JSON.stringify(saved) !== JSON.stringify(form)
    const same = form.start_cron !== '' && form.start_cron === form.stop_cron
    const incomplete = form.start_cron === '' && form.stop_cron === ''
    const canSave = dirty && !errors.start && !errors.stop && !same && !incomplete

    const update = (patch: Partial<FormState>) => setForm({ ...form, ...patch })
    const setTimes = (nextStart: { hour: number; minute: number }, nextStop: { hour: number; minute: number }, nextDays: number[]) =>
        update({ start_cron: buildCron({ ...nextStart, days: nextDays }), stop_cron: buildCron({ ...nextStop, days: nextDays }) })

    const toggleDay = (day: number) => {
        if (!start || !stop) return
        const next = days.includes(day) ? days.filter((other) => other !== day) : [...days, day]
        if (next.length === 0) return
        setTimes(start, stop, next)
    }

    const save = () =>
        updateSchedule.mutate(form, {
            onSuccess: () => toast.success('Schedule saved', 'The app will start and stop on this schedule'),
            onError: (failure) => toast.error('Could not save the schedule', describeError(failure)),
        })

    const remove = () =>
        deleteSchedule.mutate(undefined, {
            onSuccess: () => {
                toast.success('Schedule deleted', 'The app no longer starts and stops by itself')
                setConfirmDelete(false)
                setForm(null)
            },
            onError: (failure) => toast.error('Could not delete the schedule', describeError(failure)),
        })

    const upcoming = upcomingRuns(nextRuns)

    return (
        <div className="flex flex-col gap-5">
            <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,1fr)_340px]">
                <Card>
                    <CardHeader>
                        <div className="flex flex-wrap items-center justify-between gap-3">
                            <CardTitle className="text-base">Schedule</CardTitle>
                            <div className="flex items-center gap-3">
                                <label className="flex min-h-[44px] items-center gap-2 text-sm font-medium md:min-h-0">
                                    <Switch checked={form.enabled} onCheckedChange={(enabled) => update({ enabled })} aria-label="Schedule enabled" />
                                    {form.enabled ? 'On' : 'Paused'}
                                </label>
                                {schedule?.id && (
                                    <Button variant="ghost" size="icon" aria-label="Delete schedule" onClick={() => setConfirmDelete(true)}>
                                        <Trash2 className="h-4 w-4" />
                                    </Button>
                                )}
                            </div>
                        </div>
                    </CardHeader>
                    <CardContent className="flex flex-col gap-5">
                        {simple && start && stop ? (
                            <>
                                <div className="flex flex-wrap items-start gap-x-4 gap-y-3">
                                    <Field label="Starts at" error={errors.start} className="w-[150px]">
                                        <Input type="time" value={formatTime(start)} onChange={(event) => { const time = parseTime(event.target.value); if (time) setTimes(time, stop, days) }} />
                                    </Field>
                                    <Field label="Stops at" error={errors.stop} className="w-[150px]">
                                        <Input type="time" value={formatTime(stop)} onChange={(event) => { const time = parseTime(event.target.value); if (time) setTimes(start, time, days) }} />
                                    </Field>
                                    <Field label="Timezone" className="min-w-[200px] flex-1">
                                        <TimezoneSelect value={form.timezone} onChange={(timezone) => update({ timezone })} />
                                    </Field>
                                </div>

                                <div className="flex flex-col gap-3">
                                    <div className="flex flex-wrap items-center justify-between gap-2">
                                        <p className="text-[13px] font-medium">Days it runs</p>
                                        <div className="flex flex-wrap gap-1.5" role="group" aria-label="Day presets">
                                            {PRESETS.map((preset) => {
                                                const active = preset.days.length === days.length && preset.days.every((day) => days.includes(day))
                                                return (
                                                    <button
                                                        key={preset.label}
                                                        type="button"
                                                        aria-pressed={active}
                                                        onClick={() => setTimes(start, stop, [...preset.days])}
                                                        className={cn(
                                                            'min-h-[44px] rounded-full border px-3 text-[13px] font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring md:min-h-[30px]',
                                                            active ? 'border-primary bg-primary text-primary-foreground' : 'border-input text-muted-foreground hover:text-foreground'
                                                        )}
                                                    >
                                                        {preset.label}
                                                    </button>
                                                )
                                            })}
                                        </div>
                                    </div>
                                    <WeekGrid start={start} stop={stop} days={days} timezone={form.timezone} onToggle={toggleDay} />
                                    <p className="text-sm text-muted-foreground">
                                        Runs {formatHours(windowHours(start, stop))} a day, {describeDays(days)}.
                                    </p>
                                </div>
                            </>
                        ) : (
                            <>
                                <div className="grid gap-4 sm:grid-cols-2">
                                    <Field label="Start expression" hint="Minute, hour, day, month, weekday. For example 0 8 * * 1-5" error={errors.start}>
                                        <Input value={form.start_cron} onChange={(event) => update({ start_cron: event.target.value })} className="font-mono" />
                                    </Field>
                                    <Field label="Stop expression" hint="For example 0 22 * * 1-5" error={errors.stop}>
                                        <Input value={form.stop_cron} onChange={(event) => update({ stop_cron: event.target.value })} className="font-mono" />
                                    </Field>
                                </div>
                                <Field label="Timezone" className="sm:max-w-sm">
                                    <TimezoneSelect value={form.timezone} onChange={(timezone) => update({ timezone })} />
                                </Field>
                            </>
                        )}

                        {(simple || (parseSimpleCron(form.start_cron) && parseSimpleCron(form.stop_cron))) && (
                            <button type="button" onClick={() => setCustom(!custom)} className="min-h-[44px] self-start text-sm font-medium text-muted-foreground underline-offset-2 hover:text-foreground hover:underline md:min-h-0">
                                {custom ? 'Use the simple form' : 'Edit as cron expressions'}
                            </button>
                        )}

                        {(same || (incomplete && form.enabled)) && (
                            <p role="alert" className="text-sm text-status-err-fg">
                                {same ? 'Start and stop cannot be the same time.' : 'Set a start time, a stop time, or both.'}
                            </p>
                        )}
                    </CardContent>
                </Card>

                <Card aria-label="Next runs" role="region">
                    <CardHeader>
                        <CardTitle className="text-base">Next runs</CardTitle>
                        <p className="text-[13px] text-muted-foreground">in {form.timezone.replace(/_/g, ' ')}</p>
                    </CardHeader>
                    <CardContent>
                        {!form.enabled ? (
                            <p className="text-sm text-muted-foreground">The schedule is paused, so nothing will run.</p>
                        ) : upcoming.length > 0 ? (
                            <ol className="flex flex-col">
                                {upcoming.map((run) => (
                                    <li key={`${run.action}-${run.at}`} className="flex items-center gap-3 border-b border-border py-2.5 first:pt-0 last:border-b-0 last:pb-0">
                                        <span
                                            aria-hidden="true"
                                            className={cn('flex h-8 w-8 shrink-0 items-center justify-center rounded-lg', run.action === 'start' ? 'bg-status-ok-bg text-status-ok-fg' : 'bg-status-idle-bg text-status-idle-fg')}
                                        >
                                            {run.action === 'start' ? <Play className="h-4 w-4" /> : <Square className="h-4 w-4" />}
                                        </span>
                                        <div className="min-w-0 flex-1">
                                            <p className="text-sm font-medium">{run.action === 'start' ? 'Starts' : 'Stops'}</p>
                                            <p className="text-[13px] text-muted-foreground">{formatRunTime(run.at, form.timezone)}</p>
                                        </div>
                                        <span className="shrink-0 text-[13px] tabular-nums text-muted-foreground">{formatUntil(run.at)}</span>
                                    </li>
                                ))}
                            </ol>
                        ) : (
                            <p className="text-sm text-muted-foreground">{errors.start || errors.stop ? 'Fix the problem on the left to see the next runs.' : 'Set a time to see the next runs.'}</p>
                        )}
                    </CardContent>
                </Card>
            </div>

            {dirty && (
                <div role="region" aria-label="Unsaved changes" className="sticky bottom-[calc(var(--mobile-nav-h)+12px)] z-20 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-card p-3 shadow-lg md:bottom-4">
                    <span className="text-sm font-medium">{saved === null ? 'New schedule, not saved yet' : 'Unsaved changes'}</span>
                    <div className="flex flex-wrap items-center gap-2">
                        <Button variant="ghost" onClick={() => reset(saved)} disabled={updateSchedule.isPending}>
                            <Undo2 className="h-4 w-4" />
                            Discard
                        </Button>
                        <Button onClick={save} disabled={!canSave || updateSchedule.isPending}>
                            {updateSchedule.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                            Save schedule
                        </Button>
                    </div>
                </div>
            )}

            <ConfirmationDialog
                open={confirmDelete}
                onOpenChange={setConfirmDelete}
                onConfirm={remove}
                title="Delete this schedule?"
                description="The app will stop starting and stopping by itself. It keeps its current state."
                confirmText="Delete schedule"
                isLoading={deleteSchedule.isPending}
                variant="destructive"
            />
        </div>
    )
}

const formatHours = (hours: number) => `${Number.isInteger(hours) ? hours : hours.toFixed(1)} hours`

function TimezoneSelect({ value, onChange }: { value: string; onChange: (zone: string) => void }) {
    return (
        <select
            value={value}
            onChange={(event) => onChange(event.target.value)}
            className="h-[38px] min-h-[44px] w-full rounded-lg border border-input bg-card px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:min-h-0"
        >
            {timeZones(value).map((zone) => (
                <option key={zone} value={zone}>
                    {zone.replace(/_/g, ' ')}
                </option>
            ))}
        </select>
    )
}

const WEEK_ORDER = [1, 2, 3, 4, 5, 6, 0]
const NOW_REFRESH_MS = 60_000

// One row per weekday with a 24 hour bar, so the week can be read at a glance: the day is on or off, and when it
// runs is the filled part of its bar. A day is switched on and off by pressing its name.
function WeekGrid({ start, stop, days, timezone, onToggle }: { start: { hour: number; minute: number }; stop: { hour: number; minute: number }; days: number[]; timezone: string; onToggle: (day: number) => void }) {
    const segments = runningWindow(start, stop)
    // "Now" moves once a minute, on the schedule's own clock.
    const [now, setNow] = useState(() => nowInZone(timezone))
    useEffect(() => {
        setNow(nowInZone(timezone))
        const timer = setInterval(() => setNow(nowInZone(timezone)), NOW_REFRESH_MS)
        return () => clearInterval(timer)
    }, [timezone])
    const today = now.day

    return (
        <div className="flex flex-col gap-1.5" role="group" aria-label="Days and running hours">
            <div aria-hidden="true" className="relative ml-[88px] h-4 text-[11px] text-muted-foreground">
                {HOUR_TICKS.map((hour) => (
                    <span key={hour} className={cn('absolute', hour === 24 ? 'right-0' : hour === 0 ? 'left-0' : '-translate-x-1/2')} style={hour > 0 && hour < 24 ? { left: `${(hour / 24) * 100}%` } : undefined}>
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
                                on ? 'border-primary bg-primary text-primary-foreground' : 'border-dashed border-input text-muted-foreground hover:text-foreground'
                            )}
                        >
                            {on ? <Check aria-hidden="true" className="h-3.5 w-3.5" /> : <span aria-hidden="true" className="h-3.5 w-3.5" />}
                            {DAY_LABELS[day]}
                        </button>
                        <div aria-hidden="true" className={cn('relative h-5 flex-1 overflow-hidden rounded-md', on ? 'bg-muted' : 'bg-muted/40')}>
                            {on &&
                                segments.map((segment) => (
                                    <div
                                        key={segment.from}
                                        className="absolute inset-y-0 rounded-md bg-status-ok"
                                        style={{ left: `${(segment.from / MINUTES_IN_DAY) * 100}%`, width: `${((segment.to - segment.from) / MINUTES_IN_DAY) * 100}%` }}
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
            <p className="ml-[88px] flex items-center gap-1.5 text-[12px] text-muted-foreground">
                <span aria-hidden="true" className="h-3 w-0.5 bg-foreground" />
                Now, {String(Math.floor(now.minutes / 60)).padStart(2, '0')}:{String(now.minutes % 60).padStart(2, '0')} in {timezone.replace(/_/g, ' ')}
            </p>
        </div>
    )
}
