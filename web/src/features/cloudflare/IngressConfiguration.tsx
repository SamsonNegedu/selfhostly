import React, { useState } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/Card'
import { Button } from '@/shared/components/ui/Button'
import { Input } from '@/shared/components/ui/Input'
import { Badge } from '@/shared/components/ui/Badge'
import { Plus, Trash2, Save, AlertCircle, CheckCircle, Globe } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { useUpdateTunnelIngress, useCreateTunnelDNSRecord } from '@/shared/services/api'
import type { IngressRule } from '@/shared/types/api'

interface IngressConfigurationProps {
    appId: string;
    nodeId: string;
    existingIngress?: IngressRule[];
    existingHostname?: string;
    tunnelID?: string;
    onSave?: (rules: IngressRule[], hostname?: string) => void;
}

export function IngressConfiguration({ appId, nodeId, existingIngress = [], existingHostname: _existingHostname = '', tunnelID: _tunnelID, onSave }: IngressConfigurationProps) {
    const queryClient = useQueryClient()
    // Handle null/undefined values in existing ingress rules
    const safeIngress = existingIngress || []
    const sanitizedIngress = safeIngress.map(rule => ({
        ...rule,
        hostname: rule.hostname || null,
        path: rule.path || null
    }))
    const [rules, setRules] = useState<IngressRule[]>(sanitizedIngress.length > 0 ? sanitizedIngress : [{ service: '', hostname: null, path: null }])
    const [isSaving, setIsSaving] = useState(false)
    const [saveError, setSaveError] = useState<string | null>(null)
    const [saveSuccess, setSaveSuccess] = useState(false)

    const updateTunnelIngressMutation = useUpdateTunnelIngress()
    const createDNSRecordMutation = useCreateTunnelDNSRecord()

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

        // Extract all unique hostnames from rules for DNS record creation
        const hostnames = validRules
            .map(rule => rule.hostname)
            .filter((h): h is string => h !== null && h !== undefined && h.trim() !== '')
            .filter((value, index, self) => self.indexOf(value) === index) // unique values

        // Update ingress configuration
        updateTunnelIngressMutation.mutate(
            {
                appId,
                nodeId,
                ingressRules: validRules,
                hostname: hostnames[0] || undefined, // Send first hostname for backward compatibility
                targetDomain: undefined
            },
            {
                onSuccess: () => {
                    setSaveSuccess(true)
                    setSaveError(null)

                    // Immediately invalidate tunnel query for instant UI feedback (with nodeId)
                    queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', appId, nodeId] })
                    queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', appId] }) // Also invalidate without nodeId for backward compatibility

                    // Create DNS records for all hostnames
                    if (hostnames.length > 0) {
                        const dnsPromises = hostnames.map(hostname =>
                            createDNSRecordMutation.mutateAsync({
                                appId,
                                nodeId,
                                hostname
                            })
                        )

                        Promise.all(dnsPromises)
                            .then(() => {
                                // Invalidate all related queries to refresh UI (with nodeId)
                                queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', appId, nodeId] })
                                queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', appId] }) // Also invalidate without nodeId
                                queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnels'] })
                                queryClient.invalidateQueries({ queryKey: ['app', appId, nodeId] })
                                queryClient.invalidateQueries({ queryKey: ['app', appId] }) // Also invalidate without nodeId
                                queryClient.invalidateQueries({ queryKey: ['apps'] })
                                onSave?.(validRules, hostnames[0])
                            })
                            .catch((error: Error) => {
                                setSaveError(`Ingress configured but DNS creation failed: ${error.message}`)
                            })
                    } else {
                        // Invalidate all related queries to refresh UI (with nodeId)
                        queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', appId, nodeId] })
                        queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnel', appId] }) // Also invalidate without nodeId
                        queryClient.invalidateQueries({ queryKey: ['cloudflare', 'tunnels'] })
                        queryClient.invalidateQueries({ queryKey: ['app', appId, nodeId] })
                        queryClient.invalidateQueries({ queryKey: ['app', appId] }) // Also invalidate without nodeId
                        queryClient.invalidateQueries({ queryKey: ['apps'] })
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
                                Ingress configuration saved successfully. DNS records created automatically for custom domains.
                            </span>
                        </div>
                    )}

                    {/* Info Banner */}
                    <div className="border rounded-lg p-4 bg-blue-50 dark:bg-blue-900/10 border-blue-200 dark:border-blue-900/30">
                        <div className="flex items-start gap-3">
                            <Globe className="h-5 w-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
                            <div className="space-y-1">
                                <h3 className="text-sm font-medium text-blue-900 dark:text-blue-100">Custom Domain Setup</h3>
                                <p className="text-xs text-blue-700 dark:text-blue-300">
                                    Add a hostname (e.g., <code className="px-1 py-0.5 rounded bg-blue-100 dark:bg-blue-900/30">vertsh.localnest.de</code>) to any ingress rule below,
                                    and a DNS CNAME record will be created automatically pointing to your Cloudflare tunnel.
                                    Leave hostname empty to use the default tunnel URL.
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
                                            Hostname
                                            {rule.hostname && <Globe className="h-3 w-3 text-green-500" />}
                                        </label>
                                        <Input
                                            placeholder="vertsh.localnest.de"
                                            value={rule.hostname || ''}
                                            onChange={(e: React.ChangeEvent<HTMLInputElement>) => updateRule(index, 'hostname', e.target.value || undefined)}
                                        />
                                        <p className="text-xs text-muted-foreground mt-1">
                                            {rule.hostname
                                                ? <span className="text-green-600 dark:text-green-400 flex items-center gap-1">
                                                    <CheckCircle className="h-3 w-3" /> DNS record will be created
                                                </span>
                                                : 'Optional - uses tunnel URL if empty'}
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
                                        <label className="text-xs font-medium text-muted-foreground">Path (Optional)</label>
                                        <Input
                                            placeholder="/api/*"
                                            value={rule.path || ''}
                                            onChange={(e: React.ChangeEvent<HTMLInputElement>) => updateRule(index, 'path', e.target.value || undefined)}
                                        />
                                        <p className="text-xs text-muted-foreground mt-1">
                                            For path-based routing
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

                    <div className="text-xs text-muted-foreground space-y-2 bg-muted/50 rounded-lg p-3">
                        <p className="flex items-start gap-2">
                            <CheckCircle className="h-3.5 w-3.5 mt-0.5 flex-shrink-0 text-green-500" />
                            <span>A catch-all rule (404 response) is automatically added to the end of your configuration.</span>
                        </p>
                        <p className="flex items-start gap-2">
                            <CheckCircle className="h-3.5 w-3.5 mt-0.5 flex-shrink-0 text-green-500" />
                            <span>DNS CNAME records are automatically created for any hostname you enter in the ingress rules.</span>
                        </p>
                        <p className="flex items-start gap-2">
                            <AlertCircle className="h-3.5 w-3.5 mt-0.5 flex-shrink-0 text-blue-500" />
                            <span>Make sure your domain's nameservers are pointing to Cloudflare before adding a custom hostname.</span>
                        </p>
                    </div>
                </div>
            </CardContent>
        </Card>
    )
}
