import { RefreshCw } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent } from '@/shared/components/ui/Card'
import { formatUpdateTime } from '@/shared/lib/update'
import type { UpdateStatus } from '@/shared/types/api'

interface UpdateVersionCardProps {
    status: UpdateStatus
    // A run is under way, so checking would only confuse things.
    running: boolean
    checking: boolean
    onCheck: () => void
}

// The version that is running, when it was last checked, and the button to check now.
function UpdateVersionCard({ status, running, checking, onCheck }: UpdateVersionCardProps) {
    const checked = formatUpdateTime(status.checked_at)

    return (
        <Card>
            <CardContent className="flex flex-wrap items-center gap-4 p-4">
                <div className="min-w-0 flex-1">
                    <h2 className="font-semibold">Selfhostly version</h2>
                    <p className="text-compact text-muted-foreground">
                        Running{' '}
                        <span data-testid="update-current-version" className="font-mono">
                            {status.current_version}
                        </span>
                        {checked && `. Last checked ${checked}.`}
                    </p>
                </div>
                <Button
                    variant="outline"
                    data-testid="update-check-button"
                    disabled={!status.enabled || running || checking}
                    onClick={onCheck}
                >
                    <RefreshCw className={checking ? 'h-4 w-4 animate-spin' : 'h-4 w-4'} />
                    {checking ? 'Checking...' : 'Check for updates'}
                </Button>
            </CardContent>
        </Card>
    )
}

export default UpdateVersionCard
