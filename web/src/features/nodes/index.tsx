import { useNavigate } from 'react-router-dom';
import { useState } from 'react';
import { useNodes, useDeleteNode, useCurrentNode } from '../../shared/services/api';
import { Button } from '../../shared/components/ui/button';
import { Card } from '../../shared/components/ui/card';
import { Badge } from '../../shared/components/ui/badge';
import ConfirmationDialog from '../../shared/components/ui/ConfirmationDialog';
import { Server, Plus, Trash2 } from 'lucide-react';

export function NodesPage() {
  const navigate = useNavigate();
  const { data: nodes, isLoading } = useNodes();
  const { data: currentNode } = useCurrentNode();
  const deleteMutation = useDeleteNode();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [nodeToDelete, setNodeToDelete] = useState<{ id: string; name: string } | null>(null);

  const handleDeleteClick = (id: string, name: string) => {
    setNodeToDelete({ id, name });
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (!nodeToDelete) return;
    try {
      await deleteMutation.mutateAsync(nodeToDelete.id);
      setDeleteDialogOpen(false);
      setNodeToDelete(null);
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
      {nodes && nodes.length > 0 ? (
        <>
          {/* Mobile Card View */}
          <div className="block sm:hidden space-y-3">
            {nodes.map((node) => (
              <Card key={node.id} className="overflow-hidden">
                <div className="p-4 space-y-3">
                  {/* Header - Name */}
                  <div className="flex items-center gap-2">
                    <Server className="h-5 w-5 text-muted-foreground flex-shrink-0" />
                    <h3 className="font-semibold text-base truncate flex-1">{node.name}</h3>
                  </div>
                  
                  {/* Badges Row */}
                  <div className="flex items-center gap-2 flex-wrap">
                    {getStatusBadge(node.status)}
                    {node.is_primary ? (
                      <Badge variant="default" className="bg-blue-600">Primary</Badge>
                    ) : (
                      <Badge variant="secondary">Secondary</Badge>
                    )}
                  </div>
                  
                  {/* Details Section */}
                  <div className="space-y-2 text-sm">
                    <div>
                      <span className="text-muted-foreground text-xs">API Endpoint</span>
                      <p className="text-xs mt-0.5 break-all font-mono">{node.api_endpoint}</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground text-xs">Last Seen</span>
                      <p className="text-xs mt-0.5">{node.last_seen ? new Date(node.last_seen).toLocaleString() : 'Never'}</p>
                    </div>
                  </div>
                </div>
                
                {/* Delete Button - Only for secondary nodes */}
                {!node.is_primary && (
                  <div className="border-t bg-muted/30 px-4 py-3">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDeleteClick(node.id, node.name)}
                      className="w-full text-destructive hover:text-destructive hover:bg-destructive/10 h-9"
                    >
                      <Trash2 className="h-4 w-4 mr-2" />
                      Remove Node
                    </Button>
                  </div>
                )}
              </Card>
            ))}
          </div>
          
          {/* Desktop Table View */}
          <Card className="hidden sm:block">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="border-b">
                  <tr>
                    <th className="text-left p-4 font-medium text-sm">Name</th>
                    <th className="text-left p-4 font-medium text-sm">API Endpoint</th>
                    <th className="text-left p-4 font-medium text-sm">Status</th>
                    <th className="text-left p-4 font-medium text-sm">Role</th>
                    <th className="text-left p-4 font-medium text-sm">Last Seen</th>
                    <th className="text-right p-4 font-medium text-sm">Actions</th>
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
                            onClick={() => handleDeleteClick(node.id, node.name)}
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
        </>
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

      <ConfirmationDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title="Delete Node"
        description={`Are you sure you want to delete "${nodeToDelete?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        cancelText="Cancel"
        onConfirm={handleConfirmDelete}
        isLoading={deleteMutation.isPending}
        variant="destructive"
      />
    </div>
  );
}

export default NodesPage;
