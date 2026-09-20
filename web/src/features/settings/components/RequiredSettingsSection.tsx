import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import type { UpdatePlan } from '@/shared/types/api'

interface RequiredSettingsSectionProps {
    required: UpdatePlan['settings']['required_missing']
    values: Record<string, string>
    onChange: (key: string, value: string) => void
}

// The settings this release cannot start without, one field each.
function RequiredSettingsSection({ required, values, onChange }: RequiredSettingsSectionProps) {
    return (
        <section aria-label="Settings this release needs" className="flex flex-col gap-3">
            <h4 className="text-compact font-semibold">Settings this release needs</h4>
            {required.map((setting) => (
                <Field key={setting.key} label={setting.key} hint={setting.description}>
                    <Input
                        data-testid={`update-input-${setting.key}`}
                        type={setting.secret ? 'password' : 'text'}
                        value={values[setting.key] ?? ''}
                        onChange={(event) => onChange(setting.key, event.target.value)}
                        autoComplete="off"
                        spellCheck={false}
                    />
                </Field>
            ))}
        </section>
    )
}

export default RequiredSettingsSection
