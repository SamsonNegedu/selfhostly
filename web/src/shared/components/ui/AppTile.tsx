import { cn } from "@/shared/lib/utils"

const TINTS = ["blue", "red", "purple", "navy", "teal", "green"] as const
type Tint = (typeof TINTS)[number]

// Full class names so Tailwind can see them at build time.
const TINT_CLASSES: Record<Tint, string> = {
  blue: "bg-tint-blue-bg text-tint-blue-fg",
  red: "bg-tint-red-bg text-tint-red-fg",
  purple: "bg-tint-purple-bg text-tint-purple-fg",
  navy: "bg-tint-navy-bg text-tint-navy-fg",
  teal: "bg-tint-teal-bg text-tint-teal-fg",
  green: "bg-tint-green-bg text-tint-green-fg",
}

const SIZE_CLASSES = {
  sm: "h-[30px] w-[30px] text-[13px]",
  md: "h-[38px] w-[38px] text-base",
  lg: "h-[52px] w-[52px] text-xl",
} as const

const HASH_MULTIPLIER = 31

// The same name always lands on the same tint, so an app keeps its color everywhere it appears.
function tintFor(name: string): Tint {
  let hash = 0
  for (const char of name) {
    hash = (hash * HASH_MULTIPLIER + char.charCodeAt(0)) % TINTS.length
  }
  return TINTS[hash]
}

interface AppTileProps extends React.HTMLAttributes<HTMLDivElement> {
  name: string
  tint?: Tint
  size?: keyof typeof SIZE_CLASSES
}

function AppTile({ name, tint, size = "md", className, ...props }: AppTileProps) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        "flex shrink-0 items-center justify-center rounded-[9px] font-semibold",
        TINT_CLASSES[tint ?? tintFor(name)],
        SIZE_CLASSES[size],
        className
      )}
      {...props}
    >
      {name.charAt(0).toUpperCase()}
    </div>
  )
}

export { AppTile }
