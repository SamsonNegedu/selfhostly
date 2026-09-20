import { useId, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/components/ui/Select'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { useProviderFeatures, useProviders, useSettings, useUpdateSettings } from '@/shared/services/api'
import ProviderFeaturesCard from '../components/ProviderFeaturesCard'
import ProviderStatusCard from '../components/ProviderStatusCard'
import { isProviderReady, parseProviderConfig, type ProviderConfig } from '../lib/provider-config'

// Which service gives your apps a public address, and the credentials it needs. Saved credentials are never
// sent back to the browser, only a masked hint of the token.
function TunnelProviderSection() {
    const { toast } = useToast()
    const { data: settings } = useSettings()
    const { data: providerData } = useProviders()
    const update = useUpdateSettings()
    const providerFieldId = useId()
    const [provider, setProvider] = useState('')
    const [config, setConfig] = useState<ProviderConfig>({})
    const [masked, setMasked] = useState<Record<string, string>>({})
    const { data: features } = useProviderFeatures(provider)

    // Load the form from the saved settings, and again whenever they change.
    const [loadedSettings, setLoadedSettings] = useState(settings)
    if (settings && settings !== loadedSettings) {
        setLoadedSettings(settings)
        setProvider(settings.active_tunnel_provider || 'cloudflare')
        const { config, masked } = parseProviderConfig(settings.tunnel_provider_config)
        setMasked(masked)
        setConfig(config)
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
    const ready = isProviderReady(provider, values, masked)

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
            <ProviderStatusCard current={current} />

            <Card>
                <CardHeader>
                    <CardTitle className="text-base">Provider</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-col gap-4">
                    {/* Field would put the id on the Select root, so the label is tied to the trigger by hand. */}
                    <div className="flex flex-col gap-1.5">
                        <label htmlFor={providerFieldId} className="text-compact font-medium">
                            Active provider
                        </label>
                        <Select value={provider} onValueChange={setProvider} disabled={providers.length === 0}>
                            <SelectTrigger id={providerFieldId} className="sm:max-w-sm">
                                <SelectValue placeholder="No providers available" />
                            </SelectTrigger>
                            <SelectContent>
                                {providers.map((item) => (
                                    <SelectItem key={item.name} value={item.name}>
                                        {item.display_name}
                                        {item.is_configured ? ' (connected)' : ''}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>

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
                            {update.isPending && <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />}
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

            {features && <ProviderFeaturesCard features={features} />}
        </div>
    )
}

export default TunnelProviderSection
