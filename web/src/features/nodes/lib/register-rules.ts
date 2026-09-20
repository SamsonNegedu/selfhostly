import type { Node, RegisterNodeRequest } from '@/shared/types/api'

// A node already in the cluster with this ID or this name (names are compared without regard to case).
export function findConflicts(form: RegisterNodeRequest, nodes: Node[]) {
    const id = form.id.trim()
    const name = form.name.trim().toLowerCase()
    const idTaken = id !== '' && nodes.some((node) => node.id === id)
    const nameTaken = name !== '' && nodes.some((node) => node.name.toLowerCase() === name)
    return { idTaken, nameTaken }
}

// Every field filled in, and neither the ID nor the name already taken.
export function isRegistrationValid(form: RegisterNodeRequest, nodes: Node[]): boolean {
    const { idTaken, nameTaken } = findConflicts(form, nodes)
    return (
        form.id.trim() !== '' &&
        form.name.trim() !== '' &&
        form.api_endpoint.trim() !== '' &&
        form.api_key.trim() !== '' &&
        !idTaken &&
        !nameTaken
    )
}

// Shown until the person fills in the new machine's address.
const NODE_ADDRESS_PLACEHOLDER = 'http://<this-machine-address>:8080'

// The lines for the new machine's .env file, so it can add itself to this cluster.
export function joinSettings(primaryUrl: string, nodeUrl: string, token: string): string {
    return [
        'NODE_IS_PRIMARY=false',
        `PRIMARY_NODE_URL=${primaryUrl.trim()}`,
        `REGISTRATION_TOKEN=${token}`,
        `NODE_API_ENDPOINT=${nodeUrl.trim() || NODE_ADDRESS_PLACEHOLDER}`,
    ].join('\n')
}

// The first node that was not in the cluster when the token was made: the machine that joined with it.
export function findJoinedNode(nodes: Node[] | undefined, knownAtStart: Set<string>): Node | undefined {
    return nodes?.find((node) => !knownAtStart.has(node.id))
}
