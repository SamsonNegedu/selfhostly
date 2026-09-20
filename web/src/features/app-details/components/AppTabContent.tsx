import { AlertTriangle } from 'lucide-react'
import type { AppTab } from '@/shared/lib/routes'
import type { App } from '@/shared/types/api'
import { AppLogsPanel } from './AppLogsPanel'
import AppOverview from './AppOverview'
import CloudflareTab from './CloudflareTab'
import ComposeEditor from './ComposeEditor'
import EnvironmentTab from './EnvironmentTab'
import HistoryTab from './HistoryTab'
import { ScheduleEditor } from './ScheduleEditor'

// Shown in place of a tab that needs to know which node the app is on, when that is not known.
const missingNode = (what: string) => (
    <div className="flex min-h-[200px] items-center justify-center text-muted-foreground">
        <AlertTriangle className="mr-2 h-5 w-5" />
        Unable to load {what}: node_id is missing
    </div>
)

// The body of whichever tab is open.
function AppTabContent({ app, tab }: { app: App; tab: AppTab }) {
    return (
        <div>
            {tab === 'overview' && <AppOverview app={app} />}
            {tab === 'config' &&
                (app.node_id ? (
                    <ComposeEditor appId={app.id} nodeId={app.node_id} initialComposeContent={app.compose_content} />
                ) : (
                    missingNode('the compose editor')
                ))}
            {tab === 'environment' && (app.node_id ? <EnvironmentTab app={app} /> : missingNode('the environment'))}
            {tab === 'logs' &&
                (app.node_id ? <AppLogsPanel appId={app.id} nodeId={app.node_id} /> : missingNode('logs'))}
            {tab === 'access' &&
                (app.node_id ? (
                    <CloudflareTab appId={app.id} nodeId={app.node_id} composeContent={app.compose_content} />
                ) : (
                    missingNode('tunnel info')
                ))}
            {tab === 'history' && (app.node_id ? <HistoryTab app={app} /> : missingNode('the history'))}
            {tab === 'schedule' &&
                (app.node_id ? <ScheduleEditor appId={app.id} nodeId={app.node_id} /> : missingNode('the schedule'))}
        </div>
    )
}

export default AppTabContent
