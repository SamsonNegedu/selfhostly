import { useState } from 'react'
import { CheckCircle2, Loader2, MinusCircle } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { StatusPill } from '@/shared/components/ui/StatusPill'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { useProviderFeatures, useProviders, useSettings, useUpdateSettings } from '@/shared/services/api'

type ProviderConfig = Record<string, { api_token?: string; account_id?: string } | undefined>

const FEATURES: { key: 'ingress' | 'dns' | 'status_sync' | 'container'; label: string }[] = [
    { key: 'ingress', label: 'Ingress rules' },
    { key: 'dns', label: 'DNS records' },
    { key: 'status_sync', label: 'Status sync' },
    { key: 'container', label: 'Container sidecar' },
]

// Which service gives your apps a public address, and the credentials it needs. Saved credentials are never
// sent back to the browser, only a masked hint of the token.
function TunnelProviderSection() {
    const { toast } = useToast()
    const { data: settings } = useSettings()
    const { data: providerData } = useProviders()
    const update = useUpdateSettings()
    const [provider, setProvider] = useState('')
    const [config, setConfig] = useState<ProviderConfig>({})
    const [masked, setMasked] = useState<Record<string, string>>({})
    const { data: features } = useProviderFeatures(provider)

    // Load the form from the saved settings, and again whenever they change.
    const [loadedSettings, setLoadedSettings] = useState(settings)
    if (settings && settings !== loadedSettings) {
        setLoadedSettings(settings)
        setProvider(settings.active_tunnel_provider || 'cloudflare')
        try {
            const parsed: ProviderConfig = settings.tunnel_provider_config
                ? JSON.parse(settings.tunnel_provider_config)
                : {}
            const hints: Record<string, string> = {}
            for (const [name, value] of Object.entries(parsed)) {
                if (value?.api_token?.includes('****')) {
                    hints[name] = value.api_token
                    parsed[name] = { ...value, api_token: '' }
                }
            }
            setMasked(hints)
            setConfig(parsed)
        } catch {
            setMasked({})
            setConfig({})
        }
    }

    if (!settings || !providerData) {
        return (
            <div role="status" aria-label="Loading tunnel provider" className="flex flex-col gap-3">
                <Skeleton className="h-16 rounded-xl" />
                <Skeleton className="h-52 rounded-xl" />
            </div>
        )
    }

    const providers = providerData.providers
    const current = providers.find((item) => item.name === provider)
    const values = config[provider] ?? {}
    const set = (field: 'api_token' | 'account_id', value: string) =>
        setConfig({ ...config, [provider]: { ...values, [field]: value } })
    const isCloudflare = provider === 'cloudflare'
    const ready =
        !isCloudflare ||
        (((values.api_token ?? '') !== '' || masked[provider] !== undefined) && (values.account_id ?? '') !== '')

    const save = () =>
        update.mutate(
            {
                active_tunnel_provider: provider,
                tunnel_provider_config: JSON.stringify(config),
                auto_start_apps: settings.auto_start_apps,
            },
            {
                onSuccess: () =>
                    toast.success(
                        'Tunnel provider saved',
                        `${current?.display_name ?? provider} is now the active provider`,
                    ),
                onError: (failure) => toast.error('Could not save', describeError(failure)),
            },
        )

    return (
        <div className="flex flex-col gap-5">
            <Card className="flex flex-wrap items-center gap-3 p-4" aria-label="Provider status" role="region">
                <div className="min-w-0 flex-1">
                    <p className="font-semibold">{current?.display_name ?? 'No provider'}</p>
                    <p className="text-[13px] text-muted-foreground">
                        {current?.is_configured
                            ? 'Credentials are saved. Custom domains are available.'
                            : 'Add credentials below to publish apps on your own domain.'}
                    </p>
                </div>
                <StatusPill kind={current?.is_configured ? 'ok' : 'idle'}>
                    {current?.is_configured ? 'Connected' : 'Not connected'}
                </StatusPill>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle className="text-base">Provider</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-col gap-4">
                    <Field label="Active provider">
                        <select
                            value={provider}
                            onChange={(event) => setProvider(event.target.value)}
                            className="h-[38px] min-h-[44px] w-full rounded-lg border border-input bg-card px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:min-h-0 sm:max-w-sm"
                        >
                            {providers.length === 0 && <option value="">No providers available</option>}
                            {providers.map((item) => (
                                <option key={item.name} value={item.name}>
                                    {item.display_name}
                                    {item.is_configured ? ' (connected)' : ''}
                                </option>
                            ))}
                        </select>
                    </Field>

                    {isCloudflare && (
                        <div className="grid gap-4 sm:grid-cols-2">
                            <Field
                                label="API token"
                                hint={
                                    masked[provider]
                                        ? 'A token is saved. Enter a new one only to replace it.'
                                        : "Create a token with the 'Cloudflare Tunnel' permission."
                                }
                            >
                                <Input
                                    type="password"
                                    value={values.api_token ?? ''}
                                    onChange={(event) => set('api_token', event.target.value)}
                                    placeholder={masked[provider] ?? 'Paste your API token'}
                                    autoComplete="off"
                                    className="font-mono"
                                />
                            </Field>
                            <Field label="Account ID" hint="Shown in your Cloudflare dashboard.">
                                <Input
                                    value={values.account_id ?? ''}
                                    onChange={(event) => set('account_id', event.target.value)}
                                    className="font-mono"
                                    autoComplete="off"
                                />
                            </Field>
                        </div>
                    )}

                    <div>
                        <Button onClick={save} disabled={update.isPending || !ready}>
                            {update.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                            Save provider
                        </Button>
                    </div>
                    {update.error && (
                        <p role="alert" className="text-sm text-status-err-fg">
                            Could not save. {describeError(update.error)}
                        </p>
                    )}
                </CardContent>
            </Card>

            {features && (
                <Card>
                    <CardHeader>
                        <CardTitle className="text-base">What {features.display_name} can do here</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <ul className="grid gap-2 text-sm sm:grid-cols-2">
                            {FEATURES.map((feature) => {
                                const on = features.features[feature.key]
                                return (
                                    <li key={feature.key} className="flex items-center gap-2">
                                        {on ? (
                                            <CheckCircle2 aria-hidden="true" className="h-4 w-4 text-status-ok-fg" />
                                        ) : (
                                            <MinusCircle aria-hidden="true" className="h-4 w-4 text-muted-foreground" />
                                        )}
                                        <span className={on ? '' : 'text-muted-foreground'}>
                                            {feature.label}
                                            <span className="sr-only">{on ? ': supported' : ': not supported'}</span>
                                        </span>
                                    </li>
                                )
                            })}
                        </ul>
                    </CardContent>
                </Card>
            )}
        </div>
    )
}

export default TunnelProviderSection
