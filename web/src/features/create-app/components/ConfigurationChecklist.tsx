import React from 'react'
import { Check, X, AlertCircle } from 'lucide-react'

interface ChecklistItem {
    id: string
    label: string
    checked: boolean
    error?: string
}

interface ConfigurationChecklistProps {
    items: ChecklistItem[]
}

function ConfigurationChecklist({ items }: ConfigurationChecklistProps) {
    const allChecked = items.every(item => item.checked)

    return (
        <div className="space-y-3">
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold">Configuration Checklist</h3>
                {allChecked ? (
                    <span className="text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
                        <Check className="h-3 w-3" />
                        Ready to deploy
                    </span>
                ) : (
                    <span className="text-xs text-muted-foreground">
                        {items.filter(i => i.checked).length} / {items.length} complete
                    </span>
                )}
            </div>

            <div className="space-y-2">
                {items.map((item) => (
                    <div
                        key={item.id}
                        className={`
                            flex items-start gap-3 p-3 rounded-lg border
                            ${item.checked
                                ? 'bg-green-50 dark:bg-green-900/10 border-green-200 dark:border-green-900/30'
                                : 'bg-muted/50 border-border'
                            }
                        `}
                    >
                        <div className={`
                            w-5 h-5 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5
                            ${item.checked
                                ? 'bg-green-500 text-white'
                                : 'bg-muted text-muted-foreground'
                            }
                        `}>
                            {item.checked ? (
                                <Check className="h-3 w-3" />
                            ) : item.error ? (
                                <X className="h-3 w-3 text-red-500" />
                            ) : (
                                <span className="text-xs">{item.id}</span>
                            )}
                        </div>

                        <div className="flex-1 min-w-0">
                            <p className={`
                                text-sm font-medium
                                ${item.checked ? 'text-green-900 dark:text-green-100' : 'text-foreground'}
                            `}>
                                {item.label}
                            </p>
                            {item.error && (
                                <p className="text-xs text-red-600 dark:text-red-400 mt-1 flex items-center gap-1">
                                    <AlertCircle className="h-3 w-3" />
                                    {item.error}
                                </p>
                            )}
                        </div>

                        {item.checked && (
                            <Check className="h-5 w-5 text-green-500 flex-shrink-0" />
                        )}
                    </div>
                ))}
            </div>

            {!allChecked && (
                <p className="text-xs text-muted-foreground mt-2">
                    Please complete all required fields before deploying
                </p>
            )}
        </div>
    )
}

export default ConfigurationChecklist
