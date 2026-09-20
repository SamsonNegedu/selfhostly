import { Link } from 'react-router-dom'
import { Compass } from 'lucide-react'
import { buttonClasses } from '@/shared/components/ui/Button'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { ROUTES } from '@/shared/lib/routes'

// Shown for an address that does not exist, instead of silently sending the person somewhere else.
export function NotFound() {
    return (
        <EmptyState
            icon={<Compass className="h-5 w-5" />}
            title="That page does not exist"
            description="The address may be mistyped, or the page may have moved."
            action={
                <Link to={ROUTES.fleet} className={buttonClasses()}>
                    Go to Fleet
                </Link>
            }
            className="py-16"
        />
    )
}
