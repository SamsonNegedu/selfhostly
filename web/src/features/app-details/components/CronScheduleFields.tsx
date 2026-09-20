import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import type { ScheduleErrors, ScheduleFormState } from '../hooks/useScheduleForm'
import TimezoneSelect from './TimezoneSelect'

interface CronScheduleFieldsProps {
    form: ScheduleFormState
    errors: ScheduleErrors
    onChange: (patch: Partial<ScheduleFormState>) => void
}

// For schedules the simple form cannot express: the start and stop times as cron text.
function CronScheduleFields({ form, errors, onChange }: CronScheduleFieldsProps) {
    return (
        <>
            <div className="grid gap-4 sm:grid-cols-2">
                <Field
                    label="Start expression"
                    hint="Minute, hour, day, month, weekday. For example 0 8 * * 1-5"
                    error={errors.start}
                >
                    <Input
                        value={form.start_cron}
                        onChange={(event) => onChange({ start_cron: event.target.value })}
                        className="font-mono"
                    />
                </Field>
                <Field label="Stop expression" hint="For example 0 22 * * 1-5" error={errors.stop}>
                    <Input
                        value={form.stop_cron}
                        onChange={(event) => onChange({ stop_cron: event.target.value })}
                        className="font-mono"
                    />
                </Field>
            </div>
            <Field label="Timezone" className="sm:max-w-sm">
                <TimezoneSelect value={form.timezone} onChange={(timezone) => onChange({ timezone })} />
            </Field>
        </>
    )
}

export default CronScheduleFields
