import * as React from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { cn } from '@/shared/lib/utils'

interface OverviewCardProps {
    icon: React.ReactNode
    title: string
    // Classes for the content area, such as the gap between its children.
    contentClassName?: string
    children: React.ReactNode
}

// A titled card on the app's Overview, with the icon and heading every card there shares.
function OverviewCard({ icon, title, contentClassName, children }: OverviewCardProps) {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                    {icon}
                    {title}
                </CardTitle>
            </CardHeader>
            <CardContent className={cn(contentClassName)}>{children}</CardContent>
        </Card>
    )
}

export default OverviewCard
