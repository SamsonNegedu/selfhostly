import type { App, ContainerInfo } from '@/shared/types/api'

export type ContainerAction = 'restart' | 'stop' | 'delete'

export interface ContainerGroup {
    key: string
    name: string
    app?: App
    managed: boolean
    containers: ContainerInfo[]
}

export function groupContainers(containers: ContainerInfo[], apps: App[]): ContainerGroup[] {
    const groups = new Map<string, ContainerGroup>()
    for (const container of containers) {
        const managed = container.is_managed
        const key = managed ? `${container.node_id}:${container.app_name}` : 'external'
        const existing = groups.get(key)
        if (existing) existing.containers.push(container)
        else {
            groups.set(key, {
                key,
                name: managed ? container.app_name : 'Not managed by Selfhostly',
                app: managed
                    ? apps.find((app) => app.name === container.app_name && app.node_id === container.node_id)
                    : undefined,
                managed,
                containers: [container],
            })
        }
    }
    return [...groups.values()].sort((a, b) =>
        a.managed === b.managed ? a.name.localeCompare(b.name) : a.managed ? -1 : 1,
    )
}

// What the confirmation asks and says for an action on a container.
export function actionCopy(action: ContainerAction, container: ContainerInfo) {
    return {
        restart: {
            title: `${container.state === 'stopped' ? 'Start' : 'Restart'} ${container.name}?`,
            text: container.state === 'stopped' ? 'Start' : 'Restart',
            body:
                container.state === 'stopped'
                    ? 'It starts running again.'
                    : 'It is unavailable for a moment while it restarts.',
        },
        stop: {
            title: `Stop ${container.name}?`,
            text: 'Stop',
            body: 'It stays stopped until you start it again.',
        },
        delete: {
            title: `Delete ${container.name}?`,
            text: 'Delete container',
            body: container.is_managed
                ? `It belongs to ${container.app_name}. Deleting it can break that app. Stopping the whole app from Fleet is safer.`
                : 'This removes the container. Volumes may stay behind.',
        },
    }[action]
}
