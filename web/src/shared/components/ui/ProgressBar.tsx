import * as React from "react"

import { cn } from "@/shared/lib/utils"
import type { StatusKind } from "@/shared/lib/status"

const FILL_CLASSES: Record<StatusKind, string> = {
  ok: "bg-status-ok",
  warn: "bg-status-warn",
  err: "bg-status-err",
  info: "bg-status-info",
  idle: "bg-status-idle",
}

const MAX_PERCENT = 100
// A bar with any value at all shows a little fill, so a low reading is still visibly colored.
const MIN_VISIBLE_PERCENT = 4

interface ProgressBarProps extends Omit<React.HTMLAttributes<HTMLDivElement>, "children"> {
  value: number
  tone?: StatusKind
  "aria-label": string
}

function ProgressBar({ value, tone = "info", className, ...props }: ProgressBarProps) {
  const clamped = Math.min(MAX_PERCENT, Math.max(0, Math.round(value)))

  return (
    <div
      role="progressbar"
      aria-valuemin={0}
      aria-valuemax={MAX_PERCENT}
      aria-valuenow={clamped}
      className={cn("h-1.5 w-full overflow-hidden rounded-full bg-secondary", className)}
      {...props}
    >
      <div
        className={cn("h-full rounded-full transition-[width]", FILL_CLASSES[tone])}
        style={{ width: `${clamped > 0 ? Math.max(clamped, MIN_VISIBLE_PERCENT) : 0}%` }}
      />
    </div>
  )
}

export { ProgressBar }
