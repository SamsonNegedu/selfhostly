import { useMemo, useState } from 'react'
import { ChevronDown, ChevronRight, Copy, Check, AlertCircle } from 'lucide-react'
import { parse } from 'yaml'
import { Button } from '@/shared/components/ui/Button'
import { useToast } from '@/shared/components/ui/Toast'

interface EnvVarSidebarProps {
    composeContent: string
}

interface ServiceEnvVars {
    serviceName: string
    variables: { key: string; value: string; isReference: boolean }[]
    count: number
}

function EnvVarSidebar({ composeContent }: EnvVarSidebarProps) {
    const { toast } = useToast()
    const [copiedKey, setCopiedKey] = useState<string | null>(null)
    const [expandedServices, setExpandedServices] = useState<Set<string>>(new Set())

    // Parse compose YAML and extract environment variables
    const parsedData = useMemo(() => {
        try {
            const parsed = parse(composeContent)
            const services: ServiceEnvVars[] = []
            let totalVarCount = 0

            if (parsed && typeof parsed === 'object' && 'services' in parsed) {
                const servicesObj = parsed.services as Record<string, any>

                for (const [serviceName, serviceConfig] of Object.entries(servicesObj)) {
                    const envVars: { key: string; value: string; isReference: boolean }[] = []

                    // Handle environment in map format
                    if (serviceConfig.environment && typeof serviceConfig.environment === 'object') {
                        for (const [key, value] of Object.entries(serviceConfig.environment)) {
                            if (typeof value === 'string') {
                                const isReference = /\$\{[A-Z_][A-Z0-9_]*\}/.test(value)
                                envVars.push({
                                    key,
                                    value,
                                    isReference
                                })
                                totalVarCount++
                            }
                        }
                    }
                    // Handle environment in list format: ["KEY=value", "KEY2=value2"]
                    else if (serviceConfig.environment && Array.isArray(serviceConfig.environment)) {
                        for (const item of serviceConfig.environment) {
                            if (typeof item === 'string' && item.includes('=')) {
                                const [key, ...valueParts] = item.split('=')
                                const value = valueParts.join('=')
                                const isReference = /\$\{[A-Z_][A-Z0-9_]*\}/.test(value)
                                envVars.push({
                                    key,
                                    value,
                                    isReference
                                })
                                totalVarCount++
                            }
                        }
                    }

                    if (envVars.length > 0) {
                        services.push({
                            serviceName,
                            variables: envVars,
                            count: envVars.length
                        })
                    }
                }
            }

            return { services, totalVarCount, error: null }
        } catch (error) {
            return {
                services: [],
                totalVarCount: 0,
                error: error instanceof Error ? error.message : 'Failed to parse YAML'
            }
        }
    }, [composeContent])

    const toggleService = (serviceName: string) => {
        const newExpanded = new Set(expandedServices)
        if (newExpanded.has(serviceName)) {
            newExpanded.delete(serviceName)
        } else {
            newExpanded.add(serviceName)
        }
        setExpandedServices(newExpanded)
    }

    const handleCopyValue = async (value: string, key: string) => {
        try {
            await navigator.clipboard.writeText(value)
            setCopiedKey(key)
            toast.success('Copied', 'Value copied to clipboard')
            setTimeout(() => setCopiedKey(null), 2000)
        } catch (error) {
            toast.error('Failed to copy', 'Could not copy value to clipboard')
        }
    }

    // Expand all by default if few services
    useMemo(() => {
        if (parsedData.services.length <= 3) {
            const allServiceNames = new Set(parsedData.services.map(s => s.serviceName))
            if (allServiceNames.size !== expandedServices.size) {
                setExpandedServices(allServiceNames)
            }
        }
    }, [parsedData.services])

    if (parsedData.error) {
        return (
            <div className="text-xs text-muted-foreground flex items-start gap-2 p-4">
                <AlertCircle className="h-4 w-4 text-amber-500 flex-shrink-0 mt-0.5" />
                <span>Unable to parse environment variables: {parsedData.error}</span>
            </div>
        )
    }

    if (parsedData.totalVarCount === 0) {
        return (
            <div className="text-xs text-muted-foreground py-8 text-center">
                No environment variables found in your docker-compose.yml.
            </div>
        )
    }

    return (
        <div className="space-y-4">
            <div className="text-xs text-muted-foreground px-1">
                {parsedData.totalVarCount} variable{parsedData.totalVarCount !== 1 ? 's' : ''} across {parsedData.services.length} service{parsedData.services.length !== 1 ? 's' : ''}
            </div>
            <div className="space-y-2">
                {parsedData.services.map((service) => {
                    const isExpanded = expandedServices.has(service.serviceName)
                    return (
                        <div
                            key={service.serviceName}
                            className="border border-border rounded-lg overflow-hidden"
                        >
                            {/* Service header */}
                            <div className="bg-muted/30">
                                <button
                                    onClick={() => toggleService(service.serviceName)}
                                    className="w-full flex items-center gap-2 px-3 py-2.5 hover:bg-muted/50 transition-colors text-left"
                                >
                                    {isExpanded ? (
                                        <ChevronDown className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                                    ) : (
                                        <ChevronRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                                    )}
                                    <div className="flex-1 flex items-center gap-2 min-w-0">
                                        <span className="text-sm font-semibold truncate">{service.serviceName}</span>
                                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted text-muted-foreground flex-shrink-0">
                                            {service.count} {service.count === 1 ? 'var' : 'vars'}
                                        </span>
                                    </div>
                                </button>
                            </div>

                            {/* Variables list */}
                            {isExpanded && (
                                <div className="p-2 space-y-1 border-t border-border">
                                    {service.variables.map((variable) => {
                                        const isCopied = copiedKey === variable.key
                                        const displayValue = variable.value.length > 50
                                            ? variable.value.substring(0, 50) + '...'
                                            : variable.value

                                        return (
                                            <div
                                                key={variable.key}
                                                className="group flex items-center gap-2 text-xs hover:bg-muted/50 rounded px-2 py-1"
                                            >
                                                <span
                                                    className={`font-medium min-w-[120px] ${variable.isReference
                                                        ? 'text-blue-600 dark:text-blue-400'
                                                        : 'text-foreground'
                                                        }`}
                                                >
                                                    {variable.key}:
                                                </span>
                                                <span
                                                    className={`flex-1 font-mono truncate ${variable.isReference
                                                        ? 'text-blue-600 dark:text-blue-400'
                                                        : 'text-muted-foreground'
                                                        }`}
                                                    title={variable.value}
                                                >
                                                    {displayValue}
                                                </span>
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
                                                    onClick={() => handleCopyValue(variable.value, variable.key)}
                                                    title="Copy value"
                                                >
                                                    {isCopied ? (
                                                        <Check className="h-3 w-3 text-green-600" />
                                                    ) : (
                                                        <Copy className="h-3 w-3" />
                                                    )}
                                                </Button>
                                            </div>
                                        )
                                    })}
                                </div>
                            )}
                        </div>
                    )
                })}

                {/* Legend */}
                <div className="pt-2 border-t border-border text-xs text-muted-foreground">
                    <div className="flex items-center gap-4">
                        <div className="flex items-center gap-1">
                            <div className="w-3 h-3 rounded-full bg-blue-600 dark:bg-blue-400" />
                            <span>Variable reference ($&#123;VAR&#125;)</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default EnvVarSidebar
