import React from 'react'
import { Check } from 'lucide-react'

interface Step {
    id: number
    label: string
    status: 'pending' | 'current' | 'completed'
}

interface ProgressIndicatorProps {
    steps: Step[]
}

function ProgressIndicator({ steps }: ProgressIndicatorProps) {
    return (
        <div className="mb-8">
            {/* Progress bar background */}
            <div className="relative">
                <div className="h-2 bg-muted rounded-full overflow-hidden">
                    {/* Progress fill */}
                    <div
                        className="h-full bg-primary transition-all duration-500 ease-in-out"
                        style={{
                            width: `${((steps.filter(s => s.status === 'completed').length) / (steps.length - 1)) * 100}%`
                        }}
                    />
                </div>

                {/* Step indicators */}
                <div className="absolute top-0 left-0 right-0 flex justify-between -mt-1">
                    {steps.map((step) => {
                        const isCompleted = step.status === 'completed'
                        const isCurrent = step.status === 'current'
                        const isPending = step.status === 'pending'

                        return (
                            <div key={step.id} className="relative">
                                {/* Step circle */}
                                <div
                                    className={`
                                        w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium
                                        transition-all duration-300
                                        ${isCompleted ? 'bg-primary text-primary-foreground' : ''}
                                        ${isCurrent ? 'bg-primary text-primary-foreground ring-4 ring-primary/20' : ''}
                                        ${isPending ? 'bg-muted text-muted-foreground' : ''}
                                    `}
                                >
                                    {isCompleted ? <Check className="h-4 w-4" /> : step.id}
                                </div>

                                {/* Step label */}
                                <div
                                    className={`
                                        absolute -bottom-6 left-1/2 -translate-x-1/2 whitespace-nowrap text-sm
                                        transition-colors duration-300
                                        ${isCurrent ? 'font-semibold text-foreground' : 'text-muted-foreground'}
                                    `}
                                >
                                    {step.label}
                                </div>

                                {/* Current step indicator */}
                                {isCurrent && (
                                    <div className="absolute -inset-1 -z-10 w-10 h-10 bg-primary/10 rounded-full animate-pulse" />
                                )}
                            </div>
                        )
                    })}
                </div>
            </div>

            {/* Spacer for labels */}
            <div className="h-6" />
        </div>
    )
}

export default ProgressIndicator
