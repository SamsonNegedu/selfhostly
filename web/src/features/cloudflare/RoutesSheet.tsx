import { Link } from 'react-router-dom'
import { buttonClasses } from '@/shared/components/ui/Button'
import { Sheet, SheetContent, SheetDescription, SheetTitle } from '@/shared/components/ui/Sheet'
import { appHref } from '@/shared/lib/routes'
import { IngressConfiguration } from './IngressConfiguration'
import type { AddressRow } from './lib/address-rows'

interface RoutesSheetProps {
    // The address whose routes are shown. The sheet is closed when there is none.
    row: AddressRow | null
    onClose: () => void
    onSaved: () => void
}

// The routes of one address: editable for a custom domain, and explained for a Quick Tunnel.
function RoutesSheet({ row, onClose, onSaved }: RoutesSheetProps) {
    return (
        <Sheet open={row !== null} onOpenChange={(open) => !open && onClose()}>
            <SheetContent
                side="right"
                className="max-md:inset-x-0 max-md:bottom-0 max-md:top-auto max-md:h-auto max-md:max-h-[88vh] max-md:max-w-none max-md:rounded-t-[20px] max-md:border-l-0 max-md:border-t md:max-w-xl overflow-y-auto"
            >
                {row && (
                    <>
                        <SheetTitle>{row.app.name} routes</SheetTitle>
                        <SheetDescription>
                            {row.kind === 'quick'
                                ? 'This address is temporary and changes if the app restarts.'
                                : 'One hostname per service.'}
                        </SheetDescription>
                        {row.kind === 'custom' && row.tunnel ? (
                            <IngressConfiguration
                                appId={row.app.id}
                                nodeId={row.app.node_id}
                                existingIngress={row.tunnel.ingress_rules ?? []}
                                onSave={onSaved}
                                flat
                            />
                        ) : (
                            <div className="flex flex-col gap-3 text-sm">
                                <p className="font-mono">{row.app.public_url}</p>
                                <p className="text-muted-foreground">
                                    Quick Tunnels have one fixed route to the app and cannot be edited. For a stable
                                    address, switch the app to your own domain.
                                </p>
                                <Link
                                    to={appHref(row.app, 'access')}
                                    className={buttonClasses({ className: 'self-start' })}
                                >
                                    Open the app's Access tab
                                </Link>
                            </div>
                        )}
                    </>
                )}
            </SheetContent>
        </Sheet>
    )
}

export default RoutesSheet
