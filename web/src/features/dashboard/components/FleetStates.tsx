import { Link } from 'react-router-dom'
import { Globe, Plus, Server } from 'lucide-react'
import { buttonClasses } from '@/shared/components/ui/Button'
import { Card } from '@/shared/components/ui/Card'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { ROUTES } from '@/shared/lib/routes'

const SKELETON_CARDS = 3

function SkeletonCard() {
    return (
        <Card className="flex flex-col gap-3.5 p-4">
            <div className="flex items-center gap-3">
                <Skeleton className="h-[38px] w-[38px] rounded-[9px]" />
                <div className="flex-1 space-y-2">
                    <Skeleton className="h-3.5 w-1/2" />
                    <Skeleton className="h-3 w-3/4" />
                </div>
                <Skeleton className="h-6 w-16 rounded-full" />
            </div>
            <Skeleton className="h-3 w-3/5" />
            <div className="grid grid-cols-3 gap-3">
                <Skeleton className="h-8" />
                <Skeleton className="h-8" />
                <Skeleton className="h-8" />
            </div>
            <div className="flex gap-2">
                <Skeleton className="h-[34px] w-20" />
                <Skeleton className="h-[34px] w-20" />
            </div>
        </Card>
    )
}

// What the screen looks like while apps are loading, shaped like the real thing so nothing jumps.
export function FleetLoading() {
    return (
        <div className="flex flex-col gap-5" role="status" aria-label="Loading your fleet">
            <div className="flex items-center justify-between">
                <div className="space-y-2">
                    <Skeleton className="h-7 w-28" />
                    <Skeleton className="h-4 w-48" />
                </div>
                <Skeleton className="h-[34px] w-28" />
            </div>
            <div className="flex gap-2">
                <Skeleton className="h-[34px] w-20 rounded-full" />
                <Skeleton className="h-[34px] w-24 rounded-full" />
                <Skeleton className="h-[34px] w-24 rounded-full" />
            </div>
            <Skeleton className="h-5 w-44" />
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                {Array.from({ length: SKELETON_CARDS }, (_, index) => (
                    <SkeletonCard key={index} />
                ))}
            </div>
        </div>
    )
}

// The very first visit: no apps at all. It says what to do next.
export function FleetEmpty() {
    return (
        <div className="flex flex-col gap-5">
            <EmptyState
                icon={<Plus className="h-5 w-5" />}
                title="Deploy your first app"
                description="Create an app from a compose file and Selfhostly checks it, starts it, and keeps it running. You can add a public address later."
                action={
                    <Link to={ROUTES.newApp} className={buttonClasses()}>
                        <Plus className="h-4 w-4" />
                        New app
                    </Link>
                }
                className="py-14"
            />
            <div className="grid gap-4 md:grid-cols-2">
                <Card className="flex items-center gap-4 p-4">
                    <div aria-hidden="true" className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                        <Globe className="h-[18px] w-[18px]" />
                    </div>
                    <div className="flex-1">
                        <p className="font-semibold">Reach apps from anywhere</p>
                        <p className="text-[13px] text-muted-foreground">Connect Cloudflare to give apps a public address.</p>
                    </div>
                    <Link to={ROUTES.settings} className={buttonClasses({ variant: 'outline' })}>Connect</Link>
                </Card>
                <Card className="flex items-center gap-4 p-4">
                    <div aria-hidden="true" className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                        <Server className="h-[18px] w-[18px]" />
                    </div>
                    <div className="flex-1">
                        <p className="font-semibold">Add another machine</p>
                        <p className="text-[13px] text-muted-foreground">Spread apps across a Pi, a NAS or a VPS.</p>
                    </div>
                    <Link to={ROUTES.registerNode} className={buttonClasses({ variant: 'outline' })}>Add node</Link>
                </Card>
            </div>
        </div>
    )
}
