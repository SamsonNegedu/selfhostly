import { useState } from 'react'
import { Activity, History } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import type { App, ComposeVersion } from '@/shared/types/api'
import ActivityTimeline from './ActivityTimeline'
import ComposeVersionHistory from './ComposeVersionHistory'
import VersionCompareDialog from './VersionCompareDialog'

// Every saved version of the compose file, with compare and restore, next to what has happened to the app.
function HistoryTab({ app }: { app: App }) {
    const [viewing, setViewing] = useState<ComposeVersion | null>(null)

    return (
        <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-base">
                        <History className="h-4 w-4 text-muted-foreground" />
                        Versions
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="mb-3 text-sm text-muted-foreground">
                        Each save keeps the file it replaced. Restore makes an old version the current one, then update
                        the app to run it.
                    </p>
                    <ComposeVersionHistory
                        appId={app.id}
                        nodeId={app.node_id}
                        onVersionSelect={setViewing}
                        showLabels
                    />
                </CardContent>
            </Card>

            <Card className="flex max-h-[720px] flex-col">
                <CardHeader className="flex-shrink-0">
                    <CardTitle className="flex items-center gap-2 text-base">
                        <Activity className="h-4 w-4 text-muted-foreground" />
                        Activity
                    </CardTitle>
                </CardHeader>
                <CardContent className="min-h-0 flex-1 overflow-y-auto">
                    <ActivityTimeline app={app} />
                </CardContent>
            </Card>

            <VersionCompareDialog
                version={viewing}
                against={app.compose_content}
                againstLabel="the current file"
                onClose={() => setViewing(null)}
            />
        </div>
    )
}

export default HistoryTab
