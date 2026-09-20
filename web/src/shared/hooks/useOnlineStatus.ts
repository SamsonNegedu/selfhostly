import { useSyncExternalStore } from 'react'

function subscribe(notify: () => void) {
    window.addEventListener('online', notify)
    window.addEventListener('offline', notify)
    return () => {
        window.removeEventListener('online', notify)
        window.removeEventListener('offline', notify)
    }
}

// False while the browser has no network. It follows the browser's own online and offline events.
export function useOnlineStatus(): boolean {
    return useSyncExternalStore(
        subscribe,
        () => navigator.onLine,
        () => true,
    )
}
