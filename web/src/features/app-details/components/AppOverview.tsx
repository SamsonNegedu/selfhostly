import { Activity } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import type { App } from '@/shared/types/api'
import AccessCard from './AccessCard'
import ActivityTimeline from './ActivityTimeline'
import ContainersCard from './ContainersCard'
import FailedStartGuide from './FailedStartGuide'
import MetricTiles from './MetricTiles'
import ResourcesCard from './ResourcesCard'
import ScheduleCard from './ScheduleCard'

interface AppOverviewProps {
    app: App
}

// The first tab of an app: how it is doing, what it is made of, how to reach it, and what happened lately.
function AppOverview({ app }: AppOverviewProps) {
    return (
        <div className="flex flex-col gap-5">
            {app.status === 'error' && <FailedStartGuide app={app} />}

            <MetricTiles app={app} />

            <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
                <div className="flex flex-col gap-5">
                    <ContainersCard app={app} />
                    <AccessCard app={app} />
                    <ScheduleCard app={app} />
                    <ResourcesCard app={app} />
                </div>

                <Card className="flex max-h-[720px] flex-col">
                    <CardHeader className="flex-shrink-0">
                        <CardTitle className="flex items-center gap-2 text-base">
                            <Activity className="h-4 w-4 text-muted-foreground" />
                            Recent activity
                        </CardTitle>
                    </CardHeader>
                    <CardContent className="min-h-0 flex-1 overflow-y-auto">
                        <ActivityTimeline app={app} />
                    </CardContent>
                </Card>
            </div>
        </div>
    )
}

export default AppOverview
