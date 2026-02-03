import React, { useState } from 'react'
import { Button } from '@/shared/components/ui/Button'
import { Input } from '@/shared/components/ui/Input'
import { Plus, Trash2, Globe, CheckCircle, AlertCircle } from 'lucide-react'
import type { IngressRule } from '@/shared/types/api'

interface IngressRulesEditorProps {
    value: IngressRule[];
    onChange: (rules: IngressRule[]) => void;
}

export default function IngressRulesEditor({ value, onChange }: IngressRulesEditorProps) {
    const [rules, setRules] = useState<IngressRule[]>(
        value.length > 0 ? value : [{ service: 'http://localhost:8080', hostname: null, path: null }]
    )

    const updateRules = (newRules: IngressRule[]) => {
        setRules(newRules)
        onChange(newRules)
    }

    const addRule = () => {
        updateRules([...rules, { service: '', hostname: null, path: null }])
    }

    const removeRule = (index: number) => {
        if (rules.length > 1) {
            const newRules = [...rules]
            newRules.splice(index, 1)
            updateRules(newRules)
        }
    }

    const updateRule = (index: number, field: keyof IngressRule, value: string | Record<string, any> | undefined | null) => {
        const newRules = [...rules]
        newRules[index] = { ...newRules[index], [field]: value || null }
        updateRules(newRules)
    }

    return (
        <div className="space-y-4">
            {/* Info Banner */}
            <div className="border rounded-lg p-4 bg-blue-50 dark:bg-blue-900/10 border-blue-200 dark:border-blue-900/30">
                <div className="flex items-start gap-3">
                    <Globe className="h-5 w-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
                    <div className="space-y-1">
                        <h3 className="text-sm font-medium text-blue-900 dark:text-blue-100">Custom Domain Setup</h3>
                        <p className="text-xs text-blue-700 dark:text-blue-300">
                            Configure your ingress rules now to automatically apply them after the app starts.
                            Add a hostname (e.g., <code className="px-1 py-0.5 rounded bg-blue-100 dark:bg-blue-900/30">app.yourdomain.com</code>)
                            and DNS CNAME records will be created automatically. You can also configure this later.
                        </p>
                    </div>
                </div>
            </div>

            <div className="space-y-3">
                <div className="flex justify-between items-center">
                    <h3 className="text-sm font-medium">Ingress Rules</h3>
                    <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={addRule}
                        className="flex items-center gap-1"
                    >
                        <Plus className="h-3 w-3" />
                        Add Rule
                    </Button>
                </div>

                {rules.map((rule, index) => (
                    <div key={index} className="border rounded-lg p-3 space-y-3">
                        <div className="flex justify-between items-center">
                            <span className="text-xs font-medium text-muted-foreground">Rule {index + 1}</span>
                            {rules.length > 1 && (
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => removeRule(index)}
                                    className="text-destructive hover:text-destructive"
                                >
                                    <Trash2 className="h-3 w-3" />
                                </Button>
                            )}
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                            <div>
                                <label className="text-xs font-medium text-muted-foreground flex items-center gap-1">
                                    Hostname (Optional)
                                    {rule.hostname && <Globe className="h-3 w-3 text-green-500" />}
                                </label>
                                <Input
                                    placeholder="app.yourdomain.com"
                                    value={rule.hostname || ''}
                                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                                        updateRule(index, 'hostname', e.target.value || null)
                                    }
                                />
                                <p className="text-xs text-muted-foreground mt-1">
                                    {rule.hostname
                                        ? <span className="text-green-600 dark:text-green-400 flex items-center gap-1">
                                            <CheckCircle className="h-3 w-3" /> DNS record will be created
                                        </span>
                                        : 'Leave empty for default tunnel URL'}
                                </p>
                            </div>

                            <div>
                                <label className="text-xs font-medium text-muted-foreground">
                                    Service URL <span className="text-red-500">*</span>
                                </label>
                                <Input
                                    placeholder="http://localhost:8080"
                                    value={rule.service}
                                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                                        updateRule(index, 'service', e.target.value)
                                    }
                                />
                                <p className="text-xs text-muted-foreground mt-1">
                                    e.g., http://localhost:8080
                                </p>
                            </div>

                            <div>
                                <label className="text-xs font-medium text-muted-foreground">Path (Optional)</label>
                                <Input
                                    placeholder="/api/*"
                                    value={rule.path || ''}
                                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                                        updateRule(index, 'path', e.target.value || null)
                                    }
                                />
                                <p className="text-xs text-muted-foreground mt-1">
                                    For path-based routing
                                </p>
                            </div>
                        </div>
                    </div>
                ))}
            </div>

            <div className="text-xs text-muted-foreground space-y-2 bg-muted/50 rounded-lg p-3">
                <p className="flex items-start gap-2">
                    <CheckCircle className="h-3.5 w-3.5 mt-0.5 flex-shrink-0 text-green-500" />
                    <span>A catch-all rule (404 response) will be automatically added to the end of your configuration.</span>
                </p>
                <p className="flex items-start gap-2">
                    <CheckCircle className="h-3.5 w-3.5 mt-0.5 flex-shrink-0 text-green-500" />
                    <span>DNS CNAME records will be automatically created for any hostname you enter.</span>
                </p>
                <p className="flex items-start gap-2">
                    <AlertCircle className="h-3.5 w-3.5 mt-0.5 flex-shrink-0 text-blue-500" />
                    <span>Make sure your domain's nameservers are pointing to Cloudflare before adding a custom hostname.</span>
                </p>
            </div>
        </div>
    )
}
