import { useState } from 'react'
import { CheckCircle2, Loader2, Plus, Save, Trash2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { describeError } from '@/shared/lib/errors'
import { useCreateTunnelDNSRecord, useUpdateTunnelIngress } from '@/shared/services/api'
import type { IngressRule } from '@/shared/types/api'

interface IngressConfigurationProps {
    appId: string
    nodeId: string
    existingIngress?: IngressRule[]
    onSave?: (rules: IngressRule[], hostname?: string) => void
    // Drops the card and its title, for when it sits inside a sheet that already has one.
    flat?: boolean
}

const EMPTY_RULE: IngressRule = { service: '', hostname: null, path: null }
const HOSTNAME_PATTERN = /^(?=.{1,253}$)([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$/i
const SERVICE_PATTERN = /^https?:\/\/\S+$/i

function normalize(rules: IngressRule[] | undefined): IngressRule[] {
    const cleaned = (rules ?? []).map((rule) => ({ ...rule, hostname: rule.hostname || null, path: rule.path || null }))
    return cleaned.length > 0 ? cleaned : [EMPTY_RULE]
}

// Which hostname goes to which service for one app's tunnel. Saving updates the tunnel and then creates a DNS
// record for every hostname, so each address starts working without a visit to Cloudflare.
export function IngressConfiguration({
    appId,
    nodeId,
    existingIngress,
    onSave,
    flat = false,
}: IngressConfigurationProps) {
    const [rules, setRules] = useState<IngressRule[]>(() => normalize(existingIngress))
    const [saving, setSaving] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [saved, setSaved] = useState(false)
    const [touched, setTouched] = useState(false)

    const updateIngress = useUpdateTunnelIngress()
    const createDns = useCreateTunnelDNSRecord()

    const change = (index: number, patch: Partial<IngressRule>) => {
        setRules(rules.map((rule, i) => (i === index ? { ...rule, ...patch } : rule)))
        setSaved(false)
    }

    const hostnameError = (rule: IngressRule) =>
        rule.hostname && !HOSTNAME_PATTERN.test(rule.hostname)
            ? 'Enter a full hostname, such as app.example.com.'
            : undefined
    const serviceError = (rule: IngressRule) => {
        if (rule.service.trim() === '') return touched ? 'Say where the app answers.' : undefined
        return SERVICE_PATTERN.test(rule.service.trim())
            ? undefined
            : 'Start with http:// or https://, for example http://web:80.'
    }
    const invalid = rules.some((rule) => hostnameError(rule) || serviceError(rule))
    const filled = rules.filter((rule) => rule.service.trim() !== '')

    const save = async () => {
        setTouched(true)
        setError(null)
        if (filled.length === 0) return setError('Add at least one route with a service address.')
        if (invalid) return
        const cleaned = filled.map((rule) => ({
            ...rule,
            service: rule.service.trim(),
            hostname: rule.hostname?.trim() || null,
            path: rule.path?.trim() || null,
        }))
        const hostnames = [...new Set(cleaned.map((rule) => rule.hostname).filter((host): host is string => !!host))]

        setSaving(true)
        try {
            await updateIngress.mutateAsync({ appId, nodeId, ingressRules: cleaned, hostname: hostnames[0] })
        } catch (failure) {
            setError(describeError(failure))
            setSaving(false)
            return
        }
        try {
            for (const hostname of hostnames) await createDns.mutateAsync({ appId, nodeId, hostname })
        } catch (failure) {
            setError(`The routes were saved, but a DNS record could not be created. ${describeError(failure)}`)
            setSaving(false)
            return
        }
        setSaving(false)
        setSaved(true)
        onSave?.(cleaned, hostnames[0])
    }

    const form = (
        <div className="flex flex-col gap-5">
            <p className="text-sm text-muted-foreground">
                Which hostname goes to which service. A DNS record is created for each hostname, and anything that
                matches no route gets a 404.
            </p>

            {rules.map((rule, index) => (
                <fieldset
                    key={index}
                    className="flex flex-col gap-3 border-t border-border pt-4 first:border-t-0 first:pt-0"
                >
                    <legend className="sr-only">Route {index + 1}</legend>
                    <Field
                        label={rules.length > 1 ? `Hostname (route ${index + 1})` : 'Hostname'}
                        hint="Optional. Leave it empty to use the tunnel's own address."
                        error={hostnameError(rule)}
                    >
                        <Input
                            value={rule.hostname ?? ''}
                            onChange={(event) => change(index, { hostname: event.target.value.trim() || null })}
                            placeholder="app.example.com"
                            inputMode="url"
                            autoComplete="off"
                        />
                    </Field>
                    <Field
                        label="Service address"
                        hint="Where the app answers, for example http://web:80."
                        error={serviceError(rule)}
                    >
                        <Input
                            value={rule.service}
                            onChange={(event) => change(index, { service: event.target.value.trim() })}
                            placeholder="http://web:80"
                            inputMode="url"
                            className="font-mono"
                            autoComplete="off"
                        />
                    </Field>
                    <Field label="Path (optional)" hint="Send only this path to the service, for example /api/*.">
                        <Input
                            value={rule.path ?? ''}
                            onChange={(event) => change(index, { path: event.target.value.trim() || null })}
                            placeholder="/api/*"
                            autoComplete="off"
                        />
                    </Field>
                    {rules.length > 1 && (
                        <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            className="self-start text-destructive"
                            onClick={() => setRules(rules.filter((_, i) => i !== index))}
                        >
                            <Trash2 className="h-4 w-4" />
                            Remove route
                        </Button>
                    )}
                </fieldset>
            ))}

            <Button
                type="button"
                variant="outline"
                className="self-start"
                onClick={() => setRules([...rules, EMPTY_RULE])}
            >
                <Plus className="h-4 w-4" />
                Add a route
            </Button>

            {error && (
                <p role="alert" className="rounded-lg bg-status-err-bg px-3.5 py-3 text-sm text-status-err-fg">
                    {error}
                </p>
            )}
            {saved && (
                <p role="status" className="flex items-center gap-2 text-sm text-status-ok-fg">
                    <CheckCircle2 aria-hidden="true" className="h-4 w-4" />
                    Routes saved. DNS records are ready.
                </p>
            )}

            <div className="flex flex-wrap items-center gap-3 border-t border-border pt-4">
                <Button onClick={() => void save()} disabled={saving}>
                    {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                    Save routes
                </Button>
                <span className="text-[13px] text-muted-foreground">
                    Point your domain's nameservers at Cloudflare before adding a hostname.
                </span>
            </div>
        </div>
    )

    if (flat) return form

    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-base">Routes</CardTitle>
            </CardHeader>
            <CardContent>{form}</CardContent>
        </Card>
    )
}
