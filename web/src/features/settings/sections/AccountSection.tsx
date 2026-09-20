import { useState } from 'react'
import { LogOut, ShieldOff } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent } from '@/shared/components/ui/Card'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { useAuth } from '@/shared/components/auth/AuthProvider'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { logout, useRevokeSessions } from '@/shared/services/api'

// Who is signed in, and the two ways to end a session: this device only, or everyone everywhere.
function AccountSection() {
    const { user, authEnabled } = useAuth()
    const { toast } = useToast()
    const revoke = useRevokeSessions()
    const [confirming, setConfirming] = useState(false)

    const signOutEverywhere = () => {
        setConfirming(false)
        revoke.mutate(undefined, {
            onSuccess: () => {
                toast.success('Signed out everywhere', 'Everyone has to sign in again')
                logout()
            },
            onError: (failure) => toast.error('Could not sign everyone out', describeError(failure)),
        })
    }

    if (!authEnabled) {
        return (
            <Card>
                <CardContent className="flex flex-col gap-1 p-4">
                    <p className="font-semibold">Sign-in is turned off</p>
                    <p className="text-[13px] text-muted-foreground">
                        This server does not ask anyone to sign in, so there are no sessions to end. To require GitHub
                        sign-in, set <code className="font-mono">AUTH_ENABLED=true</code> and restart it.
                    </p>
                </CardContent>
            </Card>
        )
    }

    return (
        <div className="flex flex-col gap-5">
            <Card>
                <CardContent className="flex flex-wrap items-center gap-4 p-4">
                    {user?.picture ? (
                        <img src={user.picture} alt="" className="h-11 w-11 rounded-full" />
                    ) : (
                        <div
                            aria-hidden="true"
                            className="flex h-11 w-11 items-center justify-center rounded-full bg-muted text-base font-semibold"
                        >
                            {(user?.name ?? '?').slice(0, 1).toUpperCase()}
                        </div>
                    )}
                    <div className="min-w-0 flex-1">
                        <p className="font-semibold">{user?.name ?? 'Not signed in'}</p>
                        <p className="text-[13px] text-muted-foreground">Signed in with GitHub</p>
                    </div>
                    <Button variant="outline" onClick={() => logout()}>
                        <LogOut className="h-4 w-4" />
                        Sign out
                    </Button>
                </CardContent>
            </Card>

            <Card className="border-status-err/40">
                <CardContent className="flex flex-wrap items-center gap-4 p-4">
                    <div className="min-w-0 flex-1">
                        <p className="font-semibold">Sign out everywhere</p>
                        <p className="text-[13px] text-muted-foreground">
                            Ends every session on every device, including this one. Use it if a device is lost or you
                            think someone else has access.
                        </p>
                    </div>
                    <Button variant="danger" onClick={() => setConfirming(true)} disabled={revoke.isPending}>
                        <ShieldOff className="h-4 w-4" />
                        Sign out everywhere
                    </Button>
                </CardContent>
            </Card>

            <ConfirmationDialog
                open={confirming}
                onOpenChange={setConfirming}
                title="Sign out everywhere?"
                description="Everyone who is signed in, on any device, has to sign in again. This includes you."
                confirmText="Sign everyone out"
                variant="destructive"
                onConfirm={signOutEverywhere}
            />
        </div>
    )
}

export default AccountSection
