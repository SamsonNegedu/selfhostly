import { useState } from 'react';
import { RotateCw, Square, Trash2 } from 'lucide-react';
import { Button } from '@/shared/components/ui/button';
import { useRestartContainer, useStopContainer, useDeleteContainer } from '@/shared/services/api';
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog';
import { useToast } from '@/shared/components/ui/Toast';

interface ContainerActionsProps {
  containerId: string;
  containerName: string;
  containerState: string;
  appName: string;
}

function ContainerActions({ containerId, containerName, containerState, appName }: ContainerActionsProps) {
  const [restartDialogOpen, setRestartDialogOpen] = useState(false);
  const [stopDialogOpen, setStopDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const { toast } = useToast();

  const restartMutation = useRestartContainer();
  const stopMutation = useStopContainer();
  const deleteMutation = useDeleteContainer();

  const handleRestart = async () => {
    try {
      await restartMutation.mutateAsync(containerId);
      const action = isRunning ? 'restarted' : 'started';
      toast.success(`Container "${containerName}" ${action} successfully`);
      setRestartDialogOpen(false);
    } catch (error) {
      toast.error(`Failed to restart container: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const handleStop = async () => {
    try {
      await stopMutation.mutateAsync(containerId);
      toast.success(`Container "${containerName}" stopped successfully`);
      setStopDialogOpen(false);
    } catch (error) {
      toast.error(`Failed to stop container: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync(containerId);
      toast.success(`Container "${containerName}" deleted successfully`);
      setDeleteDialogOpen(false);
    } catch (error) {
      toast.error(`Failed to delete container: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const isRunning = containerState === 'running';
  const isStopped = containerState === 'stopped';
  const isUnmanaged = appName === 'unmanaged';

  return (
    <div className="flex items-center gap-2">
      {/* Restart/Start Button */}
      <Button
        variant="outline"
        size="sm"
        onClick={() => setRestartDialogOpen(true)}
        disabled={restartMutation.isPending}
        title={isStopped ? 'Start container' : 'Restart container'}
      >
        <RotateCw className={`h-4 w-4 ${restartMutation.isPending ? 'animate-spin' : ''}`} />
      </Button>

      {/* Stop Button */}
      <Button
        variant="outline"
        size="sm"
        onClick={() => setStopDialogOpen(true)}
        disabled={stopMutation.isPending || !isRunning}
        title="Stop container"
      >
        <Square className="h-4 w-4" />
      </Button>

      {/* Delete Button - Only show for unmanaged containers */}
      {isUnmanaged && (
        <Button
          variant="outline"
          size="sm"
          onClick={() => setDeleteDialogOpen(true)}
          disabled={deleteMutation.isPending}
          title="Delete unmanaged container"
          className="text-destructive hover:text-destructive"
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      )}

      {/* Restart/Start Confirmation Dialog */}
      <ConfirmationDialog
        open={restartDialogOpen}
        onOpenChange={setRestartDialogOpen}
        title={isStopped ? 'Start Container' : 'Restart Container'}
        description={
          isStopped
            ? `Are you sure you want to start "${containerName}"?`
            : `Are you sure you want to restart "${containerName}"? This will cause a brief service interruption.`
        }
        confirmText={isStopped ? 'Start' : 'Restart'}
        onConfirm={handleRestart}
        variant="default"
      />

      {/* Stop Confirmation Dialog */}
      <ConfirmationDialog
        open={stopDialogOpen}
        onOpenChange={setStopDialogOpen}
        title="Stop Container"
        description={`Are you sure you want to stop "${containerName}"? The container will remain stopped until manually started again.`}
        confirmText="Stop"
        onConfirm={handleStop}
        variant="destructive"
      />

      {/* Delete Confirmation Dialog */}
      <ConfirmationDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title="Delete Unmanaged Container"
        description={`Are you sure you want to permanently delete "${containerName}"? This action cannot be undone and will remove the container and any data stored in it (volumes may persist depending on configuration).`}
        confirmText="Delete Container"
        onConfirm={handleDelete}
        variant="destructive"
        isLoading={deleteMutation.isPending}
      />
    </div>
  );
}

export default ContainerActions;
