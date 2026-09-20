import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { CheckCircle2, Copy, KeyRound, Loader2 } from 'lucide-react'
import { Button, buttonClasses } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { CodeBlock } from '@/shared/components/ui/CodeBlock'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { useCopyToClipboard } from '@/shared/hooks/useCopyToClipboard'
import { describeError } from '@/shared/lib/errors'
import { ROUTES } from '@/shared/lib/routes'
import { useCreateJoinToken, useNodes } from '@/shared/services/api'
import { findJoinedNode, joinSettings } from '../lib/register-rules'

const POLL_MS = 3000

// The new machine adds itself with a single-use token. This page makes the token and the settings to paste,
// then waits for the machine to show up.
function AutomaticJoin() {
    const copy = useCopyToClipboard()
    const createToken = useCreateJoinToken()
    const [primaryUrl, setPrimaryUrl] = useState(window.location.origin)
    const [nodeUrl, setNodeUrl] = useState('')
    const issued = createToken.data
    const { data: nodes } = useNodes({ refetchInterval: issued ? POLL_MS : false })
    const [knownAtStart, setKnownAtStart] = useState<Set<string> | null>(null)

    // Remember who was in the cluster when the token was made, so anyone new after that is the machine that joined.
    if (issued && nodes && knownAtStart === null) setKnownAtStart(new Set(nodes.map((node) => node.id)))
    const joined = issued && knownAtStart ? findJoinedNode(nodes, knownAtStart) : undefined

    const env = useMemo(() => joinSettings(primaryUrl, nodeUrl, issued?.token ?? ''), [primaryUrl, nodeUrl, issued])

    return (
        <div className="flex flex-col gap-5">
            <Card>
                <CardHeader>
                    <CardTitle className="text-base">1. Make a join token</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-col gap-4">
                    <p className="text-sm text-muted-foreground">
                        The token works once and expires after an hour, so it is safe to paste into the new machine's
                        settings.
                    </p>
                    <div className="grid gap-4 sm:grid-cols-2">
                        <Field label="This server's address" hint="What the new machine will use to reach this one.">
                            <Input
                                value={primaryUrl}
                                onChange={(event) => setPrimaryUrl(event.target.value)}
                                inputMode="url"
                                className="font-mono"
                            />
                        </Field>
                        <Field label="The new machine's address" hint="Where this server can reach the new machine.">
                            <Input
                                value={nodeUrl}
                                onChange={(event) => setNodeUrl(event.target.value)}
                                placeholder="http://192.168.1.50:8080"
                                inputMode="url"
                                className="font-mono"
                            />
                        </Field>
                    </div>
                    <div className="flex flex-wrap items-center gap-3">
                        <Button onClick={() => createToken.mutate()} disabled={createToken.isPending}>
                            {createToken.isPending ? (
                                <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                            ) : (
                                <KeyRound className="h-4 w-4" />
                            )}
                            {issued ? 'Make a new token' : 'Make a join token'}
                        </Button>
                        {issued && (
                            <span className="text-compact text-muted-foreground">
                                Expires{' '}
                                {new Date(issued.expires_at).toLocaleTimeString([], {
                                    hour: '2-digit',
                                    minute: '2-digit',
                                })}
                            </span>
                        )}
                    </div>
                    {createToken.error && (
                        <p role="alert" className="text-sm text-status-err-fg">
                            Could not make a token. {describeError(createToken.error)}
                        </p>
                    )}
                </CardContent>
            </Card>

            {issued && (
                <Card>
                    <CardHeader>
                        <CardTitle className="text-base">2. Put this in the new machine's .env</CardTitle>
                    </CardHeader>
                    <CardContent className="flex flex-col gap-4">
                        <CodeBlock
                            lines={env.split('\n').map((text) => ({ text }))}
                            aria-label="Settings for the new machine"
                        />
                        <div className="flex flex-wrap items-center gap-3">
                            <Button
                                variant="outline"
                                onClick={() => void copy(env, 'The settings copied to your clipboard')}
                            >
                                <Copy className="h-4 w-4" />
                                Copy settings
                            </Button>
                            <span className="text-compact text-muted-foreground">
                                Install Selfhostly there, add these lines to its .env, and start it.
                            </span>
                        </div>
                    </CardContent>
                </Card>
            )}

            {issued && (
                <Card aria-label="Connection" role="region">
                    <CardContent className="flex items-center gap-3 p-4">
                        {joined ? (
                            <>
                                <CheckCircle2 aria-hidden="true" className="h-5 w-5 shrink-0 text-status-ok-fg" />
                                <div className="flex-1">
                                    <p className="font-semibold">{joined.name} joined</p>
                                    <p className="text-compact text-muted-foreground">
                                        It is {joined.status}. You can put apps on it now.
                                    </p>
                                </div>
                                <Link to={ROUTES.nodes} className={buttonClasses()}>
                                    See nodes
                                </Link>
                            </>
                        ) : (
                            <>
                                <Loader2
                                    aria-hidden="true"
                                    className="h-5 w-5 shrink-0 animate-spin text-muted-foreground"
                                />
                                <div className="flex-1">
                                    <p className="font-semibold">Waiting for the new machine</p>
                                    <p className="text-compact text-muted-foreground">
                                        This page checks every few seconds and shows it here as soon as it starts.
                                    </p>
                                </div>
                            </>
                        )}
                    </CardContent>
                </Card>
            )}
        </div>
    )
}

export default AutomaticJoin
