import React, { useState, useEffect } from 'react'
import { useSettings, useUpdateSettings } from '@/shared/services/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Checkbox } from '@/shared/components/ui'
import AppBreadcrumb from '@/shared/components/layout/Breadcrumb'

function Settings() {
    const { data: settings, isLoading } = useSettings()
    const updateSettings = useUpdateSettings()
    const [formData, setFormData] = useState({
        id: settings?.id,
        cloudflare_api_token: '',
        cloudflare_account_id: '',
        auto_start_apps: false,
    })

    useEffect(() => {
        if (settings) {
            setFormData({
                id: settings.id,
                cloudflare_api_token: '',
                cloudflare_account_id: settings.cloudflare_account_id,
                auto_start_apps: settings.auto_start_apps,
            })
        }
    }, [settings])

    if (isLoading) {
        return <div>Loading settings...</div>
    }

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        updateSettings.mutate(formData)
    }

    return (
        <div>
            {/* Breadcrumb Navigation */}
            <div className="mb-6">
                <AppBreadcrumb
                    items={[
                        { label: 'Home', path: '/apps' },
                        { label: 'Settings', isCurrentPage: true }
                    ]}
                />
            </div>

            <div className="mb-6">
                <h1 className="text-3xl font-bold">Settings</h1>
                <p className="text-muted-foreground mt-2">
                    Configure application settings
                </p>
            </div>

            <div className="max-w-2xl space-y-6">
                <Card>
                    <CardHeader>
                        <CardTitle>Cloudflare Configuration</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <form onSubmit={handleSubmit} className="space-y-4">
                            <div>
                                <label htmlFor="cloudflare_api_token" className="block text-sm font-medium mb-2">
                                    API Token
                                </label>
                                <input
                                    id="cloudflare_api_token"
                                    type="password"
                                    value={formData.cloudflare_api_token}
                                    onChange={(e) => setFormData({ ...formData, cloudflare_api_token: e.target.value })}
                                    className="w-full px-3 py-2 border border-input bg-background text-foreground rounded-md focus:outline-none focus:ring-2 focus:ring-ring placeholder:text-muted-foreground"
                                    placeholder={settings?.cloudflare_api_token ? "Leave blank to keep current(" + settings?.cloudflare_api_token + ")" : "Enter new API token"}
                                />
                            </div>
                            <div>
                                <label htmlFor="cloudflare_account_id" className="block text-sm font-medium mb-2">
                                    Account ID
                                </label>
                                <input
                                    id="cloudflare_account_id"
                                    type="text"
                                    value={formData.cloudflare_account_id}
                                    onChange={(e) => setFormData({ ...formData, cloudflare_account_id: e.target.value })}
                                    className="w-full px-3 py-2 border border-input bg-background text-foreground rounded-md focus:outline-none focus:ring-2 focus:ring-ring placeholder:text-muted-foreground"
                                    required
                                />
                            </div>
                            <Button type="submit" disabled={updateSettings.isPending}>
                                {updateSettings.isPending ? 'Saving...' : 'Save Settings'}
                            </Button>
                        </form>
                    </CardContent>
                </Card>

                <Card>
                    <CardHeader>
                        <CardTitle>General Settings</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-4">
                            <div className="flex items-center space-x-2">
                                <Checkbox
                                    id="auto_start_apps"
                                    checked={formData.auto_start_apps}
                                    onCheckedChange={(checked) => {
                                        const newValue = checked as boolean;
                                        setFormData({ ...formData, auto_start_apps: newValue });

                                        // Auto-save the setting when checkbox is toggled
                                        updateSettings.mutate({
                                            ...formData,
                                            auto_start_apps: newValue
                                        });
                                    }}
                                />
                                <label
                                    htmlFor="auto_start_apps"
                                    className="text-sm font-medium cursor-pointer select-none"
                                >
                                    Auto-start apps when server starts
                                </label>
                            </div>
                            <p className="text-sm text-muted-foreground">
                                When enabled, all apps will automatically start when the server boots up.
                            </p>
                        </div>
                    </CardContent>
                </Card>
            </div>
        </div>
    )
}

export default Settings
