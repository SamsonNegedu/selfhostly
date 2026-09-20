import * as React from 'react'
import { Link } from 'react-router-dom'
import {
    Breadcrumb,
    BreadcrumbItem,
    BreadcrumbLink,
    BreadcrumbList,
    BreadcrumbPage,
    BreadcrumbSeparator,
} from '../ui/Breadcrumb'
import { useCrumbs } from './useCrumbs'

// Where you are, in the topbar. The last crumb is the current page and is not a link.
function TopbarBreadcrumbs({ className }: { className?: string }) {
    const crumbs = useCrumbs()
    if (crumbs.length === 0) return null

    const lastIndex = crumbs.length - 1

    return (
        <Breadcrumb className={className}>
            <BreadcrumbList>
                {crumbs.map((crumb, index) => (
                    <React.Fragment key={`${crumb.label}-${index}`}>
                        {index > 0 && <BreadcrumbSeparator />}
                        <BreadcrumbItem>
                            {index === lastIndex || !crumb.to ? (
                                <BreadcrumbPage>{crumb.label}</BreadcrumbPage>
                            ) : (
                                <BreadcrumbLink asChild>
                                    <Link to={crumb.to}>{crumb.label}</Link>
                                </BreadcrumbLink>
                            )}
                        </BreadcrumbItem>
                    </React.Fragment>
                ))}
            </BreadcrumbList>
        </Breadcrumb>
    )
}

export default TopbarBreadcrumbs
