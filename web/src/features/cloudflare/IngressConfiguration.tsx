import React, { useState } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Input } from '@/shared/components/ui/input'
import { Badge } from '@/shared/components/ui/badge'
import { Plus, Trash2, Save, AlertCircle, CheckCircle, Globe } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { useUpdateTunnelIngress, useCreateDNSRecord } from '@/shared/services/api'
import type { IngressRule } from '@/shared/types/api'

interface IngressConfigurationProps {
    appId: number
    existingIngress?: IngressRule[]
    existingHostname?: string
    tunnelID?: string
    onSave?: (rules: IngressRule[], hostname?: string) => void
}

export function IngressConfiguration({ appId, existingIngress = [], existingHostname = '', tunnelID, onSave }: IngressConfigurationProps) {
    const queryClient = useQueryClient()
    // Handle null values in existing ingress rules
    const sanitizedIngress = existingIngress.map(rule => ({
        ...rule,
        hostname: rule.hostname || null,
        path: rule.path || null
    }))
    const [rules, setRules] = useState<IngressRule[]>(sanitizedIngress.length > 0 ? sanitizedIngress : [{ service: '', hostname: null, path: null }])
    const [hostname, setHostname] = useState(existingHostname || '')
    const [targetDomain, setTargetDomain] = useState('') // Default empty for cfargotunnel.com
    const [isSaving, setIsSaving] = useState(false)
    const [saveError, setSaveError] = useState<string | null>(null)
    const [saveSuccess, setSaveSuccess] = useState(false)

    const updateTunnelIngressMutation = useUpdateTunnelIngress()
    const createDNSRecordMutation = useCreateDNSRecord()

    const addRule = () => {
        setRules([...rules, { service: '', hostname: null, path: null }])
    }

    const removeRule = (index: number) => {
        if (rules.length > 1) {
            const newRules = [...rules]
            newRules.splice(index, 1)
            setRules(newRules)
        }
    }

    const updateRule = (index: number, field: keyof IngressRule, value: string | Record<string, any> | undefined | null) => {
        const newRules = [...rules]
        newRules[index] = { ...newRules[index], [field]: value || undefined }
        setRules(newRules)
    }

    const getDefaultServiceUrl = () => {
        // Try to infer service from existing data
        const app = queryClient.getQueryData(['app', appId]) as any
        if (app && app.compose_content) {
            // Try to extract port from docker-compose content
            const portMatch = app.compose_content.match(/ports:\s*\n\s*-\s*"(\d+):(\d+)"/)
            if (portMatch) {
                return `http://localhost:${portMatch[2]}`
            }
        }
        return 'http://localhost:8080'
    }

    const handleSave = () => {
        setIsSaving(true)
        setSaveError(null)
        setSaveSuccess(false)

        // Validate all rules have service defined
        const validRules = rules.filter(rule => rule.service.trim() !== '')
        if (validRules.length === 0) {
            setSaveError('At least one service must be defined')
            setIsSaving(false)
            return
        }

        // Update ingress configuration
        updateTunnelIngressMutation.mutate(
            {
                appId,
                ingressRules: validRules,
                hostname: hostname || undefined,
                targetDomain: targetDomain || undefined
            },
            {
                onSuccess: () => {
                    setSaveSuccess(true)
                    setSaveError(null)

                    // If hostname is provided, also create DNS record
                    if (hostname) {
                        createDNSRecordMutation.mutate(
                            {
                                appId,
                                hostname,
                                targetDomain: targetDomain || undefined
                            },
                            {
                                onSuccess: () => {
                                    onSave?.(validRules, hostname)
                                },
                                onError: (error: Error) => {
                                    setSaveError(`Ingress configured but DNS creation failed: ${error.message}`)
                                }
                            }
                        )
                    } else {
                        onSave?.(validRules, undefined)
                    }
                },
                onError: (error: Error) => {
                    setSaveError(error.message)
                    setSaveSuccess(false)
                },
                onSettled: () => {
                    setIsSaving(false)
                }
            }
        )
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle className="flex items-center gap-2">
                    <span>Ingress Configuration</span>
                    <Badge variant="outline" className="text-xs">
                        Public Access
                    </Badge>
                </CardTitle>
            </CardHeader>
            <CardContent>
                <div className="space-y-4">
                    {saveError && (
                        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3 flex items-center gap-2">
                            <AlertCircle className="h-4 w-4 text-red-500" />
                            <span className="text-sm text-red-700 dark:text-red-400">{saveError}</span>
                        </div>
                    )}

                    {saveSuccess && (
                        <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-3 flex items-center gap-2">
                            <CheckCircle className="h-4 w-4 text-green-500" />
                            <span className="text-sm text-green-700 dark:text-green-400">
                                {hostname ? 'Ingress and DNS configuration saved successfully' : 'Ingress configuration saved successfully'}
                            </span>
                        </div>
                    )}

                    {/* Hostname Configuration */}
                    <div className="border rounded-lg p-4 bg-blue-50 dark:bg-blue-900/20">
                        <div className="flex items-center gap-2 mb-3">
                            <Globe className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                            <h3 className="text-sm font-medium text-blue-800 dark:text-blue-200">Custom Domain Setup</h3>
                        </div>
                        <div className="space-y-3">
                            <div>
                                <label className="text-xs font-medium text-blue-700 dark:text-blue-300">Custom Domain (Optional)</label>
                                <Input
                                    placeholder="app.example.com"
                                    value={hostname}
                                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => setHostname(e.target.value)}
                                    className="mt-1"
                                />
                                <p className="text-xs text-blue-600 dark:text-blue-400 mt-1">
                                    Enter your custom domain to create a DNS record for public access.
                                    This will create a CNAME record pointing your subdomain to your Cloudflare tunnel.
                                    Make sure your domain is already pointed to Cloudflare nameservers.
                                </p>
                            </div>

                            <div>
                                <label className="text-xs font-medium text-blue-700 dark:text-blue-300">Target Domain (Optional)</label>
                                <Input
                                    placeholder="example.com"
                                    value={targetDomain}
                                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTargetDomain(e.target.value)}
                                    className="mt-1"
                                />
                                <p className="text-xs text-blue-600 dark:text-blue-400 mt-1">
                                    Leave empty to use the default Cloudflare tunnel domain (your_tunnel_id.cfargotunnel.com),
                                    or enter your own domain (e.g., example.com) if you have custom routing configured.
                                </p>
                            </div>

                            {hostname && (
                                <div className="pt-2">
                                    <p className="text-xs font-medium text-blue-700 dark:text-blue-300">DNS Record Preview:</p>
                                    <Badge variant="secondary" className="mt-1">
                                        CNAME {hostname || 'your-domain'} → {tunnelID}.cfargotunnel.com
                                        {targetDomain && (
                                            <span className="ml-1 text-xs">(Custom domain: {targetDomain})</span>
                                        )}
                                    </Badge>
                                    <p className="text-xs text-blue-600 dark:text-blue-400 mt-1">
                                        {targetDomain
                                            ? `This will create a CNAME record pointing ${hostname || 'your-domain'} to your tunnel.`
                                            : `This will create a CNAME record pointing ${hostname || 'your-domain'} directly to your Cloudflare tunnel (${tunnelID}.cfargotunnel.com).`
                                        }
                                    </p>
                                </div>
                            )}
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
                                        <label className="text-xs font-medium text-muted-foreground">Hostname</label>
                                        <Input
                                            placeholder="app.example.com"
                                            value={rule.hostname || ''}
                                            onChange={(e: React.ChangeEvent<HTMLInputElement>) => updateRule(index, 'hostname', e.target.value || undefined)}
                                        />
                                        <p className="text-xs text-muted-foreground mt-1">
                                            Leave empty for tunnel URL
                                        </p>
                                    </div>

                                    <div>
                                        <label className="text-xs font-medium text-muted-foreground">Service URL</label>
                                        <Input
                                            placeholder={getDefaultServiceUrl()}
                                            value={rule.service}
                                            onChange={(e: React.ChangeEvent<HTMLInputElement>) => updateRule(index, 'service', e.target.value)}
                                        />
                                        <p className="text-xs text-muted-foreground mt-1">
                                            e.g., http://localhost:8080
                                        </p>
                                    </div>

                                    <div>
                                        <label className="text-xs font-medium text-muted-foreground">Path</label>
                                        <Input
                                            placeholder="/api/*"
                                            value={rule.path || ''}
                                            onChange={(e: React.ChangeEvent<HTMLInputElement>) => updateRule(index, 'path', e.target.value || undefined)}
                                        />
                                        <p className="text-xs text-muted-foreground mt-1">
                                            Optional path routing
                                        </p>
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>

                    <div className="pt-4 border-t">
                        <Button
                            onClick={handleSave}
                            disabled={isSaving}
                            className="w-full"
                        >
                            {isSaving ? (
                                <>
                                    <div className="h-4 w-4 border-2 border-white border-t-transparent rounded-full animate-spin mr-2" />
                                    Saving Configuration...
                                </>
                            ) : (
                                <>
                                    <Save className="h-4 w-4 mr-2" />
                                    Save Ingress Configuration
                                </>
                            )}
                        </Button>
                    </div>

                    <div className="text-xs text-muted-foreground space-y-2">
                        <p><strong>Note:</strong> A catch-all rule (404 response) is automatically added to the end of your configuration.</p>
                        <p><strong>Tip:</strong> To access your app via a custom domain, add a DNS CNAME record pointing to your tunnel ID.</p>
                    </div>
                </div>
            </CardContent>
        </Card>
    )
}
