import { Card, CardContent } from '@/shared/components/ui/Card'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { Switch } from '@/shared/components/ui/Switch'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { useSettings, useUpdateSettings } from '@/shared/services/api'

// Behavior of the server as a whole. A change is saved the moment you make it.
function GeneralSection() {
    const { toast } = useToast()
    const { data: settings } = useSettings()
    const update = useUpdateSettings()

    if (!settings) return <Skeleton role="status" aria-label="Loading general settings" className="h-24 rounded-xl" />

    const toggle = (autoStart: boolean) =>
        update.mutate(
            { auto_start_apps: autoStart },
            {
                onSuccess: () => toast.success('Saved', autoStart ? 'Apps start when the server boots' : 'Apps stay stopped after a reboot until you start them'),
                onError: (failure) => toast.error('Could not save', describeError(failure)),
            }
        )

    return (
        <Card>
            <CardContent className="flex items-start justify-between gap-4 p-4">
                <div>
                    <label htmlFor="auto-start" className="text-sm font-medium">
                        Start apps when the server boots
                    </label>
                    <p className="mt-1 text-[13px] text-muted-foreground">After a reboot or power cut, every app that was running starts again by itself.</p>
                </div>
                <Switch id="auto-start" checked={settings.auto_start_apps} onCheckedChange={toggle} disabled={update.isPending} />
            </CardContent>
        </Card>
    )
}

export default GeneralSection
