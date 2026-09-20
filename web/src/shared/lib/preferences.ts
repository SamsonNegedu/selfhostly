// Choices that belong to this browser, kept in localStorage. The Fleet screen reads the same keys when it opens.
export const FLEET_VIEW_KEY = 'apps-view-mode'
export const FLEET_GROUP_KEY = 'fleet-group-by'
export const DENSITY_KEY = 'ui-density'

export type Density = 'comfortable' | 'compact'

export function readPreference<T extends string>(key: string, allowed: readonly T[], fallback: T): T {
    try {
        const saved = localStorage.getItem(key)
        return allowed.includes(saved as T) ? (saved as T) : fallback
    } catch {
        return fallback
    }
}

export function savePreference(key: string, value: string) {
    try {
        localStorage.setItem(key, value)
    } catch {
        // Storage can be blocked. The choice then lasts until the page closes.
    }
}

// Density is a page-wide attribute, so styles can react to it without every component knowing about it.
export function applyDensity(density: Density) {
    document.documentElement.dataset.density = density
}

export const readDensity = (): Density =>
    readPreference<Density>(DENSITY_KEY, ['comfortable', 'compact'], 'comfortable')
