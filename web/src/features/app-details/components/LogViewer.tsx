import React from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Download, RefreshCw } from 'lucide-react'

function LogViewer({ appId }: { appId: number }) {
    const [logs, setLogs] = React.useState('')
    const [isLoading, setIsLoading] = React.useState(false)

    const fetchLogs = async () => {
        setIsLoading(true)
        try {
            const response = await fetch(`/api/apps/${appId}/logs`)
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
    }, [appId])

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
                <div className="flex items-center justify-between">
                    <CardTitle className="text-xl">Logs</CardTitle>
                    <div className="flex gap-2">
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
                <div className="bg-black text-green-400 p-4 rounded-md font-mono text-sm overflow-auto max-h-[400px]">
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
