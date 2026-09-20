import { CheckCircle2, MinusCircle } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import type { ProviderFeatures } from '@/shared/types/api'

const FEATURES: { key: 'ingress' | 'dns' | 'status_sync' | 'container'; label: string }[] = [
    { key: 'ingress', label: 'Ingress rules' },
    { key: 'dns', label: 'DNS records' },
    { key: 'status_sync', label: 'Status sync' },
    { key: 'container', label: 'Container sidecar' },
]

// Which of the things Selfhostly does with a tunnel the provider supports.
function ProviderFeaturesCard({ features }: { features: ProviderFeatures }) {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-base">What {features.display_name} can do here</CardTitle>
            </CardHeader>
            <CardContent>
                <ul className="grid gap-2 text-sm sm:grid-cols-2">
                    {FEATURES.map((feature) => {
                        const on = features.features[feature.key]
                        return (
                            <li key={feature.key} className="flex items-center gap-2">
                                {on ? (
                                    <CheckCircle2 aria-hidden="true" className="h-4 w-4 text-status-ok-fg" />
                                ) : (
                                    <MinusCircle aria-hidden="true" className="h-4 w-4 text-muted-foreground" />
                                )}
                                <span className={on ? '' : 'text-muted-foreground'}>
                                    {feature.label}
                                    <span className="sr-only">{on ? ': supported' : ': not supported'}</span>
                                </span>
                            </li>
                        )
                    })}
                </ul>
            </CardContent>
        </Card>
    )
}

export default ProviderFeaturesCard
