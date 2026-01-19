import * as React from "react"
import { ChevronDown } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { Button } from "./button"

interface DropdownMenuProps {
  children: React.ReactNode
}

interface DropdownMenuTriggerProps {
  children: React.ReactNode
  asChild?: boolean
}

interface DropdownMenuContentProps {
  children: React.ReactNode
  className?: string
  align?: "start" | "center" | "end"
}

interface DropdownMenuItemProps {
  children: React.ReactNode
  asChild?: boolean
  className?: string
}

const DropdownMenu: React.FC<DropdownMenuProps> = ({ children }) => {
  const [isOpen, setIsOpen] = React.useState(false)

  const toggleOpen = () => setIsOpen(!isOpen)

  return (
    <div className="relative">
      {React.Children.map(children, (child, index) => {
        if (index === 0) {
          return React.cloneElement(child as React.ReactElement, { onClick: toggleOpen })
        }
        return isOpen && child
      })}
    </div>
  )
}

const DropdownMenuTrigger: React.FC<DropdownMenuTriggerProps> = ({ children, asChild }) => {
  return <>{children}</>
}

const DropdownMenuContent: React.FC<DropdownMenuContentProps> = ({ children, className, align = "end" }) => {
  return (
    <div
      className={cn(
        "absolute z-50 min-w-[8rem] overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-md",
        "animate-in fade-in-0 zoom-in-95",
        "data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95",
        align === "start" && "origin-top-left left-0",
        align === "center" && "origin-top",
        align === "end" && "origin-top-right right-0",
        className
      )}
      style={{ top: '100%', marginTop: '4px' }}
    >
      {children}
    </div>
  )
}

const DropdownMenuItem: React.FC<DropdownMenuItemProps> = ({ children, className }) => {
  return (
    <div
      className={cn(
        "relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors",
        "focus:bg-accent focus:text-accent-foreground",
        "data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
        className
      )}
    >
      {children}
    </div>
  )
}


export {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
}
