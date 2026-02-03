import { useState, useRef, useEffect } from 'react';
import { useNodes } from '../../services/api';
import { Badge } from './Badge';
import { Button } from './Button';
import { ChevronDown, Check, Server, Filter } from 'lucide-react';

interface NodeSelectorProps {
  selectedNodeIds: string[];
  onChange: (nodeIds: string[]) => void;
  multiSelect?: boolean;
  className?: string;
  collapsed?: boolean;
}

export function NodeSelector({
  selectedNodeIds,
  onChange,
  multiSelect = true,
  className = '',
  collapsed = false,
}: NodeSelectorProps) {
  const { data: nodes } = useNodes();
  const [isOpen, setIsOpen] = useState(false);
  const [dropdownPosition, setDropdownPosition] = useState<{ top: number; left: number; width: number } | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  // Calculate dropdown position when opening
  useEffect(() => {
    if (isOpen && buttonRef.current) {
      const rect = buttonRef.current.getBoundingClientRect();
      const navbarHeight = 72; // Approximate navbar height (adjust if needed)

      if (collapsed) {
        // Position to the right when collapsed
        // Ensure dropdown is below navbar
        const topPosition = Math.max(rect.top, navbarHeight);
        setDropdownPosition({
          top: topPosition,
          left: rect.right + 8,
          width: 256, // w-64
        });
      } else {
        // Position below when expanded
        const topPosition = Math.max(rect.bottom + 8, navbarHeight);
        setDropdownPosition({
          top: topPosition,
          left: rect.left,
          width: rect.width,
        });
      }
    }
  }, [isOpen, collapsed]);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as HTMLElement)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);

  const handleToggle = (nodeId: string) => {
    if (!multiSelect) {
      // Single select mode
      onChange([nodeId]);
      setIsOpen(false);
      return;
    }

    // Multi-select mode
    if (nodeId === 'all') {
      // Toggle "All Nodes"
      if (selectedNodeIds.includes('all')) {
        onChange([]);
      } else {
        onChange(['all']);
      }
    } else {
      // Toggle specific node
      if (selectedNodeIds.includes(nodeId)) {
        // Remove if already selected
        const newSelection = selectedNodeIds.filter(id => id !== nodeId && id !== 'all');
        onChange(newSelection);
      } else {
        // Add to selection (remove 'all' if it was there)
        const newSelection = [...selectedNodeIds.filter(id => id !== 'all'), nodeId];
        onChange(newSelection);
      }
    }
  };

  const getDisplayText = () => {
    if (selectedNodeIds.includes('all') || selectedNodeIds.length === 0) {
      return 'All Nodes';
    }
    if (selectedNodeIds.length === 1) {
      const node = nodes?.find(n => n.id === selectedNodeIds[0]);
      return node?.name || 'Unknown Node';
    }
    return `${selectedNodeIds.length} nodes selected`;
  };

  const isSelected = (nodeId: string) => {
    if (nodeId === 'all') {
      return selectedNodeIds.includes('all') || selectedNodeIds.length === 0;
    }
    return selectedNodeIds.includes(nodeId);
  };

  const onlineNodes = nodes?.filter(n => n.status === 'online') || [];

  return (
    <div className={`relative ${className}`} ref={dropdownRef}>
      <Button
        ref={buttonRef}
        variant="outline"
        onClick={() => setIsOpen(!isOpen)}
        className={`w-full ${collapsed ? 'justify-center px-2' : 'justify-between'}`}
        title={collapsed ? getDisplayText() : undefined}
      >
        {collapsed ? (
          <Filter className="h-4 w-4 flex-shrink-0" />
        ) : (
          <>
            <span className="flex items-center gap-2 min-w-0 flex-1">
              <Filter className="h-4 w-4 flex-shrink-0" />
              <span className="truncate">{getDisplayText()}</span>
            </span>
            <ChevronDown className={`h-4 w-4 flex-shrink-0 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
          </>
        )}
      </Button>

      {isOpen && dropdownPosition && (
        <div
          className="fixed z-[60] bg-popover border border-border rounded-lg shadow-lg max-h-64 overflow-y-auto"
          style={{
            top: `${dropdownPosition.top}px`,
            left: `${dropdownPosition.left}px`,
            width: `${dropdownPosition.width}px`,
          }}
        >
          {/* All Nodes Option */}
          {multiSelect && (
            <div
              className="flex items-center gap-2 px-3 py-2 hover:bg-accent cursor-pointer"
              onClick={() => handleToggle('all')}
            >
              <div className="flex h-4 w-4 flex-shrink-0 items-center justify-center border rounded">
                {isSelected('all') && <Check className="h-3 w-3" />}
              </div>
              <span className="flex-1 text-sm font-medium truncate">All Nodes</span>
              {nodes && (
                <Badge variant="secondary" className="text-xs flex-shrink-0">{nodes.length}</Badge>
              )}
            </div>
          )}

          {/* Divider */}
          {multiSelect && nodes && nodes.length > 0 && (
            <div className="border-t border-border my-1" />
          )}

          {/* Individual Nodes */}
          {onlineNodes.length > 0 ? (
            onlineNodes.map((node) => (
              <div
                key={node.id}
                className="flex items-center gap-2 px-3 py-2 hover:bg-accent cursor-pointer"
                onClick={() => handleToggle(node.id)}
              >
                {multiSelect && (
                  <div className="flex h-4 w-4 flex-shrink-0 items-center justify-center border rounded">
                    {isSelected(node.id) && <Check className="h-3 w-3" />}
                  </div>
                )}
                <Server className="h-4 w-4 flex-shrink-0 text-muted-foreground" />
                <span className="flex-1 text-sm truncate" title={node.name}>{node.name}</span>
                {node.is_primary && (
                  <Badge variant="default" className="text-xs bg-blue-600 flex-shrink-0">Primary</Badge>
                )}
                {!multiSelect && isSelected(node.id) && (
                  <Check className="h-4 w-4 flex-shrink-0 text-primary" />
                )}
              </div>
            ))
          ) : (
            <div className="px-3 py-4 text-sm text-muted-foreground text-center">
              No online nodes available
            </div>
          )}
        </div>
      )}
    </div>
  );
}
