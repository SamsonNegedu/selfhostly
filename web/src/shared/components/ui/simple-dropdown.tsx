import * as React from "react"
import { ChevronDown } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { Button } from "./button"

interface SimpleDropdownProps {
  children: React.ReactNode
  trigger: React.ReactNode
  className?: string
}

interface SimpleDropdownContextType {
  isOpen: boolean
  setIsOpen: React.Dispatch<React.SetStateAction<boolean>>
}

const SimpleDropdownContext = React.createContext<SimpleDropdownContextType | null>(null)

function SimpleDropdownProvider({ children, isOpen, setIsOpen }: {
  children: React.ReactNode
  isOpen: boolean
  setIsOpen: React.Dispatch<React.SetStateAction<boolean>>
}) {
  return (
    <SimpleDropdownContext.Provider value={{ isOpen, setIsOpen }}>
      {children}
    </SimpleDropdownContext.Provider>
  )
}

function SimpleDropdown({ children, trigger, className }: SimpleDropdownProps) {
  const [isOpen, setIsOpen] = React.useState(false)

  const toggleDropdown = () => setIsOpen(!isOpen)

  // Close dropdown when clicking outside
  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (isOpen && !event.composedPath().includes(document.getElementById('simple-dropdown-trigger') as Node)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [isOpen])

  return (
    <SimpleDropdownProvider isOpen={isOpen} setIsOpen={setIsOpen}>
      <div className="relative" id="simple-dropdown-trigger">
        <div onClick={toggleDropdown} className="cursor-pointer">
          {trigger}
        </div>
        
        {isOpen && (
          <div
            className={cn(
              "absolute right-0 z-50 mt-2 w-48 origin-top-right rounded-md border bg-white shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none",
              "animate-in fade-in-0 zoom-in-95",
              className
            )}
          >
            {children}
          </div>
        )}
      </div>
    </SimpleDropdownProvider>
  )
}

interface SimpleDropdownItemProps {
  children: React.ReactNode
  onClick?: () => void
  className?: string
  href?: string
}

function SimpleDropdownItem({ children, onClick, className, href }: SimpleDropdownItemProps) {
  const context = React.useContext(SimpleDropdownContext)
  const setIsOpen = context?.setIsOpen

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault()
    if (onClick) {
      onClick()
    }
    // Close dropdown when item is clicked
    if (setIsOpen) {
      setIsOpen(false)
    }
  }

  const content = (
    <div
      className={cn(
        "block px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 hover:text-gray-900",
        "cursor-pointer transition-colors duration-150 ease-in-out",
        className
      )}
    >
      {children}
    </div>
  )

  if (href) {
    return (
      <a href={href} onClick={handleClick} className="block">
        {content}
      </a>
    )
  }

  return content
}

export { SimpleDropdown, SimpleDropdownItem }
export default SimpleDropdown
