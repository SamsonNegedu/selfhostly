import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { useCurrentNode, useNodes } from '../services/api';

interface NodeContextType {
  selectedNodeIds: string[];
  setSelectedNodeIds: (nodeIds: string[]) => void;
  resetToDefault: () => void;
  currentNodeId: string | null;
}

const NodeContext = createContext<NodeContextType | undefined>(undefined);

const STORAGE_KEY = 'selfhostly_selected_nodes';

interface NodeContextProviderProps {
  children: ReactNode;
}

export function NodeContextProvider({ children }: NodeContextProviderProps) {
  const { data: currentNode } = useCurrentNode();
  const { data: nodes } = useNodes();
  const [selectedNodeIds, setSelectedNodeIdsState] = useState<string[]>([]);
  const [isInitialized, setIsInitialized] = useState(false);

  // Initialize from localStorage or default to current node
  // Validate stored node IDs against available nodes to filter out deleted nodes
  useEffect(() => {
    if (currentNode && !isInitialized) {
      const stored = localStorage.getItem(STORAGE_KEY);
      const availableNodeIds = nodes?.map(n => n.id) || [];

      if (stored) {
        try {
          const parsed = JSON.parse(stored);
          if (Array.isArray(parsed) && parsed.length > 0) {
            // Filter out node IDs that no longer exist
            const validNodeIds = parsed.filter((id: string) => availableNodeIds.includes(id));
            if (validNodeIds.length > 0) {
              setSelectedNodeIdsState(validNodeIds);
              // Update localStorage if we filtered out invalid IDs
              if (validNodeIds.length !== parsed.length) {
                localStorage.setItem(STORAGE_KEY, JSON.stringify(validNodeIds));
              }
            } else {
              // All stored node IDs are invalid, use current node
              setSelectedNodeIdsState([currentNode.id]);
            }
          } else {
            // Invalid stored value, use current node
            setSelectedNodeIdsState([currentNode.id]);
          }
        } catch (e) {
          // Parse error, use current node
          setSelectedNodeIdsState([currentNode.id]);
        }
      } else {
        // No stored value, use current node
        setSelectedNodeIdsState([currentNode.id]);
      }

      setIsInitialized(true);
    }
  }, [currentNode, nodes, isInitialized]);

  // Save to localStorage whenever selection changes
  const setSelectedNodeIds = (nodeIds: string[]) => {
    setSelectedNodeIdsState(nodeIds);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(nodeIds));
  };

  // Reset to current node only
  const resetToDefault = () => {
    if (currentNode) {
      setSelectedNodeIds([currentNode.id]);
    }
  };

  const value: NodeContextType = {
    selectedNodeIds,
    setSelectedNodeIds,
    resetToDefault,
    currentNodeId: currentNode?.id || null,
  };

  return <NodeContext.Provider value={value}>{children}</NodeContext.Provider>;
}

export function useNodeContext(): NodeContextType {
  const context = useContext(NodeContext);
  if (context === undefined) {
    throw new Error('useNodeContext must be used within a NodeContextProvider');
  }
  return context;
}
