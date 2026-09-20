import { useEffect, useMemo, useState } from 'react'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { useAppSchedule, useDeleteAppSchedule, useTestSchedule, useUpdateAppSchedule } from '@/shared/services/api'
import type { ScheduleNextRuns } from '@/shared/types/api'
import { ALL_DAYS, browserZone, buildCron, parseSimpleCron } from '../lib/schedule-cron'

export interface ScheduleFormState {
    enabled: boolean
    start_cron: string
    stop_cron: string
    timezone: string
}

export interface ScheduleErrors {
    start?: string
    stop?: string
}

const DEFAULT_START = { hour: 8, minute: 0 }
const DEFAULT_STOP = { hour: 22, minute: 0 }

// The state behind the schedule editor: the saved schedule, the form being edited, what the server says the times
// mean, and saving and deleting. The screen only draws it.
export function useScheduleForm(appId: string, nodeId: string) {
    const { data: schedule, isLoading, error } = useAppSchedule(appId, nodeId)
    const updateSchedule = useUpdateAppSchedule(appId, nodeId)
    const deleteSchedule = useDeleteAppSchedule(appId, nodeId)
    const testSchedule = useTestSchedule()
    const { toast } = useToast()

    const saved: ScheduleFormState | null = useMemo(
        () =>
            schedule
                ? {
                      enabled: schedule.enabled,
                      start_cron: schedule.start_cron || '',
                      stop_cron: schedule.stop_cron || '',
                      timezone: schedule.timezone || 'UTC',
                  }
                : null,
        [schedule],
    )
    const [form, setForm] = useState<ScheduleFormState | null>(saved)
    const [custom, setCustom] = useState(false)
    const [nextRuns, setNextRuns] = useState<ScheduleNextRuns | null>(null)
    const [errors, setErrors] = useState<ScheduleErrors>({})

    // Back to a saved state. A schedule the simple form cannot express opens as cron text.
    const reset = (state: ScheduleFormState | null) => {
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
            },
        )
        // The mutation object changes on every render, so only the form and the ids decide when to test.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [form?.enabled, form?.start_cron, form?.stop_cron, form?.timezone, appId, nodeId])

    const dirty = form !== null && (saved === null || JSON.stringify(saved) !== JSON.stringify(form))
    const same = form !== null && form.start_cron !== '' && form.start_cron === form.stop_cron
    const incomplete = form !== null && form.start_cron === '' && form.stop_cron === ''
    const canSave = dirty && !errors.start && !errors.stop && !same && !incomplete

    const update = (patch: Partial<ScheduleFormState>) => {
        if (form) setForm({ ...form, ...patch })
    }

    // The first schedule for an app that has none: every day, on at 08:00 and off at 22:00, in this browser's zone.
    const createSchedule = () => {
        setCustom(false)
        setForm({
            enabled: true,
            start_cron: buildCron({ ...DEFAULT_START, days: ALL_DAYS }),
            stop_cron: buildCron({ ...DEFAULT_STOP, days: ALL_DAYS }),
            timezone: browserZone(),
        })
    }

    const save = () => {
        if (!form) return
        updateSchedule.mutate(form, {
            onSuccess: () => toast.success('Schedule saved', 'The app will start and stop on this schedule'),
            onError: (failure) => toast.error('Could not save the schedule', describeError(failure)),
        })
    }

    // `onDeleted` runs once the server has removed it, for example to close the confirmation.
    const remove = (onDeleted: () => void) =>
        deleteSchedule.mutate(undefined, {
            onSuccess: () => {
                toast.success('Schedule deleted', 'The app no longer starts and stops by itself')
                onDeleted()
                setForm(null)
            },
            onError: (failure) => toast.error('Could not delete the schedule', describeError(failure)),
        })

    return {
        schedule,
        isLoading,
        error,
        saved,
        form,
        custom,
        setCustom,
        nextRuns,
        errors,
        dirty,
        same,
        incomplete,
        canSave,
        saving: updateSchedule.isPending,
        deleting: deleteSchedule.isPending,
        reset,
        update,
        createSchedule,
        save,
        remove,
    }
}
