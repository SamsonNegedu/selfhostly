/**
 * Generates a color based on a string (username)
 */
export function stringToColor(str: string): string {
  let hash = 0;
  
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
  }
  
  const h = hash % 360;
  return `hsl(${h}, 70%, 50%)`;
}

/**
 * Gets initials from a name (e.g., "John Doe" -> "JD")
 */
export function getInitials(name: string): string {
  return name
    .split(' ')
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase())
    .slice(0, 2)
    .join('');
}

/**
 * Generates an SVG avatar with background color and initials
 */
export function generateAvatarSvg(name: string, size: number = 40): string {
  const initials = getInitials(name);
  const bgColor = stringToColor(name);
  
  return `
    <svg 
      width="${size}" 
      height="${size}" 
      viewBox="0 0 ${size} ${size}" 
      xmlns="http://www.w3.org/2000/svg"
    >
      <rect width="${size}" height="${size}" fill="${bgColor}" rx="${size * 0.25}"/>
      <text 
        x="${size / 2}" 
        y="${size / 2}" 
        font-family="system-ui, -apple-system, sans-serif" 
        font-size="${size * 0.4}" 
        text-anchor="middle" 
        dominant-baseline="middle" 
        fill="white" 
        font-weight="500"
      >
        ${initials}
      </text>
    </svg>
  `;
}

/**
 * Creates a data URL for an avatar
 */
export function createAvatarDataUrl(name: string, size: number = 40): string {
  const svg = generateAvatarSvg(name, size);
  const encoded = encodeURIComponent(svg.trim());
  return `data:image/svg+xml;utf8,${encoded}`;
}
