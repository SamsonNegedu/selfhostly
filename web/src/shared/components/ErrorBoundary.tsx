import { Component, type ErrorInfo, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { buttonClasses, Button } from '@/shared/components/ui/Button'
import { ErrorState } from '@/shared/components/ui/ErrorState'
import { ROUTES } from '@/shared/lib/routes'

interface ErrorBoundaryProps {
    children: ReactNode
    // When this changes the boundary tries again, so leaving a broken page clears the error.
    resetKey?: string
}

interface ErrorBoundaryState {
    failed: boolean
}

// Catches a crash while rendering a page, so the person sees a message and a way out and not a blank screen.
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
    state: ErrorBoundaryState = { failed: false }

    static getDerivedStateFromError(): ErrorBoundaryState {
        return { failed: true }
    }

    componentDidCatch(error: Error, info: ErrorInfo) {
        console.error('A page failed to render', error, info.componentStack)
    }

    componentDidUpdate(previous: ErrorBoundaryProps) {
        if (this.state.failed && previous.resetKey !== this.props.resetKey) this.setState({ failed: false })
    }

    render() {
        if (!this.state.failed) return this.props.children
        return (
            <ErrorState
                title="This page hit a problem"
                description="Something went wrong while showing it. Reloading usually fixes it. If it keeps happening, the browser console has the details."
            >
                <div className="flex flex-wrap gap-2">
                    <Button onClick={() => window.location.reload()}>Reload the page</Button>
                    <Link to={ROUTES.fleet} onClick={() => this.setState({ failed: false })} className={buttonClasses({ variant: 'outline' })}>
                        Back to Fleet
                    </Link>
                </div>
            </ErrorState>
        )
    }
}
