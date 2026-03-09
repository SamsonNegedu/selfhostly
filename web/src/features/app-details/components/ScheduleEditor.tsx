import { useState, useEffect } from 'react';
import { Calendar, Save, Trash2, Power, AlertCircle, Clock, PlayCircle, StopCircle, Info } from 'lucide-react';
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import ConfirmationDialog from '@/shared/components/ui/ConfirmationDialog';
import { TimezoneSelector, CronBuilder } from '@/features/schedules/components';
import { useAppSchedule, useUpdateAppSchedule, useDeleteAppSchedule, useTestSchedule } from '@/shared/services/api';
import { useToast } from '@/shared/components/ui/Toast';
import { ScheduleNextRuns } from '@/shared/types/api';

interface ScheduleEditorProps {
  appId: string;
  nodeId: string;
}

export function ScheduleEditor({ appId, nodeId }: ScheduleEditorProps) {
  const { data: schedule, isLoading } = useAppSchedule(appId, nodeId);
  const updateScheduleMutation = useUpdateAppSchedule(appId, nodeId);
  const deleteScheduleMutation = useDeleteAppSchedule(appId, nodeId);
  const testScheduleMutation = useTestSchedule();
  const { toast } = useToast();

  const [formData, setFormData] = useState({
    enabled: false,
    start_cron: '',
    stop_cron: '',
    timezone: 'UTC',
  });

  const [nextRuns, setNextRuns] = useState<ScheduleNextRuns | null>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [errors, setErrors] = useState<{ start?: string; stop?: string }>({});

  // Initialize form data when schedule is loaded
  useEffect(() => {
    if (schedule) {
      setFormData({
        enabled: schedule.enabled,
        start_cron: schedule.start_cron || '',
        stop_cron: schedule.stop_cron || '',
        timezone: schedule.timezone || 'UTC',
      });
    }
  }, [schedule]);

  // Test schedule expressions when they change
  useEffect(() => {
    if (formData.enabled && (formData.start_cron || formData.stop_cron)) {
      testScheduleMutation.mutate({
        ...formData,
        app_id: appId,
        node_id: nodeId,
      }, {
        onSuccess: (data) => {
          setNextRuns(data);
          setErrors({});
        },
        onError: (error: any) => {
          const errorMsg = error.response?.data?.error || error.message;
          if (errorMsg.includes('must occur after')) {
            setErrors({ stop: 'Stop time must be after start time' });
          } else if (errorMsg.includes('cannot be the same')) {
            setErrors({ stop: 'Start and stop schedules cannot be the same' });
          } else if (errorMsg.includes('start')) {
            setErrors({ start: errorMsg });
          } else if (errorMsg.includes('stop')) {
            setErrors({ stop: errorMsg });
          } else {
            setErrors({ start: errorMsg });
          }
        },
      });
    } else {
      setNextRuns(null);
      setErrors({});
    }
  }, [formData.enabled, formData.start_cron, formData.stop_cron, formData.timezone, appId, nodeId]);

  const handleSave = () => {
    updateScheduleMutation.mutate(formData, {
      onSuccess: () => {
        toast.success('Schedule saved', 'Application schedule has been updated successfully');
      },
      onError: (error: any) => {
        toast.error('Failed to save schedule', error.response?.data?.error || error.message);
      },
    });
  };

  const handleDelete = () => {
    deleteScheduleMutation.mutate(undefined, {
      onSuccess: () => {
        toast.success('Schedule deleted', 'Application schedule has been removed');
        setShowDeleteConfirm(false);
      },
      onError: (error: any) => {
        toast.error('Failed to delete schedule', error.response?.data?.error || error.message);
      },
    });
  };

  const handleCreateSchedule = () => {
    setFormData({
      enabled: true,
      start_cron: '',
      stop_cron: '',
      timezone: formData.timezone || 'UTC',
    });
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center p-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
        </CardContent>
      </Card>
    );
  }

  const hasSchedule = schedule?.id;
  const hasSameSchedules = formData.start_cron && formData.stop_cron && formData.start_cron === formData.stop_cron;
  const hasValidExpressions = (formData.start_cron || formData.stop_cron) && !errors.start && !errors.stop && !hasSameSchedules;
  const canSave = formData.enabled && hasValidExpressions;
  const isScheduleEnabled = schedule?.enabled ?? false;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Clock className="h-5 w-5 text-muted-foreground" />
            <div>
              <CardTitle>Application Schedule</CardTitle>
              <p className="text-sm text-muted-foreground mt-1">
                Automatically start and stop this application on a schedule
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {hasSchedule && (
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium bg-muted">
                <Power className={`h-4 w-4 ${isScheduleEnabled ? 'text-green-600 dark:text-green-400' : 'text-muted-foreground'}`} />
                {isScheduleEnabled ? 'Enabled' : 'Disabled'}
              </div>
            )}
            {hasSchedule && (
              <Button
                variant="destructive"
                size="sm"
                onClick={() => setShowDeleteConfirm(true)}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-6">
        {!hasSchedule && !formData.enabled ? (
          <div className="flex flex-col items-center justify-center py-12 px-4 text-center border-2 border-dashed rounded-lg">
            <div className="rounded-full bg-muted p-3 mb-4">
              <Calendar className="h-6 w-6 text-muted-foreground" />
            </div>
            <h3 className="text-lg font-semibold mb-2">No schedule configured</h3>
            <p className="text-sm text-muted-foreground max-w-md mb-4">
              Create a schedule to automatically start and stop this application based on cron expressions.
              Perfect for development environments, cost optimization, or maintenance windows.
            </p>
            <Button onClick={handleCreateSchedule}>
              <Power className="h-4 w-4 mr-2" />
              Create Schedule
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            {/* Start and Stop Schedules - Side by Side */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {/* Start Schedule */}
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <PlayCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                  <label className="text-sm font-medium">Start Schedule</label>
                </div>
                <CronBuilder
                  value={formData.start_cron}
                  onChange={(value) => setFormData({ ...formData, start_cron: value })}
                  error={errors.start}
                  placeholder="e.g., 0 9 * * 1-5"
                />
                {errors.start && (
                  <div className="flex items-center gap-2 text-xs text-destructive bg-destructive/10 p-2 rounded-md">
                    <AlertCircle className="h-3.5 w-3.5 flex-shrink-0" />
                    {errors.start}
                  </div>
                )}
              </div>

              {/* Stop Schedule */}
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <StopCircle className="h-4 w-4 text-red-600 dark:text-red-400" />
                  <label className="text-sm font-medium">Stop Schedule</label>
                </div>
                <CronBuilder
                  value={formData.stop_cron}
                  onChange={(value) => setFormData({ ...formData, stop_cron: value })}
                  error={errors.stop}
                  placeholder="e.g., 0 18 * * 1-5"
                />
                {errors.stop && (
                  <div className="flex items-center gap-2 text-xs text-destructive bg-destructive/10 p-2 rounded-md">
                    <AlertCircle className="h-3.5 w-3.5 flex-shrink-0" />
                    {errors.stop}
                  </div>
                )}
              </div>
            </div>

            {/* Timezone and Next Runs - Side by Side */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {/* Timezone */}
              <div className="space-y-2">
                <label className="text-sm font-medium">Timezone</label>
                <TimezoneSelector
                  value={formData.timezone}
                  onChange={(timezone) => setFormData({ ...formData, timezone })}
                />
              </div>

            </div>
            {/* Next Runs Preview */}
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <Info className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">Next Scheduled Runs</span>
              </div>
              {nextRuns && (nextRuns.next_start || nextRuns.next_stop) ? (
                <div className="space-y-1.5 p-3 bg-muted rounded-lg">
                  {nextRuns.next_start && (
                    <div className="flex items-center gap-2 text-xs">
                      <PlayCircle className="h-3.5 w-3.5 text-green-600 dark:text-green-400 flex-shrink-0" />
                      <span className="text-muted-foreground">Start:</span>
                      <span className="font-medium">
                        {new Date(nextRuns.next_start).toLocaleString(undefined, {
                          month: 'short',
                          day: 'numeric',
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </span>
                    </div>
                  )}
                  {nextRuns.next_stop && (
                    <div className="flex items-center gap-2 text-xs">
                      <StopCircle className="h-3.5 w-3.5 text-red-600 dark:text-red-400 flex-shrink-0" />
                      <span className="text-muted-foreground">Stop:</span>
                      <span className="font-medium">
                        {new Date(nextRuns.next_stop).toLocaleString(undefined, {
                          month: 'short',
                          day: 'numeric',
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </span>
                    </div>
                  )}
                </div>
              ) : (!formData.start_cron && !formData.stop_cron) ? (
                <div className="p-3 bg-muted rounded-lg">
                  <p className="text-xs text-muted-foreground">
                    Configure a schedule to see next runs
                  </p>
                </div>
              ) : null}
            </div>

            {/* Validation warning */}
            {formData.enabled && !hasValidExpressions && (
              <div className="flex items-center gap-2 text-sm text-destructive bg-destructive/10 p-3 rounded-md">
                <AlertCircle className="h-4 w-4 flex-shrink-0" />
                <span>
                  {!formData.start_cron && !formData.stop_cron
                    ? 'Please configure at least one schedule (start or stop)'
                    : hasSameSchedules
                      ? 'Start and stop schedules cannot be the same time'
                      : 'Please fix the errors in your cron expressions'}
                </span>
              </div>
            )}

            {/* Save Button */}
            <div className="flex justify-end pt-2">
              <Button
                onClick={handleSave}
                disabled={updateScheduleMutation.isPending || !canSave}
              >
                {updateScheduleMutation.isPending ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-current mr-2" />
                    Saving...
                  </>
                ) : (
                  <>
                    <Save className="h-4 w-4 mr-2" />
                    Save Schedule
                  </>
                )}
              </Button>
            </div>
          </div>
        )}
      </CardContent>

      {/* Delete Confirmation Dialog */}
      <ConfirmationDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        onConfirm={handleDelete}
        title="Delete Schedule"
        description="Are you sure you want to delete this schedule? The application will no longer start or stop automatically."
        confirmText="Delete Schedule"
        isLoading={deleteScheduleMutation.isPending}
        variant="destructive"
      />
    </Card>
  );
}
