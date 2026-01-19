import React from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Loader2 } from 'lucide-react'
import { useUpdateAppContainers } from '@/shared/services/api'

function UpdateProgress({ appId }: { appId: number }) {
    const [isUpdating, setIsUpdating] = React.useState(false)
    const updateApp = useUpdateAppContainers()

    const handleUpdate = () => {
        setIsUpdating(true)
        updateApp.mutate(appId, {
            onSettled: () => setIsUpdating(false)
        })
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-xl">Update Progress</CardTitle>
            </CardHeader>
            <CardContent>
                {isUpdating ? (
                    <div className="space-y-4">
                        <div className="flex items-center space-x-2">
                            <Loader2 className="h-4 w-4 animate-spin" />
                            <span>Updating containers...</span>
                        </div>
                        <div className="h-2 bg-muted rounded-full overflow-hidden">
                            <div className="h-full bg-primary animate-[width_2s_ease-in-out]" style={{ width: '60%' }} />
                        </div>
                    </div>
                ) : (
                    <div className="space-y-4">
                        <p className="text-sm text-muted-foreground">
                            Update app containers with zero downtime
                        </p>
                        <Button onClick={handleUpdate} disabled={isUpdating || updateApp.isPending}>
                            {isUpdating || updateApp.isPending ? (
                                <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                            ) : (
                                "Start Update"
                            )}
                        </Button>
                    </div>
                )}
            </CardContent>
        </Card>
    )
}

export default UpdateProgress
