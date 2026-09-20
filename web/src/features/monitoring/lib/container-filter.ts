import type { ContainerInfo } from '@/shared/types/api'

export type StateFilter = 'all' | 'running' | 'stopped'

// The containers in the chosen state whose name, or whose app's name, contains the search text.
export function filterContainers(containers: ContainerInfo[], state: StateFilter, query: string): ContainerInfo[] {
    const text = query.trim().toLowerCase()
    return containers.filter((container) => {
        if (state !== 'all' && container.state !== state) return false
        return (
            text === '' ||
            container.name.toLowerCase().includes(text) ||
            container.app_name.toLowerCase().includes(text)
        )
    })
}
