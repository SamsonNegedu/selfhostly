import React from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { SimpleDropdown, SimpleDropdownItem } from '@/shared/components/ui/simple-dropdown'
import { Download, RefreshCw, ChevronDown } from 'lucide-react'
import { useAppServices } from '@/shared/services/api'

function LogViewer({ appId, nodeId }: { appId: string; nodeId: string }) {
    const [logs, setLogs] = React.useState('')
    const [isLoading, setIsLoading] = React.useState(false)
    const [selectedService, setSelectedService] = React.useState<string>('')
    const { data: services = [] } = useAppServices(appId, nodeId)

    const fetchLogs = async () => {
        setIsLoading(true)
        try {
            const params = new URLSearchParams({ node_id: nodeId })
            if (selectedService) {
                params.append('service', selectedService)
            }
            const response = await fetch(`/api/apps/${appId}/logs?${params.toString()}`)
            const text = await response.text()
            setLogs(text)
        } catch (error) {
            console.error('Failed to fetch logs:', error)
        } finally {
            setIsLoading(false)
        }
    }

    React.useEffect(() => {
        fetchLogs()
        const interval = setInterval(fetchLogs, 5000)
        return () => clearInterval(interval)
    }, [appId, nodeId, selectedService])

    const downloadLogs = () => {
        const blob = new Blob([logs], { type: 'text/plain' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `app-${appId}-logs.txt`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(url)
    }

    return (
        <Card>
            <CardHeader>
                <div className="flex items-center justify-between gap-4">
                    <CardTitle className="text-xl">Logs</CardTitle>
                    <div className="flex items-center gap-2">
                        {services.length > 0 && (
                            <SimpleDropdown
                                trigger={
                                    <Button variant="outline" size="sm" className="gap-2">
                                        <span>{selectedService || 'All Services'}</span>
                                        <ChevronDown className="h-4 w-4" />
                                    </Button>
                                }
                            >
                                <div className="py-1 min-w-[160px]">
                                    <SimpleDropdownItem onClick={() => setSelectedService('')}>
                                        All Services
                                    </SimpleDropdownItem>
                                    {services.map(service => (
                                        <SimpleDropdownItem 
                                            key={service} 
                                            onClick={() => setSelectedService(service)}
                                        >
                                            {service}
                                        </SimpleDropdownItem>
                                    ))}
                                </div>
                            </SimpleDropdown>
                        )}
                        <Button variant="outline" size="icon" onClick={fetchLogs} disabled={isLoading}>
                            <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
                        </Button>
                        <Button variant="outline" size="icon" onClick={downloadLogs}>
                            <Download className="h-4 w-4" />
                        </Button>
                    </div>
                </div>
            </CardHeader>
            <CardContent>
                <div className="bg-slate-950 dark:bg-black text-green-400 dark:text-green-300 p-4 rounded-md font-mono text-sm overflow-auto max-h-[600px] border border-slate-800 dark:border-slate-900">
                    {logs ? (
                        <pre className="whitespace-pre-wrap">{logs}</pre>
                    ) : (
                        <p className="text-muted-foreground">No logs available</p>
                    )}
                </div>
            </CardContent>
        </Card>
    )
}

export default LogViewer
