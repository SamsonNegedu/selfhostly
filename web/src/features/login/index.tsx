import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { AlertCircle, Github, Server, X } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { loginWithGitHub } from '@/shared/services/api'

interface LoginProblem {
    title: string
    detail: string
}

// What went wrong when GitHub sent the person back with an error. Only the allow list case gets its own wording,
// because it is the one they can do something about.
function describeProblem(error: string, description: string | null): LoginProblem {
    const text = `${error} ${description ?? ''}`.toLowerCase()
    if (
        error === 'access_denied' ||
        error === 'unauthorized' ||
        text.includes('whitelist') ||
        text.includes('not authorized')
    ) {
        return {
            title: 'This GitHub account is not allowed',
            detail: 'Only accounts on the allow list can sign in. Ask whoever runs this server to add your GitHub username.',
        }
    }
    return {
        title: 'Sign in did not finish',
        detail: description ? `${description}.` : 'Something went wrong with GitHub. Try again.',
    }
}

function Login() {
    const [params, setParams] = useSearchParams()
    const [problem, setProblem] = useState<LoginProblem | null>(null)
    const [handled, setHandled] = useState<string | null>(null)

    const error = params.get('error')
    const description = params.get('error_description')
    const errorKey = error ? `${error}|${description ?? ''}` : null
    if (error && errorKey !== handled) {
        setHandled(errorKey)
        setProblem(describeProblem(error, description))
    }

    useEffect(() => {
        if (!error) return
        // The message is kept in state, so the address can be cleaned and a refresh does not show it again.
        const next = new URLSearchParams(params)
        next.delete('error')
        next.delete('error_description')
        setParams(next, { replace: true })
    }, [error, params, setParams])

    return (
        <main className="flex min-h-screen items-center justify-center bg-background p-4">
            <div className="flex w-full max-w-sm flex-col gap-6">
                <div className="flex flex-col items-center gap-3 text-center">
                    <div
                        aria-hidden="true"
                        className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary text-primary-foreground"
                    >
                        <Server className="h-6 w-6" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-semibold tracking-tight">Selfhostly</h1>
                        <p className="text-sm text-muted-foreground">Run your own apps, from one place.</p>
                    </div>
                </div>

                {problem && (
                    <div
                        role="alert"
                        className="flex items-start gap-3 rounded-xl bg-status-err-bg p-4 text-status-err-fg"
                    >
                        <AlertCircle aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0" />
                        <div className="min-w-0 flex-1">
                            <p className="font-semibold">{problem.title}</p>
                            <p className="text-sm">{problem.detail}</p>
                        </div>
                        <button
                            type="button"
                            onClick={() => setProblem(null)}
                            aria-label="Dismiss"
                            className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md hover:bg-black/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring max-sm:h-[44px] max-sm:w-[44px]"
                        >
                            <X className="h-4 w-4" />
                        </button>
                    </div>
                )}

                <div className="flex flex-col gap-4 rounded-xl border border-border bg-card p-6 shadow-sm">
                    <h2 className="text-center text-lg font-semibold">Sign in</h2>
                    <Button onClick={loginWithGitHub} size="lg" variant="outline" className="w-full">
                        <Github className="h-5 w-5" />
                        Continue with GitHub
                    </Button>
                    <p className="text-center text-[13px] text-muted-foreground">
                        You need a GitHub account that has been allowed on this server.
                    </p>
                </div>
            </div>
        </main>
    )
}

export default Login
