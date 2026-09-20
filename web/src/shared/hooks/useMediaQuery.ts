import { useEffect, useState } from 'react'

// True while the screen matches a CSS media query, and follows it as the screen changes (rotating a phone,
// resizing a window).
export function useMediaQuery(query: string): boolean {
    const [matches, setMatches] = useState(() => window.matchMedia(query).matches)

    useEffect(() => {
        const list = window.matchMedia(query)
        const update = () => setMatches(list.matches)
        update()
        list.addEventListener('change', update)
        return () => list.removeEventListener('change', update)
    }, [query])

    return matches
}

// Below the `md` breakpoint, where the layout switches to the phone version.
export const useIsPhone = () => useMediaQuery('(max-width: 767px)')
