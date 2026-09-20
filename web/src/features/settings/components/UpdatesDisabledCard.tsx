import { ShieldAlert } from 'lucide-react'
import { Card, CardContent } from '@/shared/components/ui/Card'

// What the server says when it will not do updates from the browser, and what to do about it.
const DISABLED_MESSAGES: Record<string, string> = {
    'not enabled':
        'Updates from this page are turned off. Set UI_UPDATES_ENABLED=true and restart Selfhostly to turn them on. You can always update with selfhostlyctl upgrade.',
    'no signing key':
        'This build has no release signing key, so it cannot check that an update is genuine. Set UPDATE_PUBLIC_KEY to the key your releases are signed with.',
    'no docker socket': 'The updater needs the Docker socket, and Selfhostly cannot reach it here.',
    'secondary node': 'This is a secondary node. Update it from the primary, or with selfhostlyctl upgrade.',
    'auth disabled': 'Sign-in is turned off. Updating Selfhostly needs a signed-in user, so turn sign-in on first.',
}

// Why updates from this page are not available, and what to do about it.
function UpdatesDisabledCard({ reason }: { reason: string }) {
    return (
        <Card>
            <CardContent className="flex items-start gap-3 p-4">
                <ShieldAlert aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-muted-foreground" />
                <div>
                    <h2 className="font-semibold">Updates from this page are not available</h2>
                    <p className="text-compact text-muted-foreground">{DISABLED_MESSAGES[reason] ?? reason}</p>
                </div>
            </CardContent>
        </Card>
    )
}

export default UpdatesDisabledCard
