import { createContext, useContext, useState, ReactNode } from 'react'
import { useCurrentNode, useNodes } from '../services/api'

interface NodeContextType {
    selectedNodeIds: string[]
    setSelectedNodeIds: (nodeIds: string[]) => void
    resetToDefault: () => void
    currentNodeId: string | null
}

const NodeContext = createContext<NodeContextType | undefined>(undefined)

const STORAGE_KEY = 'selfhostly_selected_nodes'

interface NodeContextProviderProps {
    children: ReactNode
}

// Validate stored node IDs against available nodes to filter out deleted nodes. Anything unusable falls back to the
// current node.
function readInitialSelection(currentNodeId: string, availableNodeIds: string[]): string[] {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (!stored) return [currentNodeId]
    try {
        const parsed = JSON.parse(stored)
        if (!Array.isArray(parsed)) return [currentNodeId]
        const validNodeIds = parsed.filter((id: string) => availableNodeIds.includes(id))
        return validNodeIds.length > 0 ? validNodeIds : [currentNodeId]
    } catch {
        return [currentNodeId]
    }
}

export function NodeContextProvider({ children }: NodeContextProviderProps) {
    const { data: currentNode } = useCurrentNode()
    const { data: nodes } = useNodes()
    const [selectedNodeIds, setSelectedNodeIdsState] = useState<string[]>([])
    const [isInitialized, setIsInitialized] = useState(false)

    // Initialize from localStorage or default to current node, once the current node is known
    if (currentNode && !isInitialized) {
        setSelectedNodeIdsState(readInitialSelection(currentNode.id, nodes?.map((n) => n.id) || []))
        setIsInitialized(true)
    }

    // Save to localStorage whenever selection changes
    const setSelectedNodeIds = (nodeIds: string[]) => {
        setSelectedNodeIdsState(nodeIds)
        localStorage.setItem(STORAGE_KEY, JSON.stringify(nodeIds))
    }

    // Reset to current node only
    const resetToDefault = () => {
        if (currentNode) {
            setSelectedNodeIds([currentNode.id])
        }
    }

    const value: NodeContextType = {
        selectedNodeIds,
        setSelectedNodeIds,
        resetToDefault,
        currentNodeId: currentNode?.id || null,
    }

    return <NodeContext.Provider value={value}>{children}</NodeContext.Provider>
}

export function useNodeContext(): NodeContextType {
    const context = useContext(NodeContext)
    if (context === undefined) {
        throw new Error('useNodeContext must be used within a NodeContextProvider')
    }
    return context
}
