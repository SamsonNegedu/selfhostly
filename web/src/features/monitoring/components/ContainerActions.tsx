import { useState } from 'react';
import { RotateCw, Square } from 'lucide-react';
import { Button } from '@/shared/components/ui/button';
import { useRestartContainer, useStopContainer } from '@/shared/services/api';
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog';
import { useToast } from '@/shared/components/ui/Toast';

interface ContainerActionsProps {
  containerId: string;
  containerName: string;
  containerState: string;
}

function ContainerActions({ containerId, containerName, containerState }: ContainerActionsProps) {
  const [restartDialogOpen, setRestartDialogOpen] = useState(false);
  const [stopDialogOpen, setStopDialogOpen] = useState(false);
  const { toast } = useToast();

  const restartMutation = useRestartContainer();
  const stopMutation = useStopContainer();

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

  const isRunning = containerState === 'running';
  const isStopped = containerState === 'stopped';

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
    </div>
  );
}

export default ContainerActions;
