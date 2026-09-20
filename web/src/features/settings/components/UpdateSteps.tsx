import { CheckCircle2, Circle, Loader2, XCircle } from 'lucide-react'
import { stepLabel } from '@/shared/lib/update'
import type { UpdateStep, UpdateStepState } from '@/shared/types/api'

const STEP_STATE_WORDS: Record<UpdateStepState, string> = {
    pending: 'Waiting',
    running: 'In progress',
    done: 'Done',
    failed: 'Failed',
}

function StepIcon({ state }: { state: UpdateStepState }) {
    if (state === 'done') return <CheckCircle2 aria-hidden="true" className="h-4 w-4 text-status-ok-fg" />
    if (state === 'failed') return <XCircle aria-hidden="true" className="h-4 w-4 text-status-err-fg" />
    if (state === 'running') {
        return <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin text-status-info-fg" />
    }
    return <Circle aria-hidden="true" className="h-4 w-4 text-muted-foreground" />
}

// What the updater has done so far, one line per step. The state is spoken as well as shown.
function UpdateSteps({ steps }: { steps: UpdateStep[] }) {
    if (steps.length === 0) return null
    return (
        <ol aria-label="Update steps" className="flex flex-col gap-2">
            {steps.map((step) => (
                <li
                    key={step.name}
                    data-testid={`update-step-${step.name}`}
                    data-state={step.state}
                    className="flex items-start gap-2.5 text-sm"
                >
                    <span className="mt-0.5">
                        <StepIcon state={step.state} />
                    </span>
                    <span className="min-w-0">
                        <span className={step.state === 'pending' ? 'text-muted-foreground' : 'font-medium'}>
                            <span className="sr-only">{STEP_STATE_WORDS[step.state]}: </span>
                            {stepLabel(step.name)}
                        </span>
                        {step.message && (
                            <span className="block text-[13px] text-muted-foreground">{step.message}</span>
                        )}
                    </span>
                </li>
            ))}
        </ol>
    )
}

export default UpdateSteps
