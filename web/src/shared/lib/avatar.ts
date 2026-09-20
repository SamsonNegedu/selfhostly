/**
 * Gets initials from a name (e.g., "John Doe" -> "JD")
 */
export function getInitials(name: string): string {
    return name
        .split(' ')
        .filter(Boolean)
        .map((part) => part.charAt(0).toUpperCase())
        .slice(0, 2)
        .join('')
}
