import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'

// What to refresh when the server says something of that kind changed. Prefix keys cover every variant of a query.
const REFRESH: Record<string, string[]> = {
    apps: ['apps', 'app', 'tunnels'],
    jobs: ['jobs', 'job'],
    nodes: ['nodes', 'node', 'system'],
}

// Several changes usually arrive together, for example a whole deploy, so they are refreshed once.
const BATCH_MS = 250

// Keeps what is on screen current by listening to the server's change stream. The normal polling stays as a
// fallback, so a dropped connection only means updates arrive a little later.
export function useEventStream() {
    const queryClient = useQueryClient()

    useEffect(() => {
        if (typeof EventSource === 'undefined') return
        const pending = new Set<string>()
        let timer: ReturnType<typeof setTimeout> | undefined

        const flush = () => {
            timer = undefined
            for (const key of pending) void queryClient.invalidateQueries({ queryKey: [key] })
            pending.clear()
        }
        const schedule = (keys: string[]) => {
            keys.forEach((key) => pending.add(key))
            if (timer === undefined) timer = setTimeout(flush, BATCH_MS)
        }

        const source = new EventSource('/api/events', { withCredentials: true })
        let connectedBefore = false
        source.addEventListener('change', (event) => {
            try {
                const { kind } = JSON.parse((event as MessageEvent<string>).data) as { kind: string }
                schedule(REFRESH[kind] ?? [])
            } catch {
                // A malformed event is ignored. The next poll catches up.
            }
        })
        // Changes may have been missed while the connection was down, so refresh everything on reconnecting.
        source.onopen = () => {
            if (connectedBefore) schedule(Object.values(REFRESH).flat())
            connectedBefore = true
        }

        return () => {
            source.close()
            if (timer !== undefined) clearTimeout(timer)
        }
    }, [queryClient])
}
