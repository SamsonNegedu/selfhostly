import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import App from './App'
import '@fontsource/ibm-plex-sans/400.css'
import '@fontsource/ibm-plex-sans/500.css'
import '@fontsource/ibm-plex-sans/600.css'
import '@fontsource/ibm-plex-mono/400.css'
import '@fontsource/ibm-plex-mono/500.css'
import '../styles/globals.css'
import { applyDensity, readDensity } from '@/shared/lib/preferences'

applyDensity(readDensity())

// A page kept from before a deploy asks for chunk files that the new build no longer has. Reloading once
// fetches the current page. The flag stops a reload loop when the file is really missing.
const RELOADED_KEY = 'selfhostly:reloaded-for-new-version'
window.addEventListener('vite:preloadError', (event) => {
    try {
        if (sessionStorage.getItem(RELOADED_KEY)) return
        sessionStorage.setItem(RELOADED_KEY, '1')
    } catch {
        return
    }
    event.preventDefault()
    window.location.reload()
})

const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            refetchOnWindowFocus: false,
            retry: 1,
        },
    },
})

ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        <QueryClientProvider client={queryClient}>
            <App />
        </QueryClientProvider>
    </React.StrictMode>,
)
