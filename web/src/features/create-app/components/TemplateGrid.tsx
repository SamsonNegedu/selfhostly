import { AppTile } from '@/shared/components/ui/AppTile'
import { cn } from '@/shared/lib/utils'
import { TEMPLATES, type AppTemplate } from '../lib/templates'

interface TemplateGridProps {
    templateId: string | null
    onSelect: (template: AppTemplate) => void
}

// The ready-made apps to start from. The chosen one is marked, and pressing it again keeps it chosen.
function TemplateGrid({ templateId, onSelect }: TemplateGridProps) {
    return (
        <div role="group" aria-label="Templates" className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {TEMPLATES.map((item) => (
                <button
                    key={item.id}
                    type="button"
                    aria-pressed={item.id === templateId}
                    onClick={() => onSelect(item)}
                    className={cn(
                        'flex min-h-[44px] items-start gap-3 rounded-xl border bg-card p-3.5 text-left transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                        item.id === templateId ? 'border-primary ring-1 ring-primary' : 'border-border',
                    )}
                >
                    <AppTile name={item.name} size="md" />
                    <span className="flex min-w-0 flex-col">
                        <span className="text-body font-semibold">{item.name}</span>
                        <span className="text-compact text-muted-foreground">{item.description}</span>
                    </span>
                </button>
            ))}
        </div>
    )
}

export default TemplateGrid
