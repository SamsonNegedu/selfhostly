import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'

interface CommandPaletteContextValue {
    open: boolean
    setOpen: (open: boolean) => void
    toggle: () => void
}

const CommandPaletteContext = createContext<CommandPaletteContextValue | undefined>(undefined)

// Lets the topbar trigger and the keyboard shortcut open the same palette.
export function CommandPaletteProvider({ children }: { children: ReactNode }) {
    const [open, setOpen] = useState(false)
    const toggle = useCallback(() => setOpen((current) => !current), [])
    const value = useMemo(() => ({ open, setOpen, toggle }), [open, toggle])

    return <CommandPaletteContext.Provider value={value}>{children}</CommandPaletteContext.Provider>
}

export function useCommandPalette(): CommandPaletteContextValue {
    const context = useContext(CommandPaletteContext)
    if (context === undefined) {
        throw new Error('useCommandPalette must be used within a CommandPaletteProvider')
    }
    return context
}
