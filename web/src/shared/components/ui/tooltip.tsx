import * as React from "react"
import { cn } from "@/shared/lib/utils"

const TooltipProvider = ({ children }: { children: React.ReactNode }) => (
    <div className="group relative inline-block">{children}</div>
)

const TooltipTrigger = ({ children, asChild = false }: { children: React.ReactNode; asChild?: boolean }) => {
    if (asChild) {
        return <>{children}</>
    }
    return <>{children}</>
}

const TooltipContent = ({
    children,
    className,
}: { children: React.ReactNode; className?: string }) => (
    <div
        className={cn(
            "absolute z-50 hidden group-hover:block bg-popover text-popover-foreground border shadow-lg text-xs rounded py-1 px-2 -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap",
            className
        )}
    >
        {children}
    </div>
)

const Tooltip = ({ children }: { children: React.ReactNode }) => {
    return <>{children}</>
}

export default Tooltip
export { TooltipProvider, TooltipTrigger, TooltipContent }
