import { useNavigate } from 'react-router-dom';
import { useNodes, useDeleteNode, useCurrentNode } from '../../shared/services/api';
import { Button } from '../../shared/components/ui/button';
import { Card } from '../../shared/components/ui/card';
import { Badge } from '../../shared/components/ui/badge';
import { Server, Plus, Trash2 } from 'lucide-react';

export function NodesPage() {
  const navigate = useNavigate();
  const { data: nodes, isLoading } = useNodes();
  const { data: currentNode } = useCurrentNode();
  const deleteMutation = useDeleteNode();

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this node?')) return;
    try {
      await deleteMutation.mutateAsync(id);
    } catch (error) {
      console.error('Failed to delete node:', error);
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'online':
        return <Badge variant="default" className="bg-green-500">Online</Badge>;
      case 'offline':
        return <Badge variant="destructive">Offline</Badge>;
      case 'unreachable':
        return <Badge variant="secondary" className="bg-yellow-500">Unreachable</Badge>;
      default:
        return <Badge variant="secondary">{status}</Badge>;
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <div className="fade-in space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">Nodes</h1>
          <p className="text-muted-foreground mt-2">
            Manage nodes in your cluster
          </p>
        </div>
        <Button onClick={() => navigate('/nodes/new')}>
          <Plus className="h-4 w-4 mr-2" />
          Register Node
        </Button>
      </div>

      {/* Current Node Info */}
      {currentNode && (
        <div className="flex items-start gap-3 p-4 rounded-lg bg-muted/50 border-2">
          <Server className="h-5 w-5 text-primary flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <p className="text-sm font-medium mb-1">
              Current Node: {currentNode.name}
              {currentNode.is_primary && (
                <Badge variant="default" className="bg-blue-600 ml-2">Primary</Badge>
              )}
            </p>
            <p className="text-sm text-muted-foreground">
              Running on {currentNode.api_endpoint} • {nodes?.length || 0} node{nodes?.length !== 1 ? 's' : ''} in cluster
            </p>
          </div>
        </div>
      )}

      {/* Nodes List */}
      {nodes && nodes.length > 0 ? (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="border-b">
                <tr>
                  <th className="text-left p-4 font-medium">Name</th>
                  <th className="text-left p-4 font-medium">API Endpoint</th>
                  <th className="text-left p-4 font-medium">Status</th>
                  <th className="text-left p-4 font-medium">Role</th>
                  <th className="text-left p-4 font-medium">Last Seen</th>
                  <th className="text-right p-4 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {nodes.map((node) => (
                  <tr key={node.id} className="border-b last:border-b-0 hover:bg-muted/50">
                    <td className="p-4">
                      <div className="flex items-center gap-2">
                        <Server className="h-4 w-4 text-muted-foreground" />
                        <span className="font-medium">{node.name}</span>
                      </div>
                    </td>
                    <td className="p-4 text-sm text-muted-foreground">
                      {node.api_endpoint}
                    </td>
                    <td className="p-4">
                      {getStatusBadge(node.status)}
                    </td>
                    <td className="p-4">
                      {node.is_primary ? (
                        <Badge variant="default" className="bg-blue-600">Primary</Badge>
                      ) : (
                        <Badge variant="secondary">Secondary</Badge>
                      )}
                    </td>
                    <td className="p-4 text-sm text-muted-foreground">
                      {node.last_seen ? new Date(node.last_seen).toLocaleString() : 'Never'}
                    </td>
                    <td className="p-4">
                      <div className="flex gap-2 justify-end">
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={node.is_primary}
                          onClick={() => handleDelete(node.id)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      ) : (
        <Card className="p-12 text-center">
          <Server className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium mb-2">No nodes registered</h3>
          <p className="text-muted-foreground mb-4">
            Register your first secondary node to enable multi-node deployment
          </p>
          <Button onClick={() => navigate('/nodes/new')}>
            <Plus className="h-4 w-4 mr-2" />
            Register Node
          </Button>
        </Card>
      )}
    </div>
  );
}

export default NodesPage;
