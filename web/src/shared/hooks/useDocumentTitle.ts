import { useEffect } from 'react'

const APP_NAME = 'Selfhostly'

// Puts the page name in the browser tab and history, such as "Settings · Selfhostly", so tabs can be told apart
// and a screen reader hears where it is after a page change.
export function useDocumentTitle(...parts: (string | undefined)[]) {
    const title = [...parts.filter(Boolean), APP_NAME].join(' · ')
    useEffect(() => {
        document.title = title
    }, [title])
}
