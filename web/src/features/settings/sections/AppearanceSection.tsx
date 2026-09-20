import { useState } from 'react'
import * as RadioGroupPrimitive from '@radix-ui/react-radio-group'
import { Card, CardContent } from '@/shared/components/ui/Card'
import { SegmentedControl } from '@/shared/components/ui/SegmentedControl'
import { Switch } from '@/shared/components/ui/Switch'
import { useTheme } from '@/shared/components/theme/ThemeProvider'
import {
    applyDensity,
    DENSITY_KEY,
    FLEET_GROUP_KEY,
    FLEET_VIEW_KEY,
    readDensity,
    readPreference,
    savePreference,
    type Density,
} from '@/shared/lib/preferences'
import { cn } from '@/shared/lib/utils'

type ThemeChoice = 'dark' | 'light' | 'system'

// A small picture of each theme, drawn in that theme's own colors, so it looks right whichever theme is on now.
const PREVIEWS = {
    dark: { page: '#0E1116', card: '#151A21', line: '#262C36', ink: '#E8EAED' },
    light: { page: '#F6F7F9', card: '#FFFFFF', line: '#DDE1E7', ink: '#12151A' },
} as const
const STATUS_DOTS = ['#2FBF83', '#E0A100', '#F0524A']

const THEMES: { value: ThemeChoice; label: string; preview: keyof typeof PREVIEWS }[] = [
    { value: 'dark', label: 'Dark', preview: 'dark' },
    { value: 'light', label: 'Light', preview: 'light' },
    { value: 'system', label: 'System', preview: 'dark' },
]

function ThemePreview({ colors }: { colors: (typeof PREVIEWS)[keyof typeof PREVIEWS] }) {
    return (
        <span
            aria-hidden="true"
            className="flex h-[84px] w-full flex-col gap-1.5 rounded-lg border p-2.5"
            style={{ background: colors.page, borderColor: colors.line }}
        >
            <span className="h-2 w-3/5 rounded" style={{ background: colors.ink }} />
            <span className="h-6 rounded-md border" style={{ background: colors.card, borderColor: colors.line }} />
            <span className="flex items-center gap-1">
                {STATUS_DOTS.map((dot) => (
                    <span key={dot} className="h-2 w-2 rounded-full" style={{ background: dot }} />
                ))}
            </span>
        </span>
    )
}

function PreferenceRow({
    title,
    description,
    children,
}: {
    title: string
    description: string
    children: React.ReactNode
}) {
    return (
        <div className="flex items-center gap-3.5 border-t border-border py-3.5 first:border-t-0 first:pt-0 last:pb-0">
            <div className="min-w-0 flex-1">
                <p className="text-sm font-medium">{title}</p>
                <p className="text-compact text-muted-foreground">{description}</p>
            </div>
            {children}
        </div>
    )
}

// How Selfhostly looks on this device. These choices are remembered by this browser only.
function AppearanceSection() {
    const { theme, setTheme } = useTheme()
    const [view, setView] = useState(() => readPreference(FLEET_VIEW_KEY, ['grid', 'list'], 'grid'))
    const [density, setDensity] = useState<Density>(readDensity)
    const [groupByNode, setGroupByNode] = useState(
        () => readPreference(FLEET_GROUP_KEY, ['node', 'none'], 'node') === 'node',
    )

    return (
        <div className="flex flex-col gap-4">
            <Card>
                <CardContent className="flex flex-col gap-4 p-6">
                    <div>
                        <h3 className="text-base font-semibold">Theme</h3>
                        <p className="text-compact text-muted-foreground">
                            Dark is the default. System follows your device.
                        </p>
                    </div>
                    <RadioGroupPrimitive.Root
                        value={theme}
                        onValueChange={(value) => setTheme(value as ThemeChoice)}
                        aria-label="Theme"
                        className="grid grid-cols-3 gap-3"
                    >
                        {THEMES.map((option) => (
                            <RadioGroupPrimitive.Item
                                key={option.value}
                                value={option.value}
                                className={cn(
                                    'flex flex-col gap-2 rounded-xl border bg-card p-3 text-left transition-colors hover:bg-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ring-offset-background',
                                    theme === option.value ? 'border-2 border-primary p-[11px]' : 'border-border',
                                )}
                            >
                                <ThemePreview colors={PREVIEWS[option.preview]} />
                                <span className="text-compact font-medium">{option.label}</span>
                            </RadioGroupPrimitive.Item>
                        ))}
                    </RadioGroupPrimitive.Root>
                </CardContent>
            </Card>

            <Card>
                <CardContent className="flex flex-col p-6">
                    <div className="mb-4">
                        <h3 className="text-base font-semibold">Layout</h3>
                        <p className="text-compact text-muted-foreground">How Fleet looks when it opens.</p>
                    </div>
                    <PreferenceRow title="Default Fleet view" description="Cards or table">
                        <SegmentedControl
                            aria-label="Default Fleet view"
                            value={view}
                            onValueChange={(value) => {
                                setView(value as 'grid' | 'list')
                                savePreference(FLEET_VIEW_KEY, value)
                            }}
                            options={[
                                { value: 'grid', label: 'Cards' },
                                { value: 'list', label: 'Table' },
                            ]}
                        />
                    </PreferenceRow>
                    <PreferenceRow title="Density" description="Compact fits more rows on screen">
                        <SegmentedControl
                            aria-label="Density"
                            value={density}
                            onValueChange={(value) => {
                                setDensity(value as Density)
                                savePreference(DENSITY_KEY, value)
                                applyDensity(value as Density)
                            }}
                            options={[
                                { value: 'comfortable', label: 'Comfortable' },
                                { value: 'compact', label: 'Compact' },
                            ]}
                        />
                    </PreferenceRow>
                    <PreferenceRow title="Group apps by node" description="Show a header per node">
                        <Switch
                            aria-label="Group apps by node"
                            checked={groupByNode}
                            onCheckedChange={(checked) => {
                                setGroupByNode(checked)
                                savePreference(FLEET_GROUP_KEY, checked ? 'node' : 'none')
                            }}
                        />
                    </PreferenceRow>
                </CardContent>
            </Card>
        </div>
    )
}

export default AppearanceSection
