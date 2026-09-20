export interface ComposeResources {
    networks: string[]
    volumes: string[]
}

// The networks and volumes a compose file declares, read line by line. This is a rough reading for the overview, not
// a full parse: it lists the names under the top-level `networks:` and `volumes:` and, when there are no named
// volumes, counts bind mounts instead. Services come from the server, not from here.
export function readComposeResources(content: string): ComposeResources {
    const info: ComposeResources = {
        networks: [],
        volumes: [],
    }

    try {
        const lines = content.split('\n')
        let inNetworksSection = false
        let inVolumesSection = false
        let currentIndent = 0

        for (let i = 0; i < lines.length; i++) {
            const line = lines[i]
            const trimmedLine = line.trim()

            // Skip empty lines and comments
            if (!trimmedLine || trimmedLine.startsWith('#')) continue

            // Calculate indentation
            const indent = line.search(/\S/)

            // Check for top-level sections
            if (indent === 0) {
                if (trimmedLine.startsWith('networks:')) {
                    inNetworksSection = true
                    inVolumesSection = false
                    currentIndent = 0
                    continue
                } else if (trimmedLine.startsWith('volumes:')) {
                    inNetworksSection = false
                    inVolumesSection = true
                    currentIndent = 0
                    continue
                } else if (trimmedLine.startsWith('version:') || trimmedLine.startsWith('services:')) {
                    continue
                }
            }

            // Extract networks (first level under 'networks:')
            if (inNetworksSection && indent > 0) {
                if (currentIndent === 0) {
                    currentIndent = indent
                }
                if (indent === currentIndent && trimmedLine.includes(':')) {
                    const networkName = trimmedLine.split(':')[0].trim()
                    if (networkName && !info.networks.includes(networkName)) {
                        info.networks.push(networkName)
                    }
                }
            }

            // Extract volumes (first level under 'volumes:')
            if (inVolumesSection && indent > 0) {
                if (currentIndent === 0) {
                    currentIndent = indent
                }
                if (indent === currentIndent && trimmedLine.includes(':')) {
                    const volumeName = trimmedLine.split(':')[0].trim()
                    if (volumeName && !info.volumes.includes(volumeName)) {
                        info.volumes.push(volumeName)
                    }
                }
            }
        }

        // Also count bind mounts in services
        const bindMounts = (content.match(/- ['"]*[/~]/g) || []).length
        if (bindMounts > 0 && info.volumes.length === 0) {
            info.volumes = [`${bindMounts} bind mount${bindMounts > 1 ? 's' : ''}`]
        }
    } catch (error) {
        console.error('Failed to parse compose content:', error)
    }

    return info
}
