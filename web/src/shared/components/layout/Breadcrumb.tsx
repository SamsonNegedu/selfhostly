import React from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Home } from 'lucide-react'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/shared/components/ui/Breadcrumb'

interface BreadcrumbItem {
  label: string
  path?: string
  isCurrentPage?: boolean
}

interface AppBreadcrumbProps {
  items: BreadcrumbItem[]
  className?: string
}

function AppBreadcrumb({ items, className }: AppBreadcrumbProps) {
  const location = useLocation()

  // Use provided items or generate from location path
  const breadcrumbItems = items.length > 0
    ? items
    : generateBreadcrumbsFromPath(location.pathname)

  return (
    <Breadcrumb className={`hidden sm:block ${className}`}>
      <BreadcrumbList>
        {breadcrumbItems.map((item, index) => (
          <React.Fragment key={index}>
            <BreadcrumbItem>
              {item.path ? (
                <BreadcrumbLink asChild>
                  <Link to={item.path}>
                    {item.isCurrentPage ? (
                      <BreadcrumbPage>{item.label}</BreadcrumbPage>
                    ) : (
                      <>
                        {index === 0 ? (
                          <Home className="h-4 w-4 mr-1" />
                        ) : (
                          <span className="capitalize">{item.label}</span>
                        )}
                      </>
                    )}
                  </Link>
                </BreadcrumbLink>
              ) : (
                <BreadcrumbPage>{item.label}</BreadcrumbPage>
              )}
            </BreadcrumbItem>
            {index < breadcrumbItems.length - 1 && <BreadcrumbSeparator />}
          </React.Fragment>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  )
}

function generateBreadcrumbsFromPath(pathname: string): BreadcrumbItem[] {
  const pathSegments = pathname.split('/').filter(Boolean)
  const items: BreadcrumbItem[] = []

  // Add home link
  items.push({
    label: 'Home',
    path: '/apps',
  })

  // Build breadcrumbs from path segments
  let currentPath = ''

  pathSegments.forEach((segment, index) => {
    currentPath += `/${segment}`

    if (segment === 'apps' && index === pathSegments.length - 1) {
      // This is likely an app details page
      const appId = pathSegments[index + 1]
      if (appId) {
        items.push(
          { label: 'Apps', path: '/apps' },
          { label: `App ${appId}`, isCurrentPage: true }
        )
      } else {
        items.push({ label: 'Apps', isCurrentPage: true })
      }
    } else if (segment === 'apps' && index < pathSegments.length - 1) {
      // This is the apps list or an app creation page
      items.push({ label: 'Apps', path: '/apps' })
    } else if (segment === 'new' && pathSegments[index - 1] === 'apps') {
      // This is the new app page
      items.push({ label: 'New App', isCurrentPage: true })
    } else if (segment === 'cloudflare') {
      items.push({ label: 'Cloudflare', isCurrentPage: true })
    } else if (segment === 'settings') {
      items.push({ label: 'Settings', isCurrentPage: true })
    } else if (segment !== 'dashboard') {
      // Handle other segments with capitalization
      items.push({
        label: segment.charAt(0).toUpperCase() + segment.slice(1),
        isCurrentPage: index === pathSegments.length - 1
      })
    }
  })

  return items
}

export default AppBreadcrumb
