import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { YamlEditor } from '@/shared/components/ui/YamlEditor'

interface PasteComposeCardProps {
    value: string
    onChange: (value: string) => void
}

// A place to paste a docker-compose.yml, checked as it is typed.
function PasteComposeCard({ value, onChange }: PasteComposeCardProps) {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-base">Compose file</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
                <YamlEditor value={value} onChange={onChange} aria-label="Compose file" height={360} />
                {value.trim() === '' && (
                    <p className="text-compact text-muted-foreground">
                        Paste a docker-compose.yml. It is checked as you type.
                    </p>
                )}
            </CardContent>
        </Card>
    )
}

export default PasteComposeCard
