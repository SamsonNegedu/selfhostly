import { useState } from 'react'
import { CalendarClock, Loader2, Save, Trash2, Undo2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import ActionBar from '@/shared/components/ui/ActionBar'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { Switch } from '@/shared/components/ui/Switch'
import { UnsavedChangesGuard } from '@/shared/components/UnsavedChangesGuard'
import { describeError } from '@/shared/lib/errors'
import { upcomingRuns } from '@/shared/lib/schedule'
import { useScheduleForm } from '../hooks/useScheduleForm'
import { ALL_DAYS, buildCron, parseSimpleCron, type SimpleTime } from '../lib/schedule-cron'
import CronScheduleFields from './CronScheduleFields'
import NextRunsCard from './NextRunsCard'
import SimpleScheduleFields from './SimpleScheduleFields'

interface ScheduleEditorProps {
    appId: string
    nodeId: string
}

// When the app starts and stops by itself. Most schedules are a start time, a stop time and some days, so that is
// what is shown. A schedule the simple form cannot express is edited as cron text instead.
export function ScheduleEditor({ appId, nodeId }: ScheduleEditorProps) {
    const schedule = useScheduleForm(appId, nodeId)
    const [confirmDelete, setConfirmDelete] = useState(false)
    const { form, errors } = schedule

    if (schedule.isLoading) {
        return (
            <div role="status" aria-label="Loading schedule" className="flex flex-col gap-4">
                <Skeleton className="h-24 rounded-xl" />
                <Skeleton className="h-48 rounded-xl" />
            </div>
        )
    }

    if (schedule.error) {
        return (
            <p role="alert" className="text-sm text-status-err-fg">
                Could not load the schedule. {describeError(schedule.error)}
            </p>
        )
    }

    if (!form) {
        return (
            <EmptyState
                icon={<CalendarClock className="h-5 w-5" />}
                title="Runs all the time"
                description="Give it a start and a stop time and it turns itself on and off. Handy for apps you only need during the day, and for saving power."
                action={<Button onClick={schedule.createSchedule}>Create schedule</Button>}
                className="py-14"
            />
        )
    }

    const start = parseSimpleCron(form.start_cron)
    const stop = parseSimpleCron(form.stop_cron)
    const simple = !schedule.custom && start !== null && stop !== null
    const days = start?.days ?? ALL_DAYS

    const setTimes = (nextStart: SimpleTime, nextStop: SimpleTime, nextDays: number[]) =>
        schedule.update({
            start_cron: buildCron({ ...nextStart, days: nextDays }),
            stop_cron: buildCron({ ...nextStop, days: nextDays }),
        })

    const toggleDay = (day: number) => {
        if (!start || !stop) return
        const next = days.includes(day) ? days.filter((other) => other !== day) : [...days, day]
        if (next.length === 0) return
        setTimes(start, stop, next)
    }

    return (
        <div className="flex flex-col gap-5">
            <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,1fr)_340px]">
                <Card>
                    <CardHeader>
                        <div className="flex flex-wrap items-center justify-between gap-3">
                            <CardTitle className="text-base">Schedule</CardTitle>
                            <div className="flex items-center gap-3">
                                <label className="flex min-h-[44px] items-center gap-2 text-sm font-medium md:min-h-0">
                                    <Switch
                                        checked={form.enabled}
                                        onCheckedChange={(enabled) => schedule.update({ enabled })}
                                        aria-label="Schedule enabled"
                                    />
                                    {form.enabled ? 'On' : 'Paused'}
                                </label>
                                {schedule.schedule?.id && (
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        aria-label="Delete schedule"
                                        onClick={() => setConfirmDelete(true)}
                                    >
                                        <Trash2 className="h-4 w-4" />
                                    </Button>
                                )}
                            </div>
                        </div>
                    </CardHeader>
                    <CardContent className="flex flex-col gap-5">
                        {simple && start && stop ? (
                            <SimpleScheduleFields
                                form={form}
                                errors={errors}
                                start={start}
                                stop={stop}
                                days={days}
                                onChange={schedule.update}
                                onTimes={setTimes}
                                onToggleDay={toggleDay}
                            />
                        ) : (
                            <CronScheduleFields form={form} errors={errors} onChange={schedule.update} />
                        )}

                        {(simple || (parseSimpleCron(form.start_cron) && parseSimpleCron(form.stop_cron))) && (
                            <button
                                type="button"
                                onClick={() => schedule.setCustom(!schedule.custom)}
                                className="min-h-[44px] self-start text-sm font-medium text-muted-foreground underline-offset-2 hover:text-foreground hover:underline md:min-h-0"
                            >
                                {schedule.custom ? 'Use the simple form' : 'Edit as cron expressions'}
                            </button>
                        )}

                        {(schedule.same || (schedule.incomplete && form.enabled)) && (
                            <p role="alert" className="text-sm text-status-err-fg">
                                {schedule.same
                                    ? 'Start and stop cannot be the same time.'
                                    : 'Set a start time, a stop time, or both.'}
                            </p>
                        )}
                    </CardContent>
                </Card>

                <NextRunsCard
                    enabled={form.enabled}
                    timezone={form.timezone}
                    runs={upcomingRuns(schedule.nextRuns)}
                    hasError={!!(errors.start || errors.stop)}
                />
            </div>

            <UnsavedChangesGuard when={schedule.dirty} />

            {schedule.dirty && (
                <ActionBar label="Unsaved changes">
                    <span className="text-sm font-medium">
                        {schedule.saved === null ? 'New schedule, not saved yet' : 'Unsaved changes'}
                    </span>
                    <div className="flex flex-wrap items-center gap-2">
                        <Button
                            variant="ghost"
                            onClick={() => schedule.reset(schedule.saved)}
                            disabled={schedule.saving}
                        >
                            <Undo2 className="h-4 w-4" />
                            Discard
                        </Button>
                        <Button onClick={schedule.save} disabled={!schedule.canSave || schedule.saving}>
                            {schedule.saving ? (
                                <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                            ) : (
                                <Save className="h-4 w-4" />
                            )}
                            Save schedule
                        </Button>
                    </div>
                </ActionBar>
            )}

            <ConfirmationDialog
                open={confirmDelete}
                onOpenChange={setConfirmDelete}
                onConfirm={() => schedule.remove(() => setConfirmDelete(false))}
                title="Delete this schedule?"
                description="The app will stop starting and stopping by itself. It keeps its current state."
                confirmText="Delete schedule"
                isLoading={schedule.deleting}
                variant="destructive"
            />
        </div>
    )
}
