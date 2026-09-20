import { useMemo, useState } from 'react'
import { Container, Search } from 'lucide-react'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { Input } from '@/shared/components/ui/Input'
import { SegmentedControl } from '@/shared/components/ui/SegmentedControl'
import type { App, ContainerInfo } from '@/shared/types/api'
import { filterContainers, type StateFilter } from '../lib/container-filter'
import ContainersByApp from './ContainersByApp'

const STATE_OPTIONS = [
    { value: 'all', label: 'All' },
    { value: 'running', label: 'Running' },
    { value: 'stopped', label: 'Stopped' },
]

interface ContainersSectionProps {
    containers: ContainerInfo[]
    apps: App[]
    nodeName: (id: string) => string
}

// Every container on the answering nodes, with a search box and a state filter.
function ContainersSection({ containers, apps, nodeName }: ContainersSectionProps) {
    const [query, setQuery] = useState('')
    const [state, setState] = useState<StateFilter>('all')
    const visible = useMemo(() => filterContainers(containers, state, query), [containers, state, query])

    return (
        <section aria-label="Containers" className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <h2 className="text-lg font-semibold">Containers</h2>
                <div className="flex flex-wrap items-center gap-2">
                    <div className="relative">
                        <Search
                            aria-hidden="true"
                            className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
                        />
                        <Input
                            aria-label="Search containers"
                            value={query}
                            onChange={(event) => setQuery(event.target.value)}
                            placeholder="Search"
                            className="pl-9 sm:w-56"
                        />
                    </div>
                    <SegmentedControl
                        aria-label="Container state"
                        options={STATE_OPTIONS}
                        value={state}
                        onValueChange={(value) => setState(value as StateFilter)}
                    />
                </div>
            </div>
            {visible.length === 0 ? (
                <EmptyState
                    icon={<Container className="h-5 w-5" />}
                    title={containers.length === 0 ? 'No containers' : 'No containers match'}
                    description={
                        containers.length === 0
                            ? 'Nothing is running on these nodes yet.'
                            : 'Try another search or state.'
                    }
                    className="py-10"
                />
            ) : (
                <ContainersByApp containers={visible} apps={apps} nodeName={nodeName} />
            )}
        </section>
    )
}

export default ContainersSection
