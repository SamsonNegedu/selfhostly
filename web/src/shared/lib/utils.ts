import { type ClassValue, clsx } from 'clsx'
import { extendTailwindMerge } from 'tailwind-merge'

// The custom type scale in tailwind.config.js. Without this, twMerge reads `text-compact` as a text color
// and drops it or a real color class when the two meet.
const FONT_SIZES = ['caption', 'detail', 'compact', 'body', 'title', 'heading']

const twMerge = extendTailwindMerge({
    extend: { classGroups: { 'font-size': [{ text: FONT_SIZES }] } },
})

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs))
}
