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
  darkMode: ["class"],
  content: [
    './src/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      screens: {
        'xs': '475px',
      },
      fontFamily: {
        sans: ['"IBM Plex Sans"', ...defaultTheme.fontFamily.sans],
        mono: ['"IBM Plex Mono"', ...defaultTheme.fontFamily.mono],
      },
      colors: {
        border: hslVar('border'),
        input: hslVar('input'),
        ring: hslVar('ring'),
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
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
    },
  },
  plugins: [],
}
