import CloudflareManagement from './CloudflareManagement'
import AppBreadcrumb from '@/shared/components/layout/Breadcrumb'

function Cloudflare() {
    return (
        <div className="space-y-6">
            {/* Breadcrumb Navigation */}
            <div>
                <AppBreadcrumb
                    items={[
                        { label: 'Home', path: '/apps' },
                        { label: 'Cloudflare', isCurrentPage: true }
                    ]}
                />
            </div>

            <CloudflareManagement />
        </div>
    )
}

export default Cloudflare
