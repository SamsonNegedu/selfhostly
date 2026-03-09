import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Format a future date/time as a relative time string (short format)
 * Examples: "now", "in 5m", "in 2h", "in 3d", "Mar 9"
 */
export function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = date.getTime() - now.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffMins < 1) return 'now'
  if (diffMins < 60) return `in ${diffMins}m`
  if (diffHours < 24) return `in ${diffHours}h`
  if (diffDays < 7) return `in ${diffDays}d`
  return date.toLocaleDateString()
}

/**
 * Format a future date/time as a relative time string (detailed format)
 * Examples: "now", "in 5 minutes", "in 2 hours", "in 3 days", "Mar 9, 7:00 PM"
 */
export function formatRelativeTimeDetailed(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = date.getTime() - now.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffMins < 1) return 'now'
  if (diffMins < 60) return `in ${diffMins} minute${diffMins !== 1 ? 's' : ''}`
  if (diffHours < 24) return `in ${diffHours} hour${diffHours !== 1 ? 's' : ''}`
  if (diffDays < 7) return `in ${diffDays} day${diffDays !== 1 ? 's' : ''}`
  return date.toLocaleString('en-US', { 
    month: 'short', 
    day: 'numeric', 
    hour: 'numeric', 
    minute: '2-digit',
    hour12: true 
  })
}
