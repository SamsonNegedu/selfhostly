import defaultTheme from 'tailwindcss/defaultTheme'

const hslVar = (name) => `hsl(var(--${name}))`

const statusColor = (kind) => ({
    DEFAULT: hslVar(`status-${kind}`),
    bg: hslVar(`status-${kind}-bg`),
    fg: hslVar(`status-${kind}-fg`),
})

const tintColor = (name) => ({
    bg: hslVar(`tint-${name}-bg`),
    fg: hslVar(`tint-${name}-fg`),
})

/** @type {import('tailwindcss').Config} */
export default {
    darkMode: ['class'],
    content: ['./src/**/*.{js,ts,jsx,tsx}'],
    theme: {
        extend: {
            screens: {
                xs: '475px',
            },
            // The type scale, in px so phones (which shrink the root size) keep the same reading size.
            fontSize: {
                caption: '11px',
                detail: '12px',
                compact: '13px',
                body: '14px',
                title: '15px',
                heading: '16px',
            },
            fontFamily: {
                sans: ['"IBM Plex Sans"', ...defaultTheme.fontFamily.sans],
                mono: ['"IBM Plex Mono"', ...defaultTheme.fontFamily.mono],
            },
            colors: {
                border: hslVar('border'),
                input: hslVar('input'),
                ring: hslVar('ring'),
                scrim: hslVar('scrim'),
                background: hslVar('background'),
                foreground: hslVar('foreground'),
                primary: {
                    DEFAULT: hslVar('primary'),
                    foreground: hslVar('primary-foreground'),
                },
                secondary: {
                    DEFAULT: hslVar('secondary'),
                    foreground: hslVar('secondary-foreground'),
                },
                destructive: {
                    DEFAULT: hslVar('destructive'),
                    foreground: hslVar('destructive-foreground'),
                },
                muted: {
                    DEFAULT: hslVar('muted'),
                    foreground: hslVar('muted-foreground'),
                },
                accent: {
                    DEFAULT: hslVar('accent'),
                    foreground: hslVar('accent-foreground'),
                },
                popover: {
                    DEFAULT: hslVar('popover'),
                    foreground: hslVar('popover-foreground'),
                },
                card: {
                    DEFAULT: hslVar('card'),
                    foreground: hslVar('card-foreground'),
                },
                status: {
                    ok: statusColor('ok'),
                    warn: statusColor('warn'),
                    err: statusColor('err'),
                    info: statusColor('info'),
                    idle: statusColor('idle'),
                },
                tint: {
                    blue: tintColor('blue'),
                    red: tintColor('red'),
                    purple: tintColor('purple'),
                    navy: tintColor('navy'),
                    teal: tintColor('teal'),
                    green: tintColor('green'),
                },
                terminal: {
                    DEFAULT: hslVar('terminal'),
                    foreground: hslVar('terminal-foreground'),
                    muted: hslVar('terminal-muted'),
                    warn: hslVar('terminal-warn'),
                    error: hslVar('terminal-error'),
                },
            },
            // Entrance and exit motion for overlays. Radix keeps an element mounted until its exit animation ends,
            // so every open state needs a matching closed one. The individual `scale` and `translate` properties
            // are used so the dialog's own translate utilities that center it are not overwritten.
            keyframes: {
                'overlay-in': { from: { opacity: '0' }, to: { opacity: '1' } },
                'overlay-out': { from: { opacity: '1' }, to: { opacity: '0' } },
                'dialog-in': { from: { opacity: '0', scale: '0.96' }, to: { opacity: '1', scale: '1' } },
                'dialog-out': { from: { opacity: '1', scale: '1' }, to: { opacity: '0', scale: '0.96' } },
                'popover-in': { from: { opacity: '0', scale: '0.97' }, to: { opacity: '1', scale: '1' } },
                'popover-out': { from: { opacity: '1', scale: '1' }, to: { opacity: '0', scale: '0.97' } },
                'sheet-bottom-in': { from: { translate: '0 100%' }, to: { translate: '0 0' } },
                'sheet-bottom-out': { from: { translate: '0 0' }, to: { translate: '0 100%' } },
                'sheet-right-in': { from: { translate: '100% 0' }, to: { translate: '0 0' } },
                'sheet-right-out': { from: { translate: '0 0' }, to: { translate: '100% 0' } },
            },
            animation: {
                'overlay-in': 'overlay-in 150ms ease-out',
                'overlay-out': 'overlay-out 120ms ease-in forwards',
                'dialog-in': 'dialog-in 180ms cubic-bezier(0.16, 1, 0.3, 1)',
                'dialog-out': 'dialog-out 120ms ease-in forwards',
                'popover-in': 'popover-in 120ms ease-out',
                'popover-out': 'popover-out 90ms ease-in forwards',
                'sheet-bottom-in': 'sheet-bottom-in 260ms cubic-bezier(0.32, 0.72, 0, 1)',
                'sheet-bottom-out': 'sheet-bottom-out 200ms ease-in forwards',
                'sheet-right-in': 'sheet-right-in 260ms cubic-bezier(0.32, 0.72, 0, 1)',
                'sheet-right-out': 'sheet-right-out 200ms ease-in forwards',
            },
            borderRadius: {
                lg: 'var(--radius)',
                md: 'calc(var(--radius) - 2px)',
                sm: 'calc(var(--radius) - 4px)',
            },
        },
    },
    plugins: [],
}
