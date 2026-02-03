import { useMemo } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/Card'
import { Badge } from '@/shared/components/ui/Badge'
import {
    Activity,
    Clock,
    Layers,
    HardDrive,
    Network,
    Calendar,
    Server,
    CheckCircle2,
    XCircle,
    Play,
    Pause,
    AlertTriangle
} from 'lucide-react'
import ActivityTimeline from './ActivityTimeline'
import { useAppServices } from '@/shared/services/api'
import type { App } from '@/shared/types/api'

interface AppOverviewProps {
    app: App
}

interface ComposeInfo {
    networks: string[]
    volumes: string[]
}

function AppOverview({ app }: AppOverviewProps) {
    // Get services from backend endpoint for consistency with LogViewer
    const { data: services = [] } = useAppServices(app.id, app.node_id || '')

    // Parse compose content to extract networks and volumes (services come from backend)
    const composeInfo: ComposeInfo = useMemo(() => {
        const info: ComposeInfo = {
            networks: [],
            volumes: []
        }

        try {
            const lines = app.compose_content.split('\n')
            let inNetworksSection = false
            let inVolumesSection = false
            let currentIndent = 0

            for (let i = 0; i < lines.length; i++) {
                const line = lines[i]
                const trimmedLine = line.trim()

                // Skip empty lines and comments
                if (!trimmedLine || trimmedLine.startsWith('#')) continue

                // Calculate indentation
                const indent = line.search(/\S/)

                // Check for top-level sections
                if (indent === 0) {
                    if (trimmedLine.startsWith('networks:')) {
                        inNetworksSection = true
                        inVolumesSection = false
                        currentIndent = 0
                        continue
                    } else if (trimmedLine.startsWith('volumes:')) {
                        inNetworksSection = false
                        inVolumesSection = true
                        currentIndent = 0
                        continue
                    } else if (trimmedLine.startsWith('version:') || trimmedLine.startsWith('services:')) {
                        continue
                    }
                }

                // Extract networks (first level under 'networks:')
                if (inNetworksSection && indent > 0) {
                    if (currentIndent === 0) {
                        currentIndent = indent
                    }
                    if (indent === currentIndent && trimmedLine.includes(':')) {
                        const networkName = trimmedLine.split(':')[0].trim()
                        if (networkName && !info.networks.includes(networkName)) {
                            info.networks.push(networkName)
                        }
                    }
                }

                // Extract volumes (first level under 'volumes:')
                if (inVolumesSection && indent > 0) {
                    if (currentIndent === 0) {
                        currentIndent = indent
                    }
                    if (indent === currentIndent && trimmedLine.includes(':')) {
                        const volumeName = trimmedLine.split(':')[0].trim()
                        if (volumeName && !info.volumes.includes(volumeName)) {
                            info.volumes.push(volumeName)
                        }
                    }
                }
            }

            // Also count bind mounts in services
            const bindMounts = (app.compose_content.match(/- ['"]*[\/~]/g) || []).length
            if (bindMounts > 0 && info.volumes.length === 0) {
                info.volumes = [`${bindMounts} bind mount${bindMounts > 1 ? 's' : ''}`]
            }
        } catch (error) {
            console.error('Failed to parse compose content:', error)
        }

        return info
    }, [app.compose_content])

    const getStatusIcon = (status: App['status']) => {
        switch (status) {
            case 'running':
                return <Play className="h-4 w-4" />
            case 'stopped':
                return <Pause className="h-4 w-4" />
            case 'updating':
                return <Activity className="h-4 w-4 animate-spin" />
            case 'error':
                return <AlertTriangle className="h-4 w-4" />
            default:
                return <Clock className="h-4 w-4" />
        }
    }

    const getStatusColor = (status: App['status']) => {
        switch (status) {
            case 'running':
                return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
            case 'stopped':
                return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'
            case 'updating':
                return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
            case 'error':
                return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
            default:
                return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'
        }
    }

    const formatDate = (dateString: string) => {
        const date = new Date(dateString)
        const now = new Date()
        const diffMs = now.getTime() - date.getTime()
        const diffDays = Math.floor(diffMs / 86400000)

        if (diffDays === 0) return 'Today'
        if (diffDays === 1) return 'Yesterday'
        if (diffDays < 7) return `${diffDays} days ago`
        return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
    }

    return (
        <div className="space-y-6">
            {/* Quick Stats Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                {/* Status Card */}
                <Card>
                    <CardContent className="pt-6">
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-sm text-muted-foreground mb-1">Status</p>
                                <div className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium ${getStatusColor(app.status)}`}>
                                    {getStatusIcon(app.status)}
                                    <span className="capitalize">{app.status}</span>
                                </div>
                            </div>
                            <div className={`p-3 rounded-full ${app.status === 'running' ? 'bg-green-100 dark:bg-green-900/30' : 'bg-muted'}`}>
                                {app.status === 'running' ? (
                                    <CheckCircle2 className="h-6 w-6 text-green-600 dark:text-green-400" />
                                ) : (
                                    <XCircle className="h-6 w-6 text-muted-foreground" />
                                )}
                            </div>
                        </div>
                    </CardContent>
                </Card>

                {/* Services Count */}
                <Card>
                    <CardContent className="pt-6">
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-sm text-muted-foreground mb-1">Services</p>
                                <p className="text-2xl font-bold">{services.length}</p>
                            </div>
                            <div className="p-3 rounded-full bg-blue-100 dark:bg-blue-900/30">
                                <Server className="h-6 w-6 text-blue-600 dark:text-blue-400" />
                            </div>
                        </div>
                    </CardContent>
                </Card>

                {/* Networks Count */}
                <Card>
                    <CardContent className="pt-6">
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-sm text-muted-foreground mb-1">Networks</p>
                                <p className="text-2xl font-bold">{composeInfo.networks.length || 1}</p>
                            </div>
                            <div className="p-3 rounded-full bg-purple-100 dark:bg-purple-900/30">
                                <Network className="h-6 w-6 text-purple-600 dark:text-purple-400" />
                            </div>
                        </div>
                    </CardContent>
                </Card>

                {/* Uptime/Age */}
                <Card>
                    <CardContent className="pt-6">
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-sm text-muted-foreground mb-1">Created</p>
                                <p className="text-sm font-medium">{formatDate(app.created_at)}</p>
                            </div>
                            <div className="p-3 rounded-full bg-amber-100 dark:bg-amber-900/30">
                                <Calendar className="h-6 w-6 text-amber-600 dark:text-amber-400" />
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* Main Info Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* Left Column */}
                <div className="space-y-6">
                    {/* Services List */}
                    <Card>
                        <CardHeader>
                            <CardTitle className="text-lg flex items-center gap-2">
                                <Layers className="h-5 w-5 text-primary" />
                                Services
                            </CardTitle>
                        </CardHeader>
                        <CardContent>
                            {services.length > 0 ? (
                                <div className="space-y-2">
                                    {services.map((service) => (
                                        <div
                                            key={service}
                                            className="flex items-center justify-between p-3 rounded-lg bg-muted/50 hover:bg-muted transition-colors"
                                        >
                                            <div className="flex items-center gap-2">
                                                <div className="w-2 h-2 rounded-full bg-primary" />
                                                <span className="font-mono text-sm font-medium">{service}</span>
                                            </div>
                                            <Badge variant="outline" className="text-xs">
                                                active
                                            </Badge>
                                        </div>
                                    ))}
                                </div>
                            ) : (
                                <p className="text-sm text-muted-foreground text-center py-4">
                                    No services found
                                </p>
                            )}
                        </CardContent>
                    </Card>

                    {/* Resources */}
                    <Card>
                        <CardHeader>
                            <CardTitle className="text-lg flex items-center gap-2">
                                <HardDrive className="h-5 w-5 text-primary" />
                                Resources
                            </CardTitle>
                        </CardHeader>
                        <CardContent>
                            <div className="space-y-3">
                                <div className="flex items-center justify-between">
                                    <span className="text-sm text-muted-foreground">Networks</span>
                                    <div className="flex gap-1">
                                        {composeInfo.networks.length > 0 ? (
                                            composeInfo.networks.map((network, index) => (
                                                <Badge key={index} variant="secondary" className="text-xs">
                                                    {network}
                                                </Badge>
                                            ))
                                        ) : (
                                            <Badge variant="secondary" className="text-xs">
                                                default
                                            </Badge>
                                        )}
                                    </div>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-sm text-muted-foreground">Volumes</span>
                                    <span className="text-sm font-medium">
                                        {composeInfo.volumes.length > 0 ? composeInfo.volumes.join(', ') : 'None'}
                                    </span>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-sm text-muted-foreground">Last Updated</span>
                                    <span className="text-sm font-medium">{formatDate(app.updated_at)}</span>
                                </div>
                            </div>
                        </CardContent>
                    </Card>
                </div>

                {/* Right Column - Activity Timeline */}
                <Card>
                    <CardHeader>
                        <CardTitle className="text-lg flex items-center gap-2">
                            <Activity className="h-5 w-5 text-primary" />
                            Recent Activity
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <ActivityTimeline app={app} />
                    </CardContent>
                </Card>
            </div>
        </div>
    )
}

export default AppOverview
