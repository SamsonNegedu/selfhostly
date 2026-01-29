import CloudflareManagement from './CloudflareManagement'
import AppBreadcrumb from '@/shared/components/layout/Breadcrumb'

function Cloudflare() {
    return (
        <div className="space-y-4 sm:space-y-6">
            {/* Breadcrumb Navigation - Desktop only */}
            <AppBreadcrumb
                items={[
                    { label: 'Home', path: '/apps' },
                    { label: 'Cloudflare', isCurrentPage: true }
                ]}
            />

            <CloudflareManagement />
        </div>
    )
}

export default Cloudflare
