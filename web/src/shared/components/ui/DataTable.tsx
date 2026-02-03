import React, { useState, useRef, useEffect, ReactNode, useMemo } from 'react'
import { Card } from './Card'
import { Button } from './Button'
import { MoreVertical, ChevronDown, ChevronUp, Loader2, ArrowUp, ArrowDown } from 'lucide-react'

export interface ColumnDef<T> {
    key: string
    label: string
    width?: string
    render: (item: T) => ReactNode
    sortable?: boolean
    sortValue?: (item: T) => string | number | Date
}

export interface RowAction<T> {
    label: string | ((item: T) => string)
    icon?: ReactNode | ((item: T) => ReactNode)
    onClick: (item: T) => void
    variant?: 'default' | 'destructive'
    show?: (item: T) => boolean
    loading?: (item: T) => boolean
    disabled?: (item: T) => boolean
}

export interface DataTableProps<T> {
    data: T[]
    columns: ColumnDef<T>[]
    getRowKey: (item: T) => string
    actions?: RowAction<T>[]
    expandableContent?: (item: T) => ReactNode
    onRowClick?: (item: T) => void
    emptyState?: ReactNode
    isLoading?: boolean
    selectable?: boolean
    onSelectionChange?: (selectedIds: Set<string>) => void
}

type SortDirection = 'asc' | 'desc' | null

export function DataTable<T>({
    data,
    columns,
    getRowKey,
    actions = [],
    expandableContent,
    onRowClick,
    emptyState,
    isLoading = false,
    selectable = false,
    onSelectionChange,
}: DataTableProps<T>) {
    const [expandedRowId, setExpandedRowId] = useState<string | null>(null)
    const [dropdownOpenId, setDropdownOpenId] = useState<string | null>(null)
    const [dropdownPosition, setDropdownPosition] = useState<{ top: number; left: number } | null>(null)
    const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set())
    const [loadingDropdownIds, setLoadingDropdownIds] = useState<Set<string>>(new Set())
    const [sortColumn, setSortColumn] = useState<string | null>(null)
    const [sortDirection, setSortDirection] = useState<SortDirection>(null)

    const dropdownTriggerRefs = useRef<Map<string, HTMLButtonElement>>(new Map())

    // Handle row selection
    const handleRowSelect = (rowKey: string, e: React.MouseEvent) => {
        e.stopPropagation()
        setSelectedRows(prev => {
            const newSet = new Set(prev)
            if (newSet.has(rowKey)) {
                newSet.delete(rowKey)
            } else {
                newSet.add(rowKey)
            }
            onSelectionChange?.(newSet)
            return newSet
        })
    }

    // Handle row click
    const handleRowClick = (item: T, e: React.MouseEvent) => {
        const target = e.target as HTMLElement
        const isAction = target.closest('button') || target.closest('[data-dropdown-id]') || target.closest('input[type="checkbox"]')

        if (!isAction) {
            onRowClick?.(item)
        }
    }

    // Toggle expand
    const toggleExpand = (rowKey: string) => {
        setExpandedRowId(prev => prev === rowKey ? null : rowKey)
    }

    // Handle dropdown toggle
    const handleDropdownToggle = (rowKey: string, e: React.MouseEvent) => {
        e.stopPropagation()

        if (dropdownOpenId === rowKey) {
            setDropdownOpenId(null)
            setDropdownPosition(null)
        } else {
            setDropdownOpenId(null)
            setDropdownPosition(null)

            setTimeout(() => {
                setDropdownOpenId(rowKey)
                const trigger = dropdownTriggerRefs.current.get(rowKey)
                if (trigger) {
                    const rect = trigger.getBoundingClientRect()
                    setDropdownPosition({
                        top: rect.bottom + 4,
                        left: rect.right - 160
                    })
                }
            }, 0)
        }
    }

    // Helper to check if any action is loading for a specific item
    const hasLoadingAction = (item: T, itemActions: RowAction<T>[]) => {
        return itemActions.some(action => action.loading && action.loading(item))
    }

    // Close dropdown when clicking outside (but not if any action is loading)
    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (dropdownOpenId) {
                const trigger = dropdownTriggerRefs.current.get(dropdownOpenId)
                const dropdown = document.querySelector(`[data-dropdown-id="${dropdownOpenId}"]`)

                if (trigger?.contains(event.target as Node) || dropdown?.contains(event.target as Node)) {
                    return
                }

                // Find the item for this dropdown and check if any action is loading
                const item = data.find(item => getRowKey(item) === dropdownOpenId)
                if (item) {
                    const itemActions = getVisibleActions(item)
                    if (hasLoadingAction(item, itemActions)) {
                        // Don't close if any action is loading
                        return
                    }
                }

                setDropdownOpenId(null)
                setDropdownPosition(null)
            }
        }

        const handleScroll = () => {
            if (dropdownOpenId) {
                // Find the item for this dropdown and check if any action is loading
                const item = data.find(item => getRowKey(item) === dropdownOpenId)
                if (item) {
                    const itemActions = getVisibleActions(item)
                    if (hasLoadingAction(item, itemActions)) {
                        // Don't close if any action is loading
                        return
                    }
                }
                setDropdownOpenId(null)
                setDropdownPosition(null)
            }
        }

        if (dropdownOpenId) {
            document.addEventListener('mousedown', handleClickOutside)
            window.addEventListener('scroll', handleScroll, true)
            const scrollContainer = document.querySelector('.overflow-x-auto')
            if (scrollContainer) {
                scrollContainer.addEventListener('scroll', handleScroll)
            }
            return () => {
                document.removeEventListener('mousedown', handleClickOutside)
                window.removeEventListener('scroll', handleScroll, true)
                if (scrollContainer) {
                    scrollContainer.removeEventListener('scroll', handleScroll)
                }
            }
        }
    }, [dropdownOpenId, data])

    // Close dropdown when loading finishes - but only if it was open and loading started
    useEffect(() => {
        if (dropdownOpenId && loadingDropdownIds.has(dropdownOpenId)) {
            const item = data.find(item => getRowKey(item) === dropdownOpenId)
            if (item) {
                const itemActions = getVisibleActions(item)
                const isLoading = hasLoadingAction(item, itemActions)
                
                // Only auto-close if this dropdown had a loading action that just finished
                if (!isLoading) {
                    // Loading just finished, close after a brief delay
                    const timer = setTimeout(() => {
                        setDropdownOpenId(null)
                        setDropdownPosition(null)
                        setLoadingDropdownIds(prev => {
                            const next = new Set(prev)
                            next.delete(dropdownOpenId)
                            return next
                        })
                    }, 300)
                    return () => clearTimeout(timer)
                }
            }
        }
    }, [dropdownOpenId, data, actions, loadingDropdownIds])

    // Filter visible actions for a row
    const getVisibleActions = (item: T) => {
        return actions.filter(action => action.show === undefined || action.show(item))
    }

    // Handle column header click for sorting
    const handleSort = (columnKey: string) => {
        const column = columns.find(col => col.key === columnKey)
        if (!column?.sortable || !column?.sortValue) return

        if (sortColumn === columnKey) {
            // Toggle direction: asc -> desc -> null
            if (sortDirection === 'asc') {
                setSortDirection('desc')
            } else if (sortDirection === 'desc') {
                setSortDirection(null)
                setSortColumn(null)
            }
        } else {
            // New column, start with ascending
            setSortColumn(columnKey)
            setSortDirection('asc')
        }
    }

    // Sort data based on current sort state
    const sortedData = useMemo(() => {
        if (!sortColumn || !sortDirection) {
            return data
        }

        const column = columns.find(col => col.key === sortColumn)
        if (!column?.sortValue) {
            return data
        }

        const sorted = [...data].sort((a, b) => {
            const aValue = column.sortValue!(a)
            const bValue = column.sortValue!(b)

            // Handle different types
            if (aValue === null || aValue === undefined) return 1
            if (bValue === null || bValue === undefined) return -1

            let comparison = 0
            if (aValue < bValue) {
                comparison = -1
            } else if (aValue > bValue) {
                comparison = 1
            }

            return sortDirection === 'asc' ? comparison : -comparison
        })

        return sorted
    }, [data, sortColumn, sortDirection, columns])

    if (isLoading) {
        return (
            <Card className="p-12 text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
            </Card>
        )
    }

    if (data.length === 0 && emptyState) {
        return <div>{emptyState}</div>
    }

    return (
        <Card className="overflow-hidden">
            <div className="overflow-x-auto">
                <table className="w-full">
                    <thead className="bg-muted/50 border-b">
                        <tr>
                            {selectable && (
                                <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider w-10">
                                    <input
                                        type="checkbox"
                                        checked={selectedRows.size === data.length && data.length > 0}
                                        onChange={(e) => {
                                            if (e.target.checked) {
                                                const allKeys = new Set(data.map(getRowKey))
                                                setSelectedRows(allKeys)
                                                onSelectionChange?.(allKeys)
                                            } else {
                                                setSelectedRows(new Set())
                                                onSelectionChange?.(new Set())
                                            }
                                        }}
                                        className="h-4 w-4 rounded border-input bg-background cursor-pointer"
                                    />
                                </th>
                            )}
                            {columns.map((col) => {
                                const isSortable = col.sortable && col.sortValue
                                const isSorted = sortColumn === col.key
                                const isAsc = sortDirection === 'asc'
                                const isDesc = sortDirection === 'desc'

                                return (
                                    <th
                                        key={col.key}
                                        className={`px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider ${col.width || ''} ${
                                            isSortable ? 'cursor-pointer hover:bg-muted/50 transition-colors select-none' : ''
                                        }`}
                                        onClick={() => isSortable && handleSort(col.key)}
                                    >
                                        <div className="flex items-center gap-2">
                                            <span>{col.label}</span>
                                            {isSortable && (
                                                <div className="flex flex-col">
                                                    <ArrowUp
                                                        className={`h-3 w-3 transition-opacity ${
                                                            isSorted && isAsc ? 'opacity-100 text-primary' : 'opacity-30'
                                                        }`}
                                                    />
                                                    <ArrowDown
                                                        className={`h-3 w-3 -mt-1 transition-opacity ${
                                                            isSorted && isDesc ? 'opacity-100 text-primary' : 'opacity-30'
                                                        }`}
                                                    />
                                                </div>
                                            )}
                                        </div>
                                    </th>
                                )
                            })}
                            {(actions.length > 0 || expandableContent) && (
                                <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider w-10">
                                    Actions
                                </th>
                            )}
                        </tr>
                    </thead>
                    <tbody className="divide-y">
                        {sortedData.map((item) => {
                            const rowKey = getRowKey(item)
                            const isExpanded = expandedRowId === rowKey
                            const isSelected = selectedRows.has(rowKey)
                            const visibleActions = getVisibleActions(item)

                            return (
                                <React.Fragment key={rowKey}>
                                    <tr
                                        className={`group hover:bg-muted/30 transition-colors ${onRowClick ? 'cursor-pointer' : ''} ${isSelected ? 'bg-primary/5' : ''}`}
                                        onClick={(e) => handleRowClick(item, e)}
                                    >
                                        {selectable && (
                                            <td className="px-4 py-4" onClick={(e) => e.stopPropagation()}>
                                                <input
                                                    type="checkbox"
                                                    checked={isSelected}
                                                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => handleRowSelect(rowKey, e as unknown as React.MouseEvent<Element, MouseEvent>)}
                                                    className="h-4 w-4 rounded border-input bg-background cursor-pointer"
                                                />
                                            </td>
                                        )}
                                        {columns.map((col) => (
                                            <td key={col.key} className="px-4 py-4 min-w-0">
                                                {col.render(item)}
                                            </td>
                                        ))}
                                        {(actions.length > 0 || expandableContent) && (
                                            <td className="px-4 py-4">
                                                <div className="flex items-center justify-end gap-1">
                                                    {expandableContent && (
                                                        <Button
                                                            variant="ghost"
                                                            size="icon"
                                                            className="h-8 w-8"
                                                            onClick={(e) => {
                                                                e.stopPropagation()
                                                                toggleExpand(rowKey)
                                                            }}
                                                        >
                                                            {isExpanded ? (
                                                                <ChevronUp className="h-4 w-4" />
                                                            ) : (
                                                                <ChevronDown className="h-4 w-4" />
                                                            )}
                                                        </Button>
                                                    )}

                                                    {visibleActions.length > 0 && (
                                                        <div className="relative">
                                                            {(() => {
                                                                const hasLoadingAction = visibleActions.some(action => action.loading && action.loading(item))
                                                                return (
                                                                    <Button
                                                                        ref={(el) => {
                                                                            if (el) dropdownTriggerRefs.current.set(rowKey, el)
                                                                        }}
                                                                        variant="ghost"
                                                                        size="icon"
                                                                        className={`h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity ${hasLoadingAction ? 'opacity-100' : ''}`}
                                                                        onClick={(e) => handleDropdownToggle(rowKey, e)}
                                                                    >
                                                                        {hasLoadingAction ? (
                                                                            <Loader2 className="h-4 w-4 animate-spin" />
                                                                        ) : (
                                                                            <MoreVertical className="h-4 w-4" />
                                                                        )}
                                                                    </Button>
                                                                )
                                                            })()}

                                                            {dropdownOpenId === rowKey && dropdownPosition && (
                                                                <div
                                                                    data-dropdown-id={rowKey}
                                                                    className="fixed z-[100] min-w-[160px] bg-popover border border-border rounded-md shadow-lg ring-1 ring-black ring-opacity-5 dark:ring-white/10 py-1 animate-in fade-in-0 zoom-in-95"
                                                                    style={{
                                                                        top: `${dropdownPosition.top}px`,
                                                                        left: `${dropdownPosition.left}px`,
                                                                    }}
                                                                >
                                                                    {visibleActions.map((action, idx) => {
                                                                        const isLoading = action.loading ? action.loading(item) : false
                                                                        const isDisabled = action.disabled ? action.disabled(item) : false
                                                                        const actionLabel = typeof action.label === 'function' ? action.label(item) : action.label
                                                                        const actionIcon = typeof action.icon === 'function' ? action.icon(item) : action.icon
                                                                        
                                                                        return (
                                                                            <div
                                                                                key={idx}
                                                                                className={`px-4 py-2 text-sm transition-colors duration-150 ease-in-out ${
                                                                                    isLoading || isDisabled
                                                                                        ? 'opacity-50 cursor-not-allowed'
                                                                                        : 'hover:bg-accent cursor-pointer'
                                                                                } ${action.variant === 'destructive'
                                                                                    ? 'text-destructive hover:text-destructive'
                                                                                    : 'text-foreground hover:text-accent-foreground'
                                                                                }`}
                                                                                onClick={(e) => {
                                                                                    if (isLoading || isDisabled) {
                                                                                        e.stopPropagation()
                                                                                        return
                                                                                    }
                                                                                    e.stopPropagation()
                                                                                    
                                                                                    // Mark this dropdown as having triggered a loading action
                                                                                    // (if the action has a loading function, it means it's async)
                                                                                    if (action.loading) {
                                                                                        setLoadingDropdownIds(prev => new Set(prev).add(rowKey))
                                                                                    }
                                                                                    
                                                                                    action.onClick(item)
                                                                                    // Don't close dropdown immediately - let it stay open to show loading state
                                                                                    // The dropdown will close automatically when loading finishes via useEffect
                                                                                }}
                                                                            >
                                                                                <div className="flex items-center">
                                                                                    {isLoading ? (
                                                                                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                                                                                    ) : actionIcon ? (
                                                                                        <span className="mr-2">{actionIcon}</span>
                                                                                    ) : null}
                                                                                    <span>{actionLabel}</span>
                                                                                </div>
                                                                            </div>
                                                                        )
                                                                    })}
                                                                </div>
                                                            )}
                                                        </div>
                                                    )}
                                                </div>
                                            </td>
                                        )}
                                    </tr>

                                    {isExpanded && expandableContent && (
                                        <tr className="bg-muted/30">
                                            <td colSpan={columns.length + (selectable ? 1 : 0) + 1} className="px-4 py-4">
                                                {expandableContent(item)}
                                            </td>
                                        </tr>
                                    )}
                                </React.Fragment>
                            )
                        })}
                    </tbody>
                </table>
            </div>
        </Card>
    )
}

export default DataTable
