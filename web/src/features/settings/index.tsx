import { useSearchParams } from 'react-router-dom'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/components/ui/Tabs'
import AccountSection from './sections/AccountSection'
import ActivitySection from './sections/ActivitySection'
import AppearanceSection from './sections/AppearanceSection'
import GeneralSection from './sections/GeneralSection'
import TunnelProviderSection from './sections/TunnelProviderSection'

const SECTIONS = [
    { id: 'tunnel', label: 'Tunnel provider' },
    { id: 'general', label: 'General' },
    { id: 'appearance', label: 'Appearance' },
    { id: 'account', label: 'Sign in' },
    { id: 'activity', label: 'Activity' },
] as const

type SectionId = (typeof SECTIONS)[number]['id']

const isSection = (value: string | null): value is SectionId => SECTIONS.some((section) => section.id === value)

// Settings are grouped by what they change. The current group is in the address, so it can be linked to.
function Settings() {
    const [params, setParams] = useSearchParams()
    const requested = params.get('section')
    const section: SectionId = isSection(requested) ? requested : 'tunnel'

    return (
        <div className="flex max-w-3xl flex-col gap-5">
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
                <p className="text-muted-foreground">How Selfhostly connects, looks and keeps you signed in.</p>
            </div>

            <Tabs value={section} onValueChange={(value) => setParams({ section: value }, { replace: true })}>
                <TabsList aria-label="Settings sections">
                    {SECTIONS.map((item) => (
                        <TabsTrigger key={item.id} value={item.id}>
                            {item.label}
                        </TabsTrigger>
                    ))}
                </TabsList>
                <TabsContent value="tunnel">
                    <TunnelProviderSection />
                </TabsContent>
                <TabsContent value="general">
                    <GeneralSection />
                </TabsContent>
                <TabsContent value="appearance">
                    <AppearanceSection />
                </TabsContent>
                <TabsContent value="account">
                    <AccountSection />
                </TabsContent>
                <TabsContent value="activity">
                    <ActivitySection />
                </TabsContent>
            </Tabs>
        </div>
    )
}

export default Settings
