import * as React from "react"
import { ChevronDown } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { Button } from "./button"

interface DropdownProps {
  trigger: React.ReactNode
  children: React.ReactNode
  className?: string
}

export function Dropdown({ trigger, children, className }: DropdownProps) {
  const [isOpen, setIsOpen] = React.useState(false)

  const toggleDropdown = () => setIsOpen(!isOpen)

  // Close dropdown when clicking outside
  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (isOpen && !event.composedPath().includes(document.getElementById('dropdown-trigger') as Node)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [isOpen])

  return (
    <div className="relative" id="dropdown-trigger">
      <div onClick={toggleDropdown}>
        {trigger}
      </div>
      
      {isOpen && (
        <div
          className={cn(
            "absolute right-0 z-50 mt-2 w-48 rounded-md border bg-popover p-1 text-popover-foreground shadow-md",
            "animate-in fade-in-0 zoom-in-95",
            className
          )}
        >
          {children}
        </div>
      )}
    </div>
  )
}

interface DropdownItemProps {
  children: React.ReactNode
  onClick?: () => void
  className?: string
  asChild?: boolean
}

export function DropdownItem({ children, onClick, className, asChild }: DropdownItemProps) {
  const context = React.useContext(DropdownContext)
  const setIsOpen = context?.setIsOpen || (() => {})

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault()
    if (onClick) {
      onClick()
    }
    // Close dropdown when item is clicked
    setIsOpen(false)
  }

  return (
    <div
      className={cn(
        "relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors",
        "focus:bg-accent focus:text-accent-foreground",
        "hover:bg-accent hover:text-accent-foreground",
        "data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
        className
      )}
      onClick={handleClick}
    >
      {children}
    </div>
  )
}

// Create a context to share state between dropdown components
const DropdownContext = React.createContext<{
  isOpen: boolean
  setIsOpen: React.Dispatch<React.SetStateAction<boolean>>
} | null>(null)

// Add a context provider to share state
function DropdownProvider({ children, isOpen, setIsOpen }: {
  children: React.ReactNode
  isOpen: boolean
  setIsOpen: React.Dispatch<React.SetStateAction<boolean>>
}) {
  return (
    <DropdownContext.Provider value={{ isOpen, setIsOpen }}>
      {children}
    </DropdownContext.Provider>
  )
}

// Update the Dropdown component to use the context
export { DropdownProvider }
