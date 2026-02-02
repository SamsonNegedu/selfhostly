import { useState, useEffect } from 'react'
import { useSettings, useUpdateSettings, useProviders, useProviderFeatures } from '@/shared/services/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Checkbox } from '@/shared/components/ui'
import { Badge } from '@/shared/components/ui/badge'
import AppBreadcrumb from '@/shared/components/layout/Breadcrumb'
import { CheckCircle2, AlertCircle, Network, Shield } from 'lucide-react'

function Settings() {
    const { data: settings, isLoading: settingsLoading } = useSettings()
    const { data: providersData, isLoading: providersLoading } = useProviders()
    const updateSettings = useUpdateSettings()

    const [selectedProvider, setSelectedProvider] = useState<string>('')
    const [providerConfig, setProviderConfig] = useState<Record<string, any>>({})
    const [maskedTokens, setMaskedTokens] = useState<Record<string, string>>({}) // Store masked tokens for placeholders
    const [autoStartApps, setAutoStartApps] = useState(false)

    const { data: providerFeatures } = useProviderFeatures(selectedProvider)

    // Initialize form state from settings
    useEffect(() => {
        if (settings) {
            setAutoStartApps(settings.auto_start_apps)

            // Set active provider
            const activeProvider = settings.active_tunnel_provider || 'cloudflare'
            setSelectedProvider(activeProvider)

            // Parse provider config - only use tunnel_provider_config, no legacy fields
            try {
                if (settings.tunnel_provider_config) {
                    const parsed = JSON.parse(settings.tunnel_provider_config)
                    // Store masked tokens separately for placeholders, clear from config
                    const cleaned = { ...parsed }
                    const masked: Record<string, string> = {}
                    Object.keys(cleaned).forEach(provider => {
                        if (cleaned[provider]?.api_token?.includes('****')) {
                            // Store masked token for placeholder
                            masked[provider] = cleaned[provider].api_token
                            // Clear masked token from config (user needs to enter new token)
                            cleaned[provider] = {
                                ...cleaned[provider],
                                api_token: ''
                            }
                        }
                    })
                    setMaskedTokens(masked)
                    setProviderConfig(cleaned || {})
                } else {
                    // No provider config yet - start with empty config
                    setMaskedTokens({})
                    setProviderConfig({})
                }
            } catch (error) {
                console.error('Failed to parse provider config:', error)
                setMaskedTokens({})
                setProviderConfig({})
            }
        }
    }, [settings])

    if (settingsLoading || providersLoading) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <div className="text-muted-foreground">Loading settings...</div>
            </div>
        )
    }

    const providers = providersData?.providers || []
    const configuredProviders = providers.filter(p => p.is_configured)
    const currentProviderConfig = providerConfig[selectedProvider] || {}

    const handleProviderChange = (provider: string) => {
        setSelectedProvider(provider)
        // Initialize empty config for new provider if not exists
        if (!providerConfig[provider]) {
            setProviderConfig({
                ...providerConfig,
                [provider]: {}
            })
        }
    }

    const handleConfigChange = (field: string, value: string) => {
        setProviderConfig({
            ...providerConfig,
            [selectedProvider]: {
                ...currentProviderConfig,
                [field]: value
            }
        })
    }

    const handleSaveProvider = () => {
        // Clean up masked tokens before saving - don't send tokens that contain "****"
        const cleanedConfig = { ...providerConfig }
        if (cleanedConfig[selectedProvider]?.api_token?.includes('****')) {
            // Remove masked token - backend will keep existing token
            cleanedConfig[selectedProvider] = {
                ...cleanedConfig[selectedProvider],
                api_token: ''
            }
        }

        const configToSave = {
            active_tunnel_provider: selectedProvider,
            tunnel_provider_config: JSON.stringify(cleanedConfig),
            auto_start_apps: autoStartApps
        }

        updateSettings.mutate(configToSave)
    }

    const renderProviderConfigFields = () => {
        switch (selectedProvider) {
            case 'cloudflare':
                return (
                    <div className="space-y-4">
                        <div>
                            <label htmlFor="cf_api_token" className="block text-sm font-medium mb-2">
                                API Token *
                            </label>
                            <input
                                id="cf_api_token"
                                type="password"
                                value={currentProviderConfig.api_token || ''}
                                onChange={(e) => handleConfigChange('api_token', e.target.value)}
                                className="w-full px-3 py-2 border border-input bg-background text-foreground rounded-md focus:outline-none focus:ring-2 focus:ring-ring placeholder:text-muted-foreground"
                                placeholder={maskedTokens[selectedProvider] || "Enter Cloudflare API token"}
                            />
                            <p className="text-xs text-muted-foreground mt-1">
                                Create an API token with 'Cloudflare Tunnel' permissions
                            </p>
                        </div>
                        <div>
                            <label htmlFor="cf_account_id" className="block text-sm font-medium mb-2">
                                Account ID *
                            </label>
                            <input
                                id="cf_account_id"
                                type="text"
                                value={currentProviderConfig.account_id || ''}
                                onChange={(e) => handleConfigChange('account_id', e.target.value)}
                                className="w-full px-3 py-2 border border-input bg-background text-foreground rounded-md focus:outline-none focus:ring-2 focus:ring-ring placeholder:text-muted-foreground"
                                placeholder="Enter Cloudflare account ID"
                            />
                            <p className="text-xs text-muted-foreground mt-1">
                                Find this in your Cloudflare dashboard
                            </p>
                        </div>
                    </div>
                )

            default:
                return (
                    <div className="text-sm text-muted-foreground">
                        Select a provider to configure
                    </div>
                )
        }
    }

    return (
        <div>
            {/* Breadcrumb Navigation - Desktop only */}
            <AppBreadcrumb
                items={[
                    { label: 'Home', path: '/apps' },
                    { label: 'Settings', isCurrentPage: true }
                ]}
                className="mb-4 sm:mb-6"
            />

            <div className="mb-4 sm:mb-6">
                <h1 className="text-2xl sm:text-3xl font-bold">Settings</h1>
                <p className="text-muted-foreground mt-1 sm:mt-2 text-sm sm:text-base">
                    Configure tunnel providers and application settings
                </p>
            </div>

            <div className="max-w-3xl space-y-6">
                {/* Tunnel Provider Configuration */}
                <Card>
                    <CardHeader className="pb-2 sm:pb-2">
                        <div className="flex items-center gap-2">
                            <Network className="h-5 w-5" />
                            <CardTitle>Tunnel Provider</CardTitle>
                        </div>
                        <p className="text-sm text-muted-foreground mt-2">
                            Choose and configure your tunnel provider for exposing applications to the internet
                        </p>
                    </CardHeader>
                    <CardContent className="space-y-4 pt-0">
                        {/* Provider Selection */}
                        <div>
                            <label htmlFor="provider" className="block text-sm font-medium mb-1">
                                Active Provider
                            </label>
                            <select
                                id="provider"
                                value={selectedProvider}
                                onChange={(e) => handleProviderChange(e.target.value)}
                                className="w-full px-3 py-2 border border-input bg-background text-foreground rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
                            >
                                {providers.length === 0 ? (
                                    <option value="">No providers available</option>
                                ) : (
                                    providers.map(provider => (
                                        <option key={provider.name} value={provider.name}>
                                            {provider.display_name}
                                            {provider.is_configured ? ' (Configured)' : ''}
                                        </option>
                                    ))
                                )}
                            </select>
                            {/* Provider Status */}
                            {providers.length > 0 && (
                                <div className="mt-2 flex flex-wrap gap-2">
                                    {configuredProviders.length > 0 ? (
                                        <Badge variant="default" className="bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20">
                                            <CheckCircle2 className="h-3 w-3 mr-1" />
                                            {configuredProviders.length} provider{configuredProviders.length !== 1 ? 's' : ''} configured
                                        </Badge>
                                    ) : (
                                        <Badge variant="secondary" className="bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20">
                                            <AlertCircle className="h-3 w-3 mr-1" />
                                            No providers configured
                                        </Badge>
                                    )}
                                </div>
                            )}
                        </div>

                        {/* Provider Features */}
                        {providerFeatures && (
                            <div className="border border-border rounded-lg p-4 bg-muted/30">
                                <h4 className="text-sm font-semibold mb-3 flex items-center gap-2">
                                    <Shield className="h-4 w-4" />
                                    Provider Features
                                </h4>
                                <div className="grid grid-cols-2 gap-2 text-sm">
                                    <div className="flex items-center gap-2">
                                        {providerFeatures.features.ingress ?
                                            <CheckCircle2 className="h-4 w-4 text-green-500" /> :
                                            <AlertCircle className="h-4 w-4 text-muted-foreground" />
                                        }
                                        <span className={providerFeatures.features.ingress ? '' : 'text-muted-foreground'}>
                                            Ingress Rules
                                        </span>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        {providerFeatures.features.dns ?
                                            <CheckCircle2 className="h-4 w-4 text-green-500" /> :
                                            <AlertCircle className="h-4 w-4 text-muted-foreground" />
                                        }
                                        <span className={providerFeatures.features.dns ? '' : 'text-muted-foreground'}>
                                            DNS Management
                                        </span>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        {providerFeatures.features.status_sync ?
                                            <CheckCircle2 className="h-4 w-4 text-green-500" /> :
                                            <AlertCircle className="h-4 w-4 text-muted-foreground" />
                                        }
                                        <span className={providerFeatures.features.status_sync ? '' : 'text-muted-foreground'}>
                                            Status Sync
                                        </span>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        {providerFeatures.features.container ?
                                            <CheckCircle2 className="h-4 w-4 text-green-500" /> :
                                            <AlertCircle className="h-4 w-4 text-muted-foreground" />
                                        }
                                        <span className={providerFeatures.features.container ? '' : 'text-muted-foreground'}>
                                            Container Sidecar
                                        </span>
                                    </div>
                                </div>
                            </div>
                        )}

                        {/* Provider Configuration Fields */}
                        <div>
                            {renderProviderConfigFields()}
                        </div>

                        <Button
                            onClick={handleSaveProvider}
                            disabled={updateSettings.isPending}
                            className="w-full sm:w-auto"
                        >
                            {updateSettings.isPending ? 'Saving...' : 'Save Provider Settings'}
                        </Button>
                    </CardContent>
                </Card>

                {/* General Settings */}
                <Card>
                    <CardHeader className="pb-2 sm:pb-2">
                        <CardTitle>General Settings</CardTitle>
                        <p className="text-sm text-muted-foreground mt-2">
                            Configure general application behavior
                        </p>
                    </CardHeader>
                    <CardContent className="pt-0">
                        <div className="space-y-4">
                            <div className="flex gap-3 items-start">
                                <Checkbox
                                    id="auto_start_apps"
                                    checked={autoStartApps}
                                    onCheckedChange={(checked) => {
                                        const newValue = checked as boolean
                                        setAutoStartApps(newValue)

                                        // Auto-save the setting
                                        updateSettings.mutate({
                                            auto_start_apps: newValue
                                        })
                                    }}
                                    className="mt-0.5 shrink-0"
                                />
                                <div className="flex-1 pt-0.5">
                                    <label
                                        htmlFor="auto_start_apps"
                                        className="text-sm font-medium cursor-pointer select-none leading-tight block"
                                    >
                                        Auto-start applications on server boot
                                    </label>
                                    <p className="text-sm text-muted-foreground mt-1.5 leading-relaxed">
                                        When enabled, all apps will automatically start when the server boots up
                                    </p>
                                </div>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </div>
        </div>
    )
}

export default Settings
