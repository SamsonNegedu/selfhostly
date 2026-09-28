import { useId, useState } from 'react'
import { AlertTriangle, Copy, KeyRound, Loader2, Trash2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { CodeBlock } from '@/shared/components/ui/CodeBlock'
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { Input } from '@/shared/components/ui/Input'
import { Skeleton } from '@/shared/components/ui/Skeleton'
import { useToast } from '@/shared/components/ui/Toast'
import { useCopyToClipboard } from '@/shared/hooks/useCopyToClipboard'
import { formatAgo } from '@/shared/lib/attention'
import { describeError } from '@/shared/lib/errors'
import { type CreatedDeployHook, useCreateDeployHook, useDeployHooks, useRevokeDeployHook } from '@/shared/services/api'
import type { DeployHook } from '@/shared/types/api'
import { GITHUB_ACTIONS_DEPLOY_STEP, looksUnreachableFromHostedCI } from '../lib/deploy-hook-rules'

// A repository variable to set alongside the SELFHOSTLY_DEPLOY_TOKEN secret, shown with its current
// value so it can be copied straight into GitHub's Settings > Secrets and variables > Actions.
function RepoVariableRow({ name, value, onCopy }: { name: string; value: string; onCopy: () => void }) {
    return (
        <div className="flex items-center gap-2 rounded-lg border border-border px-3 py-2">
            <div className="flex min-w-0 flex-1 flex-col">
                <span className="text-xs font-medium text-muted-foreground">{name}</span>
                <span className="truncate font-mono text-sm">{value}</span>
            </div>
            <Button variant="ghost" size="icon" onClick={onCopy} aria-label={`Copy ${name}`}>
                <Copy className="h-4 w-4" />
            </Button>
        </div>
    )
}

function DeployHooksTab({ appId, nodeId }: { appId: string; nodeId: string }) {
    const { data: hooks, isLoading, error } = useDeployHooks(appId, nodeId)
    const createHook = useCreateDeployHook(appId, nodeId)
    const revokeHook = useRevokeDeployHook(appId, nodeId)
    const copy = useCopyToClipboard()
    const { toast } = useToast()

    const nameId = useId()
    const origin = window.location.origin
    const originUnreachableFromCI = looksUnreachableFromHostedCI(window.location.hostname)
    const [name, setName] = useState('')
    const [created, setCreated] = useState<CreatedDeployHook | null>(null)
    const [pendingRevoke, setPendingRevoke] = useState<DeployHook | null>(null)

    const handleCreate = () => {
        const trimmed = name.trim()
        if (!trimmed) return
        createHook.mutate(trimmed, {
            onSuccess: (hook) => {
                setCreated(hook)
                setName('')
            },
            onError: (err) => {
                toast.error('Could not create the hook', describeError(err))
            },
        })
    }

    const confirmRevoke = () => {
        if (!pendingRevoke) return
        revokeHook.mutate(pendingRevoke.id, {
            onSuccess: () => {
                toast.success('Hook revoked', `"${pendingRevoke.name}" no longer works`)
                setPendingRevoke(null)
            },
            onError: (err) => {
                toast.error('Could not revoke the hook', describeError(err))
            },
        })
    }

    return (
        <div className="flex flex-col gap-5">
            <Card>
                <CardHeader>
                    <CardTitle className="text-base">Deploy hooks</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-col gap-4">
                    <p className="text-sm text-muted-foreground">
                        A deploy hook is a secret URL trigger for this app alone. Give it to a CI pipeline - a GitHub
                        Actions workflow, for example - so it can pull the latest image and restart this app right after
                        a build finishes, with nobody clicking Update by hand. Selfhostly never talks to your Git host:
                        the pipeline calls Selfhostly, not the other way around.
                    </p>

                    {isLoading && (
                        <div role="status" aria-label="Loading deploy hooks" className="flex flex-col gap-2">
                            <Skeleton className="h-10" />
                        </div>
                    )}
                    {error && <ErrorState title="Could not load deploy hooks" error={error} className="p-4" />}
                    {hooks && hooks.length === 0 && (
                        <p className="py-2 text-sm text-muted-foreground">No deploy hooks yet.</p>
                    )}
                    {hooks && hooks.length > 0 && (
                        <ul className="flex flex-col">
                            {hooks.map((hook) => (
                                <li
                                    key={hook.id}
                                    className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-border py-3 first:pt-0 last:border-b-0 last:pb-0"
                                >
                                    <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                                        <span className="truncate font-medium">{hook.name}</span>
                                        <span className="text-compact text-muted-foreground">
                                            Created {formatAgo(hook.created_at)}
                                            {' · '}
                                            {hook.last_used_at
                                                ? `Last triggered ${formatAgo(hook.last_used_at)}${hook.last_used_ip ? ` from ${hook.last_used_ip}` : ''}`
                                                : 'Never triggered'}
                                        </span>
                                    </div>
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => setPendingRevoke(hook)}
                                        disabled={revokeHook.isPending}
                                        aria-label={`Revoke ${hook.name}`}
                                        title="Revoke this hook"
                                    >
                                        <Trash2 className="h-4 w-4" />
                                        Revoke
                                    </Button>
                                </li>
                            ))}
                        </ul>
                    )}

                    <form
                        className="flex flex-col gap-1.5 border-t border-border pt-4"
                        onSubmit={(event) => {
                            event.preventDefault()
                            handleCreate()
                        }}
                    >
                        <label htmlFor={nameId} className="text-compact font-medium">
                            Name
                        </label>
                        <div className="flex flex-col gap-3 sm:flex-row">
                            <Input
                                id={nameId}
                                value={name}
                                onChange={(event) => setName(event.target.value)}
                                placeholder="GitHub Actions - main"
                                aria-describedby={`${nameId}-hint`}
                                className="w-full font-mono sm:max-w-xs"
                            />
                            <Button
                                type="submit"
                                disabled={createHook.isPending || !name.trim()}
                                className="sm:shrink-0"
                            >
                                {createHook.isPending ? (
                                    <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                                ) : (
                                    <KeyRound className="h-4 w-4" />
                                )}
                                New deploy hook
                            </Button>
                        </div>
                        <p id={`${nameId}-hint`} className="text-xs text-muted-foreground">
                            What calls it, so you can tell hooks apart later.
                        </p>
                    </form>
                </CardContent>
            </Card>

            {created && (
                <Card>
                    <CardHeader>
                        <CardTitle className="text-base">Copy this token now</CardTitle>
                    </CardHeader>
                    <CardContent className="flex flex-col gap-4">
                        <p className="text-sm text-muted-foreground">
                            This is the only time it is shown. Store it as a secret in your CI pipeline, not in the
                            workflow file itself.
                        </p>
                        <CodeBlock lines={[{ text: created.token }]} aria-label="Deploy hook token" />
                        <div className="flex flex-wrap items-center gap-3">
                            <Button
                                variant="outline"
                                onClick={() => void copy(created.token, 'The token copied to your clipboard')}
                            >
                                <Copy className="h-4 w-4" />
                                Copy token
                            </Button>
                        </div>

                        <p className="text-sm text-muted-foreground">
                            In your repository's Settings &gt; Secrets and variables &gt; Actions, add the token above
                            as a secret named <code>SELFHOSTLY_DEPLOY_TOKEN</code>, and these three as variables:
                        </p>
                        {originUnreachableFromCI && (
                            <div className="flex items-start gap-2 rounded-lg bg-status-warn-bg px-3 py-2.5 text-compact text-status-warn-fg">
                                <AlertTriangle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0" />
                                <p>
                                    This looks like an address only reachable on your own network. GitHub's hosted
                                    runners cannot reach it - use your public domain or tunnel URL instead (or a
                                    self-hosted runner on this network).
                                </p>
                            </div>
                        )}
                        <div className="flex flex-col gap-2">
                            <RepoVariableRow
                                name="SELFHOSTLY_URL"
                                value={origin}
                                onCopy={() => void copy(origin, 'SELFHOSTLY_URL copied to your clipboard')}
                            />
                            <RepoVariableRow
                                name="SELFHOSTLY_APP_ID"
                                value={appId}
                                onCopy={() => void copy(appId, 'SELFHOSTLY_APP_ID copied to your clipboard')}
                            />
                            <RepoVariableRow
                                name="SELFHOSTLY_NODE_ID"
                                value={nodeId}
                                onCopy={() => void copy(nodeId, 'SELFHOSTLY_NODE_ID copied to your clipboard')}
                            />
                        </div>

                        <p className="text-sm text-muted-foreground">
                            Example step for a GitHub Actions workflow, after it builds and pushes the image:
                        </p>
                        <CodeBlock
                            lines={GITHUB_ACTIONS_DEPLOY_STEP.split('\n').map((text) => ({ text }))}
                            aria-label="Example GitHub Actions step"
                        />
                        <div className="flex flex-wrap items-center gap-3">
                            <Button
                                variant="outline"
                                onClick={() =>
                                    void copy(GITHUB_ACTIONS_DEPLOY_STEP, 'The workflow step copied to your clipboard')
                                }
                            >
                                <Copy className="h-4 w-4" />
                                Copy step
                            </Button>
                        </div>
                    </CardContent>
                </Card>
            )}

            <ConfirmationDialog
                open={!!pendingRevoke}
                onOpenChange={(open) => !open && setPendingRevoke(null)}
                title={`Revoke "${pendingRevoke?.name}"?`}
                description="Anything still using this token stops being able to trigger an update for this app. This cannot be undone."
                confirmText="Revoke"
                cancelText="Cancel"
                onConfirm={confirmRevoke}
                isLoading={revokeHook.isPending}
                variant="destructive"
                confirmationText={pendingRevoke?.name}
            />
        </div>
    )
}

export default DeployHooksTab
