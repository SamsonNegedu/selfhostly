import React, { createContext, useContext, useEffect, useState } from 'react'

type Theme = 'light' | 'dark' | 'system'
type ActualTheme = 'light' | 'dark'

interface ThemeContextType {
    theme: Theme
    setTheme: (theme: Theme) => void
    actualTheme: ActualTheme
}

// Keep in sync with the inline script in index.html, which applies the class before first paint.
const THEME_STORAGE_KEY = 'theme'
const DEFAULT_THEME: Theme = 'dark'
const DARK_SCHEME_QUERY = '(prefers-color-scheme: dark)'

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

function isTheme(value: string | null): value is Theme {
    return value === 'light' || value === 'dark' || value === 'system'
}

function readStoredTheme(): Theme {
    const saved = localStorage.getItem(THEME_STORAGE_KEY)
    return isTheme(saved) ? saved : DEFAULT_THEME
}

function resolveTheme(theme: Theme): ActualTheme {
    if (theme === 'system') {
        return window.matchMedia(DARK_SCHEME_QUERY).matches ? 'dark' : 'light'
    }
    return theme
}

function applyThemeClass(actual: ActualTheme) {
    const root = window.document.documentElement
    root.classList.remove('light', 'dark')
    root.classList.add(actual)
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
    const [theme, setTheme] = useState<Theme>(readStoredTheme)
    const [actualTheme, setActualTheme] = useState<ActualTheme>(() => resolveTheme(readStoredTheme()))

    useEffect(() => {
        localStorage.setItem(THEME_STORAGE_KEY, theme)
        const actual = resolveTheme(theme)
        applyThemeClass(actual)
        setActualTheme(actual)
    }, [theme])

    useEffect(() => {
        if (theme !== 'system') return

        const mediaQuery = window.matchMedia(DARK_SCHEME_QUERY)
        const handleChange = () => {
            const actual = resolveTheme('system')
            applyThemeClass(actual)
            setActualTheme(actual)
        }

        mediaQuery.addEventListener('change', handleChange)
        return () => mediaQuery.removeEventListener('change', handleChange)
    }, [theme])

    return (
        <ThemeContext.Provider value={{ theme, setTheme, actualTheme }}>
            {children}
        </ThemeContext.Provider>
    )
}

export function useTheme() {
    const context = useContext(ThemeContext)
    if (context === undefined) {
        throw new Error('useTheme must be used within a ThemeProvider')
    }
    return context
}
