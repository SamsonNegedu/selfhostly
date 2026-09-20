import * as React from "react"

import { cn } from "@/shared/lib/utils"

type LogLevel = "info" | "warn" | "error"

interface TerminalLine {
  time?: string
  level?: LogLevel
  text: string
}

const LEVEL_CLASSES: Record<LogLevel, string> = {
  info: "text-terminal-muted",
  warn: "text-terminal-warn",
  error: "text-terminal-error",
}

const LEVEL_TEXT_CLASSES: Record<LogLevel, string> = {
  info: "",
  warn: "",
  error: "text-terminal-error",
}

const LEVEL_TAG_WIDTH = 5
const NEAR_BOTTOM_PX = 24

interface TerminalProps {
  lines: TerminalLine[]
  "aria-label": string
  className?: string
  // Keeps the newest line in view as lines arrive, unless the reader has scrolled up to read.
  follow?: boolean
}

// Streaming output such as logs. It stays dark in both themes, like a real terminal, and is
// focusable so keyboard users can scroll it. It does not announce new lines to screen readers.
function Terminal({ lines, className, follow = false, ...props }: TerminalProps) {
  const ref = React.useRef<HTMLDivElement>(null)
  const pinned = React.useRef(true)

  React.useEffect(() => {
    const element = ref.current
    if (follow && element && pinned.current) element.scrollTop = element.scrollHeight
  }, [lines, follow])

  return (
    <div
      ref={ref}
      onScroll={(event) => {
        const element = event.currentTarget
        pinned.current = element.scrollHeight - element.scrollTop - element.clientHeight < NEAR_BOTTOM_PX
      }}
      role="log"
      aria-live="off"
      tabIndex={0}
      aria-label={props["aria-label"]}
      className={cn(
        "overflow-auto rounded-[10px] bg-terminal px-4 py-3.5 font-mono text-[12px] leading-5 text-terminal-foreground ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 sm:text-[12.5px]",
        className
      )}
    >
      {lines.map((line, index) => (
        <div key={index} className="whitespace-pre-wrap break-words">
          {line.time && <span className="text-terminal-muted">{line.time}  </span>}
          {line.level && (
            <span className={LEVEL_CLASSES[line.level]}>{line.level.toUpperCase().padEnd(LEVEL_TAG_WIDTH)} </span>
          )}
          <span className={line.level ? LEVEL_TEXT_CLASSES[line.level] : undefined}>{line.text}</span>
        </div>
      ))}
    </div>
  )
}

export { Terminal }
export type { TerminalLine }
