import * as React from 'react'

interface ActionBarProps {
    // Names the bar for screen readers.
    label: string
    children: React.ReactNode
}

// The bar pinned to the bottom of a page while there is something to save or submit. On phones it sits above the tab
// bar. Its position comes from the `sticky-action-bar` class in globals.css, so every bar agrees.
function ActionBar({ label, children }: ActionBarProps) {
    return (
        <div
            role="region"
            aria-label={label}
            className="sticky-action-bar z-20 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-card p-3 shadow-lg"
        >
            {children}
        </div>
    )
}

export default ActionBar
