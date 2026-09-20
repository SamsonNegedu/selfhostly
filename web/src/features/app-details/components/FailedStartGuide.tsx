import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { AlertTriangle, Loader2, Play } from 'lucide-react'
import { Button, buttonClasses } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { DiffBlock } from '@/shared/components/ui/DiffBlock'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { appHref } from '@/shared/lib/routes'
import { useApps, useStartApp, useUpdateApp } from '@/shared/services/api'
import type { App } from '@/shared/types/api'
import { collectHostPorts, detectPortConflict, replaceHostPort, suggestFreePort } from '../lib/port-conflict'

type Step = 'idle' | 'saving' | 'starting'

// What to do about an app that failed to start. It says what went wrong, and for a port that is already in
// use it offers a specific change to the compose file, shown as a diff, that you apply and retry in one step.
function FailedStartGuide({ app }: { app: App }) {
    const { toast } = useToast()
    const updateApp = useUpdateApp(app.id, app.node_id)
    const startApp = useStartApp()
    const { data: apps = [] } = useApps([app.node_id])
    const [step, setStep] = useState<Step>('idle')

    const conflictPort = detectPortConflict(app.error_message)
    // The failure names a port that the compose file no longer publishes, so it was edited after the failure.
    const editedSinceFailure = conflictPort !== null && replaceHostPort(app.compose_content, conflictPort, conflictPort) === null

    const fix = useMemo(() => {
        if (conflictPort === null) return null
        const used = collectHostPorts(apps.map((other) => other.compose_content))
        used.add(conflictPort)
        const suggested = suggestFreePort(conflictPort, used)
        if (suggested === null) return null
        const after = replaceHostPort(app.compose_content, conflictPort, suggested)
        return after === null ? null : { from: conflictPort, to: suggested, after }
    }, [apps, app.compose_content, conflictPort])

    const applyAndRetry = async () => {
        if (!fix) return
        try {
            setStep('saving')
            await updateApp.mutateAsync({ compose_content: fix.after })
            setStep('starting')
            await startApp.mutateAsync({ id: app.id, nodeId: app.node_id })
            toast.success('Fix applied', `${app.name} is starting on port ${fix.to}`)
        } catch (error) {
            toast.error('Could not apply the fix', describeError(error))
        } finally {
            setStep('idle')
        }
    }

    const retry = () =>
        startApp.mutate(
            { id: app.id, nodeId: app.node_id },
            {
                onSuccess: () => toast.success('App starting', `${app.name} is starting`),
                onError: (error) => toast.error('Could not start app', describeError(error)),
            }
        )

    const working = step !== 'idle' || startApp.isPending

    return (
        <Card className="border-status-err/50" aria-label="Why it did not start" role="region">
            <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                    <AlertTriangle className="h-4 w-4 text-status-err-fg" />
                    {fix ? `Port ${fix.from} is already in use` : editedSinceFailure ? 'Config changed since it failed' : `${app.name} did not start`}
                </CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
                <p className="text-sm text-muted-foreground">
                    {fix
                        ? `Another program on this machine already listens on ${fix.from}. Publish ${app.name} on ${fix.to} instead. That port is not used by any of your other apps on this node.`
                        : editedSinceFailure
                          ? `The last start failed on port ${conflictPort}, which the compose file no longer uses. Retry to start it with the current config.`
                          : app.error_message || 'The app reported an error. The deployment log shows each step that ran.'}
                </p>

                {fix && (
                    <div className="flex flex-col gap-2">
                        <p className="text-[13px] font-medium">Change to the compose file</p>
                        <DiffBlock before={app.compose_content} after={fix.after} aria-label="Compose file change" />
                    </div>
                )}

                <div className="flex flex-wrap items-center gap-2">
                    {fix ? (
                        <Button onClick={applyAndRetry} disabled={working}>
                            {working ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                            {step === 'saving' ? 'Saving' : step === 'starting' ? 'Starting' : `Apply and retry on ${fix.to}`}
                        </Button>
                    ) : (
                        <Button onClick={retry} disabled={working}>
                            {working ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                            Retry start
                        </Button>
                    )}
                    <Link to={appHref(app, 'logs')} className={buttonClasses({ variant: 'outline' })}>
                        View deployment log
                    </Link>
                    <Link to={appHref(app, 'config')} className={buttonClasses({ variant: 'ghost' })}>
                        Edit config
                    </Link>
                </div>
            </CardContent>
        </Card>
    )
}

export default FailedStartGuide
