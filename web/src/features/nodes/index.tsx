import { useNavigate } from 'react-router-dom';
import { useNodes, useCurrentNode } from '../../shared/services/api';
import { Button } from '../../shared/components/ui/Button';
import { Badge } from '../../shared/components/ui/Badge';
import { Server, Plus } from 'lucide-react';
import NodesListView from './components/NodesListView';

export function NodesPage() {
  const navigate = useNavigate();
  const { data: nodes, isLoading } = useNodes();
  const { data: currentNode } = useCurrentNode();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <div className="fade-in space-y-4 sm:space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3 sm:gap-0">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold">Nodes</h1>
          <p className="text-muted-foreground mt-1 sm:mt-2 text-sm sm:text-base">
            Manage nodes in your cluster
          </p>
        </div>
        <Button onClick={() => navigate('/nodes/new')} className="w-full sm:w-auto">
          <Plus className="h-4 w-4 mr-2" />
          Register Node
        </Button>
      </div>

      {/* Current Node Info */}
      {currentNode && (
        <div className="flex items-start gap-3 p-4 rounded-lg bg-muted/50 border-2">
          <Server className="h-5 w-5 text-primary flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <div className="text-sm font-medium mb-1 flex items-center gap-2">
              <span>Current Node: {currentNode.name}</span>
              {currentNode.is_primary && (
                <Badge variant="default" className="bg-blue-600">Primary</Badge>
              )}
            </div>
            <p className="text-sm text-muted-foreground">
              Running on {currentNode.api_endpoint} • {nodes?.length || 0} node{nodes?.length !== 1 ? 's' : ''} in cluster
            </p>
          </div>
        </div>
      )}

      {/* Nodes List */}
      <NodesListView nodes={nodes || []} />
    </div>
  );
}

export default NodesPage;
